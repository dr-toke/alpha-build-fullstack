package store

import (
	"context"
	"errors"
	"testing"

	"github.com/dr-toke/api/internal/domain"
)

func TestContentPublishFlow(t *testing.T) {
	ctx := context.Background()
	docID := mustContentDoc(t, domain.ContentPost)

	t.Run("unpublished doc has no current revision", func(t *testing.T) {
		var slug string
		if err := testStore.Pool.QueryRow(ctx, `SELECT slug FROM content_docs WHERE id = $1`, docID).Scan(&slug); err != nil {
			t.Fatal(err)
		}
		d, rev, err := testStore.DocBySlug(ctx, domain.ContentPost, slug, "en")
		if err != nil {
			t.Fatal(err)
		}
		if d.Status != domain.ContentDraft {
			t.Errorf("status = %s, want draft", d.Status)
		}
		if rev != nil {
			t.Error("expected nil revision before anything is published")
		}
	})

	revID, err := testStore.NewRevision(ctx, domain.ContentRevision{
		DocID: docID, Title: "First Draft", BodyMD: "hello", Author: "writer1", License: "all-rights-reserved",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("revision exists but is not yet current or published", func(t *testing.T) {
		var publishedAt *string
		if err := testStore.Pool.QueryRow(ctx, `SELECT published_at::text FROM content_revisions WHERE id = $1`, revID).Scan(&publishedAt); err != nil {
			t.Fatal(err)
		}
		if publishedAt != nil {
			t.Error("a new revision should not have published_at set until Publish is called")
		}
	})

	t.Run("Publish flips the pointer and stamps published_at atomically", func(t *testing.T) {
		if err := testStore.Publish(ctx, docID, revID); err != nil {
			t.Fatal(err)
		}
		var slug string
		if err := testStore.Pool.QueryRow(ctx, `SELECT slug FROM content_docs WHERE id = $1`, docID).Scan(&slug); err != nil {
			t.Fatal(err)
		}
		d, rev, err := testStore.DocBySlug(ctx, domain.ContentPost, slug, "en")
		if err != nil {
			t.Fatal(err)
		}
		if d.Status != domain.ContentPublished {
			t.Errorf("status = %s, want published", d.Status)
		}
		if rev == nil || rev.ID != revID {
			t.Fatalf("current revision = %v, want %s", rev, revID)
		}
		if rev.PublishedAt == nil {
			t.Error("published_at should be set on the now-current revision")
		}
		if rev.Author != "writer1" || rev.License != "all-rights-reserved" {
			t.Errorf("author/license not round-tripped: %+v", rev)
		}
	})

	t.Run("rollback is the same operation against an older revision — 03-DOMAIN-MODEL.md §11", func(t *testing.T) {
		rev2ID, err := testStore.NewRevision(ctx, domain.ContentRevision{
			DocID: docID, Title: "Second Draft", BodyMD: "v2", Author: "writer1", License: "all-rights-reserved",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := testStore.Publish(ctx, docID, rev2ID); err != nil {
			t.Fatal(err)
		}
		// Roll back to the first revision.
		if err := testStore.Publish(ctx, docID, revID); err != nil {
			t.Fatal(err)
		}
		var slug string
		if err := testStore.Pool.QueryRow(ctx, `SELECT slug FROM content_docs WHERE id = $1`, docID).Scan(&slug); err != nil {
			t.Fatal(err)
		}
		_, rev, err := testStore.DocBySlug(ctx, domain.ContentPost, slug, "en")
		if err != nil {
			t.Fatal(err)
		}
		if rev.ID != revID || rev.Title != "First Draft" {
			t.Errorf("after rollback, current revision = %+v, want the first draft", rev)
		}
	})

	t.Run("Publish with a revision belonging to a different doc fails", func(t *testing.T) {
		otherDoc := mustContentDoc(t, domain.ContentFAQ)
		err := testStore.Publish(ctx, otherDoc, revID) // revID belongs to docID, not otherDoc
		if err == nil {
			t.Error("expected an error publishing a revision that belongs to a different doc")
		}
	})
}

func TestDocBySlugNotFound(t *testing.T) {
	_, _, err := testStore.DocBySlug(context.Background(), domain.ContentPost, "does-not-exist-"+randSuffix(), "en")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("got %v, want domain.ErrNotFound", err)
	}
}

func TestPublishedDocs(t *testing.T) {
	ctx := context.Background()
	docID := mustContentDoc(t, domain.ContentFAQ)
	revID, err := testStore.NewRevision(ctx, domain.ContentRevision{DocID: docID, Title: "FAQ Item", Author: "a", License: "CC-BY-4.0"})
	if err != nil {
		t.Fatal(err)
	}
	if err := testStore.Publish(ctx, docID, revID); err != nil {
		t.Fatal(err)
	}

	got, err := testStore.PublishedDocs(ctx, domain.ContentFAQ, "en")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range got {
		if d.ID == docID {
			found = true
		}
		if d.Status != domain.ContentPublished {
			t.Errorf("PublishedDocs returned a non-published doc: %+v", d)
		}
	}
	if !found {
		t.Error("just-published doc not found in PublishedDocs")
	}
}
