package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// Sort modes — SortValue's composite rank_score by default, or scoped to
// one cannabinoid's own ₹/mg column when Basis is set
// (05-API-REFERENCE.md §1: "scopes the value sort" — ₹/mg-CBD and ₹/mg-THC
// aren't comparable, so a basis-scoped sort orders by that specific column,
// not the blended editorial rank_score). SortPrice added for the real
// frontend's "Lowest price" option (CatalogGrid.svelte's SORTS list) —
// 05-API-REFERENCE.md always named it but nothing implemented it until now.
const (
	SortNew   = "new"
	SortValue = "value"
	SortPrice = "price"
)

// ClusterFilter scopes ListClusters/CountClusters. Matches the query
// surface the real frontend (apps/web/src/lib/sections/products/
// CatalogGrid.svelte) actually sends — category/extract/brand/basis/
// verified/sort/page — not 05-API-REFERENCE.md's aspirational param list,
// which was never reconciled against the frontend code already in the
// repo. See API-DECISIONS.md.
type ClusterFilter struct {
	BrandID         *uuid.UUID
	PublishableOnly bool
	VerifiedOnly    bool
	// Category is the LEGACY category vocabulary (tincture/edible/topical/
	// smokable/vapeable/extract/beverage/nutrition/pet) — never a stored
	// column (internal/db/migrations/002: category is derived from facets
	// at read time, "exactly one writer for facet-derived data"), so this
	// is translated into a facet/concentration_type WHERE fragment by
	// categoryWhereFragment, mirroring resolve.LegacyCategory's own
	// mapping exactly rather than approximating it.
	Category string
	// Extract is the real `extract` facet value — filtered directly
	// against product_facets, no derivation needed.
	Extract string
	// Basis scopes SortValue to cbd_price_per_mg or thc_price_per_mg
	// instead of the composite rank_score. "" | "cbd" | "thc".
	Basis string
	// Sort: SortNew (default) | SortValue | SortPrice.
	Sort string

	// Page-based pagination (1-based) — the real frontend's actual model;
	// CatalogGrid.svelte only ever moves ±1 page, never jumps arbitrarily,
	// so plain OFFSET is the pragmatic choice at this catalogue's scale
	// (thousands, not millions, of rows) despite 02-FRONTEND-CONTRACT.md
	// §5's general "never OFFSET" guidance being the more scale-correct
	// answer — see API-DECISIONS.md for the full reasoning. Page <= 0
	// defaults to page 1.
	Page  int
	Limit int
}

// buildWhere constructs the WHERE clause shared by ListClusters and
// CountClusters — kept as one function specifically so the two queries can
// never drift on what "matches the filter" means (a bug class M3's own
// clusterSelectColumns/clusterScanTargets split already guards against for
// column lists; this is the same discipline for filter predicates).
func (f ClusterFilter) buildWhere(argN *int, args *[]any) string {
	next := func(v any) string {
		*argN++
		*args = append(*args, v)
		return fmt.Sprintf("$%d", *argN)
	}

	where := " WHERE 1=1"
	if f.PublishableOnly {
		where += " AND publishable = true"
	}
	if f.BrandID != nil {
		where += " AND brand_id = " + next(*f.BrandID)
	}
	if f.VerifiedOnly {
		where += " AND brand_id IN (SELECT id FROM brands WHERE verified = true)"
	}
	if f.Extract != "" {
		where += " AND id IN (SELECT cluster_id FROM product_facets WHERE facet = 'extract' AND value = " + next(f.Extract) + ")"
	}
	if f.Category != "" {
		if frag, ok := categoryWhereFragment(f.Category, next); ok {
			where += " AND " + frag
		} else {
			// Unknown category slug — no results, not an error (same "valid
			// filter, empty match" contract as an unknown brand slug in
			// internal/api/products.go).
			where += " AND false"
		}
	}

	switch f.Sort {
	case SortValue:
		switch f.Basis {
		case "cbd":
			where += " AND cbd_price_per_mg IS NOT NULL"
		case "thc":
			where += " AND thc_price_per_mg IS NOT NULL"
		default:
			// rank_score may be NULL for un-ranked rows (e.g. hemp-seed
			// oil, no computable ₹/mg) — excluded from this ordering
			// entirely, "value" sort has nothing meaningful to say about
			// them.
			where += " AND rank_score IS NOT NULL"
		}
	case SortPrice:
		where += " AND best_price_paise IS NOT NULL"
	}
	return where
}

// categoryWhereFragment reverse-maps resolve.LegacyCategory's mapping
// (form + route + concentration_type -> legacy category) into SQL,
// following that function's exact branches rather than approximating them —
// see legacy.go's own doc comment for why form=concentrate and form=edible
// each need the extra route/concentration_type check.
func categoryWhereFragment(category string, next func(any) string) (string, bool) {
	formIn := func(vals ...string) string {
		placeholders := make([]string, len(vals))
		for i, v := range vals {
			placeholders[i] = next(v)
		}
		return "id IN (SELECT cluster_id FROM product_facets WHERE facet = 'form' AND value IN (" + strings.Join(placeholders, ",") + "))"
	}
	routeIs := func(v string) string {
		return "id IN (SELECT cluster_id FROM product_facets WHERE facet = 'route' AND value = " + next(v) + ")"
	}

	switch category {
	case "tincture":
		return formIn("oil_tincture"), true
	case "topical":
		return formIn("topical"), true
	case "smokable":
		return formIn("flower"), true
	case "beverage":
		return formIn("beverage"), true
	case "pet":
		return formIn("pet"), true
	case "vapeable":
		return "(" + formIn("vape") + " OR (" + formIn("concentrate") + " AND NOT " + routeIs("oral") + "))", true
	case "extract":
		return "(" + formIn("concentrate") + " AND " + routeIs("oral") + ")", true
	case "edible":
		return "(" + formIn("capsule") + " OR (" + formIn("edible") + " AND concentration_type NOT IN ('hemp_seed','nutrition')))", true
	case "nutrition":
		return "(" + formIn("edible") + " AND concentration_type IN ('hemp_seed','nutrition'))", true
	default:
		return "", false
	}
}

// orderByClause returns the ORDER BY matching f.Sort/f.Basis — always
// tie-broken on id (ASC for SortNew's DESC-time order and SortValue/
// SortPrice's DESC/ASC-value order alike) per 02-FRONTEND-CONTRACT.md §5:
// "so the grid doesn't reshuffle between loads."
func orderByClause(f ClusterFilter) string {
	switch f.Sort {
	case SortValue:
		switch f.Basis {
		case "cbd":
			return " ORDER BY cbd_price_per_mg ASC, id ASC"
		case "thc":
			return " ORDER BY thc_price_per_mg ASC, id ASC"
		default:
			return " ORDER BY rank_score DESC, id ASC"
		}
	case SortPrice:
		return " ORDER BY best_price_paise ASC, id ASC"
	default:
		return " ORDER BY first_seen_at DESC, id ASC"
	}
}

// ListClusters returns clusters page-paginated (1-based f.Page, defaulting
// to 1) and ordered per f.Sort/f.Basis.
func (s *Store) ListClusters(ctx context.Context, f ClusterFilter) ([]domain.ProductCluster, error) {
	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	page := f.Page
	if page <= 0 {
		page = 1
	}

	argN := 0
	var args []any
	q := clusterSelectColumns + ` FROM product_clusters` + f.buildWhere(&argN, &args) + orderByClause(f)

	argN++
	args = append(args, limit)
	limitPh := fmt.Sprintf("$%d", argN)
	argN++
	args = append(args, (page-1)*limit)
	offsetPh := fmt.Sprintf("$%d", argN)
	q += fmt.Sprintf(" LIMIT %s OFFSET %s", limitPh, offsetPh)

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

// CountClusters returns how many clusters match f's filters — ignoring
// Page/Limit entirely, since this is the envelope's "total": the size of
// the whole filtered set, not the current page. Shares buildWhere with
// ListClusters so total and the actual rows returned always agree on what
// "matches the filter" means.
func (s *Store) CountClusters(ctx context.Context, f ClusterFilter) (int, error) {
	argN := 0
	var args []any
	q := `SELECT count(*) FROM product_clusters` + f.buildWhere(&argN, &args)

	var count int
	if err := s.Pool.QueryRow(ctx, q, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("store.CountClusters: %w", err)
	}
	return count, nil
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
