package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
)

func TestCreateAccountDuplicateHandle(t *testing.T) {
	ctx := context.Background()
	handle := "dupe" + randSuffix()[:8]

	if _, err := testStore.CreateAccount(ctx, handle, "hash1"); err != nil {
		t.Fatal(err)
	}
	_, err := testStore.CreateAccount(ctx, handle, "hash2")
	if !errors.Is(err, domain.ErrDuplicateHandle) {
		t.Errorf("got %v, want domain.ErrDuplicateHandle", err)
	}
}

func TestAccountByHandleCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	handle := "MixedCase" + randSuffix()[:8]
	created, err := testStore.CreateAccount(ctx, handle, "hash")
	if err != nil {
		t.Fatal(err)
	}

	got, err := testStore.AccountByHandle(ctx, strings.ToLower(handle))
	if err != nil {
		t.Fatalf("case-insensitive lookup failed: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("got a different account via case-insensitive lookup")
	}
	if got.Handle != handle {
		t.Errorf("Handle = %q, want the original case-preserved %q", got.Handle, handle)
	}
}

// TestAccountByIDAndTouchLastSeen closes a coverage gap found the same way
// as M1's Normalize/Tokens: a coverage check turned up AccountByID and
// TouchLastSeen at 0% — both real, exported, and simply never exercised by
// any test above. CommentsForPost has the same story, tested separately.
func TestAccountByIDAndTouchLastSeen(t *testing.T) {
	ctx := context.Background()
	acct := mustAccount(t)

	got, err := testStore.AccountByID(ctx, acct.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Handle != acct.Handle {
		t.Errorf("got handle=%q, want %q", got.Handle, acct.Handle)
	}

	t.Run("unknown id returns ErrNotFound", func(t *testing.T) {
		_, err := testStore.AccountByID(ctx, uuid.New())
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("got %v, want domain.ErrNotFound", err)
		}
	})

	t.Run("TouchLastSeen bumps last_seen_at forward", func(t *testing.T) {
		before := got.LastSeenAt
		time.Sleep(10 * time.Millisecond) // ensure now() strictly advances past `before`
		if err := testStore.TouchLastSeen(ctx, acct.ID); err != nil {
			t.Fatal(err)
		}
		after, err := testStore.AccountByID(ctx, acct.ID)
		if err != nil {
			t.Fatal(err)
		}
		if !after.LastSeenAt.After(before) {
			t.Errorf("last_seen_at did not advance: before=%v after=%v", before, after.LastSeenAt)
		}
	})
}

func TestCommentsForPost(t *testing.T) {
	ctx := context.Background()
	acct := mustAccount(t)
	docID := mustContentDoc(t, domain.ContentPost)

	if _, err := testStore.CreateComment(ctx, domain.Comment{
		AccountID: acct.ID, PostID: &docID, Body: "a forum reply worth reading",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := testStore.CommentsForPost(ctx, docID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d comments, want 1", len(got))
	}
	if got[0].PostID == nil || *got[0].PostID != docID {
		t.Errorf("PostID = %v, want %s", got[0].PostID, docID)
	}
}

func TestRefreshTokenLifecycle(t *testing.T) {
	ctx := context.Background()
	acct := mustAccount(t)
	hash := "hash-" + randSuffix()

	if err := testStore.CreateRefreshToken(ctx, domain.RefreshToken{
		AccountID: acct.ID, TokenHash: hash, ExpiresAt: time.Now().Add(24 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}

	t.Run("found while valid", func(t *testing.T) {
		got, err := testStore.RefreshTokenByHash(ctx, hash)
		if err != nil {
			t.Fatal(err)
		}
		if got.AccountID != acct.ID {
			t.Errorf("account mismatch")
		}
	})

	t.Run("revoked token is no longer found — auth_invalid, not a generic error", func(t *testing.T) {
		if err := testStore.RevokeRefreshToken(ctx, hash); err != nil {
			t.Fatal(err)
		}
		_, err := testStore.RefreshTokenByHash(ctx, hash)
		if !errors.Is(err, domain.ErrAuthInvalid) {
			t.Errorf("got %v, want domain.ErrAuthInvalid", err)
		}
	})

	t.Run("expired token is not found even if not revoked", func(t *testing.T) {
		expiredHash := "expired-" + randSuffix()
		if err := testStore.CreateRefreshToken(ctx, domain.RefreshToken{
			AccountID: acct.ID, TokenHash: expiredHash, ExpiresAt: time.Now().Add(-1 * time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
		_, err := testStore.RefreshTokenByHash(ctx, expiredHash)
		if !errors.Is(err, domain.ErrAuthInvalid) {
			t.Errorf("got %v, want domain.ErrAuthInvalid for an expired token", err)
		}
	})
}

func TestCommentsExactlyOneTarget(t *testing.T) {
	ctx := context.Background()
	acct := mustAccount(t)
	clusterID := mustCluster(t, "Comment Test")

	c, err := testStore.CreateComment(ctx, domain.Comment{
		AccountID: acct.ID, ClusterID: &clusterID, Body: "a perfectly good review body",
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.ID.String() == "" {
		t.Fatal("CreateComment returned an empty ID")
	}

	t.Run("appears in CommentsForCluster", func(t *testing.T) {
		got, err := testStore.CommentsForCluster(ctx, clusterID, 10, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 {
			t.Fatalf("got %d comments, want 1", len(got))
		}
	})

	t.Run("both post_id and cluster_id set is rejected by the DB CHECK", func(t *testing.T) {
		docID := mustContentDoc(t, domain.ContentPost)
		_, err := testStore.CreateComment(ctx, domain.Comment{
			AccountID: acct.ID, PostID: &docID, ClusterID: &clusterID, Body: "should fail, both targets set",
		})
		if err == nil {
			t.Error("expected a CHECK violation, got nil")
		}
	})

	t.Run("DeleteComment by the owning account succeeds", func(t *testing.T) {
		c2, err := testStore.CreateComment(ctx, domain.Comment{AccountID: acct.ID, ClusterID: &clusterID, Body: "deletable comment body"})
		if err != nil {
			t.Fatal(err)
		}
		if err := testStore.DeleteComment(ctx, c2.ID, &acct.ID, false); err != nil {
			t.Fatal(err)
		}
		got, err := testStore.CommentsForCluster(ctx, clusterID, 10, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, gc := range got {
			if gc.ID == c2.ID {
				t.Error("soft-deleted comment still returned by CommentsForCluster")
			}
		}
	})

	t.Run("DeleteComment by a different account is rejected", func(t *testing.T) {
		other := mustAccount(t)
		c3, err := testStore.CreateComment(ctx, domain.Comment{AccountID: acct.ID, ClusterID: &clusterID, Body: "not your comment to delete"})
		if err != nil {
			t.Fatal(err)
		}
		err = testStore.DeleteComment(ctx, c3.ID, &other.ID, false)
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("got %v, want domain.ErrNotFound (the WHERE account_id=$3 guard should exclude it)", err)
		}
	})

	t.Run("DeleteComment by admin works regardless of owning account", func(t *testing.T) {
		c4, err := testStore.CreateComment(ctx, domain.Comment{AccountID: acct.ID, ClusterID: &clusterID, Body: "admin will delete this one"})
		if err != nil {
			t.Fatal(err)
		}
		if err := testStore.DeleteComment(ctx, c4.ID, nil, true); err != nil {
			t.Fatal(err)
		}
		var deletedByAdmin bool
		if err := testStore.Pool.QueryRow(ctx, `SELECT deleted_by_admin FROM comments WHERE id = $1`, c4.ID).Scan(&deletedByAdmin); err != nil {
			t.Fatal(err)
		}
		if !deletedByAdmin {
			t.Error("deleted_by_admin should be true for an admin-initiated delete")
		}
	})
}
