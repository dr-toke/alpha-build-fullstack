package resolve

import (
	"testing"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
)

// TestResolvePrecedence is not in 08-BUILD-ORDERS.md's M1 file list (only
// ruleset_test.go, cannabinoids_test.go, facets_test.go, and golden_test.go
// are named) — added anyway because Resolve is explicitly "THE single
// writer" of the most consequential rule in the whole facet system
// (03-DOMAIN-MODEL.md §2's override > rule > model > default), and writing
// this test is what caught a real bug: the first draft of Resolve took a
// clusterID parameter and never actually set it on the returned
// domain.ProductFacet, so every persisted facet would have carried a
// zero-value cluster ID. Caught here, before it reached a database.
func TestResolvePrecedence(t *testing.T) {
	clusterID := uuid.New()
	override := &domain.ProductFacetOverride{Value: "topical", Reason: "analyst correction", SetBy: "admin1"}
	rule := &FacetResult{Value: "vape", Confidence: 0.8}
	model := &FacetResult{Value: "flower", Confidence: 0.6}
	def := &FacetResult{Value: "none", Confidence: 0.3}

	t.Run("override wins over everything", func(t *testing.T) {
		got := Resolve(clusterID, domain.FacetForm, FacetInputs{Override: override, Rule: rule, Model: model, Default: def}, 1)
		if got.Value != "topical" || got.Source != domain.FacetSourceOverride {
			t.Errorf("got value=%s source=%s, want topical/override", got.Value, got.Source)
		}
		if got.Confidence != 1.0 {
			t.Errorf("override confidence = %v, want 1.0 always (03-DOMAIN-MODEL.md §2)", got.Confidence)
		}
	})

	t.Run("rule wins when no override", func(t *testing.T) {
		got := Resolve(clusterID, domain.FacetForm, FacetInputs{Rule: rule, Model: model, Default: def}, 1)
		if got.Value != "vape" || got.Source != domain.FacetSourceRule {
			t.Errorf("got value=%s source=%s, want vape/rule", got.Value, got.Source)
		}
	})

	t.Run("model wins when no override or rule", func(t *testing.T) {
		got := Resolve(clusterID, domain.FacetForm, FacetInputs{Model: model, Default: def}, 1)
		if got.Value != "flower" || got.Source != domain.FacetSourceModel {
			t.Errorf("got value=%s source=%s, want flower/model", got.Value, got.Source)
		}
	})

	t.Run("default wins when nothing else proposes", func(t *testing.T) {
		got := Resolve(clusterID, domain.FacetForm, FacetInputs{Default: def}, 1)
		if got.Value != "none" || got.Source != domain.FacetSourceDefault {
			t.Errorf("got value=%s source=%s, want none/default", got.Value, got.Source)
		}
	})

	t.Run("nil when nothing proposes anything", func(t *testing.T) {
		got := Resolve(clusterID, domain.FacetForm, FacetInputs{}, 1)
		if got != nil {
			t.Errorf("got %+v, want nil", got)
		}
	})

	t.Run("a tier with an empty Value is treated as not proposing", func(t *testing.T) {
		got := Resolve(clusterID, domain.FacetForm, FacetInputs{Rule: &FacetResult{Value: ""}, Default: def}, 1)
		if got.Source != domain.FacetSourceDefault {
			t.Errorf("empty-value rule should fall through to default, got source=%s", got.Source)
		}
	})

	t.Run("ClusterID is always set on the returned facet", func(t *testing.T) {
		for _, in := range []FacetInputs{
			{Override: override}, {Rule: rule}, {Model: model}, {Default: def},
		} {
			got := Resolve(clusterID, domain.FacetForm, in, 1)
			if got.ClusterID != clusterID {
				t.Errorf("ClusterID = %v, want %v (inputs: %+v)", got.ClusterID, clusterID, in)
			}
		}
	})
}

func TestPublishable(t *testing.T) {
	highConfRoute := &domain.ProductFacet{Confidence: 0.95}
	lowConfRoute := &domain.ProductFacet{Confidence: 0.5}

	cases := []struct {
		name           string
		purchasable    bool
		formConfidence float32
		route          *domain.ProductFacet
		pricePaise     int64
		want           bool
	}{
		{"all conditions met, no route facet", true, 0.9, nil, 100, true},
		{"all conditions met, high-confidence route", true, 0.9, highConfRoute, 100, true},
		{"not purchasable", false, 0.9, nil, 100, false},
		{"form confidence below 0.85", true, 0.84, nil, 100, false},
		{"form confidence exactly 0.85 passes", true, 0.85, nil, 100, true},
		{"route present but below 0.90", true, 0.9, lowConfRoute, 100, false},
		{"zero price fails even with everything else good", true, 0.9, nil, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Publishable(c.purchasable, c.formConfidence, c.route, c.pricePaise)
			if got != c.want {
				t.Errorf("Publishable(%v, %v, %v, %v) = %v, want %v",
					c.purchasable, c.formConfidence, c.route, c.pricePaise, got, c.want)
			}
		})
	}
}
