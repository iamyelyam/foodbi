package email

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// RunProcessor is a blocking loop that drains email_outbox. Only one replica
// may hold the advisory lock — other replicas spin waiting for it. Cancel ctx
// to stop the loop; the advisory lock is released on exit.
//
// Design notes:
//   - pg_try_advisory_lock (session-level) is used, NOT xact-level — we want
//     the lock to persist across statements.
//   - Batch size 10 per tick; sleeps 2s between empty polls.
//   - Rate-limits sends to at most 10/sec (100ms sleep between sends).
//   - Backoff on retryable errors: 1s, 4s, 16s, 64s (see BackoffFor).
func RunProcessor(ctx context.Context, pool *pgxpool.Pool, client *Client, lockID int64) {
	for {
		if ctx.Err() != nil {
			return
		}
		if err := runOnce(ctx, pool, client, lockID); err != nil {
			log.Warn().Err(err).Msg("email.processor: iteration error")
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
		}
	}
}

// runOnce acquires the advisory lock, then loops polling the outbox until the
// context is cancelled or the DB connection drops. Returning nil means
// graceful shutdown; returning an error means the outer loop should retry.
func runOnce(ctx context.Context, pool *pgxpool.Pool, client *Client, lockID int64) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	var acquired bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&acquired); err != nil {
		return err
	}
	if !acquired {
		// Another replica holds the lock — sleep and retry in the outer loop.
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(5 * time.Second):
		}
		return nil
	}
	defer func() {
		// Release regardless of how we exit. Use a fresh context so a cancelled
		// ctx doesn't block the unlock.
		relCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = conn.Exec(relCtx, "SELECT pg_advisory_unlock($1)", lockID)
	}()

	if client.DryRun() {
		log.Warn().Msg("email.processor: RESEND_API_KEY not set — operating in dry-run mode (rows will be marked 'dry_run_skipped')")
	} else {
		log.Info().Msg("email.processor: started (advisory lock acquired)")
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}

		processed, err := drainBatch(ctx, pool, client)
		if err != nil {
			return err
		}
		// If we drained a full batch, loop again immediately without waiting.
		if processed >= 10 {
			// Tight loop — but respect rate limit + ctx.
			select {
			case <-ctx.Done():
				return nil
			default:
			}
		}
	}
}

type outboxRow struct {
	ID          int64
	CompanyID   string
	ToEmail     string
	TemplateKey string
	Params      []byte
	Attempts    int
}

// drainBatch pulls up to 10 pending/retrying rows and processes them. Returns
// the number of rows handled.
func drainBatch(ctx context.Context, pool *pgxpool.Pool, client *Client) (int, error) {
	// Use FOR UPDATE SKIP LOCKED so multiple goroutines (within this replica)
	// and other replicas (if advisory lock were ever released accidentally)
	// don't collide.
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}

	rows, err := tx.Query(ctx, `
		SELECT id, company_id::text, to_email, template_key, params, attempts
		FROM email_outbox
		WHERE status IN ('pending','retrying') AND next_attempt_at <= NOW()
		ORDER BY created_at
		LIMIT 10
		FOR UPDATE SKIP LOCKED`)
	if err != nil {
		_ = tx.Rollback(ctx)
		return 0, err
	}

	var batch []outboxRow
	for rows.Next() {
		var r outboxRow
		if err := rows.Scan(&r.ID, &r.CompanyID, &r.ToEmail, &r.TemplateKey, &r.Params, &r.Attempts); err != nil {
			rows.Close()
			_ = tx.Rollback(ctx)
			return 0, err
		}
		batch = append(batch, r)
	}
	rows.Close()

	// Mark picked rows as 'sending' and bump attempts within the same tx.
	for _, r := range batch {
		_, err := tx.Exec(ctx,
			`UPDATE email_outbox SET status='sending', attempts = attempts + 1 WHERE id = $1`,
			r.ID)
		if err != nil {
			_ = tx.Rollback(ctx)
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}

	// Send each row outside the tx so slow HTTP calls don't hold row locks.
	for _, r := range batch {
		if err := handleRow(ctx, pool, client, r); err != nil {
			log.Warn().Err(err).Int64("outbox_id", r.ID).Msg("email.processor: handleRow error")
		}
		// Rate limit: 10/sec ceiling.
		select {
		case <-ctx.Done():
			return len(batch), nil
		case <-time.After(100 * time.Millisecond):
		}
	}
	return len(batch), nil
}

func handleRow(ctx context.Context, pool *pgxpool.Pool, client *Client, r outboxRow) error {
	// Parse params + language.
	var params map[string]any
	if len(r.Params) > 0 {
		if err := json.Unmarshal(r.Params, &params); err != nil {
			return markFailed(ctx, pool, r.ID, "invalid params JSON: "+err.Error())
		}
	} else {
		params = map[string]any{}
	}
	lang, _ := params["_lang"].(string)

	subject, html, err := Render(r.TemplateKey, lang, params)
	if err != nil {
		return markFailed(ctx, pool, r.ID, "render: "+err.Error())
	}

	// Dry-run: skip Resend, mark special status so ops knows the email did not
	// physically leave the system.
	if client.DryRun() {
		_, exErr := pool.Exec(ctx,
			`UPDATE email_outbox SET status='dry_run_skipped', last_error=NULL WHERE id=$1`,
			r.ID)
		return exErr
	}

	sendErr := client.Send(ctx, r.ToEmail, subject, html)
	if sendErr == nil {
		_, exErr := pool.Exec(ctx,
			`UPDATE email_outbox SET status='sent', sent_at=NOW(), last_error=NULL WHERE id=$1`,
			r.ID)
		return exErr
	}

	if !IsRetryable(sendErr) {
		return markFailed(ctx, pool, r.ID, sendErr.Error())
	}

	// Retryable: schedule next attempt, or give up past MaxAttempts.
	secs, giveUp := BackoffFor(r.Attempts)
	if giveUp || r.Attempts >= MaxAttempts {
		return markFailed(ctx, pool, r.ID, sendErr.Error())
	}
	_, exErr := pool.Exec(ctx,
		`UPDATE email_outbox
		 SET status='retrying', next_attempt_at = NOW() + ($2 * INTERVAL '1 second'), last_error=$3
		 WHERE id=$1`,
		r.ID, secs, sendErr.Error())
	return exErr
}

func markFailed(ctx context.Context, pool *pgxpool.Pool, id int64, msg string) error {
	_, err := pool.Exec(ctx,
		`UPDATE email_outbox SET status='failed', last_error=$2 WHERE id=$1`,
		id, msg)
	return err
}
