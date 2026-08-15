package store

import (
	"context"
	"testing"

	"github.com/dr-toke/api/internal/domain"
)

func TestUpsertFacetsAndFacetsFor(t *testing.T) {
	ctx := context.Background()
	clusterID := mustCluster(t, "Facets Test")

	facets := []domain.ProductFacet{
		{ClusterID: clusterID, Facet: domain.FacetForm, Value: "capsule", Source: domain.FacetSourceRule, Confidence: 0.85, Evidence: map[string]any{}, ClassifierVersion: 1},
		{ClusterID: clusterID, Facet: domain.FacetRoute, Value: "oral", Source: domain.FacetSourceRule, Confidence: 0.9, Evidence: map[string]any{}, ClassifierVersion: 1},
	}

	t.Run("insert writes both rows", func(t *testing.T) {
		if err := testStore.UpsertFacets(ctx, facets); err != nil {
			t.Fatal(err)
		}
		got, err := testStore.FacetsFor(ctx, clusterID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d facets, want 2", len(got))
		}
	})

	t.Run("re-upsert on the same facet replaces, does not duplicate", func(t *testing.T) {
		updated := []domain.ProductFacet{
			{ClusterID: clusterID, Facet: domain.FacetForm, Value: "edible", Source: domain.FacetSourceOverride, Confidence: 1.0, Evidence: map[string]any{}, ClassifierVersion: 2},
		}
		if err := testStore.UpsertFacets(ctx, updated); err != nil {
			t.Fatal(err)
		}
		got, err := testStore.FacetsFor(ctx, clusterID)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 { // form (updated) + route (untouched) — still 2, not 3
			t.Fatalf("got %d facets, want 2 (one updated in place, one untouched)", len(got))
		}
		for _, f := range got {
			if f.Facet == domain.FacetForm {
				if f.Value != "edible" || f.Source != domain.FacetSourceOverride {
					t.Errorf("form facet not updated: value=%s source=%s", f.Value, f.Source)
				}
			}
		}
	})

	t.Run("empty slice is a no-op, not an error", func(t *testing.T) {
		if err := testStore.UpsertFacets(ctx, nil); err != nil {
			t.Errorf("expected nil error for empty facet slice, got %v", err)
		}
	})
}

func TestUpsertFacetsAtomicity(t *testing.T) {
	// A batch with one facet violating the facet-name CHECK constraint
	// (internal/db/migrations/003) should roll back the WHOLE batch, not
	// leave the earlier, valid facets partially written.
	ctx := context.Background()
	clusterID := mustCluster(t, "Atomicity Test")

	facets := []domain.ProductFacet{
		{ClusterID: clusterID, Facet: domain.FacetForm, Value: "capsule", Source: domain.FacetSourceRule, Confidence: 0.9, Evidence: map[string]any{}, ClassifierVersion: 1},
		{ClusterID: clusterID, Facet: "not_a_real_facet", Value: "x", Source: domain.FacetSourceRule, Confidence: 0.9, Evidence: map[string]any{}, ClassifierVersion: 1},
	}

	if err := testStore.UpsertFacets(ctx, facets); err == nil {
		t.Fatal("expected an error from the invalid facet name, got nil")
	}

	got, err := testStore.FacetsFor(ctx, clusterID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("got %d facets after a failed batch, want 0 — the valid facet before the bad one should have rolled back too", len(got))
	}
}
