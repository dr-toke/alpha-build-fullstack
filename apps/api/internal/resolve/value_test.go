package resolve

import (
	"testing"

	"github.com/dr-toke/api/internal/domain"
)

func TestPerMg(t *testing.T) {
	t.Run("computes rupees per mg from paise", func(t *testing.T) {
		// 249900 paise = 2499.00 = wait no, price_paise IS already paise;
		// 249900 paise = ₹2499.00. At 909mg that's ~2.7488/mg.
		got := PerMg(249900, 909)
		if got == nil {
			t.Fatal("got nil, want a value")
		}
		want := 2499.0 / 909.0
		if diff := *got - want; diff > 0.001 || diff < -0.001 {
			t.Errorf("PerMg = %v, want %v", *got, want)
		}
	})

	t.Run("nil for zero mg, never zero", func(t *testing.T) {
		if got := PerMg(100000, 0); got != nil {
			t.Errorf("got %v, want nil (00-CONSTITUTION.md §5: never zero for unknown)", *got)
		}
	})

	t.Run("nil for negative mg", func(t *testing.T) {
		if got := PerMg(100000, -5); got != nil {
			t.Errorf("got %v, want nil", *got)
		}
	})
}

func TestValueTier(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	cases := []struct {
		name string
		in   *float64
		want *domain.ValueTier
	}{
		{"nil stays nil", nil, nil},
		{"just under 3 is good", f(2.99), tierPtr(domain.ValueTierGood)},
		{"exactly 3 is mid, not good (< is strict)", f(3.0), tierPtr(domain.ValueTierMid)},
		{"exactly 8 is mid (<=  is inclusive)", f(8.0), tierPtr(domain.ValueTierMid)},
		{"just over 8 is high", f(8.01), tierPtr(domain.ValueTierHigh)},
		{"zero is good (< 3)", f(0), tierPtr(domain.ValueTierGood)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ValueTier(c.in)
			if (got == nil) != (c.want == nil) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			if got != nil && *got != *c.want {
				t.Errorf("got %v, want %v", *got, *c.want)
			}
		})
	}
}

func tierPtr(t domain.ValueTier) *domain.ValueTier { return &t }

func TestDominantPerMg(t *testing.T) {
	f := func(v float64) *float64 { return &v }

	t.Run("THC wins when present, ADR-013", func(t *testing.T) {
		perMg, basis := DominantPerMg(f(2.0), f(6.0), f(3.0))
		if basis != domain.BasisTHC || *perMg != 6.0 {
			t.Errorf("got perMg=%v basis=%s, want 6.0/thc", perMg, basis)
		}
	})
	t.Run("CBD wins when THC absent", func(t *testing.T) {
		perMg, basis := DominantPerMg(f(2.0), nil, f(3.0))
		if basis != domain.BasisCBD || *perMg != 2.0 {
			t.Errorf("got perMg=%v basis=%s, want 2.0/cbd", perMg, basis)
		}
	})
	t.Run("total wins when only total present", func(t *testing.T) {
		perMg, basis := DominantPerMg(nil, nil, f(3.0))
		if basis != domain.BasisTotal || *perMg != 3.0 {
			t.Errorf("got perMg=%v basis=%s, want 3.0/total", perMg, basis)
		}
	})
	t.Run("nothing present -> nil, empty basis", func(t *testing.T) {
		perMg, basis := DominantPerMg(nil, nil, nil)
		if perMg != nil || basis != "" {
			t.Errorf("got perMg=%v basis=%q, want nil/empty", perMg, basis)
		}
	})
}

func TestRankScore(t *testing.T) {
	t.Run("multiplies all four factors", func(t *testing.T) {
		got := RankScore(2.0, 0.9, 1.1, 0.8)
		want := 1.584 // 2.0 * 0.9 * 1.1 * 0.8, float64 arithmetic isn't exact
		if diff := got - want; diff > 1e-9 || diff < -1e-9 {
			t.Errorf("got %v, want %v", got, want)
		}
	})
	t.Run("zero confidence zeroes the whole score — the safety mechanism the docs describe", func(t *testing.T) {
		// This is the actual correction for "pure ₹/mg crowns a misparsed
		// mg figure" — a suspiciously great value_score from bad data gets
		// killed by low facet_confidence, not by capping value_score itself.
		got := RankScore(1000.0, 0.0, 1.0, 1.0)
		if got != 0 {
			t.Errorf("got %v, want 0", got)
		}
	})
}

func TestValueScore(t *testing.T) {
	t.Run("cheaper per-mg scores higher", func(t *testing.T) {
		cheap := ValueScore(1.0)
		expensive := ValueScore(10.0)
		if cheap <= expensive {
			t.Errorf("cheap(%v) should score higher than expensive(%v)", cheap, expensive)
		}
	})
	t.Run("zero or negative perMg scores zero, not a divide-by-zero panic", func(t *testing.T) {
		if got := ValueScore(0); got != 0 {
			t.Errorf("got %v, want 0", got)
		}
		if got := ValueScore(-1); got != 0 {
			t.Errorf("got %v, want 0", got)
		}
	})
}
