package store

import (
	"context"
	"errors"
	"testing"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
)

func TestEnqueueListQueueResolve(t *testing.T) {
	ctx := context.Background()
	clusterID := mustCluster(t, "Queue Test")

	id, err := testStore.Enqueue(ctx, domain.ReviewQueueItem{
		ClusterID: clusterID, Reason: domain.ReviewCategoryUncertain,
		Detail: "ambiguous form", ProposedValue: map[string]any{"form": "other"},
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Run("appears in the pending queue, filtered by reason", func(t *testing.T) {
		got, err := testStore.ListQueue(ctx, QueueFilter{Reason: domain.ReviewCategoryUncertain, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, it := range got {
			if it.ID == id {
				found = true
			}
		}
		if !found {
			t.Error("enqueued item not found in pending queue filtered by its reason")
		}
	})

	t.Run("does not appear when filtered by a different reason", func(t *testing.T) {
		got, err := testStore.ListQueue(ctx, QueueFilter{Reason: domain.ReviewPriceAnomaly, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range got {
			if it.ID == id {
				t.Error("item enqueued with reason=category_uncertain appeared under a price_anomaly filter")
			}
		}
	})

	t.Run("Resolve moves it out of the pending queue", func(t *testing.T) {
		if err := testStore.Resolve(ctx, id, domain.ReviewApproved, "admin1"); err != nil {
			t.Fatal(err)
		}
		got, err := testStore.ListQueue(ctx, QueueFilter{Reason: domain.ReviewCategoryUncertain, Limit: 50})
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range got {
			if it.ID == id {
				t.Error("resolved item still appears in the pending (default status) queue view")
			}
		}
	})

	t.Run("resolving an already-resolved item returns ErrNotFound", func(t *testing.T) {
		err := testStore.Resolve(ctx, id, domain.ReviewApproved, "admin1")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("got %v, want domain.ErrNotFound (the WHERE status='pending' guard should prevent double-resolving)", err)
		}
	})

	t.Run("Resolve rejects an invalid status", func(t *testing.T) {
		other := mustCluster(t, "Queue Test 2")
		id2, err := testStore.Enqueue(ctx, domain.ReviewQueueItem{ClusterID: other, Reason: domain.ReviewLowConfidence, Detail: "x", ProposedValue: map[string]any{}})
		if err != nil {
			t.Fatal(err)
		}
		if err := testStore.Resolve(ctx, id2, domain.ReviewPending, "admin1"); err == nil {
			t.Error("expected an error resolving to status=pending, got nil")
		}
	})

	t.Run("Resolve on an unknown id returns ErrNotFound", func(t *testing.T) {
		err := testStore.Resolve(ctx, uuid.New(), domain.ReviewApproved, "admin1")
		if !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("got %v, want domain.ErrNotFound", err)
		}
	})
}
