package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

// BatchByID reads back one scrape_batches row — internal/ingest/gate.go
// uses this to look up what a batch actually staged (source, product_count)
// before deciding on it.
func (s *Store) BatchByID(ctx context.Context, id uuid.UUID) (*domain.ScrapeBatch, error) {
	const q = `
		SELECT id, source_slug, started_at, finished_at, status,
		       product_count, previous_product_count, null_field_pct,
		       selector_hit_rate, price_median_shift, rejection_reason,
		       decided_by, decided_at, created_at
		FROM scrape_batches WHERE id = $1`
	var b domain.ScrapeBatch
	err := s.Pool.QueryRow(ctx, q, id).Scan(
		&b.ID, &b.SourceSlug, &b.StartedAt, &b.FinishedAt, &b.Status,
		&b.ProductCount, &b.PreviousProductCount, &b.NullFieldPct,
		&b.SelectorHitRate, &b.PriceMedianShift, &b.RejectionReason,
		&b.DecidedBy, &b.DecidedAt, &b.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store.BatchByID: %w", domain.ErrNotFound)
		}
		return nil, fmt.Errorf("store.BatchByID: %w", err)
	}
	return &b, nil
}

// LastApprovedBatchCount returns the product_count of the most recently
// approved batch for a source — ADR-010's baseline the gate compares a new
// batch against. Returns nil, nil (not an error) when a source has never
// had an approved batch yet — the gate's bootstrap case, not a failure.
func (s *Store) LastApprovedBatchCount(ctx context.Context, sourceSlug string) (*int, error) {
	var count *int
	err := s.Pool.QueryRow(ctx,
		`SELECT product_count FROM scrape_batches
		 WHERE source_slug = $1 AND status = 'approved'
		 ORDER BY decided_at DESC LIMIT 1`,
		sourceSlug,
	).Scan(&count)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("store.LastApprovedBatchCount: %w", err)
	}
	return count, nil
}

// DecideBatch records the promotion gate's outcome — ADR-010: rejection
// "alerts and holds. It never overwrites." This function only ever moves a
// batch OUT of pending_review into approved/rejected; it does not touch
// raw_products or product_listings itself, so a rejected batch's staged
// rows simply sit there, inert, for a human to inspect or re-run.
func (s *Store) DecideBatch(ctx context.Context, batchID uuid.UUID, status domain.BatchStatus, previousCount *int, reason *string, decidedBy string) error {
	if status != domain.BatchApproved && status != domain.BatchRejected {
		return fmt.Errorf("store.DecideBatch: status must be approved or rejected, got %q", status)
	}
	tag, err := s.Pool.Exec(ctx,
		`UPDATE scrape_batches
		 SET status = $2, previous_product_count = $3, rejection_reason = $4,
		     decided_by = $5, decided_at = now()
		 WHERE id = $1 AND status = 'pending_review'`,
		batchID, status, previousCount, reason, decidedBy)
	if err != nil {
		return fmt.Errorf("store.DecideBatch: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store.DecideBatch(%s): %w", batchID, domain.ErrNotFound)
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
