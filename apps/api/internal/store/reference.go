package store

import (
	"context"
	"fmt"

	"github.com/dr-toke/api/internal/domain"
)

// linkBrokenThreshold mirrors the prior alpha's own constant
// (apps/api/internal/api/reference.go: "a link that has failed this many
// consecutive checks is withheld from the public API") — harvested
// behaviour, not a new number invented here.
const linkBrokenThreshold = 3

// ListStates returns the legal-status grid, featured first then
// display_order — 03-DOMAIN-MODEL.md §7: "Delhi NCR is pinned via
// featured." excise_url is withheld (NULL) once link_failures reaches the
// broken threshold, same self-correction the prior alpha's migration
// 026_reference_content.sql already implemented — "we never send users to a
// 404; the card still renders."
func (s *Store) ListStates(ctx context.Context) ([]domain.State, error) {
	const q = `
		SELECT slug, name, status, bhang_shops, detail,
		       CASE WHEN link_failures >= $1 THEN NULL ELSE excise_url END,
		       notes, featured, display_order, last_verified, verify_interval_days,
		       link_status, link_checked_at, link_failures,
		       (last_verified + (verify_interval_days || ' days')::interval) < now() AS stale,
		       created_at, updated_at
		FROM states
		ORDER BY featured DESC, display_order ASC, name ASC`

	rows, err := s.Pool.Query(ctx, q, linkBrokenThreshold)
	if err != nil {
		return nil, fmt.Errorf("store.ListStates: %w", err)
	}
	defer rows.Close()

	var out []domain.State
	for rows.Next() {
		var st domain.State
		if err := rows.Scan(&st.Slug, &st.Name, &st.Status, &st.BhangShops, &st.Detail,
			&st.ExciseURL, &st.Notes, &st.Featured, &st.DisplayOrder, &st.LastVerified,
			&st.VerifyIntervalDays, &st.LinkStatus, &st.LinkCheckedAt, &st.LinkFailures,
			&st.Stale, &st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store.ListStates: scan: %w", err)
		}
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.ListStates: %w", err)
	}
	return out, nil
}

// ListROA returns the five routes of administration, in display order.
func (s *Store) ListROA(ctx context.Context) ([]domain.ROAMethod, error) {
	const q = `
		SELECT id, slug, method, onset, duration, bioavailability,
		       pros, cons, best_for, warning_note, display_order, last_verified,
		       created_at, updated_at
		FROM roa_methods
		ORDER BY display_order ASC`

	rows, err := s.Pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store.ListROA: %w", err)
	}
	defer rows.Close()

	var out []domain.ROAMethod
	for rows.Next() {
		var m domain.ROAMethod
		if err := rows.Scan(&m.ID, &m.Slug, &m.Method, &m.Onset, &m.Duration,
			&m.Bioavailability, &m.Pros, &m.Cons, &m.BestFor, &m.WarningNote,
			&m.DisplayOrder, &m.LastVerified, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store.ListROA: scan: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.ListROA: %w", err)
	}
	return out, nil
}

// ListAggregators returns active aggregators, featured first — same
// broken-link withholding as ListStates, applied to the aggregator's own
// url this time.
func (s *Store) ListAggregators(ctx context.Context) ([]domain.Aggregator, error) {
	const q = `
		SELECT id, slug, name, url, description, source_slug,
		       brand_count_label, product_count_label,
		       derived_brand_count, derived_product_count,
		       featured, display_order, active, last_verified, verify_interval_days,
		       link_status, link_checked_at, link_failures,
		       (last_verified + (verify_interval_days || ' days')::interval) < now() AS stale,
		       created_at, updated_at
		FROM aggregators
		WHERE active = true AND link_failures < $1
		ORDER BY featured DESC, display_order ASC, name ASC`

	rows, err := s.Pool.Query(ctx, q, linkBrokenThreshold)
	if err != nil {
		return nil, fmt.Errorf("store.ListAggregators: %w", err)
	}
	defer rows.Close()

	var out []domain.Aggregator
	for rows.Next() {
		var a domain.Aggregator
		if err := rows.Scan(&a.ID, &a.Slug, &a.Name, &a.URL, &a.Description, &a.SourceSlug,
			&a.BrandCountLabel, &a.ProductCountLabel, &a.DerivedBrandCount, &a.DerivedProductCount,
			&a.Featured, &a.DisplayOrder, &a.Active, &a.LastVerified, &a.VerifyIntervalDays,
			&a.LinkStatus, &a.LinkCheckedAt, &a.LinkFailures, &a.Stale, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store.ListAggregators: scan: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.ListAggregators: %w", err)
	}
	return out, nil
}
