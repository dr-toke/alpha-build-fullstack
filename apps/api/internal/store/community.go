package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ── Accounts ─────────────────────────────────────────────────────────────────

// CreateAccount inserts a new pseudonymous account. Returns
// domain.ErrDuplicateHandle (wrapped) on a handle collision — the caller
// (M5's /api/auth/register handler) needs to distinguish that from any
// other failure to give a useful error, per 05-API-REFERENCE.md §3's
// registration contract.
func (s *Store) CreateAccount(ctx context.Context, handle, passwordHash string) (*domain.Account, error) {
	const q = `
		INSERT INTO accounts (handle, password_hash) VALUES ($1, $2)
		RETURNING id, handle, password_hash, created_at, last_seen_at, banned, ban_reason`
	var a domain.Account
	err := s.Pool.QueryRow(ctx, q, handle, passwordHash).Scan(
		&a.ID, &a.Handle, &a.PasswordHash, &a.CreatedAt, &a.LastSeenAt, &a.Banned, &a.BanReason)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, fmt.Errorf("store.CreateAccount(%s): %w", handle, domain.ErrDuplicateHandle)
		}
		return nil, fmt.Errorf("store.CreateAccount: %w", err)
	}
	return &a, nil
}

// AccountByHandle looks up case-insensitively via handle_lower
// (internal/db/migrations/006's GENERATED column) — handles are
// case-preserved for display but must not allow two accounts differing
// only by case, per the UNIQUE INDEX on handle_lower.
func (s *Store) AccountByHandle(ctx context.Context, handle string) (*domain.Account, error) {
	const q = `
		SELECT id, handle, password_hash, created_at, last_seen_at, banned, ban_reason
		FROM accounts WHERE handle_lower = lower($1)`
	return s.scanAccount(s.Pool.QueryRow(ctx, q, handle))
}

// AccountByID is the JWT-authenticated lookup path (/api/auth/me).
func (s *Store) AccountByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	const q = `
		SELECT id, handle, password_hash, created_at, last_seen_at, banned, ban_reason
		FROM accounts WHERE id = $1`
	return s.scanAccount(s.Pool.QueryRow(ctx, q, id))
}

func (s *Store) scanAccount(row pgx.Row) (*domain.Account, error) {
	var a domain.Account
	err := row.Scan(&a.ID, &a.Handle, &a.PasswordHash, &a.CreatedAt, &a.LastSeenAt, &a.Banned, &a.BanReason)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store: account: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("store: account: %w", err)
	}
	return &a, nil
}

// TouchLastSeen bumps last_seen_at — called on any authenticated request,
// per the account row's own last_seen_at column existing at all
// (03-DOMAIN-MODEL.md §8).
func (s *Store) TouchLastSeen(ctx context.Context, accountID uuid.UUID) error {
	_, err := s.Pool.Exec(ctx, `UPDATE accounts SET last_seen_at = now() WHERE id = $1`, accountID)
	if err != nil {
		return fmt.Errorf("store.TouchLastSeen: %w", err)
	}
	return nil
}

// ── Refresh tokens ───────────────────────────────────────────────────────────

// CreateRefreshToken stores a token's HASH only — the raw token never
// touches the database, matching purchase_tokens' same discipline
// (05-API-REFERENCE.md §3).
func (s *Store) CreateRefreshToken(ctx context.Context, t domain.RefreshToken) error {
	const q = `INSERT INTO refresh_tokens (account_id, token_hash, expires_at) VALUES ($1,$2,$3)`
	_, err := s.Pool.Exec(ctx, q, t.AccountID, t.TokenHash, t.ExpiresAt)
	if err != nil {
		return fmt.Errorf("store.CreateRefreshToken: %w", err)
	}
	return nil
}

// RefreshTokenByHash looks up a non-revoked, non-expired token by its hash —
// ADR-007's rotation contract: revoke old, issue new pair on every refresh.
func (s *Store) RefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	const q = `
		SELECT id, account_id, token_hash, expires_at, created_at, revoked
		FROM refresh_tokens WHERE token_hash = $1 AND revoked = false AND expires_at > now()`
	var t domain.RefreshToken
	err := s.Pool.QueryRow(ctx, q, tokenHash).Scan(
		&t.ID, &t.AccountID, &t.TokenHash, &t.ExpiresAt, &t.CreatedAt, &t.Revoked)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store.RefreshTokenByHash: %w", domain.ErrAuthInvalid)
		}
		return nil, fmt.Errorf("store.RefreshTokenByHash: %w", err)
	}
	return &t, nil
}

// RevokeRefreshToken marks a token unusable — logout, and the "revoke old"
// half of every rotation.
func (s *Store) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE refresh_tokens SET revoked = true WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("store.RevokeRefreshToken: %w", err)
	}
	return nil
}

// ── Comments ─────────────────────────────────────────────────────────────────

// CreateComment inserts a review (ClusterID set) or forum reply (PostID
// set) — the DB CHECK (internal/db/migrations/006) enforces exactly one of
// the two is non-nil; this function doesn't duplicate that validation, it
// lets Postgres be the single source of truth for it.
func (s *Store) CreateComment(ctx context.Context, c domain.Comment) (*domain.Comment, error) {
	const q = `
		INSERT INTO comments (account_id, post_id, cluster_id, body, verified_purchase, purchase_token_id, rating)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id, account_id, post_id, cluster_id, body, verified_purchase,
		          purchase_token_id, rating, deleted, deleted_by_admin, created_at`
	var out domain.Comment
	err := s.Pool.QueryRow(ctx, q, c.AccountID, c.PostID, c.ClusterID, c.Body,
		c.VerifiedPurchase, c.PurchaseTokenID, c.Rating,
	).Scan(&out.ID, &out.AccountID, &out.PostID, &out.ClusterID, &out.Body, &out.VerifiedPurchase,
		&out.PurchaseTokenID, &out.Rating, &out.Deleted, &out.DeletedByAdmin, &out.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("store.CreateComment: %w", err)
	}
	return &out, nil
}

const commentSelectColumns = `
	SELECT id, account_id, post_id, cluster_id, body, verified_purchase,
	       purchase_token_id, rating, deleted, deleted_by_admin, created_at`

// CommentsForCluster returns non-deleted product reviews, newest first —
// 05-API-REFERENCE.md §3: "paginated, newest first."
func (s *Store) CommentsForCluster(ctx context.Context, clusterID uuid.UUID, limit, offset int) ([]domain.Comment, error) {
	q := commentSelectColumns + ` FROM comments WHERE cluster_id = $1 AND deleted = false
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	return s.queryComments(ctx, q, clusterID, capLimit(limit), offset)
}

// CommentsForPost returns non-deleted forum replies, newest first.
func (s *Store) CommentsForPost(ctx context.Context, postID uuid.UUID, limit, offset int) ([]domain.Comment, error) {
	q := commentSelectColumns + ` FROM comments WHERE post_id = $1 AND deleted = false
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`
	return s.queryComments(ctx, q, postID, capLimit(limit), offset)
}

func (s *Store) queryComments(ctx context.Context, q string, id uuid.UUID, limit, offset int) ([]domain.Comment, error) {
	rows, err := s.Pool.Query(ctx, q, id, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("store.queryComments: %w", err)
	}
	defer rows.Close()

	var out []domain.Comment
	for rows.Next() {
		var c domain.Comment
		if err := rows.Scan(&c.ID, &c.AccountID, &c.PostID, &c.ClusterID, &c.Body,
			&c.VerifiedPurchase, &c.PurchaseTokenID, &c.Rating, &c.Deleted, &c.DeletedByAdmin, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("store.queryComments: scan: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.queryComments: %w", err)
	}
	return out, nil
}

// DeleteComment soft-deletes — comments.deleted, never a real DELETE, so a
// moderation trail survives. byAdmin distinguishes an admin moderation
// action (deleted_by_admin=true) from a user deleting their own comment;
// the caller is responsible for the "own comments only" authorization check
// (05-API-REFERENCE.md §3) before calling this with byAdmin=false — this
// function enforces it too, defensively, via the WHERE clause itself when
// an accountID is supplied.
func (s *Store) DeleteComment(ctx context.Context, id uuid.UUID, requestingAccountID *uuid.UUID, byAdmin bool) error {
	q := `UPDATE comments SET deleted = true, deleted_by_admin = $2 WHERE id = $1`
	args := []any{id, byAdmin}
	if !byAdmin {
		q += ` AND account_id = $3`
		args = append(args, requestingAccountID)
	}
	tag, err := s.Pool.Exec(ctx, q, args...)
	if err != nil {
		return fmt.Errorf("store.DeleteComment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store.DeleteComment(%s): %w", id, domain.ErrNotFound)
	}
	return nil
}

func capLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 20
	}
	return limit
}
