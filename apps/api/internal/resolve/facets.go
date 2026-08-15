package resolve

import (
	"strings"

	"github.com/dr-toke/api/internal/domain"
)

// FacetResult is one facet's rule-tier proposal — not yet a stored
// domain.ProductFacet (that's precedence.go's job, after override/model
// tiers are considered too).
type FacetResult struct {
	Value      string
	Confidence float32
	Ambiguous  bool
	Reason     string // review_queue reason if ambiguous, e.g. "category_uncertain"
	Evidence   Evidence
}

// formDetection is the internal, pre-mapping result of running the ported
// legacy classifier — same shape as the harvested ClassifyWithConfidence,
// before its legacy category vocabulary is translated into the new facet
// vocabulary. Kept package-private: nothing outside facets.go should see
// the legacy bucket names.
type formDetection struct {
	primary   string // PUBLIC bucket name (publicCat already applied — edible_solid collapses to "edible", same as the harvested source's own return value)
	secondary []string
	ambiguous bool
	reason    string
	evidence  Evidence

	// viaConcentrateMarker is true when "vapeable" was set by
	// concentrate_markers ("not diluted with any carrier oil") rather than
	// an explicit vape/cartridge/dab word — resolveForm needs this because,
	// unlike every other bucket, "vapeable" splits into two new facet values
	// (vape vs concentrate) and a marker-driven match can't be recovered by
	// re-scanning the name afterward (the matched phrase is typically in the
	// DESCRIPTION, and word lists alone don't capture "which path fired").
	viaConcentrateMarker bool
}

// classify is a faithful port of the prior alpha's
// ClassifyWithConfidence (dr-toke-init/apps/api/internal/catalog/normaliser/
// categories.go) — same three failure modes it exists to fix (negated
// marketing copy, description noise beating the name, physically impossible
// combos), same control flow, same word lists (loaded from rs, never
// hardcoded). See harvest/NOTES.md for why this can't be flattened into
// data-driven rules.
func classify(rs *CategoryRuleSet, name, description, rawCategory string) formDetection {
	var ev Evidence

	nameStripped, negName := ApplyNegation(strings.ToLower(name), rs)
	descStripped, negDesc := ApplyNegation(strings.ToLower(description+" "+rawCategory), rs)
	ev.Negated = append(append(ev.Negated, negName...), negDesc...)

	lowName := nameStripped
	lowDesc := descStripped
	lowAll := lowName + " " + lowDesc
	lowRawCategory := strings.ToLower(rawCategory)

	// ── Apparel: exclusive, NAME or breadcrumb only, checked BEFORE pet ─────
	// Checked first — found via a real full-catalog audit: "The Shaman's
	// Pet ... T-Shirt" and "Poseidon's Pet Turtle ... T-Shirt" (two
	// separate real listings, evidently a whole line of pet-themed graphic
	// tees) were both landing as form="pet" and passing the publish gate,
	// because "Pet" — a playful/creative brand-name word here, nothing to
	// do with an actual pet product — matched pet_words_name_level before
	// the apparel check for "t-shirt" ever ran. harvest/rules/categories.json
	// documents both as "exclusive" and "checked before any form detection"
	// but never specifies their relative order against EACH OTHER — this
	// order was inherited, untested against this exact collision, from the
	// prior alpha's control flow. Apparel words (t-shirt, hoodie, kurta...)
	// are concrete physical-form descriptors, effectively never false
	// positives; pet_words_name_level's list ("pet," "paw," "vet," a handful
	// of animal names) is far more likely to appear inside an unrelated
	// creative product name. A genuine pet-apparel product (e.g. a literal
	// dog hoodie) would be the rare case this reorder could miscategorize,
	// against two confirmed real instances of the opposite failure — not
	// this store's actual "pet" category, which is CBD-for-pets oils/treats,
	// not literal animal clothing.
	if matched, spans := MatchWordBoundary(rs.ApparelWords, lowName); matched {
		ev.Matched = append(ev.Matched, spans...)
		return formDetection{primary: "apparel", evidence: ev}
	}
	if matched, spans := MatchWordBoundary(rs.ApparelWords, lowRawCategory); matched {
		ev.Matched = append(ev.Matched, spans...)
		return formDetection{primary: "apparel", evidence: ev}
	}

	// ── Pet: exclusive, NAME-first ──────────────────────────────────────────
	if matched, spans := MatchWordBoundary(rs.PetWordsName, lowName); matched {
		ev.Matched = append(ev.Matched, spans...)
		return formDetection{primary: "pet", evidence: ev}
	}
	if matched, spans := MatchWordBoundary(rs.PetWordsName, lowRawCategory); matched {
		ev.Matched = append(ev.Matched, spans...)
		ev.Notes = append(ev.Notes, "pet: matched source category breadcrumb")
		return formDetection{primary: "pet", evidence: ev}
	}
	descForPetCheck := rs.PetWarningStrip.ReplaceAllString(lowDesc, " ")
	if matched, spans := MatchWordBoundary(rs.PetStrongDescPhrase, descForPetCheck); matched {
		ev.Matched = append(ev.Matched, spans...)
		ev.Notes = append(ev.Notes, "pet: matched strong formulation-intent phrase in description")
		return formDetection{primary: "pet", evidence: ev}
	}

	// ── Form detection: NAME first, description fallback ────────────────────
	set, formEv := detectForms(rs, lowName)
	fromDesc := false
	if len(set) == 0 {
		set, formEv = detectForms(rs, lowDesc)
		fromDesc = len(set) > 0
	}
	ev = Merge(ev, formEv)

	// Pure concentrate signal — trusted from the description too, it's a
	// physical property, not marketing noise.
	concentrateFired := false
	if matched, spans := MatchWordBoundary(rs.ConcentrateMarkers, lowAll); matched && !set["edible_solid"] && !set["topical"] {
		set["vapeable"] = true
		concentrateFired = true
		ev.Matched = append(ev.Matched, spans...)
	}

	// ── Coherence matrix ─────────────────────────────────────────────────────
	if set["topical"] {
		deleteAll(set, rs.CoherenceMatrix.IfTopicalDelete)
		ev.Notes = append(ev.Notes, "coherence: topical present, deleted incompatible forms")
	}
	if set["edible_solid"] {
		deleteAll(set, rs.CoherenceMatrix.IfEdibleSolidDelete)
		ev.Notes = append(ev.Notes, "coherence: edible_solid present, deleted incompatible forms")
	}
	if set["tincture"] {
		deleteAll(set, rs.CoherenceMatrix.IfTinctureDelete)
	}
	if set["vapeable"] {
		if matched, _ := MatchWordBoundary(rs.FormWordLists["vapeable"], lowName); matched {
			deleteAll(set, rs.CoherenceMatrix.IfVapeableAndNameSaysVapeDelete)
		}
	}

	autoEdible := set["extract"] && !set["edible"] && !set["edible_solid"] && !set["vapeable"] && !set["tincture"] && !set["beverage"]

	// ── Nutrition fallback ────────────────────────────────────────────────────
	if len(set) == 0 {
		cbMatched, _ := MatchWordBoundary(rs.CannabinoidContext, lowAll)
		nutMatched, nutSpans := MatchWordBoundary(rs.NutritionWords, lowAll)
		if !cbMatched && nutMatched {
			set["nutrition"] = true
			ev.Matched = append(ev.Matched, nutSpans...)
		}
	}

	// ── Assemble in priority order ───────────────────────────────────────────
	var cats []string
	for _, f := range rs.FormPriority {
		if set[f] {
			cats = append(cats, publicCat(f))
		}
	}
	cats = dedupStrings(cats)
	if autoEdible {
		cats = append(cats, "edible")
		ev.Notes = append(ev.Notes, "extract with no other form -> auto-secondary edible")
	}

	if concentrateFired {
		ev.Notes = append(ev.Notes, "vapeable set via concentrate_markers, not an explicit vape word")
	}

	if len(cats) == 0 {
		return formDetection{primary: "other", ambiguous: true, reason: "could not determine product form", evidence: ev}
	}
	viaMarker := concentrateFired && cats[0] == "vapeable"
	if fromDesc {
		return formDetection{primary: cats[0], secondary: cats[1:], ambiguous: true,
			reason: "form inferred from description only — name has no form signal", evidence: ev, viaConcentrateMarker: viaMarker}
	}
	return formDetection{primary: cats[0], secondary: cats[1:], evidence: ev, viaConcentrateMarker: viaMarker}
}

func detectForms(rs *CategoryRuleSet, s string) (map[string]bool, Evidence) {
	set := map[string]bool{}
	var ev Evidence
	for form, re := range rs.FormWordLists {
		if matched, spans := MatchWordBoundary(re, s); matched {
			set[form] = true
			ev.Matched = append(ev.Matched, spans...)
		}
	}
	return set, ev
}

func deleteAll(set map[string]bool, victims map[string]bool) {
	for v := range victims {
		delete(set, v)
	}
}

func publicCat(form string) string {
	if form == "edible_solid" {
		return "edible"
	}
	return form
}

func dedupStrings(in []string) []string {
	seen := map[string]bool{}
	out := in[:0]
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// ── Legacy-bucket -> new-facet mapping ──────────────────────────────────────
//
// This is the one piece of M1 with no precedent in the harvested source at
// all: 03-DOMAIN-MODEL.md §2's form facet (oil_tincture, capsule, edible,
// topical, flower, vape, concentrate, beverage, pet, apparel, accessory) is
// a DIFFERENT, finer vocabulary than the legacy classifier's buckets
// (topical, vapeable, smokable, edible_solid, edible, tincture, extract,
// beverage, nutrition, pet, apparel, other) — the harvested word lists group
// "capsule" and "gummy" into one edible_solid bucket, and group "vape pen"
// and "dab" into one vapeable bucket, but the new facet model wants capsule
// separated from edible, and vape separated from concentrate. See
// M1-DECISIONS.md for the full reasoning and every case this was checked
// against — this is the single largest new judgment call in M1.

var capsuleWords = wordSet([]string{
	"capsule", "capsules", "softgel", "softgels", "soft gel", "tablet", "tablets",
	"pill", "pills", "vati",
})

var concentrateSubWords = wordSet([]string{
	"distillate", "shatter", "dab", "dabs", "dabbing",
})

// dabApplicatorWords are dab/vape-pen delivery-hardware words — NEW, not
// part of the harvested categories.json word lists (that file's "vapeable"
// list has "vape pen" as a two-word phrase but no standalone "pen"; its
// "extract" list has "syringe" grouped in without any route distinction).
// Checked only within the "extract" branch below, after concentrate_markers
// and concentrateSubWords have already had their chance — see that branch's
// doc comment for the real product that motivated this.
var dabApplicatorWords = wordSet([]string{"pen", "syringe"})

// resolveForm turns a formDetection into the new facet's Form/Route/Extract
// proposal. Extract is returned alongside Form because the "extract" legacy
// bucket (RSO/FECO/hash oil) doesn't correspond to a form facet value at
// all in 03-DOMAIN-MODEL.md §2 — it's what feeds the SEPARATE `extract`
// facet (full_spectrum/broad_spectrum/isolate) instead; when the legacy
// classifier's primary bucket IS "extract", the new form facet resolves to
// concentrate (it's a concentrated cannabis product) — route is oral or
// inhaled depending on inhalation signals, see that switch case's own
// comment; this is a form/route pairing the legacy code never had to make
// explicit, because it didn't have a separate route facet.
func resolveForm(name string, d formDetection) (form domain.FormValue, route domain.RouteValue, hasRoute bool, confidence float32, ambiguous bool, reason string) {
	confidence = 0.85
	if d.ambiguous {
		confidence = 0.4
	}
	lowName := strings.ToLower(name)

	switch d.primary {
	case "pet":
		return domain.FormPet, "", false, 0.95, false, "route not applicable to pet products"
	case "apparel":
		return domain.FormApparel, "", false, 0.95, false, "route not applicable to apparel"
	case "topical":
		return domain.FormTopical, domain.RouteTopical, true, confidence, d.ambiguous, d.reason
	case "tincture":
		return domain.FormOilTincture, domain.RouteSublingual, true, confidence, d.ambiguous, d.reason
	case "smokable":
		return domain.FormFlower, domain.RouteInhaled, true, confidence, d.ambiguous, d.reason
	case "beverage":
		return domain.FormBeverage, domain.RouteOral, true, confidence, d.ambiguous, d.reason
	case "extract":
		// The harvested source's own reasoning here was "raw extract with
		// no other form is most commonly ingested orally" — true for RSO/
		// FECO (classically taken orally, often on food or in a capsule),
		// but wrong for the OTHER real product type this same "extract"
		// word list conflates: a raw, undiluted concentrate (resin/wax/a
		// dab-pen syringe) meant to be vaporized, not swallowed. Found via
		// a live-catalog check flagged directly by a user who has actually
		// used one of these products ("Cannazo Uplift Plus... since no
		// carrier is there its vapable/smokable, its pure extract") — the
		// SAME "no carrier oil" reasoning this ruleset's own
		// concentrate_markers pattern already treats as an inhalation
		// signal in the vapeable branch below, applied here too for
		// consistency, plus dabApplicatorWords ("pen," "syringe") since
		// the real motivating product's only textual signal was neither a
		// concentrate_marker phrase nor a concentrateSubWords hit — its
		// variant name is literally "Uplift+ Pen."
		if d.viaConcentrateMarker {
			return domain.FormConcentrate, domain.RouteInhaled, true, confidence, d.ambiguous, d.reason
		}
		if ok, _ := MatchWordBoundary(concentrateSubWords, lowName); ok {
			return domain.FormConcentrate, domain.RouteInhaled, true, confidence, d.ambiguous, d.reason
		}
		if ok, _ := MatchWordBoundary(dabApplicatorWords, lowName); ok {
			return domain.FormConcentrate, domain.RouteInhaled, true, confidence, d.ambiguous, d.reason
		}
		return domain.FormConcentrate, domain.RouteOral, true, confidence, d.ambiguous, d.reason
	case "nutrition":
		// No "nutrition" form value exists in 03-DOMAIN-MODEL.md §2 — hemp
		// seed/protein products are still eaten (form=edible); their lack of
		// real cannabinoid content is what ConcentrationType=hemp_seed /
		// =nutrition already communicates. See M1-DECISIONS.md.
		return domain.FormEdible, domain.RouteOral, true, confidence, d.ambiguous, d.reason
	case "edible":
		// classify() already collapsed the legacy "edible_solid" bucket into
		// the public name "edible" (publicCat, matching the harvested
		// source's own behaviour) — so the capsule/gummy sub-split has to
		// happen HERE, by re-checking capsule-specific words against the
		// name, not via a d.primary=="edible_solid" case (that value never
		// reaches this switch). Caught by TestResolveFormMapping, not by
		// inspection — see M1-DECISIONS.md.
		if ok, _ := MatchWordBoundary(capsuleWords, lowName); ok {
			return domain.FormCapsule, domain.RouteOral, true, confidence, d.ambiguous, d.reason
		}
		return domain.FormEdible, domain.RouteOral, true, confidence, d.ambiguous, d.reason
	case "vapeable":
		// Same shape of bug, same fix: a marker-driven match ("not diluted
		// with any carrier oil") often has no vape/dab WORD in the name at
		// all — the marker fired on the DESCRIPTION. Re-scanning lowName for
		// concentrateSubWords alone misses that case entirely, so
		// viaConcentrateMarker (threaded from classify()) is checked first.
		if d.viaConcentrateMarker {
			return domain.FormConcentrate, domain.RouteInhaled, true, confidence, d.ambiguous, d.reason
		}
		if ok, _ := MatchWordBoundary(concentrateSubWords, lowName); ok {
			return domain.FormConcentrate, domain.RouteInhaled, true, confidence, d.ambiguous, d.reason
		}
		return domain.FormVape, domain.RouteInhaled, true, confidence, d.ambiguous, d.reason
	default: // "other" — no signal at all
		return "", "", false, 0, true, "could not determine product form"
	}
}

// ResolveForm classifies a listing's consumption form. Returns the rule-tier
// proposal for the `form` facet plus, in Evidence.Notes, the legacy
// secondary categories needed by legacy.go to reconstruct categories[].
func ResolveForm(rs *CategoryRuleSet, name, description, rawCategory string) FacetResult {
	d := classify(rs, name, description, rawCategory)
	form, _, _, confidence, ambiguous, reason := resolveForm(name, d)
	ev := d.evidence
	for _, s := range d.secondary {
		ev.Notes = append(ev.Notes, "legacy_secondary:"+s)
	}
	return FacetResult{Value: string(form), Confidence: confidence, Ambiguous: ambiguous, Reason: reason, Evidence: ev}
}

// ResolveRoute derives the route facet from the same classification —
// route gets the higher confidence bar in the publish gate
// (03-DOMAIN-MODEL.md §2: "Route gets the higher bar because it is the
// safety-relevant field"), reflected here as a flat +0.05 over the form
// confidence: route is a strictly-derived, mechanical mapping from form once
// form is known, never an independent text match, so a correct form implies
// a very slightly more certain route.
func ResolveRoute(rs *CategoryRuleSet, name, description, rawCategory string) FacetResult {
	d := classify(rs, name, description, rawCategory)
	_, route, hasRoute, confidence, ambiguous, reason := resolveForm(name, d)
	if !hasRoute {
		return FacetResult{Ambiguous: true, Reason: reason, Evidence: d.evidence}
	}
	routeConfidence := confidence + 0.05
	if routeConfidence > 1.0 {
		routeConfidence = 1.0
	}
	return FacetResult{Value: string(route), Confidence: routeConfidence, Ambiguous: ambiguous, Reason: reason, Evidence: d.evidence}
}

// ResolveExtract classifies extract type — full_spectrum, broad_spectrum, or
// isolate. Ported from the harvested ClassifyExtractType
// (dr-toke-init/apps/api/internal/catalog/normaliser/extractor.go), ruleset-
// driven word checks rather than the original's inline strings.Contains
// calls, same isolate/protein-isolate disambiguation.
func ResolveExtract(name, description string) FacetResult {
	combined := strings.ToLower(name + " " + description)
	switch {
	case strings.Contains(combined, "vijaya"):
		// Not one of the three extract facet values — Vijaya is a
		// full-spectrum-adjacent Indian Ayurvedic formulation term.
		// Treated as full_spectrum, the closest real category, flagged low
		// confidence for a human to confirm rather than asserted outright.
		return FacetResult{Value: string(domain.ExtractFullSpectrum), Confidence: 0.6, Ambiguous: true, Reason: "vijaya-labelled, mapped to full_spectrum for review"}
	case strings.Contains(combined, "full spectrum") || strings.Contains(combined, "full-spectrum"):
		return FacetResult{Value: string(domain.ExtractFullSpectrum), Confidence: 0.9}
	case strings.Contains(combined, "broad spectrum") || strings.Contains(combined, "broad-spectrum"):
		return FacetResult{Value: string(domain.ExtractBroadSpectrum), Confidence: 0.9}
	case isCannabinoidIsolate(combined):
		return FacetResult{Value: string(domain.ExtractIsolate), Confidence: 0.9}
	default:
		return FacetResult{Ambiguous: true, Reason: "no extract-type signal"}
	}
}

// isCannabinoidIsolate distinguishes a CBD isolate from "pea/whey/soy
// PROTEIN isolate" ingredients in hemp food — verbatim logic from the
// harvested source, the exact fix for "protein isolate" over-matching.
func isCannabinoidIsolate(s string) bool {
	if strings.Contains(s, "cbd isolate") || strings.Contains(s, "cannabidiol isolate") ||
		strings.Contains(s, "cbg isolate") || strings.Contains(s, "isolate tincture") {
		return true
	}
	if !strings.Contains(s, "isolate") {
		return false
	}
	if strings.Contains(s, "protein isolate") || strings.Contains(s, "whey") ||
		strings.Contains(s, "pea protein") || strings.Contains(s, "soy protein") ||
		strings.Contains(s, "protein bar") || strings.Contains(s, "protein powder") {
		return false
	}
	return strings.Contains(s, "cbd") || strings.Contains(s, "cannabinoid") || strings.Contains(s, "cannabidiol")
}

// ResolveProfile derives the `profile` facet — cbd_dominant, thc_dominant,
// or balanced. Not in 08-BUILD-ORDERS.md §7's explicit facets.go export
// list (only Form/Route/Extract/Carrier/Purchasable are named there), but
// 03-DOMAIN-MODEL.md §2 lists `profile` as one of the six facets and is
// explicit that it's "derived from mg ratio, NEVER from text" — a gap
// between the build order's file-level export list and the actual facet
// vocabulary it's supposed to fully cover. Added here rather than left
// missing; flagged in M1-DECISIONS.md.
//
// Same >=3x dominance threshold as cannabinoids.go's finalize() — reused
// deliberately rather than inventing a second threshold for what is, in
// effect, the same "how dominant is dominant" question asked from the facet
// side instead of the concentration_type side.
func ResolveProfile(cbdMg, thcMg float64) FacetResult {
	if cbdMg <= 0 && thcMg <= 0 {
		return FacetResult{Ambiguous: true, Reason: "no cannabinoid content to derive profile from"}
	}
	hi, lo := cbdMg, thcMg
	dominant := domain.ProfileCBDDominant
	if thcMg > cbdMg {
		hi, lo = thcMg, cbdMg
		dominant = domain.ProfileTHCDominant
	}
	// lo == 0 (the minority cannabinoid is exactly absent, e.g. a THC-free
	// CBD product) is the STRONGEST possible dominance, not an
	// undetermined case — caught by the starcbd-trace-thc.json golden
	// fixture (750mg CBD, 0mg THC after the trace-THC disclaimer) failing
	// against a first draft that required lo > 0 before checking the ratio,
	// which fell through to "balanced" on a divide-by-zero guard that was
	// guarding the wrong thing.
	if hi > 0 && (lo == 0 || hi/lo >= 3) {
		return FacetResult{Value: string(dominant), Confidence: 0.95}
	}
	return FacetResult{Value: string(domain.ProfileBalanced), Confidence: 0.95}
}

// ResolveCarrier classifies carrier oil. Ported from the harvested
// ClassifyCarrierOil, ruleset-independent (these four words never changed
// across the harvest, so they're inline here rather than in
// harvest/rules/categories.json — see M1-DECISIONS.md for why this one
// pattern set wasn't harvested into JSON).
func ResolveCarrier(description string) FacetResult {
	s := strings.ToLower(description)
	switch {
	case strings.Contains(s, "hemp seed") || strings.Contains(s, "hemp-seed"):
		return FacetResult{Value: string(domain.CarrierHempSeed), Confidence: 0.85}
	case strings.Contains(s, "mct") || strings.Contains(s, "coconut"):
		return FacetResult{Value: string(domain.CarrierMCT), Confidence: 0.85}
	case strings.Contains(s, "olive"):
		return FacetResult{Value: string(domain.CarrierOlive), Confidence: 0.85}
	default:
		return FacetResult{Value: string(domain.CarrierNone), Confidence: 0.3, Ambiguous: true, Reason: "no carrier-oil signal, defaulted to none"}
	}
}

// ResolvePurchasable is the 03-DOMAIN-MODEL.md §2 gate that "kills retreats,
// courses, consultations, merch" — reuses compliance.json's
// service_listing pattern conceptually, but that pattern lives in
// internal/compliance (M2), a different package. purchasable here is a
// narrower, resolve-local check: apparel is never purchasable as a cannabis
// product (it's merch), everything else defaults true unless a later
// compliance pass (M2) overrides it. This is intentionally minimal — full
// service/event-listing detection is compliance's job, not resolve's.
//
// Takes the same (name, description, rawCategory) signature as the other
// four Resolve* functions, not the internal formDetection type — it
// re-runs classify() itself rather than accepting an already-computed
// result, same redundant-but-independently-pure tradeoff the other four
// make (see M1-DECISIONS.md: "each Resolve* function re-classifies rather
// than sharing one call").
func ResolvePurchasable(rs *CategoryRuleSet, name, description, rawCategory string) FacetResult {
	d := classify(rs, name, description, rawCategory)
	if d.primary == "apparel" {
		return FacetResult{Value: "false", Confidence: 0.7, Ambiguous: true, Reason: "apparel/merch, not a cannabis product", Evidence: d.evidence}
	}
	return FacetResult{Value: "true", Confidence: 0.5, Evidence: d.evidence}
}
