package ingest

import (
	"testing"

	"github.com/dr-toke/api/internal/resolve"
)

func TestHasCannabisContext(t *testing.T) {
	rs, err := resolve.LoadRuleSet("../../harvest/rules")
	if err != nil {
		t.Fatalf("resolve.LoadRuleSet: %v", err)
	}

	t.Run("real non-cannabis wellness items found in a live catalog audit: no context", func(t *testing.T) {
		cases := []struct{ name, description string }{
			{"Butterfly Ayurveda Masala Chai | Immunity Boosting Tea - 40 Tea Bags", "A traditional spiced tea blend."},
			{"Miracles Mushroom: Reishi Mushroom Extract Powder 110g", "Adaptogenic mushroom extract for wellness."},
			{"Grip Yoga Wheel - A Perfect Prop For Any Level Of Yoga Enthusiast", "Helps stretch and massage the back."},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				if hasCannabisContext(rs, c.name, c.description) {
					t.Errorf("expected no cannabis context for %q", c.name)
				}
			})
		}
	})

	t.Run("real cannabis products: has context", func(t *testing.T) {
		cases := []struct{ name, description string }{
			{"BOHECO CBD Oil 500mg - 30ml", "Full spectrum CBD oil."},
			{"Vijaya Leaf Extract Capsules", "Made from cannabis leaf extract."},
			{"Generic Wellness Tincture", "Contains cannabidiol as the active ingredient."},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				if !hasCannabisContext(rs, c.name, c.description) {
					t.Errorf("expected cannabis context for %q", c.name)
				}
			})
		}
	})

	t.Run("context word in description alone is enough", func(t *testing.T) {
		if !hasCannabisContext(rs, "Wellness Oil", "This is a broad spectrum formulation.") {
			t.Error("expected description-only context to be detected")
		}
	})
}
