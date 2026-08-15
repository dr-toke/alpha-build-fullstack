package store

import (
	"context"
	"testing"

	"github.com/dr-toke/api/internal/domain"
)

func TestSetOverrideAndOverridesFor(t *testing.T) {
	ctx := context.Background()
	clusterID := mustCluster(t, "Overrides Test")

	o := domain.ProductFacetOverride{
		ClusterID: clusterID, Facet: domain.FacetForm, Value: "topical",
		Reason: "analyst correction: description clearly says balm", SetBy: "admin1",
	}

	t.Run("set then read back", func(t *testing.T) {
		if err := testStore.SetOverride(ctx, o); err != nil {
			t.Fatal(err)
		}
		got, err := testStore.OverridesFor(ctx, clusterID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Value != "topical" {
			t.Errorf("got %+v, want one override with value=topical", got)
		}
	})

	t.Run("setting again on the same facet replaces, does not accumulate history", func(t *testing.T) {
		o2 := o
		o2.Value = "beverage"
		o2.Reason = "corrected again after re-reading the label"
		if err := testStore.SetOverride(ctx, o2); err != nil {
			t.Fatal(err)
		}
		got, err := testStore.OverridesFor(ctx, clusterID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d overrides, want 1 (PRIMARY KEY (cluster_id, facet) should replace, not accumulate)", len(got))
		}
		if got[0].Value != "beverage" {
			t.Errorf("value = %s, want beverage (the latest override)", got[0].Value)
		}
	})

	t.Run("a different facet on the same cluster is a separate row", func(t *testing.T) {
		route := domain.ProductFacetOverride{ClusterID: clusterID, Facet: domain.FacetRoute, Value: "oral", Reason: "x", SetBy: "admin1"}
		if err := testStore.SetOverride(ctx, route); err != nil {
			t.Fatal(err)
		}
		got, err := testStore.OverridesFor(ctx, clusterID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Errorf("got %d overrides, want 2 (form + route)", len(got))
		}
	})
}
