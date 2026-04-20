// Package dblock is the central registry of PostgreSQL advisory-lock IDs used
// across the FoodBI backend. Each constant MUST be unique. When introducing a
// new cross-replica singleton (background worker, periodic job, migration
// coordinator, …), allocate the next free ID here and reference it at the
// single call-site that owns the lock.
//
// Registry:
//
//	42  EmailOutboxProcessor — drains email_outbox via Resend (Phase 6)
package dblock

// EmailOutboxProcessor is the advisory-lock ID used by the email outbox
// processor goroutine so that only one API replica drains email_outbox at a
// time. Session-level (pg_try_advisory_lock), NOT transactional.
const EmailOutboxProcessor int64 = 42
