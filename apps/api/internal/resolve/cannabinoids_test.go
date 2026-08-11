package resolve

import (
	"testing"

	"github.com/dr-toke/api/internal/domain"
)

// loadTestRuleSet loads the real harvested rules — not a fixture — so this
// test suite is exercising the actual patterns that will ship, same
// reasoning as ruleset_test.go.
func loadTestRuleSet(t *testing.T) *RuleSet {
	t.Helper()
	rs, err := LoadRuleSet(harvestRulesDir)
	if err != nil {
		t.Fatalf("LoadRuleSet: %v", err)
	}
	return rs
}

// TestCannabinoidAttribution ports every case from the prior alpha's
// cannabinoids_test.go (dr-toke-init/apps/api/internal/catalog/normaliser)
// verbatim — same product names, same descriptions, same expected values.
// This is the canonical fixture set the whole facet-refactor effort exists
// to keep green; see harvest/NOTES.md and testdata/golden's cannex-strong-
// plus fixtures for the same cases in the M0-era golden-fixture format.
func TestCannabinoidAttribution(t *testing.T) {
	rs := loadTestRuleSet(t)

	cases := []struct {
		label, name, desc string
		wantType          domain.ConcentrationType
		wantCBD, wantTHC  float64 // 0 = don't care beyond type
	}{
		{
			"explicit both, label-first (CannEx)",
			"Cannavedic - CannEx Strong Plus THC+CBD Extract (10:1 THC:CBD) 1000mg | 1ml",
			"THC-dominant formulation. Cannabinoids THC 909mg CBD 91mg. THC:CBD ratio of 10:1.",
			domain.ConcentrationTHC, 91, 909,
		},
		{
			"CBD product with trace-THC disclaimer must stay CBD",
			"StarCBD - CBD Oil Pain Relief - 750mg/30ml",
			"Pure CBD goodness. Contains less than 0.3% THC as permitted by law.",
			domain.ConcentrationCBD, 750, 0,
		},
		{
			"CBD in NAME, plain THC mention in desc — name wins",
			"Hempire CBD Tincture 2000mg",
			"Full spectrum extract with CBD, THC and minor cannabinoids working together.",
			domain.ConcentrationCBD, 2000, 0,
		},
		{
			"bare ratio + CBD-named -> split, major share to CBD",
			"Monk's Hemp - Breath (10:1) Pain Relief CBD Oil - 3000mg 30ml",
			"Peppermint flavoured relief oil.",
			domain.ConcentrationCBD, 2727.27, 272.73,
		},
		{
			"CBD+THC pair, no ratio -> assume balanced 1:1, both ₹/mg computable",
			"Cannarma Full Spectrum Cannabis Extract Oil CBD + THC - 1500mg 10ml",
			"Balanced full plant extract.",
			domain.ConcentrationTotal, 750, 750,
		},
		{
			"unoriented ratio — 1:1 would contradict label -> honest total",
			"Mystery THC+CBD Oil (2:1) - 900mg 10ml",
			"Lopsided blend, direction unstated.",
			domain.ConcentrationTotal, 0, 0,
		},
		{
			"THC-free isolate",
			"Pure Isolate CBD Oil 1000mg - THC Free",
			"Zero THC, lab tested.",
			domain.ConcentrationCBD, 1000, 0,
		},
		{
			"hemp seed — no cannabinoids",
			"Cold Pressed Hemp Seed Oil 100ml",
			"Nutritious culinary oil, omega rich.",
			domain.ConcentrationHempSeed, 0, 0,
		},
		{
			"full chemical names (IPV4000 ground truth: 2000/2000)",
			"Vijaya Extract - CBD THC Balanced -IPV4000 mg | 1:1 CBD:THC | Premium Dewaxed",
			"4000 mg of Cannabinoids 2000 mg Cannabidiol - CBD 2000 mg Tetrahydrocannabinol - THC",
			domain.ConcentrationTotal, 2000, 2000,
		},
		{
			"'50% THC' must NOT read as '0% THC' = thc-free (IPV4000 full text)",
			"Vijaya Extract - CBD THC Balanced -IPV4000 mg | 1:1 CBD:THC | Premium Dewaxed - 2000 mg - 5 ml",
			"Composition: Vijaya Extract 10000 mg of Strong Vijaya Extract 4000 mg of Cannabinoids 2000 mg Cannabidiol - CBD 2000 mg Tetrahydrocannabinol - THC Schedule E Drug 1:1 CBD:THC = 50% CBD : 50% THC Pure Extract - No carrier oil",
			domain.ConcentrationTotal, 2000, 2000,
		},
		{
			"genuine zero-THC phrasing still detected",
			"Pure CBD Isolate Drops 500mg - 0% THC",
			"Contains 0% THC. Pure isolate.",
			domain.ConcentrationCBD, 500, 0,
		},
		{
			"name total beats raw-herb weight in description",
			"Qurist De-Stress Gummies - 450mg CBD",
			"Made from 1800mg of premium hemp flower input material per batch.",
			domain.ConcentrationCBD, 450, 0,
		},
		{
			"per-drop figures scale to the name's pack total",
			"Cannaben Balance Oil - 1000 mg - 30ml",
			"Each drop delivers 2.1 mg of CBD and 2.1 mg of THC for balanced relief.",
			domain.ConcentrationTotal, 500, 500,
		},
		{
			"NAMED hemp seed oil stays seed oil despite CBD mention in desc",
			"Cannazo Hemp Seed Oil: Multipurpose Oil 100ml",
			"Unlike CBD oil, this multipurpose hemp seed oil is for skin and hair. 1 mg",
			domain.ConcentrationHempSeed, 0, 0,
		},
		{
			"balanced 1:1 bare ratio splits (Cannabryl: 4000mg ground truth 1:1)",
			"Cannabryl - 4000 mg - 1:1 - Premium",
			"The 1:1 ratio of CBD and THC is all set to deliver amazing health effects.",
			domain.ConcentrationTotal, 2000, 2000,
		},
		{
			"labelled ratio with | separator",
			"Cannavedic CannaBalance THC+CBD Oil (1:1|THC:CBD) 1000mg, 10ml",
			"Balanced formulation.",
			domain.ConcentrationTotal, 500, 500,
		},
		{
			"'Nmg of CBD' wording + per-drop distractors (Cannabryl ground truth 500/500)",
			"Cannabryl - 1000 mg - 1:1 - Premium",
			"Each 30 ml bottle delivers you 1000 mg of the cannabidiol. The 1:1 ratio of CBD and THC. With each drop, you will get 2.1 mg of CBD and 2.1 mg of THC. 1000 mg of Cannabinoids per 30 ml bottle 500mg of CBD 500mg of THC Ratio of CBD : THC is 1:1 or 50/50",
			domain.ConcentrationTotal, 500, 500,
		},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			r := ExtractCannabinoids(&rs.Cannabinoids, c.name, c.desc)
			t.Logf("cbd=%.2f thc=%.2f total=%.2f type=%s confidence=%.2f", r.CBDMg, r.THCMg, r.TotalCannabinoidsMg, r.ConcentrationType, r.Confidence)
			if r.ConcentrationType != c.wantType {
				t.Errorf("type = %s, want %s", r.ConcentrationType, c.wantType)
			}
			if c.wantCBD > 0 && !approx(r.CBDMg, c.wantCBD) {
				t.Errorf("cbd = %.2f, want %.2f", r.CBDMg, c.wantCBD)
			}
			if c.wantTHC > 0 && !approx(r.THCMg, c.wantTHC) {
				t.Errorf("thc = %.2f, want %.2f", r.THCMg, c.wantTHC)
			}
		})
	}
}

func approx(got, want float64) bool {
	d := got - want
	return d > -0.5 && d < 0.5
}

func TestIsolateAndWeightBased(t *testing.T) {
	rs := loadTestRuleSet(t)

	t.Run("gram-weight isolate: 100% CBD - 5gm must yield 5000mg CBD", func(t *testing.T) {
		r := ExtractCannabinoids(&rs.Cannabinoids, "Cannavedic - Pure CBD Isolate – 100% CBD 0%THC - 5gm", "Pure crystalline isolate.")
		if r.ConcentrationType != domain.ConcentrationCBD || r.CBDMg < 4999 || r.CBDMg > 5001 {
			t.Errorf("got cbd=%.0f type=%s, want ~5000/cbd", r.CBDMg, r.ConcentrationType)
		}
	})
}

func TestBestMG(t *testing.T) {
	cases := []struct {
		name string
		r    CannabinoidExtraction
		want float64
	}{
		{"thc-dominant uses thc", CannabinoidExtraction{CBDMg: 91, THCMg: 909, ConcentrationType: domain.ConcentrationTHC}, 909},
		{"cbd-dominant uses cbd", CannabinoidExtraction{CBDMg: 750, THCMg: 0, ConcentrationType: domain.ConcentrationCBD}, 750},
		{"total uses total", CannabinoidExtraction{TotalCannabinoidsMg: 4000, ConcentrationType: domain.ConcentrationTotal}, 4000},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.r.BestMG(); got != c.want {
				t.Errorf("BestMG() = %v, want %v", got, c.want)
			}
		})
	}
}
