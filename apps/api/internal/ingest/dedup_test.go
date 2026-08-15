package ingest

import (
	"context"
	"os/exec"
	"testing"

	"github.com/dr-toke/api/internal/domain"
)

func TestFingerprint(t *testing.T) {
	t.Run("deterministic — same inputs always produce the same fingerprint", func(t *testing.T) {
		a := Fingerprint("boheco", "CBD Oil 500mg", 30, 500)
		b := Fingerprint("boheco", "CBD Oil 500mg", 30, 500)
		if a != b {
			t.Errorf("got %s and %s, want equal", a, b)
		}
	})

	t.Run("differs when any single input differs", func(t *testing.T) {
		base := Fingerprint("boheco", "CBD Oil 500mg", 30, 500)
		cases := map[string]string{
			"brand":  Fingerprint("magiccann", "CBD Oil 500mg", 30, 500),
			"name":   Fingerprint("boheco", "CBD Oil 1000mg", 30, 500),
			"volume": Fingerprint("boheco", "CBD Oil 500mg", 60, 500),
			"mg":     Fingerprint("boheco", "CBD Oil 500mg", 30, 750),
		}
		for label, fp := range cases {
			if fp == base {
				t.Errorf("%s: fingerprint unchanged when input differs", label)
			}
		}
	})

	t.Run("name matching is case- and whitespace-insensitive, per the harvested lower+trim", func(t *testing.T) {
		a := Fingerprint("boheco", "  CBD Oil 500mg  ", 30, 500)
		b := Fingerprint("boheco", "cbd oil 500mg", 30, 500)
		if a != b {
			t.Errorf("got %s and %s, want equal (case/whitespace should not matter)", a, b)
		}
	})

	t.Run("32 hex chars — first 16 bytes of sha256, not the full digest", func(t *testing.T) {
		fp := Fingerprint("boheco", "CBD Oil 500mg", 30, 500)
		if len(fp) != 32 {
			t.Errorf("got length %d, want 32", len(fp))
		}
	})
}

// TestAssignCluster proves the check-then-create invariant against a real
// Postgres container (same discipline as staging_live_test.go's
// startTestStore) — NOT gated by INGEST_LIVE_TEST since it makes no network
// calls, only gated on docker being available.
func TestAssignCluster(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}

	st, cleanup := startTestStore(t)
	defer cleanup()
	ctx := context.Background()

	fp := Fingerprint("boheco", "CBD Oil 500mg", 30, 500)
	shell := domain.ProductCluster{
		Name:              "CBD Oil 500mg",
		ConcentrationType: domain.ConcentrationCBD,
	}

	t.Run("new fingerprint creates a cluster", func(t *testing.T) {
		id, err := AssignCluster(ctx, st, fp, shell)
		if err != nil {
			t.Fatal(err)
		}
		if id.String() == "" {
			t.Fatal("got zero-value UUID")
		}

		got, err := st.ClusterByFingerprint(ctx, fp)
		if err != nil {
			t.Fatal(err)
		}
		if got.ID != id {
			t.Errorf("got id=%s, want %s", got.ID, id)
		}
	})

	t.Run("same fingerprint reuses the existing cluster, not a second one", func(t *testing.T) {
		firstID, err := AssignCluster(ctx, st, fp, shell)
		if err != nil {
			t.Fatal(err)
		}
		secondID, err := AssignCluster(ctx, st, fp, shell)
		if err != nil {
			t.Fatal(err)
		}
		if firstID != secondID {
			t.Errorf("got two different cluster IDs (%s, %s) for the same fingerprint", firstID, secondID)
		}
	})

	t.Run("a fingerprint match refreshes derived fields, not just returns the old ID (04-PIPELINE.md §1: critical)", func(t *testing.T) {
		fp3 := Fingerprint("boheco", "Refreshable Product", 30, 500)
		cbd := 500.0
		firstID, err := AssignCluster(ctx, st, fp3, domain.ProductCluster{
			Name:              "Refreshable Product v1",
			ConcentrationType: domain.ConcentrationCBD,
			CBDMg:             &cbd,
		})
		if err != nil {
			t.Fatal(err)
		}

		newCBD := 750.0
		secondID, err := AssignCluster(ctx, st, fp3, domain.ProductCluster{
			Name:              "Refreshable Product v2 (reclassified)",
			ConcentrationType: domain.ConcentrationCBD,
			CBDMg:             &newCBD,
		})
		if err != nil {
			t.Fatal(err)
		}
		if firstID != secondID {
			t.Fatalf("refresh created a new cluster (%s -> %s), want same ID reused", firstID, secondID)
		}

		got, err := st.ClusterByFingerprint(ctx, fp3)
		if err != nil {
			t.Fatal(err)
		}
		if got.Name != "Refreshable Product v2 (reclassified)" {
			t.Errorf("got name=%q, want the refreshed name — fields were not updated on a fingerprint match", got.Name)
		}
		if got.CBDMg == nil || *got.CBDMg != 750.0 {
			t.Errorf("got cbd_mg=%v, want 750 — fields were not updated on a fingerprint match", got.CBDMg)
		}
	})

	t.Run("a different fingerprint creates a distinct cluster", func(t *testing.T) {
		fp2 := Fingerprint("boheco", "CBD Oil 1000mg", 30, 1000)
		id1, err := AssignCluster(ctx, st, fp, shell)
		if err != nil {
			t.Fatal(err)
		}
		id2, err := AssignCluster(ctx, st, fp2, shell)
		if err != nil {
			t.Fatal(err)
		}
		if id1 == id2 {
			t.Error("two distinct fingerprints resolved to the same cluster")
		}
	})
}
