package resolve

import "regexp"

// MatchWordBoundary reports whether a compiled word-boundary pattern matches
// s, and every span where it did — used instead of a bare bool everywhere a
// caller needs to build Evidence, which is everywhere in this package (see
// 03-DOMAIN-MODEL.md §2: evidence is "what makes the review queue possible").
func MatchWordBoundary(re *regexp.Regexp, s string) (matched bool, spans []Span) {
	locs := re.FindAllStringIndex(s, -1)
	if len(locs) == 0 {
		return false, nil
	}
	spans = make([]Span, len(locs))
	for i, loc := range locs {
		spans[i] = Span{Start: loc[0], End: loc[1], Text: s[loc[0]:loc[1]]}
	}
	return true, spans
}

// ApplyNegation runs both of harvest/rules/categories.json's negation
// patterns over s, in the same order as the harvested source's
// stripNegations(): the general negation-window pattern first ("no need to
// smoke or vape"), then the standalone "smoke-free"/"vape-free" pattern
// second. Returns the fully-stripped text plus every window that was
// removed, tagged with which pattern caught it, for Evidence.Negated.
func ApplyNegation(s string, cat *CategoryRuleSet) (stripped string, negated []Span) {
	s1, w1 := NegationWindows(s, cat.NegationPrimary)
	for i := range w1 {
		w1[i].Rule = "negation_strip_pattern"
	}
	s2, w2 := NegationWindows(s1, cat.NegationFormFree)
	for i := range w2 {
		w2[i].Rule = "negation_strip_pattern_2"
	}
	return s2, append(w1, w2...)
}
