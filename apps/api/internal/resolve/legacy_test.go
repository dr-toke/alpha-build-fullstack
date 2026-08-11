package resolve

import (
	"reflect"
	"testing"

	"github.com/dr-toke/api/internal/domain"
)

func TestLegacyCategory(t *testing.T) {
	cases := []struct {
		name              string
		form              domain.FormValue
		route             domain.RouteValue
		concentrationType domain.ConcentrationType
		wantCategory      string
		wantSecondary     []string
	}{
		{"pet exclusive", domain.FormPet, "", domain.ConcentrationUnknown, "pet", nil},
		{"apparel exclusive", domain.FormApparel, "", domain.ConcentrationUnknown, "apparel", nil},
		{"tincture", domain.FormOilTincture, domain.RouteSublingual, domain.ConcentrationCBD, "tincture", nil},
		{"capsule collapses to legacy edible", domain.FormCapsule, domain.RouteOral, domain.ConcentrationCBD, "edible", nil},
		{"gummy (edible) stays edible", domain.FormEdible, domain.RouteOral, domain.ConcentrationCBD, "edible", nil},
		{"hemp protein recovers legacy nutrition via concentration_type", domain.FormEdible, domain.RouteOral, domain.ConcentrationNutrition, "nutrition", nil},
		{"hemp seed oil (edible form somehow) recovers nutrition", domain.FormEdible, domain.RouteOral, domain.ConcentrationHempSeed, "nutrition", nil},
		{"concentrate + inhaled -> legacy vapeable, no secondary", domain.FormConcentrate, domain.RouteInhaled, domain.ConcentrationTHC, "vapeable", nil},
		{"concentrate + oral -> legacy extract, secondary edible", domain.FormConcentrate, domain.RouteOral, domain.ConcentrationTotal, "extract", []string{"edible"}},
		{"vape -> legacy vapeable", domain.FormVape, domain.RouteInhaled, domain.ConcentrationTHC, "vapeable", nil},
		{"flower -> legacy smokable", domain.FormFlower, domain.RouteInhaled, domain.ConcentrationUnknown, "smokable", nil},
		{"beverage", domain.FormBeverage, domain.RouteOral, domain.ConcentrationUnknown, "beverage", nil},
		{"accessory has no legacy precedent, falls back to other", domain.FormAccessory, "", domain.ConcentrationUnknown, "other", nil},
		{"unresolved form falls back to other", "", "", domain.ConcentrationUnknown, "other", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cat, sec := LegacyCategory(c.form, c.route, c.concentrationType)
			if cat != c.wantCategory {
				t.Errorf("category = %s, want %s", cat, c.wantCategory)
			}
			if !reflect.DeepEqual(sec, c.wantSecondary) {
				t.Errorf("secondary = %v, want %v", sec, c.wantSecondary)
			}
		})
	}
}

func TestLegacyCategories(t *testing.T) {
	got := LegacyCategories("extract", []string{"edible"})
	want := []string{"extract", "edible"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LegacyCategories = %v, want %v", got, want)
	}

	t.Run("no duplicates if primary equals a secondary", func(t *testing.T) {
		got := LegacyCategories("edible", []string{"edible"})
		want := []string{"edible"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("LegacyCategories = %v, want %v", got, want)
		}
	})
}
