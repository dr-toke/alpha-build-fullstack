package ingest

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dr-toke/api/internal/compliance"
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
	crs, err := compliance.LoadRuleSet("../../harvest/rules/compliance.json")
	if err != nil {
		t.Fatalf("compliance.LoadRuleSet: %v", err)
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
		// The REAL compliance check now (ADR-020) — not a heuristic. This
		// is what caught the actual "Dr. Harshal Sawarkar" listing: its
		// NAME alone doesn't match (see compliance_test.go's
		// TestEvaluateRealServiceListing), which is why Evaluate checks
		// CategoryRaw too — Shopify's product_type for that listing is
		// literally "Doctors Consultation".
		if !compliance.Evaluate(crs, p.Name, p.Description, p.CategoryRaw).Pass {
			serviceListings++
		}

		if i < 8 {
			t.Logf("  %-45s | cbd=%-6.0f thc=%-6.0f type=%-9s | form=%-10s route=%-8s conf=%.2f",
				truncate(p.Name, 45), cb.CBDMg, cb.THCMg, cb.ConcentrationType, form.Value, route.Value, form.Confidence)
		}
	}

	t.Logf("summary: %d/%d have real cannabinoid content, %d/%d resolved a form, %d/%d below the 0.85 publish-gate confidence, %d correctly caught as service listings by the REAL compliance.Evaluate (ADR-020)",
		withCannabinoids, count, withForm, count, lowConfidence, count, serviceListings)

	if withForm == 0 {
		t.Error("zero listings resolved ANY form facet out of a real sample — the classifier is not working against live data")
	}
	if serviceListings == 0 {
		t.Error("expected at least the known doctor-consultation listing to be caught in this sample — compliance.Evaluate may have regressed")
	}
}
