package resolve

import (
	"testing"

	"github.com/dr-toke/api/internal/domain"
)

// TestClassifyRealFailures ports every case from the prior alpha's
// categories_test.go TestRealFailures verbatim — same names/descriptions,
// same expected primary category and forbidden categories, tested at the
// legacy-classification level (classify()) before the new-facet mapping
// layer (resolveForm()) is applied on top. This is the level the original
// bug reports were filed against; keeping it as its own test means a future
// change that breaks the CLASSIFIER is distinguishable from one that breaks
// only the newer MAPPING layer.
func TestClassifyRealFailures(t *testing.T) {
	rs := loadTestRuleSet(t)

	cases := []struct {
		label, name, desc string
		wantPrimary       string
		forbid            []string
	}{
		{"capsule w/ negated vape copy",
			"CannaMed Cannabis Capsules 100mg (10 Capsules)",
			"Convenient and Discreet - No need to smoke or vape; take the capsule with water for an easy experience",
			"edible", []string{"vapeable", "smokable", "tincture"}},
		{"bach flower remedy",
			"Vior Naturals - Cherry Plum | Bach Flower Stock Concentrate - 10ml",
			"Fear of the mind being over-strained, of reason giving way. Bach Flower Cherry Plum",
			"other", []string{"smokable"}},
		{"dog mantra (pet exclusive)",
			"Calmosis Dog Mantra | India's Premium Vet-Approved THC-Free Tincture",
			"calming drops for dogs", "pet", []string{"smokable", "tincture", "vapeable"}},
		{"cannex concentrate",
			"Cannavedic - CannEx Strong Plus THC+CBD Extract (10:1 THC:CBD) 1000mg | 1ml",
			"pure cannabinoids and Terpenes and is not diluted with any carrier oil for a smooth experience",
			"vapeable", []string{"tincture", "smokable"}},
		{"sublingual oil",
			"Cannazo Calm Drops - CBD Dominant - Morning Bliss 30ml", "sublingual oil for anxiety",
			"tincture", []string{"smokable", "vapeable"}},
		{"gummy",
			"Indie Extracts - Full Spectrum Cannabis Gummies | 40mg", "tasty gummies made from premium extract, can be smoked... not really",
			"edible", []string{"vapeable", "smokable", "extract", "tincture"}},
		{"smoking blend",
			"Elinor Organics | Seafarer | Herbal Smoking Blend", "smoke or vaporize this herbal blend with tea-like flavours and chocolate notes",
			"smokable", []string{"edible", "beverage", "tincture"}},
		{"rso paste",
			"Hebe Shakti - Full-Spectrum RSO", "rick simpson oil paste for support",
			"extract", []string{"smokable", "vapeable"}},
		{"balm",
			"Indie Extracts - Recovery Balm | Peppermint - 50G", "cannabis infused balm. rub on sore muscles. made with cannabis extract from hemp flower",
			"topical", []string{"smokable", "edible", "tincture", "extract"}},
		{"bhang thandai",
			"Bhang Thandai Mix 100g", "traditional festival drink mix",
			"beverage", []string{"smokable"}},
		{"hemp protein",
			"Hemp Protein Powder 500g", "plant protein, omega rich",
			"nutrition", []string{"edible", "tincture"}},
		{"human product with pets-warning must NOT be pet",
			"Cannazo Vijaya Amrit Rich Oil 3325mg 15ml",
			"Premium full spectrum oil. Store in a cool, dry location. Keep out of reach of children and pets. Safety sealed.",
			"tincture", []string{"pet"}},
		{"pet product via strong description phrase",
			"Calmosis Calming Drops 200mg",
			"Specially formulated calming drops for your dog, vet approved and safe for daily use.",
			"pet", []string{"tincture"}},
		{"loose 'safe for pets' cross-sell copy must NOT be pet",
			"Cannazo Vijaya Amrit Rich Oil 3325mg 15ml",
			"Premium full-spectrum herbal wellness oil. Gentle formulation, also safe for pets and the whole family.",
			"tincture", []string{"pet"}},
		{"flavoured lip balm is topical, not edible",
			"Indus Hemp: Lip Balm - Chocolate 5G",
			"Nourishing chocolate flavoured lip balm with hemp oil.",
			"topical", []string{"edible"}},
		{"apparel cross-sell in desc must not reclassify",
			"Awshad Muscle Recovery Gel 50g",
			"Cooling gel for sore muscles. Order now and get a free tee with every purchase!",
			"topical", []string{"apparel"}},
		{"MCT-oil syringe is an extract — never smokable or topical",
			"Cannaronil Syringe 5000mg Balanced RAW",
			"Raw extract in MCT oil. Rub a drop... can be smoked by enthusiasts.",
			"extract", []string{"smokable", "topical"}},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			d := classify(&rs.Categories, c.name, c.desc, "")
			all := append([]string{d.primary}, d.secondary...)
			t.Logf("primary=%s secondary=%v ambiguous=%v reason=%s", d.primary, d.secondary, d.ambiguous, d.reason)
			if d.primary != c.wantPrimary {
				t.Errorf("primary = %s, want %s", d.primary, c.wantPrimary)
			}
			for _, f := range c.forbid {
				for _, got := range all {
					if got == f {
						t.Errorf("must NOT contain %q (got primary=%s secondary=%v)", f, d.primary, d.secondary)
					}
				}
			}
		})
	}

	t.Run("source breadcrumb 'CBD for Pets' is a trusted pet signal", func(t *testing.T) {
		d := classify(&rs.Categories, "Cannapaw Wellness Oil 200mg", "gentle daily oil", "CBD for Pets")
		if d.primary != "pet" {
			t.Errorf("primary = %s, want pet", d.primary)
		}
	})
}

// TestResolveFormMapping exercises the new-facet mapping layer specifically
// — the one part of M1 with no precedent in the harvested source (see
// facets.go's doc comment on resolveForm, and M1-DECISIONS.md).
func TestResolveFormMapping(t *testing.T) {
	rs := loadTestRuleSet(t)

	cases := []struct {
		label, name, desc string
		wantForm          domain.FormValue
		wantRoute         domain.RouteValue
	}{
		{"capsule sub-bucket of edible_solid", "CannaMed Cannabis Capsules 100mg (10 Capsules)", "take with water", domain.FormCapsule, domain.RouteOral},
		{"gummy stays edible, not capsule", "Indie Extracts - Full Spectrum Cannabis Gummies | 40mg", "tasty gummies", domain.FormEdible, domain.RouteOral},
		{"concentrate marker -> concentrate, not vape", "Cannavedic - CannEx Strong Plus THC+CBD Extract (10:1 THC:CBD) 1000mg | 1ml", "not diluted with any carrier oil", domain.FormConcentrate, domain.RouteInhaled},
		{"explicit vape pen -> vape, not concentrate", "Awshad Disposable Vape Pen 500mg", "premium vape pen for on the go", domain.FormVape, domain.RouteInhaled},
		{"dab wax -> concentrate", "Pure Dab Wax Extract 1g", "for dabbing enthusiasts", domain.FormConcentrate, domain.RouteInhaled},
		{"rso extract -> concentrate, oral route (not inhaled)", "Hebe Shakti - Full-Spectrum RSO", "rick simpson oil paste for support", domain.FormConcentrate, domain.RouteOral},
		{"tincture -> oil_tincture, sublingual", "Cannazo Calm Drops - CBD Dominant - Morning Bliss 30ml", "sublingual oil for anxiety", domain.FormOilTincture, domain.RouteSublingual},
		{"hemp protein -> edible (no 'nutrition' form value exists)", "Hemp Protein Powder 500g", "plant protein, omega rich", domain.FormEdible, domain.RouteOral},
		{"topical", "Indie Extracts - Recovery Balm | Peppermint - 50G", "rub on sore muscles", domain.FormTopical, domain.RouteTopical},
		{"beverage", "Bhang Thandai Mix 100g", "traditional festival drink mix", domain.FormBeverage, domain.RouteOral},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			form := ResolveForm(&rs.Categories, c.name, c.desc, "")
			route := ResolveRoute(&rs.Categories, c.name, c.desc, "")
			if form.Value != string(c.wantForm) {
				t.Errorf("form = %s, want %s", form.Value, c.wantForm)
			}
			if route.Value != string(c.wantRoute) {
				t.Errorf("route = %s, want %s", route.Value, c.wantRoute)
			}
		})
	}

	t.Run("pet/apparel: route is not set (Ambiguous, empty value)", func(t *testing.T) {
		route := ResolveRoute(&rs.Categories, "Calmosis Dog Mantra Vet-Approved Tincture", "calming drops for dogs", "")
		if route.Value != "" {
			t.Errorf("route.Value = %q, want empty — route doesn't apply to pet products", route.Value)
		}
	})
}

func TestResolveExtract(t *testing.T) {
	cases := []struct {
		label, name, desc string
		want              domain.ExtractValue
		wantAmbiguous     bool
	}{
		{"full spectrum", "Full Spectrum CBD Oil", "", domain.ExtractFullSpectrum, false},
		{"broad spectrum", "Broad-Spectrum CBD Tincture", "", domain.ExtractBroadSpectrum, false},
		{"genuine isolate", "Cannavedic Pure CBD Isolate 99% Cannabidiol", "", domain.ExtractIsolate, false},
		{"protein isolate is NOT a cannabinoid isolate", "Hampa Hemp Protein Bar with Pea Protein Isolate", "hemp hearts", "", true},
	}
	for _, c := range cases {
		t.Run(c.label, func(t *testing.T) {
			r := ResolveExtract(c.name, c.desc)
			if !c.wantAmbiguous && r.Value != string(c.want) {
				t.Errorf("extract = %s, want %s", r.Value, c.want)
			}
			if r.Ambiguous != c.wantAmbiguous {
				t.Errorf("ambiguous = %v, want %v", r.Ambiguous, c.wantAmbiguous)
			}
		})
	}
}

func TestResolveProfile(t *testing.T) {
	cases := []struct {
		name          string
		cbdMg, thcMg  float64
		want          domain.ProfileValue
		wantAmbiguous bool
	}{
		{"balanced 1:1", 500, 500, domain.ProfileBalanced, false},
		{"cbd dominant, thc present", 909, 91, domain.ProfileCBDDominant, false}, // hi/lo=9.99 -> cbd side is 909 here so cbd dominant, mirrors finalize()'s own >=3x rule
		{"thc dominant, exactly at 3x boundary", 100, 300, domain.ProfileTHCDominant, false},
		{"cbd zero, thc present -> thc dominant, not balanced (the bug this test locks)", 0, 500, domain.ProfileTHCDominant, false},
		{"thc zero, cbd present -> cbd dominant, not balanced", 750, 0, domain.ProfileCBDDominant, false},
		{"both zero -> ambiguous, no profile", 0, 0, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ResolveProfile(c.cbdMg, c.thcMg)
			if got.Ambiguous != c.wantAmbiguous {
				t.Errorf("ambiguous = %v, want %v", got.Ambiguous, c.wantAmbiguous)
			}
			if !c.wantAmbiguous && got.Value != string(c.want) {
				t.Errorf("profile = %s, want %s", got.Value, c.want)
			}
		})
	}
}

func TestResolveCarrier(t *testing.T) {
	cases := []struct {
		desc string
		want domain.CarrierValue
	}{
		{"MCT oil base", domain.CarrierMCT},
		{"cold-pressed coconut carrier", domain.CarrierMCT},
		{"hemp seed oil carrier", domain.CarrierHempSeed},
		{"olive oil infused", domain.CarrierOlive},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			if got := ResolveCarrier(c.desc); got.Value != string(c.want) {
				t.Errorf("carrier = %s, want %s", got.Value, c.want)
			}
		})
	}
}

func TestResolvePurchasable(t *testing.T) {
	rs := loadTestRuleSet(t)

	t.Run("apparel is not purchasable as a cannabis product", func(t *testing.T) {
		r := ResolvePurchasable(&rs.Categories, "Dr Toke Branded Hoodie", "cosy pullover hoodie", "")
		if r.Value != "false" {
			t.Errorf("purchasable = %s, want false for apparel", r.Value)
		}
	})
	t.Run("ordinary product is purchasable", func(t *testing.T) {
		r := ResolvePurchasable(&rs.Categories, "CBD Oil 750mg", "pain relief oil", "")
		if r.Value != "true" {
			t.Errorf("purchasable = %s, want true", r.Value)
		}
	})
}
