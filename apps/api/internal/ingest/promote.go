package ingest

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/dr-toke/api/internal/compliance"
	"github.com/dr-toke/api/internal/domain"
	"github.com/dr-toke/api/internal/resolve"
	"github.com/dr-toke/api/internal/store"
	"github.com/google/uuid"
)

// classifierVersion is stamped on every product_facets row written by this
// pass — 03-DOMAIN-MODEL.md §2's precedence model uses it to know which
// ruleset generation produced a proposal. 1 for the first promote build;
// bump whenever a resolve ruleset/algorithm change should be visible on
// re-inspection of already-classified products.
const classifierVersion = 1

// PromoteResult summarises one call to Promote — not persisted anywhere
// itself (scrape_batches already carries the batch-level counts); this is
// for the caller (a job runner, or a test) to log or assert against.
type PromoteResult struct {
	Promoted           int
	FilteredCompliance int
	Errors             []error
}

// Promote runs the staging -> live pipeline for ONE gate-approved batch:
// per raw product, compliance-check, classify (resolve), price-parse,
// dedup/cluster, then persist. 04-PIPELINE.md §1's stage list
// (normalise -> resolve -> compliance -> dedup -> bestdeal) is collapsed
// into one function here rather than one River job per stage — no job
// queue exists yet (River is in the stack per 00-CONSTITUTION.md but not
// wired up this milestone), and a single in-process pass is the honest
// scope for the first working pipeline. Splitting into real queued stages
// is real work for whenever River gets wired in, not simulated here with
// extra indirection.
//
// Requires the batch to already be 'approved' — DecideGate is a hard
// prerequisite this function checks itself, not something the caller is
// trusted to have done first.
func Promote(ctx context.Context, st *store.Store, rs *resolve.RuleSet, crs *compliance.RuleSet, batchID uuid.UUID) (PromoteResult, error) {
	batch, err := st.BatchByID(ctx, batchID)
	if err != nil {
		return PromoteResult{}, fmt.Errorf("ingest.Promote: %w", err)
	}
	if batch.Status != domain.BatchApproved {
		return PromoteResult{}, fmt.Errorf("ingest.Promote: batch %s is %s, not approved — run DecideGate first", batchID, batch.Status)
	}

	raw, err := st.RawProductsForBatch(ctx, batchID)
	if err != nil {
		return PromoteResult{}, fmt.Errorf("ingest.Promote: %w", err)
	}

	var result PromoteResult
	for _, p := range raw {
		promoted, err := promoteOne(ctx, st, rs, crs, batchID, p)
		switch {
		case err != nil:
			result.Errors = append(result.Errors, fmt.Errorf("%s: %w", p.SourceURL, err))
		case !promoted:
			result.FilteredCompliance++
		default:
			result.Promoted++
		}
	}
	return result, nil
}

// promoteOne classifies and persists exactly one staged listing. Returns
// (false, nil) — not an error — when compliance filters it out; the raw
// staged row (raw_products) is itself the permanent record of what was
// seen and rejected (never deleted), which is what M2-DECISIONS.md's "a
// filtered-out row should still exist as a record" actually needs. No
// review_queue entry is written for a compliance block specifically
// because review_queue.cluster_id is NOT NULL (internal/db/migrations/007)
// — the schema assumes review items attach to a cluster that already
// exists, and a service listing never gets one.
func promoteOne(ctx context.Context, st *store.Store, rs *resolve.RuleSet, crs *compliance.RuleSet, batchID uuid.UUID, p domain.RawProduct) (bool, error) {
	// Compliance first — no point classifying a listing that will never be
	// shown. 04-PIPELINE.md §1's enqueue rule ("blocked... hides any
	// existing cluster via in_stock=false") describes what happens to a
	// listing that already has a cluster from a PRIOR approved scrape; a
	// first-time block, which is all ADR-020's narrow build currently
	// produces, has no cluster to hide yet, so it's simplest and correct to
	// just not create one.
	check := compliance.Evaluate(crs, p.Name, p.Description, p.CategoryRaw)
	if !check.Pass {
		return false, nil
	}

	pricePaise, err := resolve.ParsePriceINR(p.PriceRaw)
	if err != nil {
		return false, fmt.Errorf("price: %w", err)
	}

	cb := resolve.ExtractCannabinoids(&rs.Cannabinoids, p.Name, p.Description)
	volumeML, weightG := resolve.ExtractSize(&rs.Cannabinoids, p.Name, p.Description)

	form := resolve.ResolveForm(&rs.Categories, p.Name, p.Description, p.CategoryRaw)
	route := resolve.ResolveRoute(&rs.Categories, p.Name, p.Description, p.CategoryRaw)
	extract := resolve.ResolveExtract(p.Name, p.Description)
	profile := resolve.ResolveProfile(cb.CBDMg, cb.THCMg)
	carrier := resolve.ResolveCarrier(p.Description)
	purchasable := resolve.ResolvePurchasable(&rs.Categories, p.Name, p.Description, p.CategoryRaw)
	purchasableBool := purchasable.Value == "true"

	var routeForGate *domain.ProductFacet
	if route.Value != "" {
		routeForGate = &domain.ProductFacet{Confidence: route.Confidence}
	}
	publishable := resolve.Publishable(purchasableBool, form.Confidence, routeForGate, pricePaise)

	// Found by a full live-catalog audit: 03-DOMAIN-MODEL.md §2's Publishable
	// formula correctly never requires cannabinoid presence — legitimate
	// hemp_seed/nutrition items (hemp protein powder, hemp seed oil) have no
	// cbd/thc either, and must stay publishable. But applied to cbdstore.in's
	// actual broad "wellness marketplace" inventory (not a narrow
	// cannabis-only catalog), that same permissiveness let concentration_
	// type=unknown products with NO cannabis signal AT ALL through too —
	// ordinary Ayurvedic chai, a mushroom-extract supplement, a yoga wheel —
	// because their generic form-word matches ("tea", "extract", "massage")
	// carry no cannabis-specific meaning on their own. Scoped tightly:
	// ONLY concentration_type=unknown is affected (real cbd/thc content and
	// hemp_seed/nutrition typing are untouched), and it only excludes
	// listings with NO cannabis-context word anywhere in name+description —
	// a real cannabis product from a CBD-focused store almost always
	// mentions cbd/thc/cannabis/cannabinoid/vijaya/full or broad spectrum
	// somewhere in its own marketing copy.
	if publishable && cb.ConcentrationType == domain.ConcentrationUnknown && !hasCannabisContext(rs, p.Name, p.Description) {
		publishable = false
	}

	brandID, brandTrust, err := resolveBrand(ctx, st, p.BrandRaw)
	if err != nil {
		return false, fmt.Errorf("brand lookup: %w", err)
	}

	fingerprint := Fingerprint(p.BrandRaw, p.Name, volumeML, cb.BestMG())

	cluster := buildClusterShell(p, cb, volumeML, weightG, pricePaise, brandID, brandTrust, form, publishable)

	clusterID, err := AssignCluster(ctx, st, fingerprint, cluster)
	if err != nil {
		return false, fmt.Errorf("dedup: %w", err)
	}

	facets := resolveFacets(clusterID, form, route, extract, profile, carrier, purchasable)
	if err := st.UpsertFacets(ctx, facets); err != nil {
		return false, fmt.Errorf("upsert facets: %w", err)
	}

	// The form facet is the headline safety-relevant one (03-DOMAIN-MODEL.md
	// §2's publish gate keys off it) — below the 0.85 publish threshold is
	// exactly what 06-ADMIN.md §1.2's review queue exists to triage. Only
	// this one facet, not all six, to keep the queue signal-dense rather
	// than flooding it with every extract/carrier default guess.
	if form.Confidence < 0.85 {
		if _, err := st.Enqueue(ctx, domain.ReviewQueueItem{
			ClusterID:     clusterID,
			Reason:        domain.ReviewLowConfidence,
			Detail:        fmt.Sprintf("form resolved %q at %.2f confidence: %s", form.Value, form.Confidence, form.Reason),
			ProposedValue: map[string]any{"facet": "form", "value": form.Value, "confidence": form.Confidence},
		}); err != nil {
			return false, fmt.Errorf("enqueue low-confidence review: %w", err)
		}
	}

	listing := domain.ProductListing{
		SourceSlug:          p.SourceSlug,
		SourceURL:           p.SourceURL,
		SourceSKU:           p.SourceSKU,
		ClusterID:           &clusterID,
		NameRaw:             p.Name,
		BrandRaw:            p.BrandRaw,
		DescriptionRaw:      p.Description,
		CategoryRaw:         p.CategoryRaw,
		ImageURLRaw:         p.ImageURL,
		PricePaise:          pricePaise,
		InStock:             inStockFromRawData(p.RawData),
		PromotedFromBatchID: &batchID,
	}
	if _, err := st.UpsertListing(ctx, listing); err != nil {
		return false, fmt.Errorf("upsert listing: %w", err)
	}

	return true, nil
}

// hasCannabisContext reports whether name/description mentions cannabis at
// all — reuses harvest/rules/categories.json's cannabinoid_context_words
// list (cbd/thc/vijaya/cannabinoid/cannabidiol/full spectrum/broad
// spectrum), the same word list resolve's own nutrition-fallback branch
// already uses internally, not a new pattern invented here.
func hasCannabisContext(rs *resolve.RuleSet, name, description string) bool {
	matched, _ := resolve.MatchWordBoundary(rs.Categories.CannabinoidContext, strings.ToLower(name+" "+description))
	return matched
}

// resolveBrand looks up a brand by slug, returning (nil, defaultTrust, nil)
// for an unmapped/unknown brand rather than an error — unknown_brand review
// queueing is compliance's job (ADR-019: not built until beta), so an
// unrecognised brand here is not fatal to promoting the product, it just
// gets the conservative trust default.
func resolveBrand(ctx context.Context, st *store.Store, slug string) (*uuid.UUID, float64, error) {
	brand, err := st.BrandBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, unverifiedBrandTrust, nil
		}
		return nil, 0, err
	}
	if brand.Verified {
		return &brand.ID, 1.0, nil
	}
	return &brand.ID, unverifiedBrandTrust, nil
}

// unverifiedBrandTrust / verifiedBrandTrust / RankScore's completeness
// weights below are NOT specified anywhere in the docs —
// 03-DOMAIN-MODEL.md §5 names brand_trust and completeness as RankScore
// inputs ("verified/Ayush-registered lift" and "has-image/has-COA/has-full-
// dosage respectively") but gives no formula, same gap ValueScore's own
// doc comment (internal/resolve/value.go) already flags for value_score
// itself. 0.7 (not lower) so an unverified-but-real brand's products still
// rank above near-zero rather than being effectively hidden by one
// multiplier — a full brand-trust model belongs to whatever milestone
// builds the Ayush/FSSAI verification workflow, not this first pipeline
// pass.
const unverifiedBrandTrust = 0.7

// priceAnomalyLowINRPerMg / priceAnomalyHighINRPerMg are
// harvest/rules/compliance.json's own price_anomaly thresholds
// (low_threshold_inr_per_mg / high_threshold_inr_per_mg), reused directly
// rather than inventing new numbers — see buildClusterShell's price-anomaly
// comment.
const (
	priceAnomalyLowINRPerMg  = 0.10
	priceAnomalyHighINRPerMg = 100.0
)

// buildClusterShell assembles the fully-populated domain.ProductCluster
// AssignCluster will either INSERT (new fingerprint) or use to overwrite an
// existing row's derived fields (matched fingerprint) — see dedup.go's
// AssignCluster doc comment for why refresh-on-match matters.
func buildClusterShell(p domain.RawProduct, cb resolve.CannabinoidExtraction, volumeML, weightG float64,
	pricePaise int64, brandID *uuid.UUID, brandTrust float64, form resolve.FacetResult, publishable bool,
) domain.ProductCluster {
	var cbdPerMg, thcPerMg, totalPerMg, dominantPerMg *float64
	var basisPtr *string
	var valueTier *domain.ValueTier

	// Found via a real audit of the live cbdstore.in catalog, not a
	// hypothetical: "50mg (90 Capsules)" was priced as if the whole
	// bottle were 50mg total, producing a ₹126/mg figure ~90x too high.
	// mgLikelyPerUnitUnreconciled suppresses ₹/mg-derived pricing for
	// exactly that pattern — 00-CONSTITUTION.md's "we publish less rather
	// than publish wrong" — while leaving the raw cbd_mg/thc_mg figures
	// alone (they're still honest, useful per-unit dosing info, just not
	// safe to divide the pack PRICE by).
	if !mgLikelyPerUnitUnreconciled(p.Name, p.Description, cb.Evidence) {
		cbdPerMg = resolve.PerMg(pricePaise, cb.CBDMg)
		thcPerMg = resolve.PerMg(pricePaise, cb.THCMg)
		totalPerMg = resolve.PerMg(pricePaise, cb.TotalCannabinoidsMg)
		var basis domain.Basis
		dominantPerMg, basis = resolve.DominantPerMg(cbdPerMg, thcPerMg, totalPerMg)
		valueTier = resolve.ValueTier(dominantPerMg)
		if basis != "" {
			b := string(basis)
			basisPtr = &b
		}

		// General safety net, layered on top of the specific "(N Capsules)"
		// check above — a live-catalog audit after that fix landed found
		// the SAME root cause (a per-serving/per-spray/per-drop dose priced
		// as if it were the whole pack) surfacing under other phrasings
		// ("1mg per spray" on a 30ml bottle, plain "Sublingual Drops" with
		// no parenthetical count at all) that the narrow pattern doesn't
		// and can't enumerate exhaustively. harvest/rules/compliance.json's
		// own price_anomaly tier already specifies exactly this band
		// (0.10-100 ₹/mg) as "anomalous, needs review" — reused here as a
		// suppress-rather-than-guess gate, matching 04-PIPELINE.md §5.
		if dominantPerMg != nil && (*dominantPerMg < priceAnomalyLowINRPerMg || *dominantPerMg > priceAnomalyHighINRPerMg) {
			cbdPerMg, thcPerMg, totalPerMg, dominantPerMg, basisPtr, valueTier = nil, nil, nil, nil, nil, nil
		}
	}

	// completeness: has-image + has-cannabinoid-dosage, per
	// 03-DOMAIN-MODEL.md §5's description of the input (no formula given —
	// same flagged gap as brandTrust above). 0.5 base so a product with
	// neither doesn't zero out RankScore entirely via multiplication.
	completeness := 0.5
	if p.ImageURL != nil && *p.ImageURL != "" {
		completeness += 0.25
	}
	if cb.BestMG() > 0 {
		completeness += 0.25
	}

	var rankScore *float64
	if dominantPerMg != nil {
		vs := resolve.ValueScore(*dominantPerMg)
		rs := resolve.RankScore(vs, float64(form.Confidence), brandTrust, completeness)
		rankScore = &rs
	}

	conf := float64(cb.Confidence)

	return domain.ProductCluster{
		BrandID:               brandID,
		Name:                  p.Name,
		ShortDescription:      capShortDescription(p.Description, 160),
		CBDMg:                 nilIfNotPositive(cb.CBDMg),
		THCMg:                 nilIfNotPositive(cb.THCMg),
		TotalCannabinoidsMg:   nilIfNotPositive(cb.TotalCannabinoidsMg),
		ConcentrationType:     cb.ConcentrationType,
		CannabinoidConfidence: &conf,
		CannabinoidEvidence:   resolve.EvidenceToMap(cb.Evidence),
		VolumeML:              nilIfNotPositive(volumeML),
		WeightG:               nilIfNotPositive(weightG),
		BestPricePaise:        &pricePaise,
		BestPricePerMg:        dominantPerMg,
		CBDPricePerMg:         cbdPerMg,
		THCPricePerMg:         thcPerMg,
		PricePerMgBasis:       basisPtr,
		ValueTier:             valueTier,
		RankScore:             rankScore,
		Publishable:           publishable,
	}
}

// rePackQuantity matches "(N capsules)"/"(N gummies)"/"pack of N"-style
// wording — a strong signal that a bare mg figure elsewhere in the text is
// PER-UNIT dosing (mg per capsule/gummy/tablet), not the product's total
// cannabinoid content.
var rePackQuantity = regexp.MustCompile(`(?i)\(\s*\d+\s*(capsules?|gummies|gummy|tablets?|candi(?:es|y)|sachets?|pieces?)\s*\)|pack\s+of\s+\d+`)

// mgLikelyPerUnitUnreconciled reports whether cb's mg figures should NOT be
// divided into a per-mg price. harvest/rules/cannabinoids.json's
// per-serving reconciliation (cannabinoids.go's "per-serving figure
// reconciled to pack total" branch) only fires when a SEPARATE, larger
// total-mg figure is also present in the text to reconcile against; a
// listing whose only stated dose is per-unit, with no pack-total anywhere,
// gives that reconciliation nothing to find — the ported extractor
// (faithful to the prior alpha) has no notion of multiplying dose × unit
// count from "(90 Capsules)" wording. Real, not hypothetical: this is what
// was inflating "50mg (90 Capsules)" to a ₹126/mg figure ~90x too high in
// the live cbdstore.in catalog.
func mgLikelyPerUnitUnreconciled(name, description string, ev resolve.Evidence) bool {
	if !rePackQuantity.MatchString(name) && !rePackQuantity.MatchString(description) {
		return false
	}
	for _, note := range ev.Notes {
		if note == "per-serving figure reconciled to pack total" {
			return false // the extractor already found and applied a real pack total
		}
	}
	return true
}

// resolveFacets runs every FacetResult through precedence.Resolve as the
// Rule tier (resolve's ruleset-driven output IS the rule engine — there is
// no separate ML model this milestone, per FacetInputs.Model's own doc
// comment: "the normal case for Model (M1 has no ML model yet)") and
// collects whichever facets actually proposed a value. A facet with no
// signal (empty Value, e.g. ResolveRoute when hasRoute is false) is simply
// omitted, not written as an empty row.
func resolveFacets(clusterID uuid.UUID, form, route, extract, profile, carrier, purchasable resolve.FacetResult) []domain.ProductFacet {
	proposals := []struct {
		facet  domain.Facet
		result resolve.FacetResult
	}{
		{domain.FacetForm, form},
		{domain.FacetRoute, route},
		{domain.FacetExtract, extract},
		{domain.FacetProfile, profile},
		{domain.FacetCarrier, carrier},
		{domain.FacetPurchasable, purchasable},
	}

	var out []domain.ProductFacet
	for _, p := range proposals {
		r := p.result
		f := resolve.Resolve(clusterID, p.facet, resolve.FacetInputs{Rule: &r}, classifierVersion)
		if f != nil {
			out = append(out, *f)
		}
	}
	return out
}

func nilIfNotPositive(v float64) *float64 {
	if v <= 0 {
		return nil
	}
	return &v
}

// capShortDescription enforces 02-FRONTEND-CONTRACT.md §8's "~160 chars,
// server-side" cap — truncation belongs where the canonical text lives, not
// left to the frontend to guess at consistently across the grid, compare
// table, and detail page. Rune-safe (byte-slicing a UTF-8 string can split
// a multi-byte character); returns nil for empty input, matching the
// nullable-never-empty-string convention the rest of the domain layer uses
// for optional text.
func capShortDescription(s string, max int) *string {
	if s == "" {
		return nil
	}
	r := []rune(s)
	if len(r) <= max {
		return &s
	}
	out := string(r[:max])
	return &out
}

// inStockFromRawData reads shopify.go's rd["in_stock"] bool back out of the
// jsonb-round-tripped RawData bag — RawListing/RawProduct carry no
// first-class InStock field (adapter.go's own doc comment: field names
// mirror domain.RawProduct, and neither has one), so this is where that
// value actually surfaces for promotion. Missing/wrong-typed defaults to
// true (assume in-stock) rather than false — an adapter that doesn't report
// stock status shouldn't silently hide every one of its products.
func inStockFromRawData(raw map[string]any) bool {
	v, ok := raw["in_stock"].(bool)
	if !ok {
		return true
	}
	return v
}
