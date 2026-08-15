package ingest

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/dr-toke/api/internal/domain"
)

func TestEvaluateGate(t *testing.T) {
	t.Run("bootstrap: no previous batch approves unconditionally", func(t *testing.T) {
		d := EvaluateGate(5, nil)
		if !d.Approved {
			t.Errorf("got rejected (%s), want approved for a source's first-ever batch", d.Reason)
		}
	})

	t.Run("previous count of zero also bootstraps rather than dividing by zero", func(t *testing.T) {
		zero := 0
		d := EvaluateGate(400, &zero)
		if !d.Approved {
			t.Errorf("got rejected (%s), want approved", d.Reason)
		}
	})

	t.Run("count holds steady approves", func(t *testing.T) {
		prev := 400
		d := EvaluateGate(400, &prev)
		if !d.Approved {
			t.Errorf("got rejected (%s), want approved", d.Reason)
		}
	})

	t.Run("count increases approves", func(t *testing.T) {
		prev := 400
		d := EvaluateGate(500, &prev)
		if !d.Approved {
			t.Errorf("got rejected (%s), want approved", d.Reason)
		}
	})

	t.Run("a drop right at the 30% threshold still approves (strictly greater-than rejects)", func(t *testing.T) {
		prev := 400
		d := EvaluateGate(280, &prev) // exactly 30% drop
		if !d.Approved {
			t.Errorf("got rejected (%s), want approved at exactly 30%%", d.Reason)
		}
	})

	t.Run("a drop just past 30% rejects with a reason", func(t *testing.T) {
		prev := 400
		d := EvaluateGate(279, &prev) // 30.25% drop
		if d.Approved {
			t.Error("got approved, want rejected")
		}
		if d.Reason == "" {
			t.Error("rejection carries no reason — ADR-010 requires a reason code, always")
		}
	})

	t.Run("the motivating failure mode: a store's markup breaks and the scraper returns 10% of the catalogue", func(t *testing.T) {
		prev := 400
		d := EvaluateGate(40, &prev)
		if d.Approved {
			t.Error("got approved — this is exactly the silent-90%-wipe scenario ADR-010 exists to catch")
		}
	})
}

// TestDecideGate proves the store-writing wrapper against a real Postgres
// container — bootstrap approves the first batch, then a second batch with
// a big count drop against that first approved baseline gets rejected, and
// the rejection is actually persisted (status, reason, previous_count).
func TestDecideGate(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}

	st, cleanup := startTestStore(t)
	defer cleanup()
	ctx := context.Background()

	slug := fmt.Sprintf("gate-test-%d", time.Now().UnixNano())
	if _, err := st.Pool.Exec(ctx,
		`INSERT INTO scrape_sources (slug, name, platform, base_url) VALUES ($1,$2,'shopify','https://example.com')`,
		slug, "Gate Test Source"); err != nil {
		t.Fatal(err)
	}

	// First batch: bootstrap, no prior approved run.
	b1, err := st.CreateBatch(ctx, slug)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.FinishBatch(ctx, b1, 400); err != nil {
		t.Fatal(err)
	}
	d1, err := DecideGate(ctx, st, b1)
	if err != nil {
		t.Fatal(err)
	}
	if !d1.Approved {
		t.Fatalf("bootstrap batch got rejected: %s", d1.Reason)
	}
	got1, err := st.BatchByID(ctx, b1)
	if err != nil {
		t.Fatal(err)
	}
	if got1.Status != domain.BatchApproved {
		t.Errorf("persisted status = %s, want approved", got1.Status)
	}

	// Second batch: catastrophic drop against the now-approved baseline.
	b2, err := st.CreateBatch(ctx, slug)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.FinishBatch(ctx, b2, 40); err != nil {
		t.Fatal(err)
	}
	d2, err := DecideGate(ctx, st, b2)
	if err != nil {
		t.Fatal(err)
	}
	if d2.Approved {
		t.Fatal("got approved, want rejected for a 90% count drop")
	}
	got2, err := st.BatchByID(ctx, b2)
	if err != nil {
		t.Fatal(err)
	}
	if got2.Status != domain.BatchRejected {
		t.Errorf("persisted status = %s, want rejected", got2.Status)
	}
	if got2.RejectionReason == nil || *got2.RejectionReason == "" {
		t.Error("rejection_reason not persisted")
	}
	if got2.PreviousProductCount == nil || *got2.PreviousProductCount != 400 {
		t.Errorf("previous_product_count = %v, want 400", got2.PreviousProductCount)
	}

	// A rejected batch must not have touched raw_products/product_listings —
	// DecideGate only ever writes to scrape_batches itself.
	var liveCount int
	if err := st.Pool.QueryRow(ctx, `SELECT count(*) FROM product_listings WHERE source_slug = $1`, slug).Scan(&liveCount); err != nil {
		t.Fatal(err)
	}
	if liveCount != 0 {
		t.Errorf("product_listings has %d rows for a rejected source — gate must never write live", liveCount)
	}
}
