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
		got := Evaluate(rs, name, "")
		if !got.Pass {
			t.Error("name-only check unexpectedly caught this listing — if the harvested pattern changed, update this test's premise, don't just delete it")
		}
	})

	t.Run("name + real category correctly blocks it", func(t *testing.T) {
		got := Evaluate(rs, name, category)
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
	got := Evaluate(rs, "CBD Oil 750mg Full Spectrum Tincture", "Tinctures")
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
	got := Evaluate(rs, "World-Class CBD Oil", "Oils")
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
			if got := Evaluate(rs, name, ""); got.Pass {
				t.Errorf("expected %q to be caught as a service listing, got Pass=true", name)
			}
		})
	}
}
