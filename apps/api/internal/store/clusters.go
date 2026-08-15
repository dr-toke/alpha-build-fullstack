package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ClusterByID returns a cluster by its durable UUID. If the cluster was
// merged away, it does NOT return domain.ErrNotFound — 03-DOMAIN-MODEL.md §4
// / 02-FRONTEND-CONTRACT.md §4 are explicit that a merged cluster is a 200
// with {"moved_to": new_id}, never a 404, "so the frontend rewrites instead
// of 404ing." Returns *domain.ClusterMovedError (via errors.As) in that case.
//
// Checks cluster_merges FIRST, not as a not-found fallback — Merge()
// deliberately never deletes the old product_clusters row (a hard delete
// would cascade-destroy its facets/comments/click history via ON DELETE
// CASCADE, and nothing in 03-DOMAIN-MODEL.md §4 asks for that), so the old
// row is still directly findable after a merge. A first draft checked
// cluster_merges only when the direct lookup missed, which meant a merged
// cluster's stale row was returned as if nothing had happened — caught by
// TestClusterByID's merge case, not by inspection.
func (s *Store) ClusterByID(ctx context.Context, id uuid.UUID) (*domain.ProductCluster, error) {
	var newID uuid.UUID
	mergeErr := s.Pool.QueryRow(ctx,
		`SELECT new_id FROM cluster_merges WHERE old_id = $1`, id,
	).Scan(&newID)
	if mergeErr == nil {
		return nil, &domain.ClusterMovedError{OldID: id, NewID: newID}
	}
	if !errors.Is(mergeErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("store.ClusterByID: checking cluster_merges: %w", mergeErr)
	}

	c, err := s.scanCluster(ctx, id)
	if err == nil {
		return c, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("store.ClusterByID: %w", err)
	}
	return nil, fmt.Errorf("store.ClusterByID: %w", domain.ErrNotFound)
}

func (s *Store) scanCluster(ctx context.Context, id uuid.UUID) (*domain.ProductCluster, error) {
	const q = clusterSelectColumns + ` FROM product_clusters WHERE id = $1`
	var c domain.ProductCluster
	err := s.Pool.QueryRow(ctx, q, id).Scan(clusterScanTargets(&c)...)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

const clusterSelectColumns = `
	SELECT id, fingerprint, brand_id, name, short_description,
	       cbd_mg, thc_mg, total_cannabinoids_mg, concentration_type,
	       cannabinoid_confidence, cannabinoid_evidence,
	       volume_ml, weight_g,
	       best_price_paise, best_price_per_mg, cbd_price_per_mg, thc_price_per_mg,
	       price_per_mg_basis, value_tier, rank_score,
	       image_id, coa_available, prescription_required, publishable,
	       first_seen_at, updated_at, created_at`

// clusterScanTargets returns, in the exact order clusterSelectColumns
// selects them, the addresses Scan should write into — kept as one function
// so the two never drift out of sync silently.
func clusterScanTargets(c *domain.ProductCluster) []any {
	return []any{
		&c.ID, &c.Fingerprint, &c.BrandID, &c.Name, &c.ShortDescription,
		&c.CBDMg, &c.THCMg, &c.TotalCannabinoidsMg, &c.ConcentrationType,
		&c.CannabinoidConfidence, &c.CannabinoidEvidence,
		&c.VolumeML, &c.WeightG,
		&c.BestPricePaise, &c.BestPricePerMg, &c.CBDPricePerMg, &c.THCPricePerMg,
		&c.PricePerMgBasis, &c.ValueTier, &c.RankScore,
		&c.ImageID, &c.COAAvailable, &c.PrescriptionRequired, &c.Publishable,
		&c.FirstSeenAt, &c.UpdatedAt, &c.CreatedAt,
	}
}

// CreateCluster inserts a new product_clusters row and returns its durable
// UUID — the identity 03-DOMAIN-MODEL.md §4 says is "assigned on first
// sight... never a recomputed hash," which is exactly why this is a plain
// INSERT, not an upsert: internal/db/migrations/008 deliberately leaves
// `fingerprint` without a UNIQUE constraint (a merged-away cluster keeps its
// old fingerprint forever), so "does a live cluster with this fingerprint
// already exist" is ClusterByFingerprint's job, called BEFORE this by
// internal/ingest's AssignCluster — this function never checks itself.
func (s *Store) CreateCluster(ctx context.Context, c domain.ProductCluster) (uuid.UUID, error) {
	const q = `
		INSERT INTO product_clusters
			(fingerprint, brand_id, name, short_description,
			 cbd_mg, thc_mg, total_cannabinoids_mg, concentration_type,
			 cannabinoid_confidence, cannabinoid_evidence,
			 volume_ml, weight_g,
			 best_price_paise, best_price_per_mg, cbd_price_per_mg, thc_price_per_mg,
			 price_per_mg_basis, value_tier, rank_score,
			 image_id, coa_available, prescription_required, publishable)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		RETURNING id`

	// cannabinoid_evidence is NOT NULL jsonb — same nil-map bug class as
	// facets.go's UpsertFacets and content.go's NewRevision; fixed here on
	// first write rather than waiting to hit it live a third time.
	evidence := c.CannabinoidEvidence
	if evidence == nil {
		evidence = map[string]any{}
	}

	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, q,
		c.Fingerprint, c.BrandID, c.Name, c.ShortDescription,
		c.CBDMg, c.THCMg, c.TotalCannabinoidsMg, c.ConcentrationType,
		c.CannabinoidConfidence, evidence,
		c.VolumeML, c.WeightG,
		c.BestPricePaise, c.BestPricePerMg, c.CBDPricePerMg, c.THCPricePerMg,
		c.PricePerMgBasis, c.ValueTier, c.RankScore,
		c.ImageID, c.COAAvailable, c.PrescriptionRequired, c.Publishable,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("store.CreateCluster: %w", err)
	}
	return id, nil
}

// UpdateClusterDerived overwrites every derived field on an EXISTING
// cluster (found via ClusterByFingerprint) with freshly resolved values —
// 04-PIPELINE.md §1's explicit, capitalised instruction: "the existing-
// cluster (fingerprint match) branch must refresh all derived fields —
// category/facets, cannabinoids, everything — not just bump last_seen_at
// ... This is critical. Keep it." Does NOT touch fingerprint, id,
// first_seen_at, or created_at — a re-scrape landing on the same
// fingerprint is the same product seen again, not a new one.
func (s *Store) UpdateClusterDerived(ctx context.Context, id uuid.UUID, c domain.ProductCluster) error {
	const q = `
		UPDATE product_clusters SET
			brand_id = $2, name = $3, short_description = $4,
			cbd_mg = $5, thc_mg = $6, total_cannabinoids_mg = $7, concentration_type = $8,
			cannabinoid_confidence = $9, cannabinoid_evidence = $10,
			volume_ml = $11, weight_g = $12,
			best_price_paise = $13, best_price_per_mg = $14, cbd_price_per_mg = $15, thc_price_per_mg = $16,
			price_per_mg_basis = $17, value_tier = $18, rank_score = $19,
			image_id = $20, coa_available = $21, prescription_required = $22, publishable = $23,
			updated_at = now()
		WHERE id = $1`

	evidence := c.CannabinoidEvidence
	if evidence == nil {
		evidence = map[string]any{}
	}

	tag, err := s.Pool.Exec(ctx, q, id,
		c.BrandID, c.Name, c.ShortDescription,
		c.CBDMg, c.THCMg, c.TotalCannabinoidsMg, c.ConcentrationType,
		c.CannabinoidConfidence, evidence,
		c.VolumeML, c.WeightG,
		c.BestPricePaise, c.BestPricePerMg, c.CBDPricePerMg, c.THCPricePerMg,
		c.PricePerMgBasis, c.ValueTier, c.RankScore,
		c.ImageID, c.COAAvailable, c.PrescriptionRequired, c.Publishable,
	)
	if err != nil {
		return fmt.Errorf("store.UpdateClusterDerived: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store.UpdateClusterDerived(%s): %w", id, domain.ErrNotFound)
	}
	return nil
}

// ClusterByFingerprint looks up a LIVE cluster by its dedup key
// (harvest/rules/dedup.md) — internal/ingest's AssignCluster calls this
// before CreateCluster to decide new-vs-existing. "Live" excludes any
// cluster_merges.old_id: a merged-away row keeps its fingerprint forever
// (see migrations/008's own comment), so a fresh product landing on that
// same fingerprint must resolve to the cluster it was merged INTO, not the
// stale row. Returns domain.ErrNotFound (not ClusterMovedError — that's
// ClusterByID's public-URL contract, not this internal lookup's) when no
// live cluster carries the fingerprint yet.
func (s *Store) ClusterByFingerprint(ctx context.Context, fingerprint string) (*domain.ProductCluster, error) {
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx,
		`SELECT id FROM product_clusters WHERE fingerprint = $1 ORDER BY created_at DESC LIMIT 1`,
		fingerprint,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store.ClusterByFingerprint: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("store.ClusterByFingerprint: %w", err)
	}

	// Follow a merge exactly like ClusterByID does — needed here specifically
	// because the row this fingerprint is stamped on may be the OLD,
	// merged-away one (it keeps its fingerprint forever, migrations/008's own
	// comment), so the fingerprint alone can point at a dead row.
	var newID uuid.UUID
	mergeErr := s.Pool.QueryRow(ctx,
		`SELECT new_id FROM cluster_merges WHERE old_id = $1`, id,
	).Scan(&newID)
	if mergeErr == nil {
		id = newID
	} else if !errors.Is(mergeErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("store.ClusterByFingerprint: checking cluster_merges: %w", mergeErr)
	}

	c, err := s.scanCluster(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("store.ClusterByFingerprint: %w", err)
	}
	return c, nil
}

// ClusterFilter scopes ListClusters. Deliberately minimal for M3 — full
// facet-filter query construction (?form=&route=&brand=&basis=, per
// 05-API-REFERENCE.md §1) is internal/api's job once M5 defines the exact
// param surface; this covers what's storage-layer-obvious now (publishable
// gate, brand, keyset pagination on rank_score) rather than guessing ahead
// of the handler that will actually call it. See M3-DECISIONS.md.
type ClusterFilter struct {
	BrandID         *uuid.UUID
	PublishableOnly bool
	// Cursor: keyset pagination on (rank_score DESC, id ASC) —
	// 02-FRONTEND-CONTRACT.md §5, never OFFSET. Both nil means "first page."
	CursorRankScore *float64
	CursorID        *uuid.UUID
	Limit           int
}

// ListClusters returns clusters ranked by rank_score, keyset-paginated.
// Stable tie-break on id ASC per 02-FRONTEND-CONTRACT.md §5 — "so the grid
// doesn't reshuffle between loads."
func (s *Store) ListClusters(ctx context.Context, f ClusterFilter) ([]domain.ProductCluster, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 24
	}

	q := clusterSelectColumns + ` FROM product_clusters WHERE 1=1`
	args := []any{}
	argN := 0
	next := func(v any) string {
		argN++
		args = append(args, v)
		return fmt.Sprintf("$%d", argN)
	}

	if f.PublishableOnly {
		q += ` AND publishable = true`
	}
	if f.BrandID != nil {
		q += ` AND brand_id = ` + next(*f.BrandID)
	}
	if f.CursorRankScore != nil && f.CursorID != nil {
		// rank_score may be NULL for un-ranked rows; NULLS LAST keeps them
		// out of the normal keyset walk entirely rather than producing an
		// undefined comparison against a cursor value.
		rs := next(*f.CursorRankScore)
		id := next(*f.CursorID)
		q += fmt.Sprintf(` AND (rank_score, id) < (%s, %s)`, rs, id)
	}
	q += ` AND rank_score IS NOT NULL ORDER BY rank_score DESC, id ASC LIMIT ` + next(limit)

	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("store.ListClusters: %w", err)
	}
	defer rows.Close()

	var out []domain.ProductCluster
	for rows.Next() {
		var c domain.ProductCluster
		if err := rows.Scan(clusterScanTargets(&c)...); err != nil {
			return nil, fmt.Errorf("store.ListClusters: scan: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.ListClusters: %w", err)
	}
	return out, nil
}

// Merge records that oldID's identity now lives at newID — 03-DOMAIN-MODEL.md
// §4. Never guess: the caller (an admin action) supplies both IDs
// explicitly; this function does not infer merge targets from similarity.
func (s *Store) Merge(ctx context.Context, oldID, newID uuid.UUID) error {
	if oldID == newID {
		return fmt.Errorf("store.Merge: old and new cluster IDs are identical (%s)", oldID)
	}
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO cluster_merges (old_id, new_id) VALUES ($1, $2)`, oldID, newID)
	if err != nil {
		return fmt.Errorf("store.Merge: %w", err)
	}
	return nil
}
