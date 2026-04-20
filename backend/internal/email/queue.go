package email

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/foodbi/backend/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Enqueuer is the interface the auth service depends on. Kept small for
// testability — tests pass a fake that records calls.
type Enqueuer interface {
	EnqueueOTP(ctx context.Context, tx pgx.Tx, user *models.User, code string) error
	EnqueuePasswordReset(ctx context.Context, tx pgx.Tx, user *models.User, token, resetURL string) error
	EnqueueInvite(ctx context.Context, tx pgx.Tx, companyID uuid.UUID, toEmail, role, companyName, lang, acceptURL string) error
}

// DefaultEnqueuer wraps the package-level Enqueue call so that *Client or a
// noop value can be used as an Enqueuer. The Client itself isn't strictly
// required for enqueue (enqueue only writes to the DB), but we attach the
// methods to *Client so main.go has a single object to wire.
func (c *Client) EnqueueOTP(ctx context.Context, tx pgx.Tx, user *models.User, code string) error {
	return EnqueueOTP(ctx, tx, user, code)
}

func (c *Client) EnqueuePasswordReset(ctx context.Context, tx pgx.Tx, user *models.User, token, resetURL string) error {
	return EnqueuePasswordReset(ctx, tx, user, token, resetURL)
}

func (c *Client) EnqueueInvite(ctx context.Context, tx pgx.Tx, companyID uuid.UUID, toEmail, role, companyName, lang, acceptURL string) error {
	return EnqueueInvite(ctx, tx, companyID, toEmail, role, companyName, lang, acceptURL)
}

// EnqueueInput is the raw shape for inserting into email_outbox.
type EnqueueInput struct {
	CompanyID   uuid.UUID
	UserID      *uuid.UUID
	Type        string
	ToEmail     string
	TemplateKey string
	Lang        string
	Params      map[string]any
}

// Enqueue INSERTs a row into email_outbox using the caller's transaction.
// It MUST NOT open its own tx — the contract is that user-creation and email
// enqueue commit atomically.
func Enqueue(ctx context.Context, tx pgx.Tx, in EnqueueInput) error {
	if tx == nil {
		return fmt.Errorf("email.Enqueue: tx is required")
	}
	if in.ToEmail == "" {
		return fmt.Errorf("email.Enqueue: to_email required")
	}
	if in.Type == "" || in.TemplateKey == "" {
		return fmt.Errorf("email.Enqueue: type and template_key required")
	}

	// Store lang inside params so the processor can render without another
	// DB lookup. Keep an explicit "_lang" key separate from user-facing params
	// to avoid collisions.
	if in.Params == nil {
		in.Params = map[string]any{}
	}
	in.Params["_lang"] = normalizeLang(in.Lang)

	paramsJSON, err := json.Marshal(in.Params)
	if err != nil {
		return fmt.Errorf("email.Enqueue: marshal params: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO email_outbox (company_id, user_id, type, to_email, template_key, params)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		in.CompanyID, in.UserID, in.Type, in.ToEmail, in.TemplateKey, paramsJSON)
	if err != nil {
		return fmt.Errorf("email.Enqueue: insert outbox: %w", err)
	}
	return nil
}

// EnqueueOTP enqueues a verification-code email for a newly registered user.
func EnqueueOTP(ctx context.Context, tx pgx.Tx, user *models.User, code string) error {
	if user == nil {
		return fmt.Errorf("EnqueueOTP: user is nil")
	}
	uid := user.ID
	return Enqueue(ctx, tx, EnqueueInput{
		CompanyID:   user.CompanyID,
		UserID:      &uid,
		Type:        TemplateOTP,
		ToEmail:     user.Email,
		TemplateKey: TemplateOTP,
		Lang:        DefaultLanguage, // users row doesn't carry lang yet at register time; default ru
		Params: map[string]any{
			"FirstName": user.FirstName,
			"Code":      code,
		},
	})
}

// EnqueuePasswordReset enqueues a reset-link email.
func EnqueuePasswordReset(ctx context.Context, tx pgx.Tx, user *models.User, token, resetURL string) error {
	if user == nil {
		return fmt.Errorf("EnqueuePasswordReset: user is nil")
	}
	uid := user.ID
	return Enqueue(ctx, tx, EnqueueInput{
		CompanyID:   user.CompanyID,
		UserID:      &uid,
		Type:        TemplatePasswordReset,
		ToEmail:     user.Email,
		TemplateKey: TemplatePasswordReset,
		Lang:        DefaultLanguage,
		Params: map[string]any{
			"FirstName": user.FirstName,
			"Token":     token,
			"ResetURL":  resetURL,
		},
	})
}

// EnqueueInvite enqueues an invite acceptance email.
func EnqueueInvite(ctx context.Context, tx pgx.Tx, companyID uuid.UUID, toEmail, role, companyName, lang, acceptURL string) error {
	return Enqueue(ctx, tx, EnqueueInput{
		CompanyID:   companyID,
		UserID:      nil,
		Type:        TemplateInvite,
		ToEmail:     toEmail,
		TemplateKey: TemplateInvite,
		Lang:        lang,
		Params: map[string]any{
			"CompanyName": companyName,
			"Role":        role,
			"AcceptURL":   acceptURL,
		},
	})
}

// BackoffFor returns the next-attempt delay for a given attempt number.
// Attempts 1..4 → 1s, 4s, 16s, 64s (exponential, base 4). After attempt >= 4
// the processor marks the row failed.
func BackoffFor(attempt int) (seconds int, giveUp bool) {
	switch attempt {
	case 1:
		return 1, false
	case 2:
		return 4, false
	case 3:
		return 16, false
	case 4:
		return 64, false
	default:
		return 0, true
	}
}

// MaxAttempts is the retry ceiling before a row is marked 'failed'.
const MaxAttempts = 4
