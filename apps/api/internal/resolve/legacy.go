package resolve

import "github.com/dr-toke/api/internal/domain"

// LegacyCategory maps a cluster's RESOLVED facets (post-precedence, so a
// human override is reflected here same as everywhere else — 03-DOMAIN-MODEL.md
// §2: "category := form value, mapped through the legacy name table") back
// to the pre-refactor category vocabulary the frontend's catalog.ts still
// reads directly (CATEGORY_COLORS: tincture, smokable, vapeable, edible,
// topical, extract, nutrition, beverage, pet, apparel, other) — ADR-002's
// dual-write promise: "the frontend never breaks mid-refactor."
//
// Two things here have no precedent in the harvested source, both forced by
// the new facet vocabulary being finer-grained than the legacy one:
//
//  1. form=concentrate is ambiguous on its own — it's the merge point of TWO
//     legacy buckets (facets.go: "vapeable" via dab/shatter/distillate, and
//     "extract" via RSO/FECO/hash oil). Route disambiguates: oral means it
//     came from the extract bucket (legacy name "extract", auto-secondary
//     "edible" — 03-DOMAIN-MODEL.md §2's literal words: "extract (RSO/FECO/
//     hash oil) also emits edible as a secondary, as today"); inhaled means
//     the vapeable/dab bucket (legacy name "vapeable", no secondary).
//  2. form=edible alone would LOSE the legacy "nutrition" category entirely
//     — resolveForm collapsed the legacy "nutrition" bucket into form=edible
//     (03-DOMAIN-MODEL.md §2 has no "nutrition" form value, see facets.go),
//     but the frontend's CATEGORY_COLORS map still has a nutrition entry and
//     ADR-002 promises the legacy view never breaks. concentration_type is
//     what recovers the distinction: hemp_seed/nutrition-typed edibles map
//     back to legacy "nutrition", everything else stays "edible".
func LegacyCategory(form domain.FormValue, route domain.RouteValue, concentrationType domain.ConcentrationType) (category string, secondary []string) {
	switch form {
	case domain.FormPet:
		return "pet", nil
	case domain.FormApparel:
		return "apparel", nil
	case domain.FormTopical:
		return "topical", nil
	case domain.FormOilTincture:
		return "tincture", nil
	case domain.FormFlower:
		return "smokable", nil
	case domain.FormVape:
		return "vapeable", nil
	case domain.FormBeverage:
		return "beverage", nil
	case domain.FormConcentrate:
		if route == domain.RouteOral {
			return "extract", []string{"edible"}
		}
		return "vapeable", nil
	case domain.FormCapsule:
		return "edible", nil
	case domain.FormEdible:
		if concentrationType == domain.ConcentrationHempSeed || concentrationType == domain.ConcentrationNutrition {
			return "nutrition", nil
		}
		return "edible", nil
	case domain.FormAccessory:
		// No legacy bucket ever produced "accessory" — it's a new-only form
		// value with no harvested precedent (nothing in the prior alpha's
		// word lists mapped to it). "other" is the closest honest fallback.
		return "other", nil
	default: // empty/unresolved form
		return "other", nil
	}
}

// LegacyCategories assembles the full categories[] array — primary first,
// per 03-DOMAIN-MODEL.md §2 ("category := form value... categories[] :=
// [form, …secondary forms, extract-derived tags]") and
// 02-FRONTEND-CONTRACT.md §9 ("first category is primary... the rest are
// small grey tags").
func LegacyCategories(category string, secondary []string) []string {
	return dedupStrings(append([]string{category}, secondary...))
}
