package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Shopify is the generic adapter for any Shopify storefront's public
// /products.json API — works for both single-brand direct stores and
// multi-brand aggregators (VendorMap set). Config-driven from a
// *ScraperSpec loaded out of harvest/scrapers/*.yaml — no store-specific Go
// code, per 08-BUILD-ORDERS.md §1: porting this once covers cbdstore now
// and the ~10 other Shopify stores catalogued in harvest/NOTES.md's
// deferred-stores list for free, later, as new YAML files.
//
// Faithful port of dr-toke-init/apps/api/internal/catalog/scraper/shopify.go
// — every quirk in harvest/scrapers/cbdstore.yaml's `quirks` list is
// preserved deliberately, most importantly the per-variant URL
// construction (see toRawListings' doc comment).
type Shopify struct {
	source      string
	domain      string
	brandSlug   string // set for single-brand direct stores; empty for aggregators
	vendorMap   map[string]string
	userAgent   string
	rateLimitMS int
	client      *http.Client
}

// NewShopify builds a Shopify adapter from a loaded spec. brandSlug is only
// meaningful for role="direct" stores (none harvested yet — see
// harvest/NOTES.md's deferred list); role="aggregator" stores (cbdstore)
// resolve brand per-product via spec.VendorMap instead.
func NewShopify(spec *ScraperSpec, userAgent string, rateLimitMS int) *Shopify {
	domain := strings.TrimPrefix(strings.TrimPrefix(spec.BaseURL, "https://"), "http://")
	domain = strings.TrimRight(domain, "/")

	s := &Shopify{
		source:      spec.Slug,
		domain:      domain,
		vendorMap:   spec.VendorMap,
		userAgent:   userAgent,
		rateLimitMS: rateLimitMS,
		client:      &http.Client{Timeout: 30 * time.Second},
	}
	if spec.Role == "direct" {
		s.brandSlug = spec.Slug
	}
	return s
}

func (s *Shopify) Source() string { return s.source }

// ── Shopify's own /products.json response shape ─────────────────────────────

type shopifyProduct struct {
	ID          int64            `json:"id"`
	Title       string           `json:"title"`
	Handle      string           `json:"handle"`
	BodyHTML    string           `json:"body_html"`
	Vendor      string           `json:"vendor"`
	ProductType string           `json:"product_type"`
	Tags        []string         `json:"tags"`
	Variants    []shopifyVariant `json:"variants"`
	Images      []shopifyImage   `json:"images"`
}

type shopifyVariant struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Price string `json:"price"`
	SKU   string `json:"sku"`
	// Available is Shopify's own computed in-stock boolean, present on
	// every public /products.json response. inventory_quantity/
	// inventory_policy are ALSO sometimes present (harvest/scrapers/
	// cbdstore.yaml's original field mapping assumed they always would be)
	// but Shopify commonly hides them by default now — confirmed live
	// against cbdstore.in's actual current API response, which returns
	// "available": true with NO inventory_quantity/inventory_policy keys at
	// all. Available is the field that's actually reliably present; the
	// other two are kept as a fallback for stores still exposing them.
	Available         bool   `json:"available"`
	InventoryQuantity int    `json:"inventory_quantity"`
	InventoryPolicy   string `json:"inventory_policy"`
}

type shopifyImage struct {
	Src string `json:"src"`
}

type shopifyPage struct {
	Products []shopifyProduct `json:"products"`
}

// ScrapeAll pages through /products.json?limit=250 until a page returns
// fewer than 250 products (Shopify's own "last page" signal — no total
// count is exposed by this endpoint), rate-limited between pages.
func (s *Shopify) ScrapeAll(ctx context.Context, onListing func(RawListing) error) error {
	page := 1
	total := 0

	for {
		url := fmt.Sprintf("https://%s/products.json?limit=250&page=%d", s.domain, page)
		products, err := s.fetchPage(ctx, url)
		if err != nil {
			return fmt.Errorf("shopify %s page %d: %w", s.source, page, err)
		}
		if len(products) == 0 {
			break
		}

		for _, p := range products {
			for _, raw := range s.toRawListings(p) {
				if raw.Name == "" {
					continue
				}
				if err := onListing(raw); err != nil {
					return err
				}
				total++
			}
		}

		slog.Debug("shopify page done", "source", s.source, "page", page, "products", len(products))

		if len(products) < 250 {
			break
		}
		page++

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(s.rateLimitMS) * time.Millisecond):
		}
	}

	slog.Info("shopify scrape done", "source", s.source, "total_variants", total)
	return nil
}

// toRawListings emits ONE RawListing PER VARIANT, not per product — the
// single most important behaviour ported from the harvested source.
// Shopify lists every size of a product as one entry sharing a single
// handle/URL; without a per-variant URL, every variant of a product
// upserts onto the same (source_slug, source_url) row via
// store.UpsertListing's UNIQUE constraint and the LAST one scraped wins — a
// 15ml/4500mg listing ends up showing the 3ml/₹999 price. source_url below
// (?variant={id}) is also Shopify's own canonical deep-link for that exact
// size, so it opens the right variant for a buyer too, not just a
// deduplication trick.
func (s *Shopify) toRawListings(p shopifyProduct) []RawListing {
	if len(p.Variants) == 0 {
		return nil // draft/hidden products occasionally have no variants at all
	}

	brandSlug := s.brandSlug
	if brandSlug == "" && s.vendorMap != nil {
		// Two-tier: exact vendor string first (the map deliberately has
		// both "Magiccann" and "Magiccann " — a trailing-space variant seen
		// in real vendor data), THEN a lower-cased fallback. Collapsing
		// this to one case-insensitive lookup would silently drop that
		// trailing-space entry's meaning if it's ever the only key that
		// matches a given store's data.
		brandSlug = s.vendorMap[p.Vendor]
		if brandSlug == "" {
			brandSlug = s.vendorMap[strings.ToLower(p.Vendor)]
		}
	}
	if brandSlug == "" {
		// Unmapped vendor — NOT an error. slugify and let the (not-yet-
		// built, M2) compliance filter queue it as unknown_brand for human
		// review, exactly as harvest/rules/compliance.json's unknown_brand
		// rule expects. A scraper's job is to observe, not to gatekeep.
		brandSlug = slugifyVendor(p.Vendor)
	}

	imageURL := ""
	if len(p.Images) > 0 {
		imageURL = p.Images[0].Src
	}

	desc := stripHTML(p.BodyHTML)
	tags := strings.Join(p.Tags, ", ")
	rawData := map[string]any{
		"shopify_id":   p.ID,
		"handle":       p.Handle,
		"product_type": p.ProductType,
		"tags":         p.Tags,
		"vendor":       p.Vendor,
	}

	var out []RawListing
	for _, v := range p.Variants {
		name := p.Title
		if v.Title != "" && v.Title != "Default Title" {
			name = p.Title + " - " + v.Title
		}

		rd := copyMap(rawData)
		rd["variant_id"] = v.ID
		rd["variant_title"] = v.Title
		rd["in_stock"] = v.Available || v.InventoryQuantity > 0 || v.InventoryPolicy == "continue"

		out = append(out, RawListing{
			SourceSlug:  s.source,
			SourceURL:   fmt.Sprintf("https://%s/products/%s?variant=%d", s.domain, p.Handle, v.ID),
			SourceSKU:   v.SKU,
			Name:        name,
			BrandRaw:    brandSlug,
			PriceRaw:    "₹" + v.Price, // normaliser strips ₹/commas — don't pre-format here
			Description: desc + "\n\nTags: " + tags,
			ImageURL:    imageURL,
			CategoryRaw: p.ProductType,
			RawData:     rd,
		})
	}
	return out
}

func (s *Shopify) fetchPage(ctx context.Context, url string) ([]shopifyProduct, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var page shopifyPage
	if err := json.Unmarshal(body, &page); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}
	return page.Products, nil
}

var reHTMLTags = regexp.MustCompile(`<[^>]+>`)
var reSpaces = regexp.MustCompile(`\s+`)

// stripHTML is a blunt tag-strip, not a real HTML parser — matches the
// harvested source exactly, "fine for Shopify's fairly clean body_html"
// per harvest/scrapers/cbdstore.yaml's own quirks note.
func stripHTML(s string) string {
	s = reHTMLTags.ReplaceAllString(s, " ")
	s = reSpaces.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

var reNonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugifyVendor(vendor string) string {
	s := strings.ToLower(vendor)
	s = reNonAlnum.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func copyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
