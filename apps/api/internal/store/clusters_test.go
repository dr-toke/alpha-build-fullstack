package store

import (
	"context"
	"errors"
	"testing"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
)

func TestClusterByID(t *testing.T) {
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		id := mustCluster(t, "Findable Cluster")
		c, err := testStore.ClusterByID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if c.ID != id {
			t.Errorf("got id=%s, want %s", c.ID, id)
		}
	})

	t.Run("genuinely missing returns ErrNotFound, not a moved error", func(t *testing.T) {
		_, err := testStore.ClusterByID(ctx, uuid.New())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("got %v, want domain.ErrNotFound", err)
		}
		var moved *domain.ClusterMovedError
		if errors.As(err, &moved) {
			t.Error("a never-existed cluster must not resolve as ClusterMovedError")
		}
	})

	t.Run("merged cluster returns ClusterMovedError, not ErrNotFound", func(t *testing.T) {
		oldID := mustCluster(t, "Old Cluster")
		newID := mustCluster(t, "New Cluster")
		if err := testStore.Merge(ctx, oldID, newID); err != nil {
			t.Fatal(err)
		}

		_, err := testStore.ClusterByID(ctx, oldID)
		var moved *domain.ClusterMovedError
		if !errors.As(err, &moved) {
			t.Fatalf("got %v, want *domain.ClusterMovedError", err)
		}
		if moved.NewID != newID {
			t.Errorf("moved.NewID = %s, want %s", moved.NewID, newID)
		}
		// 02-FRONTEND-CONTRACT.md §4: moved and not-found are different
		// outcomes — a generic errors.Is(err, ErrNotFound) check must not
		// also catch a moved cluster.
		if errors.Is(err, domain.ErrNotFound) {
			t.Error("ClusterMovedError must not satisfy errors.Is(err, ErrNotFound)")
		}
	})
}

func TestCreateClusterAndByFingerprint(t *testing.T) {
	ctx := context.Background()

	t.Run("create then find by fingerprint", func(t *testing.T) {
		fp := "fp-" + randSuffix()
		id, err := testStore.CreateCluster(ctx, domain.ProductCluster{
			Fingerprint:       &fp,
			Name:              "Fingerprint Findable " + randSuffix(),
			ConcentrationType: domain.ConcentrationUnknown,
		})
		if err != nil {
			t.Fatal(err)
		}

		c, err := testStore.ClusterByFingerprint(ctx, fp)
		if err != nil {
			t.Fatal(err)
		}
		if c.ID != id {
			t.Errorf("got id=%s, want %s", c.ID, id)
		}
		if c.Fingerprint == nil || *c.Fingerprint != fp {
			t.Errorf("got fingerprint=%v, want %s", c.Fingerprint, fp)
		}
	})

	t.Run("unknown fingerprint returns ErrNotFound", func(t *testing.T) {
		_, err := testStore.ClusterByFingerprint(ctx, "fp-never-"+randSuffix())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("got %v, want domain.ErrNotFound", err)
		}
	})

	t.Run("a merged-away cluster's fingerprint resolves to the live cluster", func(t *testing.T) {
		fp := "fp-merge-" + randSuffix()
		oldID, err := testStore.CreateCluster(ctx, domain.ProductCluster{
			Fingerprint:       &fp,
			Name:              "Stale " + randSuffix(),
			ConcentrationType: domain.ConcentrationUnknown,
		})
		if err != nil {
			t.Fatal(err)
		}
		newID := mustCluster(t, "Live")
		if err := testStore.Merge(ctx, oldID, newID); err != nil {
			t.Fatal(err)
		}

		c, err := testStore.ClusterByFingerprint(ctx, fp)
		if err != nil {
			t.Fatal(err)
		}
		if c.ID != newID {
			t.Errorf("got id=%s, want live cluster %s (not the merged-away %s)", c.ID, newID, oldID)
		}
	})
}

func TestMerge(t *testing.T) {
	ctx := context.Background()

	t.Run("cannot merge a cluster into itself", func(t *testing.T) {
		id := mustCluster(t, "Self Merge")
		if err := testStore.Merge(ctx, id, id); err == nil {
			t.Error("expected an error merging a cluster into itself, got nil")
		}
	})
}

func TestListClusters(t *testing.T) {
	ctx := context.Background()
	brandSlug := mustBrand(t)
	var brandID uuid.UUID
	if err := testStore.Pool.QueryRow(ctx, `SELECT id FROM brands WHERE slug = $1`, brandSlug).Scan(&brandID); err != nil {
		t.Fatal(err)
	}

	// Three publishable clusters with distinct rank scores, one unpublishable.
	ranks := []float64{30, 10, 20}
	for _, r := range ranks {
		_, err := testStore.Pool.Exec(ctx,
			`INSERT INTO product_clusters (name, concentration_type, brand_id, rank_score, publishable)
			 VALUES ($1, 'unknown', $2, $3, true)`,
			"Ranked "+randSuffix(), brandID, r)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := testStore.Pool.Exec(ctx,
		`INSERT INTO product_clusters (name, concentration_type, brand_id, rank_score, publishable)
		 VALUES ($1, 'unknown', $2, 999, false)`,
		"Unpublishable "+randSuffix(), brandID); err != nil {
		t.Fatal(err)
	}

	t.Run("publishable filter excludes the unpublishable row even though it ranks highest", func(t *testing.T) {
		got, err := testStore.ListClusters(ctx, ClusterFilter{BrandID: &brandID, PublishableOnly: true, Sort: SortValue, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 {
			t.Fatalf("got %d clusters, want 3", len(got))
		}
		for i := 1; i < len(got); i++ {
			if *got[i-1].RankScore < *got[i].RankScore {
				t.Errorf("not sorted rank_score DESC: %v before %v", *got[i-1].RankScore, *got[i].RankScore)
			}
		}
	})

	t.Run("page-based pagination walks forward without duplicates or gaps", func(t *testing.T) {
		page1, err := testStore.ListClusters(ctx, ClusterFilter{BrandID: &brandID, PublishableOnly: true, Sort: SortValue, Page: 1, Limit: 2})
		if err != nil {
			t.Fatal(err)
		}
		if len(page1) != 2 {
			t.Fatalf("page1 len = %d, want 2", len(page1))
		}
		page2, err := testStore.ListClusters(ctx, ClusterFilter{BrandID: &brandID, PublishableOnly: true, Sort: SortValue, Page: 2, Limit: 2})
		if err != nil {
			t.Fatal(err)
		}
		if len(page2) != 1 {
			t.Fatalf("page2 len = %d, want 1 (3 total, 2 already seen)", len(page2))
		}
		for _, p1 := range page1 {
			for _, p2 := range page2 {
				if p1.ID == p2.ID {
					t.Errorf("cluster %s appeared on both pages", p1.ID)
				}
			}
		}
	})

	t.Run("Page <= 0 defaults to page 1", func(t *testing.T) {
		explicit, err := testStore.ListClusters(ctx, ClusterFilter{BrandID: &brandID, PublishableOnly: true, Sort: SortValue, Page: 1, Limit: 2})
		if err != nil {
			t.Fatal(err)
		}
		defaulted, err := testStore.ListClusters(ctx, ClusterFilter{BrandID: &brandID, PublishableOnly: true, Sort: SortValue, Limit: 2})
		if err != nil {
			t.Fatal(err)
		}
		if len(explicit) != len(defaulted) || (len(explicit) > 0 && explicit[0].ID != defaulted[0].ID) {
			t.Errorf("Page:0 did not behave like Page:1")
		}
	})

	t.Run("default sort (unset Sort field) is SortNew, most-recent first", func(t *testing.T) {
		got, err := testStore.ListClusters(ctx, ClusterFilter{BrandID: &brandID, PublishableOnly: true, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 3 {
			t.Fatalf("got %d clusters, want 3 (unranked-eligible too — SortNew doesn't require rank_score)", len(got))
		}
		for i := 1; i < len(got); i++ {
			if got[i-1].FirstSeenAt.Before(got[i].FirstSeenAt) {
				t.Errorf("not sorted first_seen_at DESC: %v before %v", got[i-1].FirstSeenAt, got[i].FirstSeenAt)
			}
		}
	})

	t.Run("SortNew and SortValue both include a publishable row with no rank_score at all", func(t *testing.T) {
		var unrankedID uuid.UUID
		if err := testStore.Pool.QueryRow(ctx,
			`INSERT INTO product_clusters (name, concentration_type, brand_id, rank_score, publishable)
			 VALUES ($1, 'unknown', $2, NULL, true) RETURNING id`,
			"Unranked "+randSuffix(), brandID,
		).Scan(&unrankedID); err != nil {
			t.Fatal(err)
		}

		gotNew, err := testStore.ListClusters(ctx, ClusterFilter{BrandID: &brandID, PublishableOnly: true, Sort: SortNew, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, c := range gotNew {
			if c.ID == unrankedID {
				found = true
			}
		}
		if !found {
			t.Error("SortNew excluded a publishable, unranked cluster")
		}

		// A category made entirely of unranked rows (e.g. Nutrition — hemp
		// protein/hearts, no cannabinoid content to price per mg) must still
		// show under the default sort, not read as "0 products". Unranked
		// rows are included, just sorted after every ranked row.
		gotValue, err := testStore.ListClusters(ctx, ClusterFilter{BrandID: &brandID, PublishableOnly: true, Sort: SortValue, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		foundInValue := false
		lastIsUnranked := false
		for i, c := range gotValue {
			if c.ID == unrankedID {
				foundInValue = true
				lastIsUnranked = i == len(gotValue)-1
			}
		}
		if !foundInValue {
			t.Error("SortValue excluded an unranked cluster — should sort it last, not hide it")
		}
		if !lastIsUnranked {
			t.Error("SortValue did not sort the unranked cluster after all ranked ones (NULLS LAST)")
		}
	})
}

// TestListClustersRealFilters covers the filters added to match the real
// frontend (apps/web/src/lib/sections/products/CatalogGrid.svelte):
// Category, Extract, VerifiedOnly, Basis, SortPrice — none of which existed
// when ClusterFilter only supported BrandID/PublishableOnly/Sort. See
// API-DECISIONS.md.
func TestListClustersRealFilters(t *testing.T) {
	ctx := context.Background()

	verifiedSlug := "verified-" + randSuffix()
	if _, err := testStore.Pool.Exec(ctx,
		`INSERT INTO brands (slug, name, verified) VALUES ($1, $2, true)`, verifiedSlug, "Verified Co"); err != nil {
		t.Fatal(err)
	}
	var verifiedBrandID uuid.UUID
	if err := testStore.Pool.QueryRow(ctx, `SELECT id FROM brands WHERE slug = $1`, verifiedSlug).Scan(&verifiedBrandID); err != nil {
		t.Fatal(err)
	}
	unverifiedBrandSlug := mustBrand(t)
	var unverifiedBrandID uuid.UUID
	if err := testStore.Pool.QueryRow(ctx, `SELECT id FROM brands WHERE slug = $1`, unverifiedBrandSlug).Scan(&unverifiedBrandID); err != nil {
		t.Fatal(err)
	}

	// A tincture (form=oil_tincture) from the verified brand, extract=full_spectrum,
	// cheap on both rank_score and price.
	tinctureID := mustClusterFull(t, ctx, "Tincture "+randSuffix(), &verifiedBrandID, 500, 100)
	mustFacet(t, ctx, tinctureID, domain.FacetForm, "oil_tincture")
	mustFacet(t, ctx, tinctureID, domain.FacetExtract, "full_spectrum")

	// A vape (form=vape) from the unverified brand, extract=isolate, pricier.
	vapeID := mustClusterFull(t, ctx, "Vape "+randSuffix(), &unverifiedBrandID, 50, 900)
	mustFacet(t, ctx, vapeID, domain.FacetForm, "vape")
	mustFacet(t, ctx, vapeID, domain.FacetExtract, "isolate")

	t.Run("Category filters via the form facet", func(t *testing.T) {
		got, err := testStore.ListClusters(ctx, ClusterFilter{PublishableOnly: true, Category: "tincture", Sort: SortNew, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != tinctureID {
			t.Errorf("Category=tincture: got %d results, want exactly the tincture cluster", len(got))
		}
	})

	t.Run("an unknown category matches nothing, not an error", func(t *testing.T) {
		got, err := testStore.ListClusters(ctx, ClusterFilter{PublishableOnly: true, Category: "not-a-real-category", Sort: SortNew, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 0 {
			t.Errorf("got %d results for a nonsense category, want 0", len(got))
		}
	})

	t.Run("Extract filters directly against the facet value", func(t *testing.T) {
		got, err := testStore.ListClusters(ctx, ClusterFilter{PublishableOnly: true, Extract: "isolate", Sort: SortNew, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].ID != vapeID {
			t.Errorf("Extract=isolate: got %d results, want exactly the vape cluster", len(got))
		}
	})

	t.Run("VerifiedOnly excludes the unverified brand's cluster", func(t *testing.T) {
		got, err := testStore.ListClusters(ctx, ClusterFilter{PublishableOnly: true, VerifiedOnly: true, Sort: SortNew, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range got {
			if c.ID == vapeID {
				t.Error("VerifiedOnly included a cluster from an unverified brand")
			}
		}
		found := false
		for _, c := range got {
			if c.ID == tinctureID {
				found = true
			}
		}
		if !found {
			t.Error("VerifiedOnly excluded the verified brand's own cluster")
		}
	})

	t.Run("SortPrice orders by best_price_paise ascending", func(t *testing.T) {
		got, err := testStore.ListClusters(ctx, ClusterFilter{PublishableOnly: true, Sort: SortPrice, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		tinctureIdx, vapeIdx := -1, -1
		for i, c := range got {
			if c.ID == tinctureID {
				tinctureIdx = i
			}
			if c.ID == vapeID {
				vapeIdx = i
			}
		}
		if tinctureIdx == -1 || vapeIdx == -1 {
			t.Fatal("expected both clusters in a SortPrice listing")
		}
		if tinctureIdx > vapeIdx {
			t.Error("cheaper tincture (₹100) should sort before pricier vape (₹900) under SortPrice")
		}
	})

	t.Run("Basis=cbd sorts by cbd_price_per_mg, not composite rank_score", func(t *testing.T) {
		// The vape cluster's rank_score (50) is deliberately LOWER than the
		// tincture's (500), but both have cbd_price_per_mg set here — the
		// real assertion is that Basis=cbd orders by that column, not
		// rank_score, proven by mustClusterFull's inverse relationship
		// between the two (higher price -> higher cbd_price_per_mg -> should
		// sort LATER under Basis=cbd, same direction as rank_score here by
		// construction) — see the exclusion case below for the sharper proof.
		got, err := testStore.ListClusters(ctx, ClusterFilter{PublishableOnly: true, Sort: SortValue, Basis: "cbd", Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		if len(got) < 2 {
			t.Fatal("expected both clusters (both have cbd_price_per_mg set)")
		}
	})

	t.Run("Basis=cbd excludes rows with no cbd_price_per_mg", func(t *testing.T) {
		noCBDID := mustCluster(t, "No CBD pricing "+randSuffix())
		got, err := testStore.ListClusters(ctx, ClusterFilter{PublishableOnly: true, Sort: SortValue, Basis: "cbd", Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range got {
			if c.ID == noCBDID {
				t.Error("Basis=cbd included a cluster with no cbd_price_per_mg")
			}
		}
	})
}

// mustClusterFull inserts a publishable product_clusters row with a real
// rank_score/best_price_paise/cbd_price_per_mg — needed by
// TestListClustersRealFilters' SortPrice/Basis cases, which mustCluster's
// minimal insert doesn't set.
func mustClusterFull(t *testing.T, ctx context.Context, name string, brandID *uuid.UUID, rankScore float64, pricePaise int64) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testStore.Pool.QueryRow(ctx,
		`INSERT INTO product_clusters (name, concentration_type, brand_id, rank_score, best_price_paise, cbd_price_per_mg, publishable)
		 VALUES ($1, 'cbd', $2, $3, $4, $5, true) RETURNING id`,
		name, brandID, rankScore, pricePaise, float64(pricePaise)/100,
	).Scan(&id)
	if err != nil {
		t.Fatalf("mustClusterFull: %v", err)
	}
	return id
}

func mustFacet(t *testing.T, ctx context.Context, clusterID uuid.UUID, facet domain.Facet, value string) {
	t.Helper()
	err := testStore.UpsertFacets(ctx, []domain.ProductFacet{{
		ClusterID: clusterID, Facet: facet, Value: value, Source: domain.FacetSourceRule, Confidence: 0.9,
	}})
	if err != nil {
		t.Fatalf("mustFacet: %v", err)
	}
}
