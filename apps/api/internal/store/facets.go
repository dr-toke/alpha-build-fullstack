package store

import (
	"context"
	"fmt"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
)

// FacetsFor returns every resolved facet for a cluster — at most six rows,
// one per facet name (PRIMARY KEY (cluster_id, facet),
// internal/db/migrations/003).
func (s *Store) FacetsFor(ctx context.Context, clusterID uuid.UUID) ([]domain.ProductFacet, error) {
	const q = `
		SELECT cluster_id, facet, value, source, confidence, evidence,
		       classifier_version, decided_at
		FROM product_facets
		WHERE cluster_id = $1`

	rows, err := s.Pool.Query(ctx, q, clusterID)
	if err != nil {
		return nil, fmt.Errorf("store.FacetsFor: %w", err)
	}
	defer rows.Close()

	var out []domain.ProductFacet
	for rows.Next() {
		var f domain.ProductFacet
		if err := rows.Scan(&f.ClusterID, &f.Facet, &f.Value, &f.Source,
			&f.Confidence, &f.Evidence, &f.ClassifierVersion, &f.DecidedAt); err != nil {
			return nil, fmt.Errorf("store.FacetsFor: scan: %w", err)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.FacetsFor: %w", err)
	}
	return out, nil
}

// UpsertFacets writes resolve.Resolve()'s output — one row per facet,
// replacing whatever was there. This is called with the OUTCOME of
// precedence (override > rule > model > default already applied by
// internal/resolve, M1) — this function has no precedence logic of its own,
// it only persists what it's given. One statement per facet inside a single
// transaction: all-or-nothing, so a cluster is never left with three facets
// from classifier v14 and three from v13 because a write failed partway.
func (s *Store) UpsertFacets(ctx context.Context, facets []domain.ProductFacet) error {
	if len(facets) == 0 {
		return nil
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store.UpsertFacets: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op if Commit already succeeded

	const q = `
		INSERT INTO product_facets
			(cluster_id, facet, value, source, confidence, evidence, classifier_version, decided_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7, now())
		ON CONFLICT (cluster_id, facet) DO UPDATE SET
			value              = EXCLUDED.value,
			source             = EXCLUDED.source,
			confidence         = EXCLUDED.confidence,
			evidence           = EXCLUDED.evidence,
			classifier_version = EXCLUDED.classifier_version,
			decided_at         = now()`

	for _, f := range facets {
		// evidence is NOT NULL jsonb — a nil Go map encodes as SQL NULL, not
		// '{}', and violates the constraint. Same bug class as
		// content.go's NewRevision (see its comment); fixed here
		// proactively rather than waiting for this exact crash a second time.
		evidence := f.Evidence
		if evidence == nil {
			evidence = map[string]any{}
		}
		if _, err := tx.Exec(ctx, q, f.ClusterID, f.Facet, f.Value, f.Source,
			f.Confidence, evidence, f.ClassifierVersion); err != nil {
			return fmt.Errorf("store.UpsertFacets: facet=%s: %w", f.Facet, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store.UpsertFacets: commit: %w", err)
	}
	return nil
}
