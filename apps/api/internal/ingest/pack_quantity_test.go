package ingest

import (
	"testing"

	"github.com/dr-toke/api/internal/resolve"
)

func TestMgLikelyPerUnitUnreconciled(t *testing.T) {
	t.Run("the real motivating case: '(90 Capsules)' with no reconciliation note", func(t *testing.T) {
		got := mgLikelyPerUnitUnreconciled(
			"CannaMed- Medical Cannabis Capsules 50mg (90 Capsules)",
			"Each capsule is packed with carefully measured doses of high-quality cannabis.",
			resolve.Evidence{},
		)
		if !got {
			t.Error("got false, want true — this is the exact live cbdstore.in case that produced a ~90x-too-high ₹/mg")
		}
	})

	t.Run("no pack-quantity wording at all: trusted", func(t *testing.T) {
		got := mgLikelyPerUnitUnreconciled(
			"BOHECO CBD Oil 500mg - 30ml",
			"Full spectrum CBD oil, MCT carrier, 500mg CBD.",
			resolve.Evidence{},
		)
		if got {
			t.Error("got true, want false — no pack-quantity pattern present")
		}
	})

	t.Run("pack-quantity wording present, but reconciliation already fired: trusted", func(t *testing.T) {
		got := mgLikelyPerUnitUnreconciled(
			"Cure By Design Gummies 1500mg (30 Gummies)",
			"Each gummy contains 50mg. Total 1500mg per pack.",
			resolve.Evidence{Notes: []string{"per-serving figure reconciled to pack total"}},
		)
		if got {
			t.Error("got true, want false — the extractor already reconciled a real pack total, don't second-guess it")
		}
	})

	t.Run("'pack of N' wording without parens also matches", func(t *testing.T) {
		got := mgLikelyPerUnitUnreconciled(
			"Cannazo Calm Tablets 25mg - Pack of 10",
			"",
			resolve.Evidence{},
		)
		if !got {
			t.Error("got false, want true — 'pack of 10' should be caught same as '(10 tablets)'")
		}
	})

	t.Run("pack-quantity wording only in description, not name, still matches", func(t *testing.T) {
		got := mgLikelyPerUnitUnreconciled(
			"Wellness Gummy Jar 200mg",
			"Comes in a jar of (40 gummies) for daily use.",
			resolve.Evidence{},
		)
		if !got {
			t.Error("got false, want true — pattern should be checked against both name and description")
		}
	})
}
