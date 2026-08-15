package store

import (
	"context"
	"fmt"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
)

// Enqueue writes one pending review item. 04-PIPELINE.md §5's reason
// vocabulary is enforced by the CHECK constraint on review_queue.reason
// (internal/db/migrations/007) — a typo'd reason fails here, at write time,
// not silently in an admin filter that never matches it.
func (s *Store) Enqueue(ctx context.Context, item domain.ReviewQueueItem) (uuid.UUID, error) {
	const q = `
		INSERT INTO review_queue (cluster_id, reason, detail, proposed_value, status)
		VALUES ($1,$2,$3,$4,'pending')
		RETURNING id`

	// proposed_value is NOT NULL jsonb — same nil-map-encodes-as-NULL trap
	// as content.go's NewRevision and facets.go's UpsertFacets; fixed here
	// too rather than leaving a third copy of the same crash waiting.
	proposedValue := item.ProposedValue
	if proposedValue == nil {
		proposedValue = map[string]any{}
	}

	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, q, item.ClusterID, item.Reason, item.Detail, proposedValue).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("store.Enqueue: %w", err)
	}
	return id, nil
}

// QueueFilter scopes ListQueue. 06-ADMIN.md §1.2: "Filter by reason" is the
// review UI's primary navigation; Status defaults to pending (empty string)
// since a queue view that silently includes already-resolved rows isn't
// useful for the keyboard-driven triage flow that section describes.
type QueueFilter struct {
	Reason domain.ReviewReason // "" = any reason
	Status domain.ReviewStatus // "" = pending only (see doc comment above)
	Limit  int
}

// ListQueue returns queue items oldest-first — a queue is a queue; FIFO is
// the only ordering that doesn't quietly let old items rot at the bottom.
func (s *Store) ListQueue(ctx context.Context, f QueueFilter) ([]domain.ReviewQueueItem, error) {
	status := f.Status
	if status == "" {
		status = domain.ReviewPending
	}
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	q := `
		SELECT id, cluster_id, reason, detail, proposed_value, status,
		       resolved_by, resolved_at, created_at
		FROM review_queue
		WHERE status = $1`
	args := []any{status}
	if f.Reason != "" {
		q += ` AND reason = $2 ORDER BY created_at ASC LIMIT $3`
		args = append(args, f.Reason, limit)
	} else {
		q += ` ORDER BY created_at ASC LIMIT $2`
		args = append(args, limit)
	}

	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store.ListQueue: %w", err)
	}
	defer rows.Close()

	var out []domain.ReviewQueueItem
	for rows.Next() {
		var it domain.ReviewQueueItem
		if err := rows.Scan(&it.ID, &it.ClusterID, &it.Reason, &it.Detail, &it.ProposedValue,
			&it.Status, &it.ResolvedBy, &it.ResolvedAt, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("store.ListQueue: scan: %w", err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.ListQueue: %w", err)
	}
	return out, nil
}

// Resolve marks a queue item approved or rejected. Does not itself write a
// product_facet_overrides row or an admin_audit_log entry — 06-ADMIN.md
// §1.2 says approving a review item writes an override AND appends a golden
// fixture; same cross-package-boundary reasoning as overrides.go's
// SetOverride: this function's job is the review_queue row only, the caller
// (an M9 admin handler) orchestrates all three writes together.
func (s *Store) Resolve(ctx context.Context, id uuid.UUID, status domain.ReviewStatus, resolvedBy string) error {
	if status != domain.ReviewApproved && status != domain.ReviewRejected {
		return fmt.Errorf("store.Resolve: status must be approved or rejected, got %q", status)
	}
	tag, err := s.Pool.Exec(ctx,
		`UPDATE review_queue SET status = $2, resolved_by = $3, resolved_at = now()
		 WHERE id = $1 AND status = 'pending'`,
		id, status, resolvedBy)
	if err != nil {
		return fmt.Errorf("store.Resolve: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store.Resolve(%s): %w", id, domain.ErrNotFound)
	}
	return nil
}
