package ingest

import (
	"context"
	"fmt"

	"github.com/dr-toke/api/internal/domain"
	"github.com/dr-toke/api/internal/store"
	"github.com/google/uuid"
)

// countDropThreshold is 04-PIPELINE.md §2's first (and, for now, only
// implemented) auto-reject rule: "product count drops more than 30% vs. the
// last successful run."
const countDropThreshold = 0.30

// GateDecision is EvaluateGate's pure verdict — kept separate from the
// store write (DecideGate) so the threshold logic is testable without a
// database.
type GateDecision struct {
	Approved bool
	Reason   string // empty when Approved; a human-readable explanation otherwise
}

// EvaluateGate implements ADR-010's promotion gate, deliberately scoped to
// ONE of 04-PIPELINE.md §2's four auto-reject thresholds — product-count
// drop. The other three (>15% of fields newly null, ₹/mg median shifting
// beyond band, selector-hit-rate below 80%) each need data nothing before
// this milestone computes: per-field null tracking, the bestdeal job's ₹/mg
// distribution, and a scraper-reported hit-rate. Building three unmeasured
// checks would mean auto-rejecting (or, worse, silently never rejecting) on
// numbers that are always zero — worse than not having the check. This is
// the same "partial, not shortcut" pattern as M2's compliance narrow
// exception and M4's Shopify-only ingest scope.
//
// previousCount == nil is the bootstrap case — a source's very first batch
// has nothing to compare against and auto-approves, per ADR-010's spirit
// ("the gate compares against the last successful run"; with no successful
// run yet, there is nothing to protect against a false drop from).
func EvaluateGate(currentCount int, previousCount *int) GateDecision {
	if previousCount == nil || *previousCount == 0 {
		return GateDecision{Approved: true}
	}
	drop := float64(*previousCount-currentCount) / float64(*previousCount)
	if drop > countDropThreshold {
		return GateDecision{
			Approved: false,
			Reason: fmt.Sprintf(
				"product count dropped %.0f%% (%d -> %d), exceeds the %.0f%% threshold (04-PIPELINE.md §2)",
				drop*100, *previousCount, currentCount, countDropThreshold*100,
			),
		}
	}
	return GateDecision{Approved: true}
}

// DecideGate loads a batch, runs EvaluateGate against its source's last
// approved count, and writes the decision back — 'auto' as decided_by,
// matching 04-PIPELINE.md §2's "thresholds are opening estimates... a human
// accepts" model where auto-decisions are the default path and a human
// override is a separate, later action (not built here — that's an admin
// panel action against the same DecideBatch this calls).
func DecideGate(ctx context.Context, st *store.Store, batchID uuid.UUID) (GateDecision, error) {
	batch, err := st.BatchByID(ctx, batchID)
	if err != nil {
		return GateDecision{}, fmt.Errorf("ingest.DecideGate: %w", err)
	}
	if batch.ProductCount == nil {
		return GateDecision{}, fmt.Errorf("ingest.DecideGate: batch %s has no product_count — call FinishBatch first", batchID)
	}

	previous, err := st.LastApprovedBatchCount(ctx, batch.SourceSlug)
	if err != nil {
		return GateDecision{}, fmt.Errorf("ingest.DecideGate: %w", err)
	}

	decision := EvaluateGate(*batch.ProductCount, previous)

	status := domain.BatchApproved
	var reason *string
	if !decision.Approved {
		status = domain.BatchRejected
		reason = &decision.Reason
	}
	if err := st.DecideBatch(ctx, batchID, status, previous, reason, "auto"); err != nil {
		return GateDecision{}, fmt.Errorf("ingest.DecideGate: %w", err)
	}
	return decision, nil
}
