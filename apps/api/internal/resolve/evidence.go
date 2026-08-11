// Package resolve is the rule engine: cannabinoid extraction, facet
// classification, precedence, and value/rank scoring. Pure functions only —
// no SQL, no HTTP, no I/O beyond LoadRuleSet reading harvest/rules/*.json at
// startup. 08-BUILD-ORDERS.md §7, M1: "where the false-positive problem
// actually dies... every file testable in isolation."
package resolve

// Span is one matched region of source text — a rule firing, a negation
// window applied, or a value captured. Offsets are byte offsets into the
// text that was matched against (name, description, or name+" "+description
// depending on the caller — see each Resolve*/Extract* function's doc).
type Span struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Text  string `json:"text"`
	Rule  string `json:"rule"` // which named pattern in a RuleSet fired
}

// Evidence is what makes a facet or cannabinoid decision auditable —
// 03-DOMAIN-MODEL.md §2: "It is what makes the review queue possible and a
// six-month-old bug debuggable." Stored verbatim as product_facets.evidence
// / product_clusters.cannabinoid_evidence (jsonb).
type Evidence struct {
	// Matched is every span that contributed to the final value.
	Matched []Span `json:"matched,omitempty"`
	// Negated is every span that matched a pattern but was discarded because
	// it fell inside a negation window (harvest/rules/categories.json:
	// negation_strip_pattern) — recorded, not silently dropped, so a review
	// can see WHY "vape" in "no need to vape" didn't count.
	Negated []Span `json:"negated,omitempty"`
	// Notes carries anything that doesn't fit a span — e.g. cannabinoids.go's
	// per-serving-to-pack reconciliation factor, or facets.go's coherence-
	// matrix deletions ("removed vapeable: topical present").
	Notes []string `json:"notes,omitempty"`
}

// Merge combines two Evidence values, concatenating their slices. Used when
// a decision draws on matches from more than one source text (e.g. name AND
// description) or more than one stage (detection, then coherence-matrix
// pruning) and both need to survive into the stored record.
func Merge(a, b Evidence) Evidence {
	return Evidence{
		Matched: append(append([]Span{}, a.Matched...), b.Matched...),
		Negated: append(append([]Span{}, a.Negated...), b.Negated...),
		Notes:   append(append([]string{}, a.Notes...), b.Notes...),
	}
}
