package resolve

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/dr-toke/api/internal/domain"
)

// goldenGoldenDir points at the real testdata/golden/ fixtures — the ones
// converted from the prior alpha's actual passing tests during the harvest
// session (harvest/NOTES.md), plus one (hemp-seed-oil.json) corrected during
// M1 once running the real classifier against it revealed the harvest-time
// guess ("nutrition" as a form value) didn't match what the code — written
// after the fixture — actually produces. See M1-DECISIONS.md.
const goldenDir = "../../testdata/golden"

type goldenFixture struct {
	Source string `json:"source"`
	Raw    struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"raw"`
	Expect map[string]any `json:"expect"`
}

// TestGoldenFixtures runs the FULL pipeline — cannabinoid extraction, facet
// classification, new-facet mapping, legacy category reconstruction — against
// every fixture in testdata/golden/, checking whichever "expect" keys each
// fixture happens to declare. Fixtures are heterogeneous on purpose (some
// assert cannabinoid math, some assert form/route, some assert legacy
// category compatibility) rather than forced into one rigid schema —
// 11-HARVEST.md §2.6's example format was illustrative, not a contract every
// fixture must match key-for-key.
//
// This is the test 03-DOMAIN-MODEL.md §2 describes as the whole point of the
// override system: "Every override auto-appends a fixture... a classifier
// version cannot ship unless it passes 100%." M1 has no override-writing
// path yet (that's M3/M9's job), but the CI gate this test represents is
// exactly what that mechanism plugs into later.
func TestGoldenFixtures(t *testing.T) {
	rs, err := LoadRuleSet(harvestRulesDir)
	if err != nil {
		t.Fatalf("LoadRuleSet: %v", err)
	}

	entries, err := os.ReadDir(goldenDir)
	if err != nil {
		t.Fatalf("reading %s: %v", goldenDir, err)
	}
	if len(entries) == 0 {
		t.Fatal("no golden fixtures found — testdata/golden/ should not be empty")
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		t.Run(e.Name(), func(t *testing.T) {
			path := filepath.Join(goldenDir, e.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var fx goldenFixture
			if err := json.Unmarshal(data, &fx); err != nil {
				t.Fatalf("parsing %s: %v", path, err)
			}
			checkGoldenFixture(t, rs, fx)
		})
	}
}

func checkGoldenFixture(t *testing.T, rs *RuleSet, fx goldenFixture) {
	t.Helper()
	name, desc := fx.Raw.Title, fx.Raw.Description
	exp := fx.Expect

	cb := ExtractCannabinoids(&rs.Cannabinoids, name, desc)

	if want, ok := exp["cbd_mg"]; ok {
		checkFloatOrNull(t, "cbd_mg", cb.CBDMg, cb.ConcentrationType == domain.ConcentrationHempSeed || cb.ConcentrationType == domain.ConcentrationUnknown, want)
	}
	if want, ok := exp["thc_mg"]; ok {
		checkFloatOrNull(t, "thc_mg", cb.THCMg, cb.ConcentrationType == domain.ConcentrationHempSeed || cb.ConcentrationType == domain.ConcentrationUnknown, want)
	}
	if want, ok := exp["concentration_type"]; ok {
		if string(cb.ConcentrationType) != want {
			t.Errorf("concentration_type = %s, want %s", cb.ConcentrationType, want)
		}
	}
	if want, ok := exp["profile"]; ok {
		p := ResolveProfile(cb.CBDMg, cb.THCMg)
		if p.Value != want {
			t.Errorf("profile = %s, want %s", p.Value, want)
		}
	}
	if want, ok := exp["price_per_mg"]; ok && want == nil {
		if perMg := PerMg(100000, cb.BestMG()); perMg != nil {
			t.Errorf("price_per_mg = %v, want nil (BestMG=%v)", *perMg, cb.BestMG())
		}
	}

	d := classify(&rs.Categories, name, desc, "")
	form, route, _, _, ambiguous, _ := resolveForm(name, d)
	category, secondary := LegacyCategory(form, route, cb.ConcentrationType)
	categories := LegacyCategories(category, secondary)

	if want, ok := exp["form"]; ok {
		got := string(form)
		if form == "" {
			got = "other" // unresolved form has no domain.FormValue; the legacy/API-facing name is "other"
		}
		if got != want {
			t.Errorf("form = %s, want %s", got, want)
		}
	}
	if want, ok := exp["route"]; ok {
		if string(route) != want {
			t.Errorf("route = %s, want %s", route, want)
		}
	}
	if want, ok := exp["ambiguous"]; ok {
		if ambiguous != want {
			t.Errorf("ambiguous = %v, want %v", ambiguous, want)
		}
	}
	if want, ok := exp["primary_category_legacy"]; ok {
		if category != want {
			t.Errorf("legacy category = %s, want %s", category, want)
		}
	}
	if wantList, ok := exp["forbid_categories_legacy"]; ok {
		for _, w := range wantList.([]any) {
			for _, got := range categories {
				if got == w {
					t.Errorf("legacy categories %v must NOT contain %q", categories, w)
				}
			}
		}
	}
	if want, ok := exp["purchasable"]; ok {
		p := ResolvePurchasable(&rs.Categories, name, desc, "")
		gotBool := p.Value == "true"
		if gotBool != want {
			t.Errorf("purchasable = %v, want %v", gotBool, want)
		}
	}
}

// checkFloatOrNull handles the JSON-null-means-nil-pointer convention every
// golden fixture with cannabinoid expectations uses.
func checkFloatOrNull(t *testing.T, field string, got float64, gotIsNullish bool, want any) {
	t.Helper()
	if want == nil {
		if got != 0 {
			t.Errorf("%s = %v, want null/0 (hemp_seed or unknown)", field, got)
		}
		return
	}
	wantF, ok := want.(float64)
	if !ok {
		t.Errorf("%s: fixture expect value %v is not a number", field, want)
		return
	}
	if diff := got - wantF; diff > 0.5 || diff < -0.5 {
		t.Errorf("%s = %v, want %v", field, got, wantF)
	}
}
