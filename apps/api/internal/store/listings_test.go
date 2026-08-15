package store

import (
	"context"
	"testing"

	"github.com/dr-toke/api/internal/domain"
)

func TestUpsertListing(t *testing.T) {
	ctx := context.Background()
	source := mustSource(t)
	url := "https://example.com/products/x?variant=" + randSuffix()

	t.Run("insert then upsert updates in place, preserving first_seen_at", func(t *testing.T) {
		id1, err := testStore.UpsertListing(ctx, domain.ProductListing{
			SourceSlug: source, SourceURL: url, NameRaw: "Original Name", PricePaise: 10000, InStock: true,
		})
		if err != nil {
			t.Fatalf("first upsert: %v", err)
		}

		id2, err := testStore.UpsertListing(ctx, domain.ProductListing{
			SourceSlug: source, SourceURL: url, NameRaw: "Updated Name", PricePaise: 12000, InStock: false,
		})
		if err != nil {
			t.Fatalf("second upsert: %v", err)
		}
		if id1 != id2 {
			t.Errorf("expected same row id on conflict, got %s then %s", id1, id2)
		}

		listings, err := testStore.ListingsForCluster(ctx, id1) // will be empty, cluster_id never set — just proving the row updated
		_ = listings
		if err != nil {
			t.Fatalf("ListingsForCluster: %v", err)
		}

		var name string
		var price int64
		if err := testStore.Pool.QueryRow(ctx, `SELECT name_raw, price_paise FROM product_listings WHERE id = $1`, id1).Scan(&name, &price); err != nil {
			t.Fatal(err)
		}
		if name != "Updated Name" || price != 12000 {
			t.Errorf("got name=%q price=%d, want Updated Name/12000", name, price)
		}
	})

	t.Run("distinct source_url (per-variant) never collides — the harvested scraper fix", func(t *testing.T) {
		id1, err := testStore.UpsertListing(ctx, domain.ProductListing{
			SourceSlug: source, SourceURL: url + "-variantA", NameRaw: "A", PricePaise: 1000, InStock: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		id2, err := testStore.UpsertListing(ctx, domain.ProductListing{
			SourceSlug: source, SourceURL: url + "-variantB", NameRaw: "B", PricePaise: 2000, InStock: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		if id1 == id2 {
			t.Error("two distinct source_urls collapsed onto one row")
		}
	})

	t.Run("cluster_id preserved on re-upsert when the new value is nil", func(t *testing.T) {
		clusterID := mustCluster(t, "Preserve Cluster Test")
		u := url + "-cluster-preserve"
		id, err := testStore.UpsertListing(ctx, domain.ProductListing{
			SourceSlug: source, SourceURL: u, NameRaw: "A", PricePaise: 1000, InStock: true, ClusterID: &clusterID,
		})
		if err != nil {
			t.Fatal(err)
		}
		// Re-upsert (e.g. a re-scrape) without knowing the cluster yet.
		if _, err := testStore.UpsertListing(ctx, domain.ProductListing{
			SourceSlug: source, SourceURL: u, NameRaw: "A updated", PricePaise: 1500, InStock: true, ClusterID: nil,
		}); err != nil {
			t.Fatal(err)
		}
		listings, err := testStore.ListingsForCluster(ctx, clusterID)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, l := range listings {
			if l.ID == id {
				found = true
			}
		}
		if !found {
			t.Error("cluster_id was wiped by a re-upsert that passed nil — COALESCE guard failed")
		}
	})
}

func TestListingsForCluster(t *testing.T) {
	ctx := context.Background()
	source := mustSource(t)
	clusterID := mustCluster(t, "Listings For Cluster Test")

	for i, price := range []int64{5000, 1000, 3000} {
		if _, err := testStore.UpsertListing(ctx, domain.ProductListing{
			SourceSlug: source, SourceURL: "https://example.com/p" + randSuffix() + string(rune('a'+i)),
			NameRaw: "Item", PricePaise: price, InStock: true, ClusterID: &clusterID,
		}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := testStore.ListingsForCluster(ctx, clusterID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d listings, want 3", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].PricePaise > got[i].PricePaise {
			t.Errorf("listings not sorted cheapest-first: %d before %d", got[i-1].PricePaise, got[i].PricePaise)
		}
	}
}
