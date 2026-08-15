package resolve

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/dr-toke/api/internal/domain"
)

// ParsePriceINR turns a scraped price string into int64 paise —
// 00-CONSTITUTION.md §5's "Money is int64 paise. Never float" applies from
// the moment a price leaves the scraper, not just at the API boundary.
// Not specified anywhere in the docs (no ADR, no build-order line covers
// this conversion) — NEW, needed because RawProduct.PriceRaw (domain
// types.go) is a plain string and nothing before this milestone ever
// turned it into the ProductListing.PricePaise the rest of the system
// (store, resolve.PerMg, ValueTier) assumes exists.
//
// Strips ₹/Rs/INR prefixes and thousands-comma separators — matches
// shopify.go's own "₹" + v.Price construction (its comment: "normaliser
// strips ₹/commas — don't pre-format here"), this is that normaliser.
// Rounds to the nearest paisa rather than truncating, since Shopify's own
// decimal strings ("999.00") are already exact to the paisa in practice —
// rounding only matters for the rare malformed input.
func ParsePriceINR(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("resolve: empty price")
	}
	s = strings.NewReplacer(
		"₹", "",
		"Rs.", "",
		"Rs", "",
		"INR", "",
		",", "",
		" ", "",
	).Replace(s)
	if s == "" {
		return 0, fmt.Errorf("resolve: no digits in price %q", raw)
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("resolve: parse price %q: %w", raw, err)
	}
	if v < 0 {
		return 0, fmt.Errorf("resolve: negative price %q", raw)
	}
	return int64(math.Round(v * 100)), nil
}

// PerMg computes ₹/mg from a paise price and an mg figure. Returns nil (not
// zero) whenever mg is unknown or non-positive — 00-CONSTITUTION.md §5:
// "Unknown numerics are NULL... never zero" and 03-DOMAIN-MODEL.md §5:
// "cbd_price_per_mg: best_price / cbd_mg when cbd_mg > 0, else NULL."
//
// pricePaise/100 converts to rupees before dividing — the displayed figure
// throughout the docs and the frontend is ₹/mg ("₹2.10/mg"), not paise/mg;
// 05-API-REFERENCE.md's example payload shows "cbd_price_per_mg": 2.10 as a
// plain decimal, confirming rupees not paise here specifically (unlike
// price_paise itself, which stays integer paise everywhere else).
func PerMg(pricePaise int64, mg float64) *float64 {
	if mg <= 0 {
		return nil
	}
	v := float64(pricePaise) / 100.0 / mg
	return &v
}

// ValueTier bands a ₹/mg figure into the frontend-canonical tiers —
// ADR-012: these are pmgColor()'s live bands (<3/3-8/>8), not the four-tier
// scheme 03-DOMAIN-MODEL.md originally specified. nil in, nil out — a
// product with no computable ₹/mg has no tier, it does not default to one.
func ValueTier(perMg *float64) *domain.ValueTier {
	if perMg == nil {
		return nil
	}
	var t domain.ValueTier
	switch {
	case *perMg < 3:
		t = domain.ValueTierGood
	case *perMg <= 8:
		t = domain.ValueTierMid
	default:
		t = domain.ValueTierHigh
	}
	return &t
}

// DominantPerMg picks the single ₹/mg figure and basis used wherever the
// system needs ONE number for a product carrying multiple cannabinoids and
// no explicit ?basis= was given — ADR-013's priority, THC > CBD > total,
// reversing 03-DOMAIN-MODEL.md's original CBD > THC > total order per
// stakeholder direction (₹/mg-THC matters most to the target demographic).
func DominantPerMg(cbdPerMg, thcPerMg, totalPerMg *float64) (perMg *float64, basis domain.Basis) {
	if thcPerMg != nil {
		return thcPerMg, domain.BasisTHC
	}
	if cbdPerMg != nil {
		return cbdPerMg, domain.BasisCBD
	}
	if totalPerMg != nil {
		return totalPerMg, domain.BasisTotal
	}
	return nil, ""
}

// ValueScore is the raw, uncorrected "cheaper is better" signal RankScore
// multiplies against. NOT specified anywhere in the docs — 03-DOMAIN-MODEL.md
// §5 gives RankScore's composition (value_score × facet_confidence ×
// brand_trust × completeness) but never value_score's own formula. Simple
// inverse (1/₹-per-mg, so a lower price-per-mg yields a higher score) is the
// most direct reading of "value" that composes cleanly with the other three
// multiplicative dampeners — those three are what the doc's own reasoning
// ("pure ₹/mg crowns whatever product had its mg misparsed high") relies on
// to correct a too-good-to-be-true score, not a cap or clamp inside
// ValueScore itself. Flagged in M1-DECISIONS.md — the multiplicative shape
// is spec'd, this one factor's exact formula is not.
func ValueScore(perMg float64) float64 {
	if perMg <= 0 {
		return 0
	}
	return 1 / perMg
}

// RankScore is 03-DOMAIN-MODEL.md §5's exact formula:
//
//	rank_score = value_score × facet_confidence × brand_trust × completeness
//
// All four factors are expected in comparable ranges (roughly 0..1 for the
// three multipliers; value_score is unbounded above since cheap ₹/mg has no
// ceiling) — callers are responsible for producing brand_trust and
// completeness themselves (verified/Ayush-registered lift, and
// has-image/has-COA/has-full-dosage respectively), neither of which is
// resolve's concern; this function only ever multiplies four numbers.
func RankScore(valueScore, facetConfidence, brandTrust, completeness float64) float64 {
	return valueScore * facetConfidence * brandTrust * completeness
}
