package ingest

import (
	"context"
	"os/exec"
	"testing"

	"github.com/dr-toke/api/internal/compliance"
	"github.com/dr-toke/api/internal/domain"
	"github.com/dr-toke/api/internal/resolve"
)

// TestPromote proves the full staging -> live pipeline against a real
// Postgres container, using synthetic-but-realistic listings (not a live
// network scrape — shopify_live_test.go/staging_live_test.go already prove
// the scrape+stage half against real cbdstore.in; this proves the
// classify+dedup+persist half, which needs to be deterministic and fast
// enough to run every time, not gated behind INGEST_LIVE_TEST).
func TestPromote(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}

	st, cleanup := startTestStore(t)
	defer cleanup()
	ctx := context.Background()

	rs, err := resolve.LoadRuleSet("../../harvest/rules")
	if err != nil {
		t.Fatalf("resolve.LoadRuleSet: %v", err)
	}
	crs, err := compliance.LoadRuleSet("../../harvest/rules/compliance.json")
	if err != nil {
		t.Fatalf("compliance.LoadRuleSet: %v", err)
	}

	slug := "promote-test"
	if _, err := st.Pool.Exec(ctx,
		`INSERT INTO scrape_sources (slug, name, platform, base_url) VALUES ($1,$2,'shopify','https://example.com')`,
		slug, "Promote Test Source"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Pool.Exec(ctx,
		`INSERT INTO brands (slug, name, verified) VALUES ('boheco', 'BOHECO', true)
		 ON CONFLICT (slug) DO UPDATE SET verified = true`); err != nil {
		t.Fatal(err)
	}

	t.Run("a real product promotes to a publishable, priced, faceted cluster", func(t *testing.T) {
		batchID, err := st.CreateBatch(ctx, slug)
		if err != nil {
			t.Fatal(err)
		}
		_, err = st.StageRawProduct(ctx, batchID, domain.RawProduct{
			SourceSlug:  slug,
			SourceURL:   "https://example.com/products/cbd-oil-500mg?variant=1",
			Name:        "BOHECO CBD Oil 500mg - 30ml",
			BrandRaw:    "boheco",
			PriceRaw:    "₹1999.00",
			Description: "Full spectrum CBD oil in an MCT carrier, 30ml bottle, 500mg CBD, sublingual drops.",
			CategoryRaw: "Tinctures",
			RawData:     map[string]any{"in_stock": true},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := st.FinishBatch(ctx, batchID, 1); err != nil {
			t.Fatal(err)
		}
		decision, err := DecideGate(ctx, st, batchID)
		if err != nil {
			t.Fatal(err)
		}
		if !decision.Approved {
			t.Fatalf("gate rejected: %s", decision.Reason)
		}

		result, err := Promote(ctx, st, rs, crs, batchID)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Errors) != 0 {
			t.Fatalf("promote errors: %v", result.Errors)
		}
		if result.Promoted != 1 {
			t.Fatalf("got Promoted=%d, want 1", result.Promoted)
		}
		if result.FilteredCompliance != 0 {
			t.Fatalf("got FilteredCompliance=%d, want 0", result.FilteredCompliance)
		}

		var listingCount int
		if err := st.Pool.QueryRow(ctx,
			`SELECT count(*) FROM product_listings WHERE source_slug = $1`, slug,
		).Scan(&listingCount); err != nil {
			t.Fatal(err)
		}
		if listingCount != 1 {
			t.Fatalf("got %d product_listings rows, want 1", listingCount)
		}

		row := st.Pool.QueryRow(ctx,
			`SELECT cbd_mg, best_price_paise, publishable, rank_score
			 FROM product_clusters pc
			 JOIN product_listings pl ON pl.cluster_id = pc.id
			 WHERE pl.source_slug = $1`, slug)
		var cbdMg *float64
		var pricePaise *int64
		var publishable bool
		var rankScore *float64
		if err := row.Scan(&cbdMg, &pricePaise, &publishable, &rankScore); err != nil {
			t.Fatal(err)
		}
		if cbdMg == nil || *cbdMg != 500 {
			t.Errorf("got cbd_mg=%v, want 500", cbdMg)
		}
		if pricePaise == nil || *pricePaise != 199900 {
			t.Errorf("got best_price_paise=%v, want 199900 (₹1999.00)", pricePaise)
		}
		if rankScore == nil {
			t.Error("rank_score is nil for a fully-priced, fully-classified product")
		}

		var facetCount int
		if err := st.Pool.QueryRow(ctx,
			`SELECT count(*) FROM product_facets pf
			 JOIN product_listings pl ON pl.cluster_id = pf.cluster_id
			 WHERE pl.source_slug = $1`, slug,
		).Scan(&facetCount); err != nil {
			t.Fatal(err)
		}
		if facetCount == 0 {
			t.Error("no product_facets rows written for a real product")
		}
	})

	t.Run("a service listing is filtered, never reaches product_listings or product_clusters", func(t *testing.T) {
		batchID, err := st.CreateBatch(ctx, slug)
		if err != nil {
			t.Fatal(err)
		}
		_, err = st.StageRawProduct(ctx, batchID, domain.RawProduct{
			SourceSlug:  slug,
			SourceURL:   "https://example.com/products/dr-consultation?variant=1",
			Name:        "Dr. Harshal Sawarkar – BAMS Ayurvedic Physician | Vijaya-Based Medicine",
			BrandRaw:    "boheco",
			PriceRaw:    "₹500.00",
			Description: "Book a consultation.",
			CategoryRaw: "Doctors Consultation",
			RawData:     map[string]any{"in_stock": true},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := st.FinishBatch(ctx, batchID, 1); err != nil {
			t.Fatal(err)
		}
		decision, err := DecideGate(ctx, st, batchID)
		if err != nil {
			t.Fatal(err)
		}
		if !decision.Approved {
			t.Fatalf("gate rejected: %s", decision.Reason)
		}

		result, err := Promote(ctx, st, rs, crs, batchID)
		if err != nil {
			t.Fatal(err)
		}
		if result.Promoted != 0 {
			t.Errorf("got Promoted=%d, want 0 for a service listing", result.Promoted)
		}
		if result.FilteredCompliance != 1 {
			t.Errorf("got FilteredCompliance=%d, want 1", result.FilteredCompliance)
		}

		var count int
		if err := st.Pool.QueryRow(ctx,
			`SELECT count(*) FROM product_listings WHERE source_url = 'https://example.com/products/dr-consultation?variant=1'`,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Errorf("service listing reached product_listings — compliance filter did not hold")
		}
	})

	t.Run("a pack-quantity listing gets no ₹/mg pricing instead of a wildly wrong one", func(t *testing.T) {
		batchID, err := st.CreateBatch(ctx, slug)
		if err != nil {
			t.Fatal(err)
		}
		// The exact real cbdstore.in case: "50mg" here is PER CAPSULE, not
		// the pack total (4500mg for 90 capsules) — nothing in the text
		// states the real total, so the extractor (faithfully ported, no
		// unit-multiplication logic) has no way to reconcile it. Dividing
		// price by the raw 50mg would produce a ~90x-too-high ₹/mg.
		_, err = st.StageRawProduct(ctx, batchID, domain.RawProduct{
			SourceSlug:  slug,
			SourceURL:   "https://example.com/products/cannamed-capsules?variant=1",
			Name:        "CannaMed- Medical Cannabis Capsules 50mg (90 Capsules)",
			BrandRaw:    "boheco",
			PriceRaw:    "₹3150.00",
			Description: "CannaMed capsules with CBD, THC. Each capsule is packed with carefully measured doses of high-quality cannabis.",
			CategoryRaw: "Capsules",
			RawData:     map[string]any{"in_stock": true},
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := st.FinishBatch(ctx, batchID, 1); err != nil {
			t.Fatal(err)
		}
		if _, err := DecideGate(ctx, st, batchID); err != nil {
			t.Fatal(err)
		}
		result, err := Promote(ctx, st, rs, crs, batchID)
		if err != nil {
			t.Fatal(err)
		}
		if result.Promoted != 1 {
			t.Fatalf("got Promoted=%d, want 1 (errors: %v)", result.Promoted, result.Errors)
		}

		var cbdMg, thcMg, cbdPerMg, thcPerMg, bestPerMg, rankScore *float64
		var pricePaise *int64
		row := st.Pool.QueryRow(ctx,
			`SELECT cbd_mg, thc_mg, cbd_price_per_mg, thc_price_per_mg, best_price_per_mg, rank_score, best_price_paise
			 FROM product_clusters pc
			 JOIN product_listings pl ON pl.cluster_id = pc.id
			 WHERE pl.source_url = 'https://example.com/products/cannamed-capsules?variant=1'`)
		if err := row.Scan(&cbdMg, &thcMg, &cbdPerMg, &thcPerMg, &bestPerMg, &rankScore, &pricePaise); err != nil {
			t.Fatal(err)
		}

		if cbdPerMg != nil || thcPerMg != nil || bestPerMg != nil || rankScore != nil {
			t.Errorf("₹/mg-derived fields not suppressed: cbd_per_mg=%v thc_per_mg=%v best_per_mg=%v rank_score=%v",
				cbdPerMg, thcPerMg, bestPerMg, rankScore)
		}
		if pricePaise == nil || *pricePaise != 315000 {
			t.Errorf("best_price_paise = %v, want 315000 — the real price should still be recorded", pricePaise)
		}
		if cbdMg == nil {
			t.Error("cbd_mg was nulled out too — only the derived ₹/mg fields should be suppressed, not the raw dosing info")
		}
	})

	t.Run("promoting a non-approved batch is rejected", func(t *testing.T) {
		batchID, err := st.CreateBatch(ctx, slug)
		if err != nil {
			t.Fatal(err)
		}
		// Deliberately not finished/gated — still 'running'.
		if _, err := Promote(ctx, st, rs, crs, batchID); err == nil {
			t.Error("expected an error promoting a batch that was never gate-approved")
		}
	})
}
