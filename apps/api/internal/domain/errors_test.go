package domain

import (
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

// TestSentinelErrorsAreDistinct guards against a copy-paste mistake making
// two of the nine 02-FRONTEND-CONTRACT.md §3 error codes accidentally
// collapse onto the same underlying error value, which errors.Is would
// then treat as interchangeable.
func TestSentinelErrorsAreDistinct(t *testing.T) {
	sentinels := []error{
		ErrNotFound, ErrValidationFailed, ErrInvalidFilter, ErrAuthRequired,
		ErrAuthInvalid, ErrBanned, ErrRateLimited, ErrUnavailable,
		ErrDuplicateHandle, ErrPurchaseTokenAlreadyClaimed,
	}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i == j {
				continue
			}
			if errors.Is(a, b) {
				t.Errorf("sentinel %d (%v) unexpectedly matches sentinel %d (%v) via errors.Is", i, a, j, b)
			}
		}
	}
}

func TestSentinelErrorsSurviveWrapping(t *testing.T) {
	wrapped := fmt.Errorf("store.GetCluster: %w", ErrNotFound)
	if !errors.Is(wrapped, ErrNotFound) {
		t.Error("wrapped ErrNotFound should still satisfy errors.Is(err, ErrNotFound) — this is the fmt.Errorf(\"%w\") convention 08-BUILD-ORDERS.md §4 requires everywhere")
	}
}

func TestClusterMovedError(t *testing.T) {
	oldID := uuid.New()
	newID := uuid.New()
	err := &ClusterMovedError{OldID: oldID, NewID: newID}

	if got := err.Error(); got == "" {
		t.Error("Error() returned empty string")
	}

	// A moved cluster must be distinguishable from a genuinely missing one —
	// 02-FRONTEND-CONTRACT.md §4 treats "moved" (200 + moved_to) and
	// "not found" (404) as different status-code outcomes, not the same
	// failure. If ClusterMovedError ever satisfied errors.Is(_, ErrNotFound),
	// a handler doing a generic not-found check could silently 404 a
	// perfectly valid merged-product request instead of redirecting it.
	if errors.Is(err, ErrNotFound) {
		t.Error("ClusterMovedError must NOT satisfy errors.Is(err, ErrNotFound) — moved and not-found are different API outcomes")
	}

	// It must still be extractable via errors.As, including through a wrap,
	// since that's how internal/api's product-detail handler is expected to
	// recover OldID/NewID to build the {"moved_to": ...} response body.
	wrapped := fmt.Errorf("store.GetCluster: %w", err)
	var target *ClusterMovedError
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As failed to extract *ClusterMovedError through a wrap")
	}
	if target.NewID != newID {
		t.Errorf("extracted NewID = %v, want %v", target.NewID, newID)
	}
}

// TestEnumValuesMatchMigrationChecks pins every Go enum constant used as a
// facet/type/status literal to the exact string the corresponding migration's
// CHECK constraint expects. This test has no database in it — it can't catch
// a migration that drifted from these strings — but it catches the cheaper,
// more common mistake: someone renaming a Go constant's VALUE (not just its
// identifier) without grepping for every SQL CHECK that has to agree with it.
// The full cross-check against a live CHECK constraint is
// internal/db/migrations/migrations_test.go's job, which actually inserts
// each of these values against real Postgres.
func TestEnumValuesMatchMigrationChecks(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"ConcentrationCBD", string(ConcentrationCBD), "cbd"},
		{"ConcentrationTHC", string(ConcentrationTHC), "thc"},
		{"ConcentrationTotal", string(ConcentrationTotal), "total"},
		{"ConcentrationHempSeed", string(ConcentrationHempSeed), "hemp_seed"},
		{"ConcentrationNutrition", string(ConcentrationNutrition), "nutrition"},
		{"ConcentrationUnknown", string(ConcentrationUnknown), "unknown"},

		{"ValueTierGood", string(ValueTierGood), "good"},
		{"ValueTierMid", string(ValueTierMid), "mid"},
		{"ValueTierHigh", string(ValueTierHigh), "high"},

		{"FacetForm", string(FacetForm), "form"},
		{"FacetRoute", string(FacetRoute), "route"},
		{"FacetExtract", string(FacetExtract), "extract"},
		{"FacetProfile", string(FacetProfile), "profile"},
		{"FacetCarrier", string(FacetCarrier), "carrier"},
		{"FacetPurchasable", string(FacetPurchasable), "purchasable"},

		{"FacetSourceOverride", string(FacetSourceOverride), "override"},
		{"FacetSourceRule", string(FacetSourceRule), "rule"},
		{"FacetSourceModel", string(FacetSourceModel), "model"},
		{"FacetSourceDefault", string(FacetSourceDefault), "default"},

		{"LegalStatusLegal", string(LegalStatusLegal), "legal"},
		{"LegalStatusTolerated", string(LegalStatusTolerated), "tolerated"},
		{"LegalStatusGrey", string(LegalStatusGrey), "grey"},
		{"LegalStatusLimited", string(LegalStatusLimited), "limited"},
		{"LegalStatusIllegal", string(LegalStatusIllegal), "illegal"},

		{"ReviewUnknownBrand", string(ReviewUnknownBrand), "unknown_brand"},
		{"ReviewPriceAnomaly", string(ReviewPriceAnomaly), "price_anomaly"},
		{"ReviewComplianceUncertain", string(ReviewComplianceUncertain), "compliance_uncertain"},
		{"ReviewTerminologyReview", string(ReviewTerminologyReview), "terminology_review"},
		{"ReviewCategoryUncertain", string(ReviewCategoryUncertain), "category_uncertain"},
		{"ReviewLowConfidence", string(ReviewLowConfidence), "low_confidence"},

		{"ScrapePlatform shopify", string(PlatformShopify), "shopify"},
		{"ScrapePlatform woocommerce", string(PlatformWooCommerce), "woocommerce"},
		{"ScrapePlatform custom", string(PlatformCustom), "custom"},

		{"BatchRunning", string(BatchRunning), "running"},
		{"BatchPendingReview", string(BatchPendingReview), "pending_review"},
		{"BatchApproved", string(BatchApproved), "approved"},
		{"BatchRejected", string(BatchRejected), "rejected"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q (must match the CHECK constraint literal in internal/db/migrations)", c.name, c.got, c.want)
		}
	}
}
