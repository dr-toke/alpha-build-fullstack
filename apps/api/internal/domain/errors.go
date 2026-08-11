package domain

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Sentinel errors, one per stable machine error code in
// 02-FRONTEND-CONTRACT.md §3 ("not_found, moved, rate_limited, unavailable,
// invalid_filter, auth_required, auth_invalid, banned, validation_failed").
// internal/store returns these (wrapped with fmt.Errorf("<context>: %w", err)
// per 08-BUILD-ORDERS.md §4's convention); internal/api/errors.go (M5.9) maps
// each one to its HTTP status and JSON error code — this file is the only
// place the mapping's source vocabulary is defined, so the two layers can
// never drift out of sync with each other.
var (
	// ErrNotFound -> 404, code "not_found".
	ErrNotFound = errors.New("not found")

	// ErrValidationFailed -> 400, code "validation_failed".
	ErrValidationFailed = errors.New("validation failed")

	// ErrInvalidFilter -> 400, code "invalid_filter".
	ErrInvalidFilter = errors.New("invalid filter")

	// ErrAuthRequired -> 401, code "auth_required".
	ErrAuthRequired = errors.New("authentication required")

	// ErrAuthInvalid -> 401, code "auth_invalid".
	ErrAuthInvalid = errors.New("invalid credentials")

	// ErrBanned -> 403, code "banned".
	ErrBanned = errors.New("account is banned")

	// ErrRateLimited -> 429, code "rate_limited". Callers attach Retry-After
	// themselves; this sentinel only signals which envelope to write.
	ErrRateLimited = errors.New("rate limited")

	// ErrUnavailable -> 503, code "unavailable". 02-FRONTEND-CONTRACT.md §4:
	// this must never be conflated with an empty result set — "nothing
	// matches your filters" is a 200 with an empty data array, not this.
	ErrUnavailable = errors.New("service unavailable")

	// ErrDuplicateHandle -> 400, code "validation_failed". Distinct sentinel
	// (rather than reusing ErrValidationFailed directly) so the registration
	// handler can give a specific message without string-matching.
	ErrDuplicateHandle = errors.New("handle already taken")

	// ErrPurchaseTokenAlreadyClaimed -> 400, code "validation_failed".
	// Distinct from ErrNotFound: the token exists and hashed-matched, but
	// claimed_by is already set — a different failure than "no such token".
	ErrPurchaseTokenAlreadyClaimed = errors.New("purchase token already claimed")
)

// ClusterMovedError signals that a requested cluster ID was merged into
// another. GET /api/products/{old} must return 200 with {"moved_to": new_id},
// never a 404 — 03-DOMAIN-MODEL.md §4, 02-FRONTEND-CONTRACT.md §4. Carries
// the target ID so internal/api can build that body without a second query.
type ClusterMovedError struct {
	OldID uuid.UUID
	NewID uuid.UUID
}

func (e *ClusterMovedError) Error() string {
	return fmt.Sprintf("cluster %s moved to %s", e.OldID, e.NewID)
}

// Is lets errors.Is(err, ErrNotFound) report false for a moved cluster even
// though both are "the ID you asked for isn't directly there" — status
// semantics are load-bearing (02-FRONTEND-CONTRACT.md §4) and a moved cluster
// is never a 404. No Is method is defined against ErrNotFound on purpose:
// the two must stay distinguishable by errors.As, not conflate under Is.
func (e *ClusterMovedError) Unwrap() error { return nil }
