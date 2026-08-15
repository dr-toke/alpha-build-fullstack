package store

import (
	"context"
	"fmt"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
)

// UpsertListing writes one listing, keyed on the (source_slug, source_url)
// UNIQUE constraint (internal/db/migrations/001) — the constraint that makes
// the harvested per-variant-URL scraper fix (harvest/scrapers/cbdstore.yaml)
// meaningful: two variants of the same product never collide onto one row.
//
// On conflict, cluster_id is preserved via COALESCE when the caller passes
// nil — a re-scrape of an already-known, already-clustered listing must not
// wipe out dedup's prior work just because the promote step doesn't know the
// cluster yet. first_seen_at and created_at are never touched on conflict;
// last_seen_at and updated_at always bump to now().
func (s *Store) UpsertListing(ctx context.Context, l domain.ProductListing) (uuid.UUID, error) {
	const q = `
		INSERT INTO product_listings
			(source_slug, source_url, source_sku, cluster_id, name_raw, brand_raw,
			 description_raw, category_raw, image_url_raw, price_paise, affiliate_url,
			 in_stock, promoted_from_batch_id, first_seen_at, last_seen_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13, now(), now())
		ON CONFLICT (source_slug, source_url) DO UPDATE SET
			source_sku              = EXCLUDED.source_sku,
			cluster_id              = COALESCE(EXCLUDED.cluster_id, product_listings.cluster_id),
			name_raw                = EXCLUDED.name_raw,
			brand_raw               = EXCLUDED.brand_raw,
			description_raw         = EXCLUDED.description_raw,
			category_raw            = EXCLUDED.category_raw,
			image_url_raw           = EXCLUDED.image_url_raw,
			price_paise             = EXCLUDED.price_paise,
			affiliate_url           = EXCLUDED.affiliate_url,
			in_stock                = EXCLUDED.in_stock,
			promoted_from_batch_id  = EXCLUDED.promoted_from_batch_id,
			last_seen_at            = now(),
			updated_at              = now()
		RETURNING id`

	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, q,
		l.SourceSlug, l.SourceURL, l.SourceSKU, l.ClusterID, l.NameRaw, l.BrandRaw,
		l.DescriptionRaw, l.CategoryRaw, l.ImageURLRaw, l.PricePaise, l.AffiliateURL,
		l.InStock, l.PromotedFromBatchID,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("store.UpsertListing: %w", err)
	}
	return id, nil
}

// ListingsForCluster returns every listing belonging to a cluster, cheapest
// in-stock first — the ordering the product detail page's `listings[]`
// array wants (05-API-REFERENCE.md's product payload).
func (s *Store) ListingsForCluster(ctx context.Context, clusterID uuid.UUID) ([]domain.ProductListing, error) {
	const q = `
		SELECT id, source_slug, source_url, source_sku, cluster_id, name_raw,
		       brand_raw, description_raw, category_raw, image_url_raw,
		       price_paise, affiliate_url, in_stock, promoted_from_batch_id,
		       first_seen_at, last_seen_at, created_at, updated_at
		FROM product_listings
		WHERE cluster_id = $1
		ORDER BY in_stock DESC, price_paise ASC`

	rows, err := s.Pool.Query(ctx, q, clusterID)
	if err != nil {
		return nil, fmt.Errorf("store.ListingsForCluster: %w", err)
	}
	defer rows.Close()

	var out []domain.ProductListing
	for rows.Next() {
		var l domain.ProductListing
		if err := rows.Scan(
			&l.ID, &l.SourceSlug, &l.SourceURL, &l.SourceSKU, &l.ClusterID, &l.NameRaw,
			&l.BrandRaw, &l.DescriptionRaw, &l.CategoryRaw, &l.ImageURLRaw,
			&l.PricePaise, &l.AffiliateURL, &l.InStock, &l.PromotedFromBatchID,
			&l.FirstSeenAt, &l.LastSeenAt, &l.CreatedAt, &l.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("store.ListingsForCluster: scan: %w", err)
		}
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.ListingsForCluster: %w", err)
	}
	return out, nil
}
