package store

import (
	"context"
	"testing"
)

// The seed data from internal/db/migrations/004 (10 states, 5 ROA methods,
// 3 aggregators, harvested verbatim from the prior alpha) is already present
// in every test run via TestMain's migration pass — these tests check the
// self-correction machinery (stale computation, broken-link withholding)
// against that real seed data rather than inserting fresh rows.

func TestListStates(t *testing.T) {
	ctx := context.Background()
	got, err := testStore.ListStates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 10 {
		t.Fatalf("got %d states, want 10 (harvested seed data)", len(got))
	}

	t.Run("Delhi NCR is featured and sorts first", func(t *testing.T) {
		if got[0].Slug != "delhi-ncr" || !got[0].Featured {
			t.Errorf("first state = %s (featured=%v), want delhi-ncr featured", got[0].Slug, got[0].Featured)
		}
	})

	t.Run("stale is computed — seed data is from 2025-05-01, long past any reasonable verify_interval_days", func(t *testing.T) {
		for _, s := range got {
			if !s.Stale {
				t.Errorf("state %s: stale = false, want true (last_verified 2025-05-01, verify_interval_days=%d)", s.Slug, s.VerifyIntervalDays)
			}
		}
	})

	t.Run("states without an excise_url start as no_url, not unknown", func(t *testing.T) {
		found := false
		for _, s := range got {
			if s.ExciseURL == nil {
				found = true
				if s.LinkStatus != "no_url" {
					t.Errorf("state %s: link_status = %s, want no_url", s.Slug, s.LinkStatus)
				}
			}
		}
		if !found {
			t.Skip("no state without excise_url in seed data to check")
		}
	})
}

func TestListStatesWithholdsExciseURLPastBrokenThreshold(t *testing.T) {
	ctx := context.Background()
	// Push delhi-ncr's link_failures past the broken threshold and confirm
	// the URL is withheld even though it's set in the row.
	if _, err := testStore.Pool.Exec(ctx,
		`UPDATE states SET link_failures = $1 WHERE slug = 'delhi-ncr'`, linkBrokenThreshold); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = testStore.Pool.Exec(ctx, `UPDATE states SET link_failures = 0 WHERE slug = 'delhi-ncr'`)
	})

	got, err := testStore.ListStates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range got {
		if s.Slug == "delhi-ncr" {
			if s.ExciseURL != nil {
				t.Error("excise_url should be withheld (nil) once link_failures reaches the broken threshold — 'we never send users to a 404'")
			}
			return
		}
	}
	t.Fatal("delhi-ncr not found in results")
}

func TestListROA(t *testing.T) {
	ctx := context.Background()
	got, err := testStore.ListROA(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Fatalf("got %d ROA methods, want 5", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].DisplayOrder > got[i].DisplayOrder {
			t.Errorf("not sorted by display_order: %d before %d", got[i-1].DisplayOrder, got[i].DisplayOrder)
		}
	}
	// The edibles-delay golden rule is content, not a hardcoded frontend
	// string — 03-DOMAIN-MODEL.md §7. Confirm it actually made it into the DB.
	for _, m := range got {
		if m.Slug == "edibles-capsules" {
			if m.WarningNote == nil || *m.WarningNote == "" {
				t.Error("edibles-capsules should carry the golden-rule warning note")
			}
			return
		}
	}
	t.Fatal("edibles-capsules ROA method not found")
}

func TestListAggregators(t *testing.T) {
	ctx := context.Background()
	got, err := testStore.ListAggregators(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d aggregators, want 3 (harvested seed data)", len(got))
	}

	t.Run("cbdstore-india has a non-nil source_slug, others are nil (PoC scope)", func(t *testing.T) {
		for _, a := range got {
			switch a.Slug {
			case "cbdstore-india":
				if a.SourceSlug == nil || *a.SourceSlug != "cbdstore" {
					t.Errorf("cbdstore-india.source_slug = %v, want cbdstore", a.SourceSlug)
				}
			case "itshemp", "cannameds-india":
				if a.SourceSlug != nil {
					t.Errorf("%s.source_slug = %v, want nil (scraper not built yet)", a.Slug, *a.SourceSlug)
				}
			}
		}
	})
}
