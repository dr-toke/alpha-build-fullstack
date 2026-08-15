package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
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

// brandPayload / listingPayload / productPayload are 05-API-REFERENCE.md
// §1's "Product payload (essentials)" — a subset of the documented fields.
// NOT included: `image` (always null — no image pipeline exists yet, M6
// pending, and null is the honest nullable-unknown value per
// 02-FRONTEND-CONTRACT.md §8, not a fabricated placeholder object).
type brandPayload struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Verified bool   `json:"verified"`
	Ayush    bool   `json:"ayush"`
	FSSAI    bool   `json:"fssai"`
}

type listingPayload struct {
	ID         uuid.UUID `json:"id"`
	SourceSlug string    `json:"source_slug"`
	PricePaise int64     `json:"price_paise"`
	URL        string    `json:"url"`
	InStock    bool      `json:"in_stock"`
}

type productPayload struct {
	ID                   uuid.UUID                `json:"id"`
	Name                 string                   `json:"name"`
	ShortDescription     *string                  `json:"short_description"`
	Brand                *brandPayload            `json:"brand"`
	Category             string                   `json:"category"`
	Categories           []string                 `json:"categories"`
	Facets               map[string]string        `json:"facets"`
	CBDMg                *float64                 `json:"cbd_mg"`
	THCMg                *float64                 `json:"thc_mg"`
	ConcentrationType    domain.ConcentrationType `json:"concentration_type"`
	BestPricePaise       *int64                   `json:"best_price_paise"`
	CBDPricePerMg        *float64                 `json:"cbd_price_per_mg"`
	THCPricePerMg        *float64                 `json:"thc_price_per_mg"`
	BestPricePerMg       *float64                 `json:"best_price_per_mg"`
	PricePerMgBasis      *string                  `json:"price_per_mg_basis"`
	ValueTier            *domain.ValueTier        `json:"value_tier"`
	PrescriptionRequired bool                     `json:"prescription_required"`
	InStock              bool                     `json:"in_stock"`
	Image                any                      `json:"image"`
	Listings             []listingPayload         `json:"listings"`
	UpdatedAt            time.Time                `json:"updated_at"`
}

// ListProducts handles GET /api/products — 05-API-REFERENCE.md §1.
// Implemented params: `brand`, `sort` (new|value), `cursor`, `limit`.
// NOT implemented in this pass: `category`, `form`/`route`/`extract`/
// `profile`/`carrier` facet filters, `basis`-scoped ₹/mg sorting,
// `verified`. See API-DECISIONS.md — the store-layer filter surface
// (ClusterFilter) only carries BrandID/PublishableOnly/Sort/cursor/Limit
// today; wiring the rest through is straightforward but real, additional
// work, not done here under the beta-first-slice priority.
func (h *Handlers) ListProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	filter := store.ClusterFilter{
		PublishableOnly: true, // public endpoint — 03-DOMAIN-MODEL.md §2's gate is not optional here
		Limit:           parseLimit(r),
	}

	switch q.Get("sort") {
	case store.SortValue:
		filter.Sort = store.SortValue
	case "", store.SortNew:
		filter.Sort = store.SortNew
	default:
		writeInvalidFilter(w, "sort must be one of: new, value")
		return
	}

	if brandSlug := q.Get("brand"); brandSlug != "" {
		brand, err := h.Store.BrandBySlug(ctx, brandSlug)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				// An unknown brand slug is a valid filter that just matches
				// nothing — 02-FRONTEND-CONTRACT.md §4: "no results for this
				// filter" is 200+empty, not an error.
				WriteJSON(w, http.StatusOK, Envelope{Data: []productPayload{}, Page: 1, Limit: filter.Limit, Total: 0, HasMore: false})
				return
			}
			writeUnavailable(w, "could not look up brand")
			return
		}
		filter.BrandID = &brand.ID
	}

	if cursor := q.Get("cursor"); cursor != "" {
		key, id, err := decodeCursor(cursor)
		if err != nil {
			writeInvalidFilter(w, "malformed cursor")
			return
		}
		if filter.Sort == store.SortValue {
			v, err := strconv.ParseFloat(key, 64)
			if err != nil {
				writeInvalidFilter(w, "malformed cursor")
				return
			}
			filter.CursorRankScore = &v
		} else {
			t, err := time.Parse(time.RFC3339Nano, key)
			if err != nil {
				writeInvalidFilter(w, "malformed cursor")
				return
			}
			filter.CursorFirstSeenAt = &t
		}
		filter.CursorID = &id
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

	payloads := make([]productPayload, 0, len(clusters))
	for _, c := range clusters {
		p, err := buildProductPayload(ctx, h.Store, c)
		if err != nil {
			writeUnavailable(w, "could not assemble product")
			return
		}
		payloads = append(payloads, p)
	}

	env := Envelope{Data: payloads, Page: 1, Limit: filter.Limit, Total: total}
	if len(clusters) == filter.Limit {
		last := clusters[len(clusters)-1]
		env.HasMore = true
		if filter.Sort == store.SortValue {
			env.NextCursor = encodeCursor(fmt.Sprintf("%v", *last.RankScore), last.ID)
		} else {
			env.NextCursor = encodeCursor(last.FirstSeenAt.Format(time.RFC3339Nano), last.ID)
		}
	}
	WriteJSON(w, http.StatusOK, env)
}

// GetProduct handles GET /api/products/{id} — 05-API-REFERENCE.md §1:
// "Returns { moved_to } if merged." 02-FRONTEND-CONTRACT.md §4: a merged
// product is 200, never a 404 — the frontend rewrites its stored id rather
// than showing a broken link. Genuinely unknown is 404/not_found.
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

	payload, err := buildProductPayload(ctx, h.Store, *cluster)
	if err != nil {
		writeUnavailable(w, "could not assemble product")
		return
	}
	WriteJSON(w, http.StatusOK, payload)
}

func buildProductPayload(ctx context.Context, st *store.Store, c domain.ProductCluster) (productPayload, error) {
	var brand *brandPayload
	if c.BrandID != nil {
		b, err := st.BrandByID(ctx, *c.BrandID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return productPayload{}, fmt.Errorf("brand: %w", err)
		}
		if b != nil {
			brand = &brandPayload{Slug: b.Slug, Name: b.Name, Verified: b.Verified, Ayush: b.Ayush, FSSAI: b.FSSAI}
		}
	}

	facetRows, err := st.FacetsFor(ctx, c.ID)
	if err != nil {
		return productPayload{}, fmt.Errorf("facets: %w", err)
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
		return productPayload{}, fmt.Errorf("listings: %w", err)
	}
	listings := make([]listingPayload, 0, len(listingRows))
	inStock := false
	for _, l := range listingRows {
		listings = append(listings, listingPayload{
			ID: l.ID, SourceSlug: l.SourceSlug, PricePaise: l.PricePaise, URL: l.SourceURL, InStock: l.InStock,
		})
		if l.InStock {
			inStock = true
		}
	}

	return productPayload{
		ID:                   c.ID,
		Name:                 c.Name,
		ShortDescription:     c.ShortDescription,
		Brand:                brand,
		Category:             category,
		Categories:           resolve.LegacyCategories(category, secondary),
		Facets:               facets,
		CBDMg:                c.CBDMg,
		THCMg:                c.THCMg,
		ConcentrationType:    c.ConcentrationType,
		BestPricePaise:       c.BestPricePaise,
		CBDPricePerMg:        c.CBDPricePerMg,
		THCPricePerMg:        c.THCPricePerMg,
		BestPricePerMg:       c.BestPricePerMg,
		PricePerMgBasis:      c.PricePerMgBasis,
		ValueTier:            c.ValueTier,
		PrescriptionRequired: c.PrescriptionRequired,
		InStock:              inStock,
		Image:                nil,
		Listings:             listings,
		UpdatedAt:            c.UpdatedAt,
	}, nil
}
