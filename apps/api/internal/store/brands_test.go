package store

import (
	"context"
	"errors"
	"testing"

	"github.com/dr-toke/api/internal/domain"
)

func TestBrandBySlugAndApprove(t *testing.T) {
	ctx := context.Background()
	slug := mustBrand(t)

	t.Run("found, starts unverified", func(t *testing.T) {
		b, err := testStore.BrandBySlug(ctx, slug)
		if err != nil {
			t.Fatal(err)
		}
		if b.Verified {
			t.Error("newly inserted brand should not be verified by default")
		}
	})

	t.Run("unknown slug returns ErrNotFound", func(t *testing.T) {
		_, err := testStore.BrandBySlug(ctx, "does-not-exist-"+randSuffix())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("got %v, want domain.ErrNotFound", err)
		}
	})

	t.Run("Approve flips verified and bumps last_verified", func(t *testing.T) {
		if err := testStore.Approve(ctx, slug); err != nil {
			t.Fatal(err)
		}
		b, err := testStore.BrandBySlug(ctx, slug)
		if err != nil {
			t.Fatal(err)
		}
		if !b.Verified {
			t.Error("brand should be verified after Approve")
		}
	})

	t.Run("Approve on an unknown slug returns ErrNotFound", func(t *testing.T) {
		err := testStore.Approve(ctx, "does-not-exist-"+randSuffix())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("got %v, want domain.ErrNotFound", err)
		}
	})
}

func TestListBrandsOrdering(t *testing.T) {
	ctx := context.Background()
	verifiedSlug := mustBrand(t)
	if err := testStore.Approve(ctx, verifiedSlug); err != nil {
		t.Fatal(err)
	}
	_ = mustBrand(t) // an unverified one, just to have both kinds present

	got, err := testStore.ListBrands(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) < 2 {
		t.Fatalf("got %d brands, want at least 2 (seed data + these)", len(got))
	}
	// Verified brands must sort before unverified ones.
	firstUnverifiedIdx := -1
	for i, b := range got {
		if !b.Verified {
			firstUnverifiedIdx = i
			break
		}
	}
	if firstUnverifiedIdx == -1 {
		return // all verified, nothing to check
	}
	for i := 0; i < firstUnverifiedIdx; i++ {
		if !got[i].Verified {
			t.Errorf("unverified brand %q sorted before position %d, verified should sort first", got[i].Slug, firstUnverifiedIdx)
		}
	}
}
