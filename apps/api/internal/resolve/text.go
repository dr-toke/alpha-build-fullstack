package resolve

import (
	"regexp"
	"strings"
)

var reWhitespace = regexp.MustCompile(`\s+`)

// Normalize lowercases and collapses whitespace. Every matcher in this
// package operates on normalized text — harvest/rules/*.json's patterns were
// transcribed with (?i) case-insensitivity from source that already assumed
// this, and word-boundary matching is meaningless across inconsistent
// whitespace.
func Normalize(s string) string {
	return strings.TrimSpace(reWhitespace.ReplaceAllString(strings.ToLower(s), " "))
}

var reTokenSplit = regexp.MustCompile(`[^a-z0-9]+`)

// Tokens splits normalized text into non-empty word tokens. Not used by the
// regex-driven matchers in this package (harvest/rules/categories.json's
// word lists are matched as compiled word-boundary alternations directly
// against the string, not token-by-token — see match.go) — this exists for
// anything downstream that needs a token count or a token-level diff (e.g.
// an admin dry-run view highlighting which words changed a classification).
func Tokens(s string) []string {
	raw := reTokenSplit.Split(Normalize(s), -1)
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

// NegationWindows finds every match of a caller-supplied negation pattern
// (harvest/rules/categories.json's negation_strip_pattern /
// negation_strip_pattern_2 — loaded via RuleSet, not hardcoded here: this
// file is pattern-agnostic on purpose, ruleset.go owns what the patterns
// are), replaces each match with a single space in the returned text, and
// reports every replaced span so the caller can record what was negated in
// an Evidence.Negated list rather than silently discarding it.
//
// Callers apply this BEFORE any word-boundary matching runs — categories.go
// harvested behaviour (harvest/rules/categories.json: negation_note):
// "Both patterns are applied to name AND description BEFORE any form
// detection runs, not as a post-hoc filter."
func NegationWindows(s string, negation *regexp.Regexp) (stripped string, windows []Span) {
	locs := negation.FindAllStringIndex(s, -1)
	if len(locs) == 0 {
		return s, nil
	}
	var b strings.Builder
	last := 0
	for _, loc := range locs {
		start, end := loc[0], loc[1]
		b.WriteString(s[last:start])
		b.WriteByte(' ')
		windows = append(windows, Span{Start: start, End: end, Text: s[start:end]})
		last = end
	}
	b.WriteString(s[last:])
	return b.String(), windows
}
