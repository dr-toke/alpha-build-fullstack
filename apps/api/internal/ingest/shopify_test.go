package ingest

import "testing"

func testSpec() *ScraperSpec {
	return &ScraperSpec{
		Slug: "cbdstore", BaseURL: "https://cbdstore.in", Platform: "shopify",
		Role: "aggregator",
		VendorMap: map[string]string{
			"BOHECO": "boheco", "Boheco": "boheco",
			"Magiccann": "magiccann", "Magiccann ": "magiccann",
		},
	}
}

func TestToRawListingsPerVariantURL(t *testing.T) {
	// The single most important behaviour this package ports —
	// harvest/scrapers/cbdstore.yaml's CRITICAL quirk.
	s := NewShopify(testSpec(), "test-agent", 1)
	p := shopifyProduct{
		ID: 1, Title: "CBD Oil", Handle: "cbd-oil", Vendor: "BOHECO",
		BodyHTML: "<p>Great oil</p>", Tags: []string{"oil", "wellness"},
		Variants: []shopifyVariant{
			{ID: 100, Title: "15ml", Price: "999.00", InventoryQuantity: 5},
			{ID: 200, Title: "30ml", Price: "1799.00", InventoryQuantity: 0, InventoryPolicy: "continue"},
		},
	}

	got := s.toRawListings(p)
	if len(got) != 2 {
		t.Fatalf("got %d listings, want 2 (one per variant)", len(got))
	}
	if got[0].SourceURL == got[1].SourceURL {
		t.Fatal("both variants got the same source_url — this is exactly the ₹0.28/mg-style bug the harvest note warns about")
	}
	if got[0].SourceURL != "https://cbdstore.in/products/cbd-oil?variant=100" {
		t.Errorf("SourceURL = %q, want the ?variant=100 deep link", got[0].SourceURL)
	}
	if got[0].Name != "CBD Oil - 15ml" {
		t.Errorf("Name = %q, want title+variant title joined", got[0].Name)
	}
	if got[0].PriceRaw != "₹999.00" {
		t.Errorf("PriceRaw = %q, want ₹-prefixed raw Shopify decimal, unmodified", got[0].PriceRaw)
	}
}

func TestToRawListingsDefaultTitleNotAppended(t *testing.T) {
	s := NewShopify(testSpec(), "test-agent", 1)
	p := shopifyProduct{
		ID: 1, Title: "Single Variant Product", Handle: "svp", Vendor: "BOHECO",
		Variants: []shopifyVariant{{ID: 1, Title: "Default Title", Price: "500.00", InventoryQuantity: 1}},
	}
	got := s.toRawListings(p)
	if got[0].Name != "Single Variant Product" {
		t.Errorf("Name = %q, want the bare title (Default Title must not be appended)", got[0].Name)
	}
}

func TestToRawListingsZeroVariantsSkipped(t *testing.T) {
	s := NewShopify(testSpec(), "test-agent", 1)
	got := s.toRawListings(shopifyProduct{ID: 1, Title: "Draft Product", Handle: "draft"})
	if got != nil {
		t.Errorf("got %v, want nil for a product with zero variants (draft/hidden products)", got)
	}
}

func TestVendorMapTwoTier(t *testing.T) {
	spec := testSpec()
	s := NewShopify(spec, "test-agent", 1)

	cases := []struct {
		vendor string
		want   string
	}{
		{"BOHECO", "boheco"},                               // exact match
		{"Magiccann ", "magiccann"},                        // exact match, trailing space preserved as its own key
		{"boheco", "boheco"},                               // lower-cased fallback (spec has "BOHECO", not "boheco")
		{"Totally Unknown Brand", "totally-unknown-brand"}, // falls through to slugify, not an error
	}
	for _, c := range cases {
		p := shopifyProduct{ID: 1, Title: "X", Handle: "x", Vendor: c.vendor,
			Variants: []shopifyVariant{{ID: 1, Price: "1.00", InventoryQuantity: 1}}}
		got := s.toRawListings(p)
		if len(got) == 0 || got[0].BrandRaw != c.want {
			t.Errorf("vendor %q: brand = %v, want %q", c.vendor, got, c.want)
		}
	}
}

func TestInStockRawDataFlag(t *testing.T) {
	s := NewShopify(testSpec(), "test-agent", 1)
	p := shopifyProduct{ID: 1, Title: "X", Handle: "x", Vendor: "BOHECO", Variants: []shopifyVariant{
		{ID: 1, Available: true},                                                     // the field real Shopify stores actually expose (cbdstore.in confirmed live)
		{ID: 2, Available: false, InventoryQuantity: 5},                              // fallback: quantity > 0
		{ID: 3, Available: false, InventoryQuantity: 0, InventoryPolicy: "continue"}, // fallback: backorder allowed
		{ID: 4, Available: false, InventoryQuantity: 0, InventoryPolicy: "deny"},     // out of stock on every signal
	}}
	got := s.toRawListings(p)
	want := []bool{true, true, true, false}
	for i, w := range want {
		if got[i].RawData["in_stock"] != w {
			t.Errorf("variant %d: in_stock = %v, want %v", i, got[i].RawData["in_stock"], w)
		}
	}
}

func TestStripHTML(t *testing.T) {
	got := stripHTML("<p>Hello <b>world</b></p>\n\n<div>  extra   space</div>")
	want := "Hello world extra space"
	if got != want {
		t.Errorf("stripHTML = %q, want %q", got, want)
	}
}

func TestSlugifyVendor(t *testing.T) {
	if got := slugifyVendor("Some Brand & Co."); got != "some-brand-co" {
		t.Errorf("slugifyVendor = %q, want some-brand-co", got)
	}
}
