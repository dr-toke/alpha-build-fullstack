package resolve

import (
	"reflect"
	"regexp"
	"testing"
)

// TestNormalizeAndTokens exists because a coverage check during the M1
// recheck found both at 0% — no _test.go for text.go was ever written, and
// nothing else in the package calls either function (Tokens isn't used by
// the regex-driven matchers here at all; see text.go's own doc comment).
// Exported API with zero coverage is a real gap regardless of whether
// anything depends on it yet — this closes it.
func TestNormalize(t *testing.T) {
	cases := []struct{ in, want string }{
		{"  Hello   World  ", "hello world"},
		{"CBD Oil\n\t500mg", "cbd oil 500mg"},
		{"", ""},
		{"already normal", "already normal"},
	}
	for _, c := range cases {
		if got := Normalize(c.in); got != c.want {
			t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTokens(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"CBD Oil 500mg", []string{"cbd", "oil", "500mg"}},
		{"", nil},
		{"---", nil},
		{"full-spectrum, broad_spectrum!", []string{"full", "spectrum", "broad_spectrum"}},
	}
	for _, c := range cases {
		got := Tokens(c.in)
		if len(got) == 0 && len(c.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("Tokens(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestNegationWindowsDirect(t *testing.T) {
	// match.go's ApplyNegation exercises this indirectly via the real
	// harvested patterns; this test checks NegationWindows itself, in
	// isolation, against a hand-built pattern — the layer boundary the
	// package is actually organized around (text.go is pattern-agnostic).
	re := regexp.MustCompile(`\bno\b`)

	t.Run("no match returns input unchanged, nil windows", func(t *testing.T) {
		stripped, windows := NegationWindows("yes indeed", re)
		if stripped != "yes indeed" || windows != nil {
			t.Errorf("got stripped=%q windows=%v, want unchanged/nil", stripped, windows)
		}
	})

	t.Run("single match replaced with a space, span recorded", func(t *testing.T) {
		stripped, windows := NegationWindows("say no thanks", re)
		if stripped != "say   thanks" {
			t.Errorf("stripped = %q", stripped)
		}
		if len(windows) != 1 || windows[0].Text != "no" {
			t.Errorf("windows = %+v, want one span with Text=no", windows)
		}
	})

	t.Run("multiple matches all replaced and all recorded", func(t *testing.T) {
		stripped, windows := NegationWindows("no no no", re)
		if len(windows) != 3 {
			t.Errorf("got %d windows, want 3", len(windows))
		}
		if regexp.MustCompile(`\bno\b`).MatchString(stripped) {
			t.Errorf("stripped text %q should no longer match the negation pattern", stripped)
		}
	})
}
