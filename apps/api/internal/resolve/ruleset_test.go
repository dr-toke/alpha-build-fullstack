package resolve

import "testing"

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
