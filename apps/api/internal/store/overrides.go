package store

import (
	"context"
	"fmt"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
)

// OverridesFor returns every permanent human correction on a cluster.
// Overrides apply AFTER every pipeline run, unconditionally — never
// recomputed, never expired (03-DOMAIN-MODEL.md §2) — internal/resolve's
// precedence.Resolve() is the only thing that decides whether an override
// wins; this function only reads what exists.
func (s *Store) OverridesFor(ctx context.Context, clusterID uuid.UUID) ([]domain.ProductFacetOverride, error) {
	const q = `
		SELECT cluster_id, facet, value, reason, set_by, set_at
		FROM product_facet_overrides
		WHERE cluster_id = $1`

	rows, err := s.Pool.Query(ctx, q, clusterID)
	if err != nil {
		return nil, fmt.Errorf("store.OverridesFor: %w", err)
	}
	defer rows.Close()

	var out []domain.ProductFacetOverride
	for rows.Next() {
		var o domain.ProductFacetOverride
		if err := rows.Scan(&o.ClusterID, &o.Facet, &o.Value, &o.Reason, &o.SetBy, &o.SetAt); err != nil {
			return nil, fmt.Errorf("store.OverridesFor: scan: %w", err)
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.OverridesFor: %w", err)
	}
	return out, nil
}

// SetOverride writes one permanent correction — PRIMARY KEY (cluster_id,
// facet), so setting a second override on the same facet replaces the
// first rather than accumulating a history (set_at is overwritten to now();
// if an override's own history ever matters, that's what admin_audit_log,
// M0, is for).
//
// Deliberately does NOT also append a testdata/golden fixture here, even
// though 03-DOMAIN-MODEL.md §2 describes the two as inseparable ("every
// override auto-appends a fixture"). Doing so needs the raw listing
// title/description this function doesn't have — only a caller holding the
// full listing context (an M9 admin handler) does — and reaching into file
// I/O from what is otherwise a pure DB write would cross the exact
// modularity boundary 01-ARCHITECTURE.md §6 draws ("a file reaching across
// two of those is a refactor, not a feature"). The caller is expected to
// invoke SetOverride and golden.AppendFixture together. Flagged in
// M3-DECISIONS.md — worth confirming this split is right once M9 actually
// writes that caller.
func (s *Store) SetOverride(ctx context.Context, o domain.ProductFacetOverride) error {
	const q = `
		INSERT INTO product_facet_overrides (cluster_id, facet, value, reason, set_by, set_at)
		VALUES ($1,$2,$3,$4,$5, now())
		ON CONFLICT (cluster_id, facet) DO UPDATE SET
			value   = EXCLUDED.value,
			reason  = EXCLUDED.reason,
			set_by  = EXCLUDED.set_by,
			set_at  = now()`

	_, err := s.Pool.Exec(ctx, q, o.ClusterID, o.Facet, o.Value, o.Reason, o.SetBy)
	if err != nil {
		return fmt.Errorf("store.SetOverride: %w", err)
	}
	return nil
}
