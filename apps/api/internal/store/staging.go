package store

import (
	"context"
	"fmt"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
)

// staging.go is a small addition beyond M3's original 11-file list
// (store.go..golden.go) — found necessary while building M4's ingest
// package: 01-ARCHITECTURE.md §6's "SQL lives only in the store layer"
// leaves internal/ingest nowhere legitimate to put the INSERT statements
// ADR-010's promotion gate needs (scrape_batches, raw_products), so those
// belong here, not in internal/ingest/staging.go despite that file's name
// suggesting otherwise — see that file's doc comment. Flagged in
// M4-DECISIONS.md.

// CreateBatch starts a new scrape_batches row — the promotion gate's unit
// of decision (ADR-010). Status starts 'running'; FinishBatch moves it to
// 'pending_review' once the scrape completes.
func (s *Store) CreateBatch(ctx context.Context, sourceSlug string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx,
		`INSERT INTO scrape_batches (source_slug, status) VALUES ($1, 'running') RETURNING id`,
		sourceSlug).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("store.CreateBatch: %w", err)
	}
	return id, nil
}

// StageRawProduct writes ONE scraped listing to raw_products — staging,
// never live. 04-PIPELINE.md §1: "scrape: Fetch raw listings per store into
// staging. Never writes live." This function is the only thing in the
// codebase allowed to INSERT into raw_products, same single-writer
// discipline as internal/resolve.Resolve() for facets.
func (s *Store) StageRawProduct(ctx context.Context, batchID uuid.UUID, p domain.RawProduct) (uuid.UUID, error) {
	const q = `
		INSERT INTO raw_products
			(batch_id, source_slug, source_url, source_sku, name, brand_raw,
			 price_raw, description, image_url, category_raw, raw_data)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id`
	rawData := p.RawData
	if rawData == nil {
		rawData = map[string]any{} // same NOT NULL jsonb trap as facets.go/queue.go/content.go
	}
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, q, batchID, p.SourceSlug, p.SourceURL, p.SourceSKU, p.Name,
		p.BrandRaw, p.PriceRaw, p.Description, p.ImageURL, p.CategoryRaw, rawData).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("store.StageRawProduct: %w", err)
	}
	return id, nil
}

// FinishBatch closes out a batch — moves it to pending_review and records
// how many listings landed, the raw material ADR-010's gate (internal/
// ingest/gate.go, not yet built) evaluates against the previous run's count.
func (s *Store) FinishBatch(ctx context.Context, batchID uuid.UUID, productCount int) error {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE scrape_batches SET status = 'pending_review', product_count = $2, finished_at = now()
		 WHERE id = $1 AND status = 'running'`,
		batchID, productCount)
	if err != nil {
		return fmt.Errorf("store.FinishBatch: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store.FinishBatch(%s): %w", batchID, domain.ErrNotFound)
	}
	return nil
}

// RawProductsForBatch reads back everything staged under one batch —
// LoadBatch's underlying query (internal/ingest/staging.go wraps this for
// callers that only have a batch ID, not a Store).
func (s *Store) RawProductsForBatch(ctx context.Context, batchID uuid.UUID) ([]domain.RawProduct, error) {
	const q = `
		SELECT id, batch_id, source_slug, source_url, source_sku, name, brand_raw,
		       price_raw, description, image_url, category_raw, raw_data, scraped_at
		FROM raw_products WHERE batch_id = $1 ORDER BY scraped_at ASC`
	rows, err := s.Pool.Query(ctx, q, batchID)
	if err != nil {
		return nil, fmt.Errorf("store.RawProductsForBatch: %w", err)
	}
	defer rows.Close()

	var out []domain.RawProduct
	for rows.Next() {
		var p domain.RawProduct
		if err := rows.Scan(&p.ID, &p.BatchID, &p.SourceSlug, &p.SourceURL, &p.SourceSKU, &p.Name,
			&p.BrandRaw, &p.PriceRaw, &p.Description, &p.ImageURL, &p.CategoryRaw, &p.RawData, &p.ScrapedAt); err != nil {
			return nil, fmt.Errorf("store.RawProductsForBatch: scan: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.RawProductsForBatch: %w", err)
	}
	return out, nil
}
