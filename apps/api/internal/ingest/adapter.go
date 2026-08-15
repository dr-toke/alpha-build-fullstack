// Package ingest is where scraper adapters live — 08-BUILD-ORDERS.md §1:
// "three adapters, not fourteen scrapers." A store is a harvest/scrapers/
// *.yaml config file plus whichever generic platform adapter it names
// (shopify.go, woocommerce.go); nothing here is store-specific Go code.
package ingest

import "context"

// RawListing is one scraper's unprocessed output — a single variant, before
// it becomes a domain.RawProduct (which additionally carries a BatchID and a
// DB-assigned ID that don't exist until StageBatch writes it). Field names
// intentionally mirror domain.RawProduct so the eventual conversion is a
// straight copy, not a remapping.
type RawListing struct {
	SourceSlug  string
	SourceURL   string
	SourceSKU   string
	Name        string
	BrandRaw    string
	PriceRaw    string
	Description string
	ImageURL    string
	CategoryRaw string
	RawData     map[string]any
}

// Adapter is what every platform-specific scraper implements —
// harvest/scrapers/cbdstore.yaml's shopify platform today, woocommerce and
// custom later. ScrapeAll streams every listing via onListing rather than
// returning a slice, so a 3,500-product aggregator doesn't have to be held
// in memory at once — StageBatch (M4, not yet built) is expected to write
// each one to the database as it arrives.
type Adapter interface {
	Source() string
	ScrapeAll(ctx context.Context, onListing func(RawListing) error) error
}
