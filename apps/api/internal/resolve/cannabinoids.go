package resolve

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/dr-toke/api/internal/domain"
)

// CannabinoidExtraction is ExtractCannabinoids' pure-function result — not
// yet written anywhere. A caller (M3's store layer or M5's normalise job)
// maps this onto domain.ProductCluster's CBDMg/THCMg/TotalCannabinoidsMg/
// ConcentrationType/CannabinoidConfidence/CannabinoidEvidence fields.
type CannabinoidExtraction struct {
	CBDMg               float64
	THCMg               float64
	TotalCannabinoidsMg float64
	ConcentrationType   domain.ConcentrationType
	// Confidence is NOT part of the ported algorithm — the harvested source
	// (dr-toke-init's ExtractCannabinoids) never computed one; it always
	// returned a definite type. 03-DOMAIN-MODEL.md §3 requires
	// cannabinoid_confidence though, so this is a new, judgment-call
	// heuristic keyed to which branch resolved the value, loosely anchored
	// to the illustrative numbers in 11-HARVEST.md §2.2's example
	// (explicit ~0.95, generic fallback ~0.5). Flagged in M1-DECISIONS.md —
	// review before trusting it for the confidence gate in
	// 03-DOMAIN-MODEL.md §2.
	Confidence float32
	Evidence   Evidence
}

// BestMG returns the mg of the DOMINANT cannabinoid — i.e. the basis named
// by ConcentrationType — mirroring the harvested CannabinoidResult.BestMG()
// exactly: "A 10:1 THC:CBD product must price per mg of THC, not its trace
// CBD."
func (r CannabinoidExtraction) BestMG() float64 {
	switch r.ConcentrationType {
	case domain.ConcentrationCBD:
		return r.CBDMg
	case domain.ConcentrationTHC:
		return r.THCMg
	case domain.ConcentrationTotal:
		return r.TotalCannabinoidsMg
	}
	if r.CBDMg > 0 {
		return r.CBDMg
	}
	if r.THCMg > 0 {
		return r.THCMg
	}
	return r.TotalCannabinoidsMg
}

// ExtractSize returns the pack size implied by a listing's name+description
// — volumeML for liquids, weightG for solids, whichever pattern matches
// first (0 for the one that doesn't apply). NOT part of the harvested
// CannabinoidExtraction — the prior alpha's ExtractCannabinoids computed
// this internally (as `base`, below) purely to scale percent-based
// cannabinoid figures and never returned it. Exported here as its own
// function, NEW for M4/M5, because harvest/rules/dedup.md's Fingerprint
// needs a volumeML input and nothing before this milestone exposed one —
// reuses the exact same rs.Patterns["volume_ml"]/["weight_g"] this file
// already loads and validates, rather than inventing a second extraction
// path.
func ExtractSize(rs *CannabinoidRuleSet, name, description string) (volumeML, weightG float64) {
	combined := name + " " + description
	volumeML = firstMatchFloat(rs.Patterns["volume_ml"], combined)
	weightG = firstMatchFloat(rs.Patterns["weight_g"], combined)
	return
}

// ExtractCannabinoids parses cannabinoid content from a listing's name and
// description. Faithful port of the prior alpha's
// internal/catalog/normaliser/cannabinoids.go — the control flow (branch
// order, ratio orientation, per-serving reconciliation, name-first identity)
// is real Go, not data-driven, per harvest/NOTES.md's explicit finding that
// this algorithm cannot be flattened into a priority-match list without
// losing behaviour. Every named pattern is loaded from rs, none hardcoded —
// 00-CONSTITUTION.md §6.
//
// It captures CBD and THC independently (not first-match-wins) and does NOT
// early-return after the first explicit match within that step — see
// harvest/rules/cannabinoids.json's "Do not early-return" note; a product
// can carry both cannabinoids and both must be captured before any return.
func ExtractCannabinoids(rs *CannabinoidRuleSet, name, description string) CannabinoidExtraction {
	p := rs.Patterns
	var ev Evidence

	combined := name + " " + description
	combined = p["chem_name_cannabidiol"].ReplaceAllString(combined, "CBD")
	combined = p["chem_name_thc"].ReplaceAllString(combined, "THC")
	low := strings.ToLower(combined)
	thcFree := p["thc_free"].MatchString(combined)

	// NAME-FIRST hemp seed: a product NAMED "Hemp Seed Oil" is seed oil,
	// full stop, even if the description name-drops CBD for comparison.
	lowNameEarly := strings.ToLower(name)
	if p["hemp_seed"].MatchString(name) && !strings.Contains(lowNameEarly, "cbd") && !strings.Contains(lowNameEarly, "vijaya") {
		ev.Notes = append(ev.Notes, "hemp_seed: name-first match, no cbd/vijaya qualifier in name")
		return CannabinoidExtraction{ConcentrationType: domain.ConcentrationHempSeed, Confidence: 1.0, Evidence: ev}
	}

	// Scrub the "CBD + THC" PAIR so a number adjacent to it can't be
	// misattributed to one side by the per-cannabinoid matchers below.
	scrubbed := p["cannabinoid_pair"].ReplaceAllString(combined, " cannabinoid-blend ")

	var cbd, thc float64

	cbd, cbdSpans := explicitMG(p["cbd_label_first"], p["cbd_num_first"], scrubbed)
	var thcSpans []Span
	if !thcFree {
		thc, thcSpans = explicitMG(p["thc_label_first"], p["thc_num_first"], scrubbed)
	}
	ev.Matched = append(ev.Matched, cbdSpans...)
	ev.Matched = append(ev.Matched, thcSpans...)

	base := firstMatchFloat(p["volume_ml"], combined)
	if base == 0 {
		base = firstMatchFloat(p["weight_g"], combined)
	}
	if cbd == 0 && base > 0 {
		if m := p["pct_cbd"].FindStringSubmatch(scrubbed); m != nil {
			cbd = round2(parseF(m[1]) / 100 * base * 1000)
			ev.Notes = append(ev.Notes, "cbd from percent+volume/weight")
		}
	}
	if thc == 0 && !thcFree && base > 0 {
		if m := p["pct_thc"].FindStringSubmatch(scrubbed); m != nil {
			// <=0.3% THC is the hemp legal limit — a compliance disclaimer,
			// not a dosed cannabinoid.
			if pct := parseF(m[1]); pct > 0.3 {
				thc = round2(pct / 100 * base * 1000)
				ev.Notes = append(ev.Notes, "thc from percent+volume/weight")
			}
		}
	}

	// The headline total is NAME-FIRST: descriptions carry raw-herb weights
	// and pack-variant figures that are NOT this product's cannabinoid content.
	nameNorm := p["chem_name_thc"].ReplaceAllString(p["chem_name_cannabidiol"].ReplaceAllString(name, "CBD"), "THC")
	total := maxMatchFloat(p["generic_mg"], nameNorm)
	if total == 0 {
		total = maxMatchFloat(p["generic_mg"], combined)
	}

	if cbd > 0 || thc > 0 {
		// Per-serving reconciliation: "2.1mg of CBD per drop" is far below
		// the pack-level name total — scale up preserving the ratio.
		if total > 0 && (cbd+thc)*5 <= total {
			f := total / (cbd + thc)
			cbd, thc = round2(cbd*f), round2(thc*f)
			ev.Notes = append(ev.Notes, "per-serving figure reconciled to pack total")
		}
		return finalize(cbd, thc, thcFree, 0.95, ev)
	}

	// Labelled ratio + a single total mg -> split the total by the ratio.
	// NOT zeroed by thcFree — a labelled ratio is the product's own dosing
	// claim, more specific than generic THC-free boilerplate.
	if total > 0 {
		if thcParts, cbdParts, ok := extractRatio(p, combined); ok {
			sum := thcParts + cbdParts
			if sum > 0 {
				thc = round2(thcParts / sum * total)
				cbd = round2(cbdParts / sum * total)
				ev.Notes = append(ev.Notes, "labelled ratio applied to name-first total")
				return finalize(cbd, thc, false, 0.9, ev)
			}
		}
	}

	// Identity resolved NAME-FIRST, same principle as category classification.
	lowName := strings.ToLower(name)
	thcTrace := p["trace_thc"].MatchString(combined)
	thcDominant := p["thc_dominant_wording"].MatchString(combined) && !thcFree
	cbdNamed := strings.Contains(lowName, "cbd")
	thcNamed := strings.Contains(lowName, "thc") && !thcFree && !p["thc_free"].MatchString(lowName)

	dominant := ""
	switch {
	case thcDominant:
		dominant = "thc"
	case cbdNamed && !thcNamed:
		dominant = "cbd"
	case thcNamed && !cbdNamed:
		dominant = "thc"
	case strings.Contains(low, "cbd") && (thcFree || thcTrace || !strings.Contains(low, "thc")):
		dominant = "cbd"
	case strings.Contains(low, "thc") && !strings.Contains(low, "cbd") && !thcFree:
		dominant = "thc"
	}

	// Bare ratio ("(10:1)", "1:3", "1:1"). EQUAL needs no orientation at all.
	if total > 0 && !thcFree {
		if m := p["ratio_bare"].FindStringSubmatch(combined); m != nil {
			a, b := parseF(m[1]), parseF(m[2])
			bothInPlay := strings.Contains(low, "cbd") && strings.Contains(low, "thc") && !thcTrace
			switch {
			case a > 0 && a == b && bothInPlay:
				ev.Notes = append(ev.Notes, "equal bare ratio, balanced split, no orientation needed")
				return finalize(total/2, total/2, false, 0.85, ev)
			case a > 0 && b > 0 && a != b && dominant != "":
				hi, lo := a, b
				if b > a {
					hi, lo = b, a
				}
				major := round2(hi / (hi + lo) * total)
				minor := round2(lo / (hi + lo) * total)
				ev.Notes = append(ev.Notes, "unequal bare ratio oriented by name-first dominant: "+dominant)
				if dominant == "cbd" {
					return finalize(major, minor, thcFree, 0.75, ev)
				}
				return finalize(minor, major, thcFree, 0.75, ev)
			}
		}
	}

	// Hemp seed / nutrition — no cannabinoids, checked again here (not just
	// name-first-early) because the description-level signal can still fire.
	if p["hemp_seed"].MatchString(combined) && !strings.Contains(low, "cbd") && !strings.Contains(low, "vijaya") {
		ev.Notes = append(ev.Notes, "hemp_seed: description-level match")
		return CannabinoidExtraction{ConcentrationType: domain.ConcentrationHempSeed, Confidence: 0.9, Evidence: ev}
	}

	// Generic mg, attributed by the identity resolved above.
	if total > 0 {
		bothMentioned := strings.Contains(low, "cbd") && strings.Contains(low, "thc") && !thcFree && !thcTrace
		if dominant == "" && bothMentioned && !p["ratio_bare"].MatchString(combined) {
			ev.Notes = append(ev.Notes, "explicit CBD+THC pair, no printed split, assumed balanced 1:1")
			return finalize(total/2, total/2, thcFree, 0.6, ev)
		}
		switch {
		case dominant == "cbd":
			ev.Notes = append(ev.Notes, "generic mg attributed to cbd by name-first identity")
			return CannabinoidExtraction{CBDMg: total, TotalCannabinoidsMg: total, ConcentrationType: domain.ConcentrationCBD, Confidence: 0.6, Evidence: ev}
		case dominant == "thc":
			ev.Notes = append(ev.Notes, "generic mg attributed to thc by name-first identity")
			return CannabinoidExtraction{THCMg: total, TotalCannabinoidsMg: total, ConcentrationType: domain.ConcentrationTHC, Confidence: 0.6, Evidence: ev}
		case strings.Contains(low, "vijaya") || strings.Contains(low, "cannabis") ||
			strings.Contains(low, "cannabinoid") ||
			(strings.Contains(low, "cbd") && strings.Contains(low, "thc")):
			ev.Notes = append(ev.Notes, "generic mg, genuinely mixed/unknown split — honest total")
			return CannabinoidExtraction{TotalCannabinoidsMg: total, ConcentrationType: domain.ConcentrationTotal, Confidence: 0.5, Evidence: ev}
		default:
			ev.Notes = append(ev.Notes, "a number with no cannabinoid context")
			return CannabinoidExtraction{ConcentrationType: domain.ConcentrationUnknown, Confidence: 0, Evidence: ev}
		}
	}

	return CannabinoidExtraction{ConcentrationType: domain.ConcentrationUnknown, Confidence: 0, Evidence: ev}
}

// finalize sets the dominant basis from the CBD/THC split. A cannabinoid
// that is >=3x the other makes the product cbd- or thc-dominant; otherwise
// total. Both numbers are ALWAYS returned, never zeroed by the dominance
// call — 00-CONSTITUTION.md §5: never zero for unknown, and the dominance
// label must not destroy the minority cannabinoid's real value.
func finalize(cbd, thc float64, thcFree bool, confidence float32, ev Evidence) CannabinoidExtraction {
	if thcFree {
		thc = 0
	}
	r := CannabinoidExtraction{
		CBDMg:               round2(cbd),
		THCMg:               round2(thc),
		TotalCannabinoidsMg: round2(cbd + thc),
		Confidence:          confidence,
		Evidence:            ev,
	}
	switch {
	case cbd > 0 && thc > 0:
		hi, lo := cbd, thc
		if thc > cbd {
			hi, lo = thc, cbd
		}
		switch {
		case lo > 0 && hi/lo >= 3 && thc > cbd:
			r.ConcentrationType = domain.ConcentrationTHC
		case lo > 0 && hi/lo >= 3:
			r.ConcentrationType = domain.ConcentrationCBD
		default:
			r.ConcentrationType = domain.ConcentrationTotal
		}
	case cbd > 0:
		r.ConcentrationType = domain.ConcentrationCBD
	case thc > 0:
		r.ConcentrationType = domain.ConcentrationTHC
	default:
		r.ConcentrationType = domain.ConcentrationUnknown
	}
	return r
}

// extractRatio returns (thcParts, cbdParts, ok) for a THC:CBD ratio written
// in either order — "(10:1 THC:CBD)" and "THC:CBD ratio of 10:1" both parse.
func extractRatio(p map[string]*regexp.Regexp, s string) (float64, float64, bool) {
	if m := p["ratio_num_first"].FindStringSubmatch(s); m != nil {
		return orientRatio(m[3], parseF(m[1]), parseF(m[2]))
	}
	if m := p["ratio_label_first"].FindStringSubmatch(s); m != nil {
		return orientRatio(m[1], parseF(m[2]), parseF(m[3]))
	}
	return 0, 0, false
}

func orientRatio(label string, a, b float64) (float64, float64, bool) {
	if a <= 0 && b <= 0 {
		return 0, 0, false
	}
	if strings.HasPrefix(strings.ToLower(strings.ReplaceAll(label, " ", "")), "thc") {
		return a, b, true // thc:cbd = a:b
	}
	return b, a, true // cbd:thc = a:b -> thc=b, cbd=a
}

// explicitMG returns the mg for one cannabinoid, preferring the label-first
// form over number-first, and the LARGEST match within each form — texts
// list both per-serving and pack-level figures, and the pack-level figure is
// the larger one.
func explicitMG(labelFirst, numFirst *regexp.Regexp, s string) (float64, []Span) {
	if v, spans := maxSubmatch(labelFirst, s); v > 0 {
		return v, spans
	}
	return maxSubmatch(numFirst, s)
}

func maxSubmatch(re *regexp.Regexp, s string) (float64, []Span) {
	best := 0.0
	var bestSpan []Span
	for _, m := range re.FindAllStringSubmatchIndex(s, -1) {
		valStr := s[m[2]:m[3]]
		v := parseF(valStr)
		if v > best && v < 100000 {
			best = v
			bestSpan = []Span{{Start: m[0], End: m[1], Text: s[m[0]:m[1]]}}
		}
	}
	return best, bestSpan
}

func maxMatchFloat(re *regexp.Regexp, s string) float64 {
	best := 0.0
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		v := parseF(m[1])
		if v > best && v < 100000 {
			best = v
		}
	}
	return best
}

func firstMatchFloat(re *regexp.Regexp, s string) float64 {
	if m := re.FindStringSubmatch(s); m != nil {
		return parseF(m[1])
	}
	return 0
}

func parseF(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
