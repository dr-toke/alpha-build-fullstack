package ingest

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/dr-toke/api/internal/domain"
	"github.com/dr-toke/api/internal/store"
	"github.com/google/uuid"
)

// Fingerprint is harvest/rules/dedup.md's exact-match dedup key, ported
// byte-for-byte from the prior alpha's internal/catalog/dedup/dedup.go
// (confirmed via its only real caller, internal/jobs/normalise.go:65 —
// `dedup.Fingerprint(brandSlug, rawName, volML, cb.BestMG())`). rawName is
// passed through AS SCRAPED — the doc's own "normalised name" language is
// aspirational; the only normalisation that ever happened is the
// lower+trim done right here.
//
// SHA-256, truncated to the first 16 bytes (32 hex chars), not the full
// digest — matches the harvested algorithm exactly, not a stronger hash of
// our choosing.
func Fingerprint(brandSlug, rawName string, volumeML, concentrationMG float64) string {
	key := fmt.Sprintf("%s|%s|%.1f|%.1f",
		brandSlug,
		strings.ToLower(strings.TrimSpace(rawName)),
		volumeML,
		concentrationMG,
	)
	h := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", h[:16])
}

// AssignCluster resolves one raw product to a durable cluster UUID —
// check-then-create against the fingerprint, the application-level
// invariant internal/db/migrations/008's own comment describes (fingerprint
// is deliberately NOT a DB-unique column, since a merged-away cluster keeps
// its old fingerprint forever).
//
// cluster is the fully-populated shell to insert IF this fingerprint is
// new — by the time promote.go calls this, resolve's cannabinoid/facet/
// price pipeline has already run, so AssignCluster only ever persists,
// never computes.
//
// harvest/rules/dedup.md flags fuzzy/near-duplicate matching (JaroWinkler,
// WithinPct) as dead code in the prior alpha — scaffolded, never wired in.
// This is exact-fingerprint-only, same as what actually ran in production
// before. A product whose scraped name gains a stray space or a "-" vs "–"
// between scrapes creates a second cluster rather than merging — a known,
// documented gap (harvest/rules/dedup.md's "Known false-merge / false-split
// cases"), not something this function silently papers over.
func AssignCluster(ctx context.Context, st *store.Store, fingerprint string, cluster domain.ProductCluster) (uuid.UUID, error) {
	existing, err := st.ClusterByFingerprint(ctx, fingerprint)
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return uuid.Nil, fmt.Errorf("ingest.AssignCluster: %w", err)
	}

	cluster.Fingerprint = &fingerprint
	id, err := st.CreateCluster(ctx, cluster)
	if err != nil {
		return uuid.Nil, fmt.Errorf("ingest.AssignCluster: %w", err)
	}
	return id, nil
}
