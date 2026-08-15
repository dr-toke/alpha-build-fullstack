package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// golden.go is the one file in this package that is NOT SQL —
// 01-ARCHITECTURE.md §6's modularity rule ("SQL lives only in the store
// layer") is about where SQL lives, not a claim that store only ever does
// SQL. 08-BUILD-ORDERS.md §7 places AppendFixture here explicitly (3.11),
// so this is a deliberate exception in the build order itself, not one this
// implementation introduced.
//
// Fixtures written here use a DIFFERENT naming scheme than the hand-authored
// harvest-time fixtures already in testdata/golden/ (cannex-strong-plus-
// cannabinoids.json, etc.) — those are named descriptively and were
// converted from the prior alpha's real test cases (harvest/NOTES.md).
// AppendFixture names by cluster ID because that's what 03-DOMAIN-MODEL.md
// §2 specifies ("auto-appends a fixture to testdata/golden/{cluster_id}
// .json") — it's a RUNTIME mechanism: every time an admin sets an override
// on a real cluster (M9), a fixture locking that decision lands in the same
// directory internal/resolve/golden_test.go (M1) already walks, so the CI
// golden set grows automatically as humans correct real mistakes. Both
// naming schemes coexist in testdata/golden/ by design.

// GoldenFixture matches 11-HARVEST.md §2.6's format exactly.
type GoldenFixture struct {
	Source         string         `json:"source"`
	Raw            GoldenRaw      `json:"raw"`
	Expect         map[string]any `json:"expect"`
	RegressionNote string         `json:"regression_note,omitempty"`
}

// GoldenRaw is the raw scraped text a fixture locks a decision against.
type GoldenRaw struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Price       string `json:"price,omitempty"`
}

// AppendFixture writes (or merges into) dir/{clusterID}.json. Merging, not
// overwriting, matters because a cluster can accumulate overrides on
// different facets over time (form corrected today, extract corrected next
// month) — each call adds to Expect rather than discarding what a previous
// override already locked in.
func AppendFixture(dir string, clusterID uuid.UUID, source string, raw GoldenRaw, expect map[string]any, regressionNote string) error {
	path := filepath.Join(dir, clusterID.String()+".json")

	fx := GoldenFixture{Source: source, Raw: raw, Expect: map[string]any{}}
	if existing, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(existing, &fx); err != nil {
			return fmt.Errorf("store.AppendFixture: parsing existing %s: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("store.AppendFixture: reading %s: %w", path, err)
	}

	// New values win — the override being appended right now is the most
	// recent human correction; an older locked expectation for the SAME
	// facet is exactly what's being superseded.
	for k, v := range expect {
		fx.Expect[k] = v
	}
	if regressionNote != "" {
		fx.RegressionNote = regressionNote
	}
	fx.Source = source
	fx.Raw = raw

	data, err := json.MarshalIndent(fx, "", "  ")
	if err != nil {
		return fmt.Errorf("store.AppendFixture: encoding: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("store.AppendFixture: writing %s: %w", path, err)
	}
	return nil
}
