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
		got, err := testStore.ListClusters(ctx, ClusterFilter{BrandID: &brandID, PublishableOnly: true, Limit: 10})
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

	t.Run("keyset pagination walks forward without duplicates or gaps", func(t *testing.T) {
		page1, err := testStore.ListClusters(ctx, ClusterFilter{BrandID: &brandID, PublishableOnly: true, Limit: 2})
		if err != nil {
			t.Fatal(err)
		}
		if len(page1) != 2 {
			t.Fatalf("page1 len = %d, want 2", len(page1))
		}
		last := page1[len(page1)-1]
		page2, err := testStore.ListClusters(ctx, ClusterFilter{
			BrandID: &brandID, PublishableOnly: true, Limit: 2,
			CursorRankScore: last.RankScore, CursorID: &last.ID,
		})
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
}
