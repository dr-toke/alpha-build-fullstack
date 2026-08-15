package compliance

import (
	"testing"

	"github.com/dr-toke/api/internal/domain"
)

const rulesPath = "../../harvest/rules/compliance.json"

func TestLoadRuleSet(t *testing.T) {
	rs, err := LoadRuleSet(rulesPath)
	if err != nil {
		t.Fatalf("LoadRuleSet: %v", err)
	}
	if rs.ServiceListing == nil {
		t.Fatal("ServiceListing pattern not compiled")
	}
	if rs.HardBlock == nil {
		t.Fatal("HardBlock pattern not compiled")
	}
}

func TestLoadRuleSetMissingFile(t *testing.T) {
	if _, err := LoadRuleSet("/nonexistent/compliance.json"); err == nil {
		t.Error("expected an error for a nonexistent file, got nil")
	}
}

// TestEvaluateRealServiceListing is the exact real case that motivated
// building this package: a live cbdstore.in doctor-consultation booking
// found while proving M4's scraper worked. Confirms the honest finding
// that motivated checking CATEGORY, not just name — this listing's title
// alone does not contain any service_listing keyword.
func TestEvaluateRealServiceListing(t *testing.T) {
	rs, err := LoadRuleSet(rulesPath)
	if err != nil {
		t.Fatal(err)
	}

	name := "Dr. Harshal Sawarkar – BAMS Ayurvedic Physician | Vijaya-Based Medicine, Chronic Pain, Skin & Hair, Digestive & Lifestyle Disorders"
	category := "Doctors Consultation"

	t.Run("name alone does NOT catch it — the finding that justified checking category", func(t *testing.T) {
		got := Evaluate(rs, name, "", "")
		if !got.Pass {
			t.Error("name-only check unexpectedly caught this listing — if the harvested pattern changed, update this test's premise, don't just delete it")
		}
	})

	t.Run("name + real category correctly blocks it", func(t *testing.T) {
		got := Evaluate(rs, name, "", category)
		if got.Pass {
			t.Fatal("expected Pass=false for a real doctor-consultation listing")
		}
		if got.Reason != domain.ReviewComplianceUncertain {
			t.Errorf("Reason = %s, want %s", got.Reason, domain.ReviewComplianceUncertain)
		}
	})
}

func TestEvaluateOrdinaryProduct(t *testing.T) {
	rs, err := LoadRuleSet(rulesPath)
	if err != nil {
		t.Fatal(err)
	}
	got := Evaluate(rs, "CBD Oil 750mg Full Spectrum Tincture", "A full spectrum CBD tincture.", "Tinctures")
	if !got.Pass {
		t.Errorf("expected an ordinary product to pass, got %+v", got)
	}
}

func TestEvaluateWordBoundary(t *testing.T) {
	rs, err := LoadRuleSet(rulesPath)
	if err != nil {
		t.Fatal(err)
	}
	// "class" alone must NOT match (harvest/rules/compliance.json's own
	// note: "'class' alone deliberately excluded — matches 'world-class'").
	got := Evaluate(rs, "World-Class CBD Oil", "", "Oils")
	if !got.Pass {
		t.Errorf("bare 'class' substring should not match, got %+v", got)
	}
}

func TestEvaluateVariousServiceKeywords(t *testing.T) {
	rs, err := LoadRuleSet(rulesPath)
	if err != nil {
		t.Fatal(err)
	}
	cases := []string{
		"Wellness Retreat - 3 Day Package",
		"CBD Basics Workshop",
		"Meditation Session with Guru Ji",
		"Festival Tickets - Holi Special",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			if got := Evaluate(rs, name, "", ""); got.Pass {
				t.Errorf("expected %q to be caught as a service listing, got Pass=true", name)
			}
		})
	}
}

// TestEvaluateHardBlock covers the second narrow exception to ADR-019 —
// harvest/rules/compliance.json's hard_block tier, wired up after a full
// audit of the live cbdstore.in catalog surfaced products with no filter
// covering prohibited-claim wording at all.
func TestEvaluateHardBlock(t *testing.T) {
	rs, err := LoadRuleSet(rulesPath)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("catches the harvested pattern's own documented examples", func(t *testing.T) {
		cases := []struct {
			name, description string
		}{
			{"Miracle CBD Tonic", "This product cures cancer and reverses ageing."},
			{"Ultimate Relief Oil", "Guaranteed high every time, or your money back."},
			{"Wellness Elixir", "Our cure-all formula reverses diabetes naturally."},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				got := Evaluate(rs, c.name, c.description, "")
				if got.Pass {
					t.Errorf("expected name=%q description=%q to be hard-blocked, got Pass=true", c.name, c.description)
				}
				if got.Reason != domain.ReviewComplianceUncertain {
					t.Errorf("Reason = %s, want %s", got.Reason, domain.ReviewComplianceUncertain)
				}
			})
		}
	})

	t.Run("checks description, not just name — where the harvested examples actually live in real listings", func(t *testing.T) {
		got := Evaluate(rs, "Premium Cannabis Oil", "Clinically proven to cure diabetes and reverse disease.", "Tinctures")
		if got.Pass {
			t.Error("expected a description-only prohibited claim to be caught")
		}
	})

	t.Run("an ordinary therapeutic-but-not-prohibited claim passes", func(t *testing.T) {
		// "helps with chronic pain" / "supports wellness" is completely
		// normal, permitted CBD marketing language — hard_block's pattern
		// targets absolute/illegal claims (cures X, guaranteed high), not
		// routine structure/function wording. This is the real distinction
		// found while auditing the live catalog: hundreds of pain/anxiety/
		// sleep-support products must NOT be caught by this.
		got := Evaluate(rs, "CBD Oil for Chronic Pain Relief", "May help support relaxation and ease occasional discomfort.", "Tinctures")
		if !got.Pass {
			t.Errorf("ordinary structure/function wording should not be hard-blocked, got %+v", got)
		}
	})

	t.Run("hard_block takes priority over service_listing when both could match", func(t *testing.T) {
		got := Evaluate(rs, "Miracle Cure Workshop", "This cures cancer.", "")
		if got.Pass {
			t.Fatal("expected Pass=false")
		}
		if got.Detail == "" || got.Detail[:17] != "prohibited claim:" {
			t.Errorf("got Detail=%q, want it to start with 'prohibited claim:' (hard_block should win, not service_listing)", got.Detail)
		}
	})
}
