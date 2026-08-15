package store

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
)

// randSuffix keeps test data unique across the whole shared-container suite
// (main_test.go) without needing per-test transactions/truncation — see
// main_test.go's doc comment for why that tradeoff was made.
func randSuffix() string {
	return fmt.Sprintf("%08x", rand.Uint32())
}

// mustCluster inserts a minimal product_clusters row for tests that need a
// real cluster to hang facets/comments/overrides/listings off of.
func mustCluster(t *testing.T, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testStore.Pool.QueryRow(context.Background(),
		`INSERT INTO product_clusters (name, concentration_type) VALUES ($1, 'unknown') RETURNING id`,
		name+" "+randSuffix(),
	).Scan(&id)
	if err != nil {
		t.Fatalf("mustCluster: %v", err)
	}
	return id
}

// mustBrand inserts a brand with a unique slug, returns the slug.
func mustBrand(t *testing.T) string {
	t.Helper()
	slug := "test-brand-" + randSuffix()
	_, err := testStore.Pool.Exec(context.Background(),
		`INSERT INTO brands (slug, name) VALUES ($1, $2)`, slug, "Test Brand")
	if err != nil {
		t.Fatalf("mustBrand: %v", err)
	}
	return slug
}

// mustAccount inserts a pseudonymous account with a unique handle.
func mustAccount(t *testing.T) *domain.Account {
	t.Helper()
	a, err := testStore.CreateAccount(context.Background(), "user"+randSuffix()[:8], "hash")
	if err != nil {
		t.Fatalf("mustAccount: %v", err)
	}
	return a
}

// mustSource inserts a scrape_sources row with a unique slug (scrape_batches
// and product_listings both FK to it).
func mustSource(t *testing.T) string {
	t.Helper()
	slug := "src" + randSuffix()[:8]
	_, err := testStore.Pool.Exec(context.Background(),
		`INSERT INTO scrape_sources (slug, name, platform, base_url) VALUES ($1,$2,'shopify','https://example.com')`,
		slug, "Test Source")
	if err != nil {
		t.Fatalf("mustSource: %v", err)
	}
	return slug
}

// mustContentDoc inserts a content_docs row with a unique slug.
func mustContentDoc(t *testing.T, kind domain.ContentKind) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := testStore.Pool.QueryRow(context.Background(),
		`INSERT INTO content_docs (kind, slug) VALUES ($1, $2) RETURNING id`,
		kind, "doc-"+randSuffix(),
	).Scan(&id)
	if err != nil {
		t.Fatalf("mustContentDoc: %v", err)
	}
	return id
}
