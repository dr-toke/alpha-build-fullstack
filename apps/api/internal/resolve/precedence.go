package resolve

import (
	"time"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
)

// FacetInputs bundles every tier that might propose a value for one facet on
// one cluster. Any tier may be nil — that's the normal case for Model (M1
// has no ML model yet) and often for Override (most products are never
// human-corrected) and Default (some facets, like `form`, have no sensible
// default; see Resolve's doc comment).
type FacetInputs struct {
	Override *domain.ProductFacetOverride
	Rule     *FacetResult
	Model    *FacetResult
	Default  *FacetResult
}

// Resolve is THE single writer of facet precedence — 03-DOMAIN-MODEL.md §2:
// "Precedence is absolute: override > rule > model > default. Enforced in
// one function... and nowhere else." Every other file in this package
// (facets.go's ResolveForm etc.) only PROPOSES; nothing else in the codebase
// is allowed to pick a winner among tiers. If a second code path ever picks
// between override/rule/model/default, that is the bug ADR-002/ADR-003
// exist to prevent.
//
// Pure function: returns the domain.ProductFacet that SHOULD be persisted,
// but does not persist it — 08-BUILD-ORDERS.md §7, M1 is "no DB, no HTTP."
// A nil result means no tier proposed anything at all; the caller (M3/M5)
// decides whether that's a review_queue row or a skip.
func Resolve(clusterID uuid.UUID, facet domain.Facet, in FacetInputs, classifierVersion int) *domain.ProductFacet {
	now := time.Now().UTC()

	if in.Override != nil {
		return &domain.ProductFacet{
			ClusterID:         clusterID,
			Facet:             facet,
			Value:             in.Override.Value,
			Source:            domain.FacetSourceOverride,
			Confidence:        1.0, // 03-DOMAIN-MODEL.md §2: "override is always 1.0"
			Evidence:          map[string]any{"reason": in.Override.Reason, "set_by": in.Override.SetBy},
			ClassifierVersion: classifierVersion,
			DecidedAt:         now,
		}
	}
	if in.Rule != nil && in.Rule.Value != "" {
		return &domain.ProductFacet{
			ClusterID:         clusterID,
			Facet:             facet,
			Value:             in.Rule.Value,
			Source:            domain.FacetSourceRule,
			Confidence:        in.Rule.Confidence,
			Evidence:          EvidenceToMap(in.Rule.Evidence),
			ClassifierVersion: classifierVersion,
			DecidedAt:         now,
		}
	}
	if in.Model != nil && in.Model.Value != "" {
		return &domain.ProductFacet{
			ClusterID:         clusterID,
			Facet:             facet,
			Value:             in.Model.Value,
			Source:            domain.FacetSourceModel,
			Confidence:        in.Model.Confidence,
			Evidence:          EvidenceToMap(in.Model.Evidence),
			ClassifierVersion: classifierVersion,
			DecidedAt:         now,
		}
	}
	if in.Default != nil && in.Default.Value != "" {
		return &domain.ProductFacet{
			ClusterID:         clusterID,
			Facet:             facet,
			Value:             in.Default.Value,
			Source:            domain.FacetSourceDefault,
			Confidence:        in.Default.Confidence,
			Evidence:          EvidenceToMap(in.Default.Evidence),
			ClassifierVersion: classifierVersion,
			DecidedAt:         now,
		}
	}
	return nil
}

// EvidenceToMap converts Evidence into the map[string]any shape every jsonb
// evidence/cannabinoid_evidence column expects. Exported (was
// Resolve()-private until M4/M5) because internal/ingest's promote.go needs
// the exact same conversion for domain.ProductCluster.CannabinoidEvidence —
// one conversion function, not a second copy drifting out of sync with this
// one.
func EvidenceToMap(ev Evidence) map[string]any {
	m := map[string]any{}
	if len(ev.Matched) > 0 {
		m["matched"] = ev.Matched
	}
	if len(ev.Negated) > 0 {
		m["negated"] = ev.Negated
	}
	if len(ev.Notes) > 0 {
		m["notes"] = ev.Notes
	}
	return m
}

// Publishable evaluates 03-DOMAIN-MODEL.md §2's confidence gate:
//
//	publishable := facets.purchasable
//	            AND form.confidence  >= 0.85
//	            AND (route IS NULL OR route.confidence >= 0.90)
//	            AND price_paise > 0
//
// Pure function mirroring the domain.ProductCluster.Publishable cached
// column's intended source of truth (internal/db/migrations/002, M0) —
// whatever writes that column (a future M3/M5 job) should call this, not
// re-derive the formula inline a second time.
func Publishable(purchasable bool, formConfidence float32, route *domain.ProductFacet, pricePaise int64) bool {
	if !purchasable {
		return false
	}
	if formConfidence < 0.85 {
		return false
	}
	if route != nil && route.Confidence < 0.90 {
		return false
	}
	return pricePaise > 0
}
