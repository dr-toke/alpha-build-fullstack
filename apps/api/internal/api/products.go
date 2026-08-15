package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/dr-toke/api/internal/domain"
	"github.com/dr-toke/api/internal/resolve"
	"github.com/dr-toke/api/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handlers holds what every handler needs — just the store for this
// minimal build (no auth, no rate limiter, no job enqueuer yet).
type Handlers struct {
	Store *store.Store
}

// ApiBrandSummary / ApiCannabinoids / ApiListing / ApiProduct /
// ProductListResponse are Go mirrors of apps/web/src/lib/api/catalog.ts's
// TypeScript interfaces of the same names, field for field — that file is
// the real contract (ported verbatim from the trusted collaborator's
// design), not 05-API-REFERENCE.md's aspirational shape. Field order and
// naming here follow catalog.ts exactly so a future OpenAPI/codegen pass
// has nothing to reconcile. See API-DECISIONS.md.
type ApiBrandSummary struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Ayush    bool   `json:"ayush"`
	FSSAI    bool   `json:"fssai"`
	Verified bool   `json:"verified"`
}

type ApiCannabinoids struct {
	CBDMg               *float64 `json:"cbd_mg"`
	THCMg               *float64 `json:"thc_mg"`
	TotalCannabinoidsMg *float64 `json:"total_cannabinoids_mg"`
	ConcentrationType   string   `json:"concentration_type"`
}

type ApiListing struct {
	ListingID    string  `json:"listing_id,omitempty"`
	Source       string  `json:"source"`
	PriceINR     float64 `json:"price_inr"`
	SourceURL    string  `json:"source_url"`
	AffiliateURL *string `json:"affiliate_url"`
	InStock      bool    `json:"in_stock"`
}

type ApiProduct struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Slug                 string          `json:"slug"`
	Brand                ApiBrandSummary `json:"brand"`
	Category             string          `json:"category"`
	Categories           []string        `json:"categories"`
	ExtractType          string          `json:"extract_type"`
	CarrierOil           string          `json:"carrier_oil"`
	Cannabinoids         ApiCannabinoids `json:"cannabinoids"`
	VolumeML             *float64        `json:"volume_ml"`
	WeightG              *float64        `json:"weight_g"`
	BestPriceINR         *float64        `json:"best_price_inr"`
	BestPricePerMg       *float64        `json:"best_price_per_mg"`
	CBDPricePerMg        *float64        `json:"cbd_price_per_mg"`
	THCPricePerMg        *float64        `json:"thc_price_per_mg"`
	PricePerMgBasis      string          `json:"price_per_mg_basis"`
	BestListing          *ApiListing     `json:"best_listing"`
	OtherListings        []ApiListing    `json:"other_listings"`
	ImageURL             *string         `json:"image_url"`
	COAAvailable         bool            `json:"coa_available"`
	InStock              bool            `json:"in_stock"`
	PrescriptionRequired bool            `json:"prescription_required"`
	FirstSeenAt          time.Time       `json:"first_seen_at"`
}

type ProductListResponse struct {
	Products []ApiProduct `json:"products"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PerPage  int          `json:"per_page"`
}

// ListProducts handles GET /api/products — matches CatalogGrid.svelte's
// query builder exactly: category, extract, brand, basis, verified, sort
// (default "value", NOT "new" — CatalogGrid.svelte's own client-side
// default when the URL carries no sort param; see API-DECISIONS.md), page,
// limit.
func (h *Handlers) ListProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	filter := store.ClusterFilter{
		PublishableOnly: true, // public endpoint — 03-DOMAIN-MODEL.md §2's gate is not optional here
		Category:        q.Get("category"),
		Extract:         q.Get("extract"),
		VerifiedOnly:    q.Get("verified") == "true",
		Page:            parsePage(r),
		Limit:           parseLimit(r),
	}

	switch q.Get("basis") {
	case "cbd", "thc":
		filter.Basis = q.Get("basis")
	}

	switch q.Get("sort") {
	case store.SortNew:
		filter.Sort = store.SortNew
	case store.SortPrice:
		filter.Sort = store.SortPrice
	default:
		filter.Sort = store.SortValue
	}

	if brandSlug := q.Get("brand"); brandSlug != "" {
		brand, err := h.Store.BrandBySlug(ctx, brandSlug)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				// An unknown brand slug is a valid filter that just matches
				// nothing — not an error.
				WriteJSON(w, http.StatusOK, ProductListResponse{Products: []ApiProduct{}, Total: 0, Page: filter.Page, PerPage: filter.Limit})
				return
			}
			writeUnavailable(w, "could not look up brand")
			return
		}
		filter.BrandID = &brand.ID
	}

	clusters, err := h.Store.ListClusters(ctx, filter)
	if err != nil {
		writeUnavailable(w, "could not list products")
		return
	}
	total, err := h.Store.CountClusters(ctx, filter)
	if err != nil {
		writeUnavailable(w, "could not count products")
		return
	}

	products := make([]ApiProduct, 0, len(clusters))
	for _, c := range clusters {
		p, err := buildApiProduct(ctx, h.Store, c)
		if err != nil {
			writeUnavailable(w, "could not assemble product")
			return
		}
		products = append(products, p)
	}

	WriteJSON(w, http.StatusOK, ProductListResponse{
		Products: products,
		Total:    total,
		Page:     filter.Page,
		PerPage:  filter.Limit,
	})
}

// GetProduct handles GET /api/products/{id} — wrapped as { product: ... }
// per product/+page.svelte's `Remote<{ product: ApiProduct }>()`.
// 05-API-REFERENCE.md's "Returns { moved_to } if merged" / 02-FRONTEND-CONTRACT.md
// §4's "200, never 404" for a merged cluster both hold regardless of the
// wrapper key — the frontend's own product page doesn't currently branch
// on moved_to at all (a gap in the shipped code, not this backend's
// problem to silently paper over), but the contract is honored here so a
// future frontend change can rely on it.
func (h *Handlers) GetProduct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeInvalidFilter(w, "invalid product id")
		return
	}

	cluster, err := h.Store.ClusterByID(ctx, id)
	if err != nil {
		var moved *domain.ClusterMovedError
		if errors.As(err, &moved) {
			WriteJSON(w, http.StatusOK, map[string]any{"moved_to": moved.NewID})
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			writeNotFound(w, "no such product")
			return
		}
		writeUnavailable(w, "could not load product")
		return
	}

	product, err := buildApiProduct(ctx, h.Store, *cluster)
	if err != nil {
		writeUnavailable(w, "could not assemble product")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]ApiProduct{"product": product})
}

func buildApiProduct(ctx context.Context, st *store.Store, c domain.ProductCluster) (ApiProduct, error) {
	facetRows, err := st.FacetsFor(ctx, c.ID)
	if err != nil {
		return ApiProduct{}, fmt.Errorf("facets: %w", err)
	}
	facets := make(map[string]string, len(facetRows))
	var form domain.FormValue
	var route domain.RouteValue
	for _, f := range facetRows {
		facets[string(f.Facet)] = f.Value
		switch f.Facet {
		case domain.FacetForm:
			form = domain.FormValue(f.Value)
		case domain.FacetRoute:
			route = domain.RouteValue(f.Value)
		}
	}
	category, secondary := resolve.LegacyCategory(form, route, c.ConcentrationType)

	listingRows, err := st.ListingsForCluster(ctx, c.ID)
	if err != nil {
		return ApiProduct{}, fmt.Errorf("listings: %w", err)
	}

	brand, err := buildBrandSummary(ctx, st, c.BrandID, listingRows)
	if err != nil {
		return ApiProduct{}, fmt.Errorf("brand: %w", err)
	}

	// ListingsForCluster orders in_stock DESC, price_paise ASC — so [0] is
	// the cheapest in-stock listing if one exists, otherwise the cheapest
	// listing overall. best_listing is only ever the FORMER (catalog.ts:
	// "Currently out of stock across tracked sources" when nil); everything
	// else — including out-of-stock rows — goes in other_listings, matching
	// product/+page.svelte's own rendering of `l.in_stock ? '' : '(out of
	// stock)'` inside that list.
	var best *ApiListing
	var others []ApiListing
	var bestPriceINR *float64
	inStock := false
	if len(listingRows) > 0 {
		bp := float64(listingRows[0].PricePaise) / 100
		bestPriceINR = &bp
		if listingRows[0].InStock {
			l := toApiListing(listingRows[0])
			best = &l
			inStock = true
			for _, lr := range listingRows[1:] {
				others = append(others, toApiListing(lr))
			}
		} else {
			for _, lr := range listingRows {
				others = append(others, toApiListing(lr))
			}
		}
	}
	if others == nil {
		others = []ApiListing{}
	}

	basis := ""
	if c.PricePerMgBasis != nil {
		basis = *c.PricePerMgBasis
	}

	// Interim direct hotlink of the raw scraped image — NOT the real answer
	// (02-FRONTEND-CONTRACT.md §6: "Raw MinIO hostnames... proxy through
	// /media/*" applies just as much to hotlinking a source store directly;
	// the real fix is M6's fetch/transcode/hash pipeline, still pending).
	// For a first live beta, showing the store's own product photo is a
	// large, obvious improvement over every card reading "No image" — the
	// data was already being scraped and stored (ProductListing.ImageURLRaw)
	// and simply wasn't surfaced here. Flagged, not silently left broken.
	var imageURL *string
	for _, lr := range listingRows {
		if lr.ImageURLRaw != nil && *lr.ImageURLRaw != "" {
			imageURL = lr.ImageURLRaw
			break
		}
	}

	return ApiProduct{
		ID:          c.ID.String(),
		Name:        c.Name,
		Slug:        slugify(c.Name),
		Brand:       brand,
		Category:    category,
		Categories:  resolve.LegacyCategories(category, secondary),
		ExtractType: facets[string(domain.FacetExtract)],
		CarrierOil:  facets[string(domain.FacetCarrier)],
		Cannabinoids: ApiCannabinoids{
			CBDMg:               c.CBDMg,
			THCMg:               c.THCMg,
			TotalCannabinoidsMg: c.TotalCannabinoidsMg,
			ConcentrationType:   string(c.ConcentrationType),
		},
		VolumeML:             c.VolumeML,
		WeightG:              c.WeightG,
		BestPriceINR:         bestPriceINR,
		BestPricePerMg:       c.BestPricePerMg,
		CBDPricePerMg:        c.CBDPricePerMg,
		THCPricePerMg:        c.THCPricePerMg,
		PricePerMgBasis:      basis,
		BestListing:          best,
		OtherListings:        others,
		ImageURL:             imageURL,
		COAAvailable:         c.COAAvailable,
		InStock:              inStock,
		PrescriptionRequired: c.PrescriptionRequired,
		FirstSeenAt:          c.FirstSeenAt,
	}, nil
}

// buildBrandSummary is ALWAYS non-nil in the response — product/+page.svelte
// and ProductCard.svelte read p.brand.verified/.slug/.name unconditionally,
// with no null guard (ApiBrandSummary is a non-optional field in
// catalog.ts's ApiProduct). A cluster with no matched brands row (an
// unmapped vendor slugifyVendor'd by shopify.go — internal/ingest's own
// comment: "let the... compliance filter queue it as unknown_brand for
// human review") falls back to the raw scraped brand text from its first
// listing rather than sending null and crashing the frontend.
func buildBrandSummary(ctx context.Context, st *store.Store, brandID *uuid.UUID, listings []domain.ProductListing) (ApiBrandSummary, error) {
	if brandID != nil {
		b, err := st.BrandByID(ctx, *brandID)
		if err != nil {
			if !errors.Is(err, domain.ErrNotFound) {
				return ApiBrandSummary{}, err
			}
		} else {
			return ApiBrandSummary{Slug: b.Slug, Name: b.Name, Ayush: b.Ayush, FSSAI: b.FSSAI, Verified: b.Verified}, nil
		}
	}
	raw := "unknown"
	if len(listings) > 0 && listings[0].BrandRaw != "" {
		raw = listings[0].BrandRaw
	}
	return ApiBrandSummary{Slug: raw, Name: titleCase(raw)}, nil
}

// toApiListing's ListingID is the listing's own UUID, not its SKU —
// BuyButton.svelte POSTs `{ listing_id: listingId }` to the (not-yet-built)
// /api/checkout/initiate, which needs product_listings.id to record a
// click_events row (internal/db/migrations/007: click_events.listing_id
// uuid NOT NULL REFERENCES product_listings(id)). Checkout itself isn't
// built this milestone, but sending the right identifier now means the
// frontend needs zero changes when it is.
func toApiListing(l domain.ProductListing) ApiListing {
	return ApiListing{
		ListingID:    l.ID.String(),
		Source:       l.SourceSlug,
		PriceINR:     float64(l.PricePaise) / 100,
		SourceURL:    l.SourceURL,
		AffiliateURL: l.AffiliateURL,
		InStock:      l.InStock,
	}
}

var reSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// slugify is display/URL cosmetic only — 05-API-REFERENCE.md's product
// payload names a `slug` field, but no route in the shipped frontend
// actually reads product.slug (routing is by `id`, e.g. ProductCard.svelte's
// `/product?id=${p.id}`); present for TypeScript-contract completeness, not
// load-bearing.
func slugify(name string) string {
	s := strings.ToLower(name)
	s = reSlugChars.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func titleCase(slug string) string {
	parts := strings.Split(strings.ReplaceAll(slug, "-", " "), " ")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}
