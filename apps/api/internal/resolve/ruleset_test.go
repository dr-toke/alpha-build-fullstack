package resolve

import (
	"os"
	"strings"
	"testing"
)

// harvestRulesDir points at the real harvest/rules/ directory — not a
// fixture copy. If someone edits harvest/rules/cannabinoids.json and breaks
// the JSON shape or a regex, this test fails immediately, not the next time
// a human happens to run the server.
const harvestRulesDir = "../../harvest/rules"

func TestLoadRuleSet(t *testing.T) {
	rs, err := LoadRuleSet(harvestRulesDir)
	if err != nil {
		t.Fatalf("LoadRuleSet(%s): %v", harvestRulesDir, err)
	}

	t.Run("cannabinoid patterns all present", func(t *testing.T) {
		want := []string{
			"cbd_label_first", "cbd_num_first", "thc_label_first", "thc_num_first",
			"pct_cbd", "pct_thc", "volume_ml", "weight_g", "generic_mg", "hemp_seed",
			"ratio_num_first", "ratio_label_first", "thc_dominant_wording", "ratio_bare",
			"chem_name_cannabidiol", "chem_name_thc", "thc_free", "trace_thc", "cannabinoid_pair",
		}
		for _, name := range want {
			if rs.Cannabinoids.Patterns[name] == nil {
				t.Errorf("missing compiled pattern %q", name)
			}
		}
	})

	t.Run("cannabinoid patterns actually match what they should", func(t *testing.T) {
		if !rs.Cannabinoids.Patterns["cbd_label_first"].MatchString("CBD 91mg") {
			t.Error("cbd_label_first should match 'CBD 91mg'")
		}
		if !rs.Cannabinoids.Patterns["thc_free"].MatchString("this product is THC-free") {
			t.Error("thc_free should match 'THC-free'")
		}
		// The specific regression this pattern exists to prevent — see
		// harvest/rules/cannabinoids.json's "nul_before_zero_in_thc_free" note.
		if rs.Cannabinoids.Patterns["thc_free"].MatchString("50% THC") {
			t.Error("thc_free must NOT match '50% THC' — the \\b before 0 is load-bearing")
		}
	})

	t.Run("category form word lists all present and functional", func(t *testing.T) {
		forms := []string{"edible_solid", "edible", "topical", "smokable", "vapeable", "tincture", "beverage", "extract"}
		for _, f := range forms {
			re := rs.Categories.FormWordLists[f]
			if re == nil {
				t.Errorf("missing compiled word list for form %q", f)
			}
		}
		if !rs.Categories.FormWordLists["edible_solid"].MatchString("100mg capsules") {
			t.Error("edible_solid should match 'capsules'")
		}
		// The bug this whole design fixes: bare "flower" (as in Bach Flower
		// remedies) must NOT be in the smokable list.
		if rs.Categories.FormWordLists["smokable"].MatchString("bach flower remedy") {
			t.Error("smokable must NOT match bare 'flower' — harvest/rules/categories.json deliberately excludes it")
		}
		if !rs.Categories.FormWordLists["smokable"].MatchString("cannabis flower") {
			t.Error("smokable SHOULD match 'cannabis flower' (qualified, not bare)")
		}
	})

	t.Run("negation pattern strips 'no need to smoke or vape'", func(t *testing.T) {
		stripped, windows := NegationWindows("no need to smoke or vape", rs.Categories.NegationPrimary)
		if len(windows) == 0 {
			t.Fatal("expected at least one negation window")
		}
		if rs.Categories.FormWordLists["vapeable"].MatchString(stripped) {
			t.Errorf("after negation-stripping, %q should not match vapeable, still does", stripped)
		}
	})

	t.Run("coherence matrix deletions loaded", func(t *testing.T) {
		if !rs.Categories.CoherenceMatrix.IfTopicalDelete["edible_solid"] {
			t.Error("topical should delete edible_solid in the coherence matrix")
		}
		if !rs.Categories.CoherenceMatrix.IfEdibleSolidDelete["vapeable"] {
			t.Error("edible_solid should delete vapeable in the coherence matrix")
		}
	})

	t.Run("pet and apparel are exclusive", func(t *testing.T) {
		exclusive := toSet(rs.Categories.Exclusive)
		if !exclusive["pet"] || !exclusive["apparel"] {
			t.Errorf("Exclusive = %v, want pet and apparel", rs.Categories.Exclusive)
		}
	})

	t.Run("secondary implications: extract implies edible", func(t *testing.T) {
		sec := rs.Categories.SecondaryImplications["extract"]
		if len(sec) != 1 || sec[0] != "edible" {
			t.Errorf("SecondaryImplications[extract] = %v, want [edible]", sec)
		}
	})
}

func TestLoadRuleSetMissingDir(t *testing.T) {
	if _, err := LoadRuleSet("/nonexistent/path/does/not/exist"); err == nil {
		t.Error("expected an error loading a nonexistent rules directory, got nil")
	}
}

// TestLoadRuleSetMissingRequiredKey proves the fail-fast validation added
// during the M1 recheck actually fails fast: a harvest/rules/*.json missing
// a key that cannabinoids.go/facets.go index directly (rs.Patterns["x"],
// rs.FormWordLists["x"], no per-call nil check) must error at LoadRuleSet
// time with a clear message, not panic later the first time business logic
// happens to reach that code path.
func TestLoadRuleSetMissingRequiredKey(t *testing.T) {
	t.Run("cannabinoids.json missing a required pattern", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir+"/cannabinoids.json", `{"patterns": {"cbd_label_first": "CBD"}}`)
		writeFile(t, dir+"/categories.json", validCategoriesJSON)

		_, err := LoadRuleSet(dir)
		if err == nil {
			t.Fatal("expected an error for missing required cannabinoid patterns, got nil")
		}
		if !strings.Contains(err.Error(), "missing required pattern") {
			t.Errorf("error = %q, want it to name the missing pattern clearly", err.Error())
		}
	})

	t.Run("categories.json missing a required form word list", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir+"/cannabinoids.json", validCannabinoidsJSON)
		writeFile(t, dir+"/categories.json", `{"form_word_lists": {"topical": ["balm"]}}`)

		_, err := LoadRuleSet(dir)
		if err == nil {
			t.Fatal("expected an error for missing required form word lists, got nil")
		}
		if !strings.Contains(err.Error(), "missing required form_word_lists entry") {
			t.Errorf("error = %q, want it to name the missing entry clearly", err.Error())
		}
	})
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Minimal-but-complete stand-ins covering every required key, used only to
// isolate the OTHER file's validation failure in the two subtests above.
var validCannabinoidsJSON = `{"patterns": {
	"cbd_label_first": "CBD", "cbd_num_first": "CBD", "thc_label_first": "THC", "thc_num_first": "THC",
	"pct_cbd": "CBD%", "pct_thc": "THC%", "volume_ml": "ml", "weight_g": "g", "generic_mg": "mg",
	"hemp_seed": "hemp", "ratio_num_first": "ratio", "ratio_label_first": "ratio",
	"thc_dominant_wording": "dominant", "ratio_bare": "bare", "chem_name_cannabidiol": "cannabidiol",
	"chem_name_thc": "tetrahydrocannabinol", "thc_free": "free", "trace_thc": "trace", "cannabinoid_pair": "pair"
}}`

var validCategoriesJSON = `{"form_word_lists": {
	"edible_solid": ["x"], "edible": ["x"], "topical": ["x"], "smokable": ["x"],
	"vapeable": ["x"], "tincture": ["x"], "beverage": ["x"], "extract": ["x"]
}, "pet_strong_description_phrase": "x", "pet_warning_sentence_strip": "x",
"concentrate_markers": "x", "negation_strip_pattern": "x", "negation_strip_pattern_2": "x"}`
