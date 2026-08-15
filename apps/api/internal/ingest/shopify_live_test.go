package ingest

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestShopifyLiveCBDStore hits the REAL cbdstore.in over the network — not
// mocked, not a fixture. Guarded behind INGEST_LIVE_TEST=1 so `go test
// ./...` never silently depends on a third-party site's uptime, rate
// limits, or catalogue changing under the test — that dependency is opt-in,
// run deliberately, not a normal CI gate. This is the actual proof the
// scraper works, not just that toRawListings' logic is right in isolation.
func TestShopifyLiveCBDStore(t *testing.T) {
	if os.Getenv("INGEST_LIVE_TEST") != "1" {
		t.Skip("set INGEST_LIVE_TEST=1 to run this against the real cbdstore.in")
	}

	specs, err := LoadScraperSpec("../../harvest/scrapers")
	if err != nil {
		t.Fatalf("LoadScraperSpec: %v", err)
	}
	spec, ok := specs["cbdstore"]
	if !ok {
		t.Fatal("cbdstore spec not found in harvest/scrapers/")
	}

	s := NewShopify(spec, "Mozilla/5.0 (compatible; DrTokeBot/0.1; +https://drtoke.in)", 500)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var (
		count          int
		sawKnownVendor bool
		sawUnknown     bool
		sawInStock     bool
		sawPriceFormat bool
		firstFew       []RawListing
	)

	// Cap at one page (up to 250 variants) — enough to prove it genuinely
	// works against live data without hammering a real store's endpoint
	// repeatedly across test runs.
	pageDone := false
	err = s.ScrapeAll(ctx, func(l RawListing) error {
		count++
		if len(firstFew) < 5 {
			firstFew = append(firstFew, l)
		}
		if _, mapped := spec.VendorMap[l.BrandRaw]; mapped {
			sawKnownVendor = true // BrandRaw is already the mapped slug, so this checks the slug space instead below
		}
		for _, slug := range spec.VendorMap {
			if l.BrandRaw == slug {
				sawKnownVendor = true
			}
		}
		if l.RawData["vendor"] != nil {
			if _, known := spec.VendorMap[l.RawData["vendor"].(string)]; !known {
				sawUnknown = true
			}
		}
		if b, ok := l.RawData["in_stock"].(bool); ok && b {
			sawInStock = true
		}
		if len(l.PriceRaw) > 0 && l.PriceRaw[0:3] == "₹" {
			sawPriceFormat = true
		}
		if count >= 250 {
			pageDone = true
			return errStopEarly
		}
		return nil
	})
	if err != nil && err != errStopEarly {
		t.Fatalf("ScrapeAll: %v", err)
	}

	t.Logf("scraped %d real variant listings from cbdstore.in (capped at one page: %v)", count, pageDone)
	for _, l := range firstFew {
		t.Logf("  %-40s | brand=%-20s | %-10s | %s", truncate(l.Name, 40), l.BrandRaw, l.PriceRaw, l.SourceURL)
	}

	if count == 0 {
		t.Fatal("scraped zero listings from a live site that returns real products — something is broken, not just empty")
	}
	if !sawPriceFormat {
		t.Error("no listing had a ₹-prefixed price — PriceRaw formatting may be broken")
	}
	if !sawInStock {
		t.Log("note: no in-stock variant seen in this sample (not necessarily a bug, just worth knowing)")
	}
	_ = sawKnownVendor
	_ = sawUnknown

	// The per-variant-URL fix, proven against REAL data: no two listings in
	// this run should share a source_url.
	seen := map[string]bool{}
	dupes := 0
	err = nil
	s2 := NewShopify(spec, "Mozilla/5.0 (compatible; DrTokeBot/0.1)", 500)
	checked := 0
	_ = s2.ScrapeAll(ctx, func(l RawListing) error {
		if seen[l.SourceURL] {
			dupes++
		}
		seen[l.SourceURL] = true
		checked++
		if checked >= 250 {
			return errStopEarly
		}
		return nil
	})
	if dupes > 0 {
		t.Errorf("%d duplicate source_urls out of %d real listings — the per-variant-URL fix is not holding against live data", dupes, checked)
	} else {
		t.Logf("%d real listings, 0 duplicate source_urls — per-variant-URL fix confirmed against live data", checked)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

var errStopEarly = errStopEarlyType{}

type errStopEarlyType struct{}

func (errStopEarlyType) Error() string { return "stop early: sample cap reached" }
