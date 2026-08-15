package ingest

import (
	"context"
	"fmt"

	"github.com/dr-toke/api/internal/domain"
	"github.com/dr-toke/api/internal/store"
	"github.com/google/uuid"
)

// staging.go orchestrates ONE scrape run: create a batch, run the adapter,
// persist every listing, close the batch out. It contains no SQL itself —
// every write goes through store.go's CreateBatch/StageRawProduct/
// FinishBatch — 01-ARCHITECTURE.md §6: "SQL lives only in the store layer."
// This file's job is the orchestration loop (adapter -> store), nothing else.

// StageBatch runs adapter to completion, staging every listing it produces.
// Never writes to product_listings (live) — 04-PIPELINE.md §1: "scrape...
// Never writes live." Returns the batch ID and how many listings landed
// even on a partial failure (a scrape that dies halfway through still
// leaves a batch record showing what it got before failing, useful for
// diagnosing which page/listing broke it).
func StageBatch(ctx context.Context, st *store.Store, adapter Adapter) (batchID uuid.UUID, count int, err error) {
	batchID, err = st.CreateBatch(ctx, adapter.Source())
	if err != nil {
		return uuid.Nil, 0, fmt.Errorf("ingest.StageBatch: %w", err)
	}

	scrapeErr := adapter.ScrapeAll(ctx, func(l RawListing) error {
		p := domain.RawProduct{
			SourceSlug:  l.SourceSlug,
			SourceURL:   l.SourceURL,
			SourceSKU:   strOrNil(l.SourceSKU),
			Name:        l.Name,
			BrandRaw:    l.BrandRaw,
			PriceRaw:    l.PriceRaw,
			Description: l.Description,
			ImageURL:    strOrNil(l.ImageURL),
			CategoryRaw: l.CategoryRaw,
			RawData:     l.RawData,
		}
		if _, err := st.StageRawProduct(ctx, batchID, p); err != nil {
			return fmt.Errorf("staging %s: %w", l.SourceURL, err)
		}
		count++
		return nil
	})

	// Finish the batch even after a scrape error — a partial batch is still
	// useful staged data for a human to inspect (06-ADMIN.md §1.4's source
	// health view wants to show exactly this), not something to discard.
	if finishErr := st.FinishBatch(ctx, batchID, count); finishErr != nil {
		if scrapeErr != nil {
			return batchID, count, fmt.Errorf("ingest.StageBatch: scrape failed (%v) and finishing the batch also failed: %w", scrapeErr, finishErr)
		}
		return batchID, count, fmt.Errorf("ingest.StageBatch: %w", finishErr)
	}
	if scrapeErr != nil {
		return batchID, count, fmt.Errorf("ingest.StageBatch: %w", scrapeErr)
	}
	return batchID, count, nil
}

// LoadBatch reads back everything staged under one batch — what
// 06-ADMIN.md §1.4's source-health view and (eventually) the promotion
// gate both read from.
func LoadBatch(ctx context.Context, st *store.Store, batchID uuid.UUID) ([]domain.RawProduct, error) {
	return st.RawProductsForBatch(ctx, batchID)
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
