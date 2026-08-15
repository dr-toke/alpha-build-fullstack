package ingest

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dr-toke/api/internal/resolve"
)

// TestScrapeAndClassifyLiveCBDStore is the actual end-to-end proof: real
// network scrape -> staged into real Postgres -> run through the REAL
// resolve pipeline (M1's harvested rules, not a mock) -> genuine
// cannabinoid/facet output on real products. Everything before this test
// (shopify_live_test.go, staging_live_test.go) proved data movement; this
// proves the "filtering" — the actual classification decisions M1 exists
// to make — against real catalogue text, not hand-written fixtures.
//
// Same INGEST_LIVE_TEST=1 + docker guard as the other live tests.
func TestScrapeAndClassifyLiveCBDStore(t *testing.T) {
	if os.Getenv("INGEST_LIVE_TEST") != "1" {
		t.Skip("set INGEST_LIVE_TEST=1 to run this against real cbdstore.in + a real Postgres container")
	}

	st, cleanup := startTestStore(t)
	defer cleanup()

	rs, err := resolve.LoadRuleSet("../../harvest/rules")
	if err != nil {
		t.Fatalf("resolve.LoadRuleSet: %v", err)
	}

	specs, err := LoadScraperSpec("../../harvest/scrapers")
	if err != nil {
		t.Fatalf("LoadScraperSpec: %v", err)
	}
	spec := specs["cbdstore"]
	if spec == nil {
		t.Fatal("cbdstore spec not found")
	}

	real := NewShopify(spec, "Mozilla/5.0 (compatible; DrTokeBot/0.1; +https://drtoke.in)", 500)
	adapter := &cappedAdapter{inner: real, max: 60}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	batchID, count, err := StageBatch(ctx, st, adapter)
	if err != nil {
		t.Fatalf("StageBatch: %v", err)
	}
	t.Logf("staged %d real listings, now classifying with the real M1 pipeline:", count)

	staged, err := LoadBatch(ctx, st, batchID)
	if err != nil {
		t.Fatal(err)
	}

	var (
		withCannabinoids int
		withForm         int
		lowConfidence    int
		serviceListings  int
	)

	for i, p := range staged {
		cb := resolve.ExtractCannabinoids(&rs.Cannabinoids, p.Name, p.Description)
		form := resolve.ResolveForm(&rs.Categories, p.Name, p.Description, p.CategoryRaw)
		route := resolve.ResolveRoute(&rs.Categories, p.Name, p.Description, p.CategoryRaw)

		if cb.ConcentrationType != "unknown" && cb.ConcentrationType != "hemp_seed" {
			withCannabinoids++
		}
		if form.Value != "" {
			withForm++
		}
		if form.Confidence < 0.85 {
			lowConfidence++
		}
		// The "Dr. Harshal Sawarkar" class of listing this whole
		// conversation has been using as the example — this is what
		// PROVES it's a real, present problem in the actual scraped
		// sample, not a hypothetical. compliance.json's service_listing
		// pattern isn't wired in yet (ADR-019 — M2 is a placeholder), so
		// this counts it via the same word list, informationally, to show
		// what compliance WILL catch once it exists.
		if resolve.ResolveExtract(p.Name, p.Description).Ambiguous && looksLikeServiceListing(p.Name) {
			serviceListings++
		}

		if i < 8 {
			t.Logf("  %-45s | cbd=%-6.0f thc=%-6.0f type=%-9s | form=%-10s route=%-8s conf=%.2f",
				truncate(p.Name, 45), cb.CBDMg, cb.THCMg, cb.ConcentrationType, form.Value, route.Value, form.Confidence)
		}
	}

	t.Logf("summary: %d/%d have real cannabinoid content, %d/%d resolved a form, %d/%d below the 0.85 publish-gate confidence, %d look like service listings (compliance's future job, not resolve's)",
		withCannabinoids, count, withForm, count, lowConfidence, count, serviceListings)

	if withForm == 0 {
		t.Error("zero listings resolved ANY form facet out of a real sample — the classifier is not working against live data")
	}
}

// looksLikeServiceListing is NOT the real compliance check
// (harvest/rules/compliance.json's service_listing pattern, which belongs
// in internal/compliance, M2, not built) — a narrow, local approximation
// just for this test's summary line, so the demo can show the problem is
// real without reaching into a package that doesn't exist yet.
func looksLikeServiceListing(name string) bool {
	lower := strings.ToLower(name)
	for _, kw := range []string{"dr.", "consultation", "therapist", "physician"} {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}
