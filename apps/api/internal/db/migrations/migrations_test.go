// Package migrations_test proves the M0 schema — not by re-reading the SQL,
// but by actually running it. It starts a throwaway Postgres 16 container,
// applies every migration up, exercises the constraints that matter (the
// ones a human is most likely to get subtly wrong: exactly-one-of CHECKs,
// enum CHECKs, ON DELETE CASCADE, UNIQUE constraints that encode a real
// scraper bug fix), rolls everything back down, and tears the container down.
//
// Requires a working `docker` on PATH. Skips (does not fail) if docker isn't
// available, so this doesn't break `go test ./...` on a machine without it —
// but wherever it DOES run, a green result means the schema genuinely works
// against real Postgres, not just "gofmt didn't complain."
package migrations_test

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/pressly/goose/v3"
)

// startPostgres launches a throwaway postgres:16 container on a
// docker-assigned free port, waits for it to accept connections, and returns
// a ready-to-use *sql.DB plus a cleanup func. Fails the test (not skips) on
// any error past the initial docker-availability check — once we've decided
// to run this test, a docker failure is a real failure, not an environment
// quirk to shrug off.
func startPostgres(t *testing.T) *sql.DB {
	t.Helper()

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available, skipping migration integration test")
	}

	name := fmt.Sprintf("drtoke-migtest-%d", time.Now().UnixNano())
	run := exec.Command("docker", "run", "-d", "--rm", "--name", name,
		"-e", "POSTGRES_PASSWORD=test",
		"-e", "POSTGRES_USER=drtoke",
		"-e", "POSTGRES_DB=drtoke_test",
		"-p", "127.0.0.1::5432", // docker-assigned free host port
		"postgres:16",
	)
	if out, err := run.CombinedOutput(); err != nil {
		t.Fatalf("docker run: %v\n%s", err, out)
	}
	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-f", name).Run()
	})

	portOut, err := exec.Command("docker", "port", name, "5432/tcp").CombinedOutput()
	if err != nil {
		t.Fatalf("docker port: %v\n%s", err, portOut)
	}
	// Output like "127.0.0.1:54321" (docker may print an IPv6 line too; take the first).
	addr := strings.TrimSpace(strings.Split(string(portOut), "\n")[0])
	parts := strings.Split(addr, ":")
	port := parts[len(parts)-1]

	dsn := fmt.Sprintf("postgres://drtoke:test@127.0.0.1:%s/drtoke_test?sslmode=disable", port)

	var db *sql.DB
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		db, err = sql.Open("pgx", dsn)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			pingErr := db.PingContext(ctx)
			cancel()
			if pingErr == nil {
				return db
			}
			_ = db.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("postgres did not become ready within 30s: %v", err)
	return nil
}

func tableExists(t *testing.T, db *sql.DB, name string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)`,
		name,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("checking table %s: %v", name, err)
	}
	return exists
}

// TestMigrationsUpDown applies all seven migrations, confirms every table
// they claim to create actually exists, rolls all the way back down,
// confirms they're gone, then re-applies — proving both directions work, not
// just Up (a broken Down is still a real migration bug: it's what runs the
// moment anyone needs to revert a bad deploy).
func TestMigrationsUpDown(t *testing.T) {
	db := startPostgres(t)
	defer db.Close()

	goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("SetDialect: %v", err)
	}

	wantTables := []string{
		"scrape_sources", "scrape_batches", "raw_products", "product_listings", "purchase_tokens",
		"media_assets", "product_clusters", "cluster_merges",
		"product_facets", "product_facet_overrides",
		"brands", "states", "roa_methods", "aggregators",
		"content_docs", "content_revisions",
		"accounts", "refresh_tokens", "comments", "admin_users", "admin_audit_log",
		"click_events", "survey_counts", "survey_meta", "review_queue",
	}

	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("goose up: %v", err)
	}
	for _, tbl := range wantTables {
		if !tableExists(t, db, tbl) {
			t.Errorf("after up: table %q does not exist", tbl)
		}
	}

	if err := goose.DownTo(db, ".", 0); err != nil {
		t.Fatalf("goose down: %v", err)
	}
	for _, tbl := range wantTables {
		if tableExists(t, db, tbl) {
			t.Errorf("after down: table %q still exists", tbl)
		}
	}

	// Back up — later tests in this package need a live schema, and this
	// also confirms Up is idempotently re-runnable after a full Down, which
	// is exactly the "restart clean" path a real rollback-then-redeploy takes.
	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("goose up (second pass): %v", err)
	}
}

// TestSeedData confirms the harvested reference content actually landed —
// the PoC scraper source, the 12 ground-truth brands, and the aggregator
// whose source_slug should be non-NULL because its scraper exists.
func TestSeedData(t *testing.T) {
	db := startPostgres(t)
	defer db.Close()
	mustMigrateUp(t, db)

	var brandCount int
	if err := db.QueryRow(`SELECT count(*) FROM brands`).Scan(&brandCount); err != nil {
		t.Fatal(err)
	}
	if brandCount != 12 {
		t.Errorf("brands count = %d, want 12", brandCount)
	}

	var sourceSlug string
	if err := db.QueryRow(`SELECT slug FROM scrape_sources`).Scan(&sourceSlug); err != nil {
		t.Fatal(err)
	}
	if sourceSlug != "cbdstore" {
		t.Errorf("only scrape_sources row = %q, want cbdstore (PoC scope, harvest/NOTES.md)", sourceSlug)
	}

	var cbdstoreAggSource sql.NullString
	if err := db.QueryRow(`SELECT source_slug FROM aggregators WHERE slug='cbdstore-india'`).Scan(&cbdstoreAggSource); err != nil {
		t.Fatal(err)
	}
	if !cbdstoreAggSource.Valid || cbdstoreAggSource.String != "cbdstore" {
		t.Errorf("cbdstore-india.source_slug = %v, want 'cbdstore'", cbdstoreAggSource)
	}

	var itshempAggSource sql.NullString
	if err := db.QueryRow(`SELECT source_slug FROM aggregators WHERE slug='itshemp'`).Scan(&itshempAggSource); err != nil {
		t.Fatal(err)
	}
	if itshempAggSource.Valid {
		t.Errorf("itshemp.source_slug = %v, want NULL (scraper not built yet, harvest/NOTES.md deferred list)", itshempAggSource.String)
	}
}

// TestConstraints exercises exactly the constraints a schema like this is
// most likely to get subtly wrong: an exactly-one-of CHECK that only checks
// "not both nil" and forgets "not both set" is a real, common bug class.
func TestConstraints(t *testing.T) {
	db := startPostgres(t)
	defer db.Close()
	mustMigrateUp(t, db)

	var clusterID, accountID, postID, docID string
	if err := db.QueryRow(`INSERT INTO product_clusters (name, concentration_type) VALUES ('Test', 'cbd') RETURNING id`).Scan(&clusterID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO accounts (handle, password_hash) VALUES ('testuser', 'x') RETURNING id`).Scan(&accountID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO content_docs (kind, slug) VALUES ('post', 'test-post') RETURNING id`).Scan(&docID); err != nil {
		t.Fatal(err)
	}
	postID = docID

	t.Run("comments: exactly one of post_id/cluster_id — neither set fails", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO comments (account_id, body) VALUES ($1, 'this should fail, neither target set')`, accountID)
		if err == nil {
			t.Error("expected constraint violation, got none")
		}
	})

	t.Run("comments: exactly one of post_id/cluster_id — BOTH set fails", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO comments (account_id, post_id, cluster_id, body) VALUES ($1, $2, $3, 'this should also fail, both targets set')`,
			accountID, postID, clusterID)
		if err == nil {
			t.Error("expected constraint violation when BOTH post_id and cluster_id are set, got none — the exactly-one-of CHECK is under-constrained")
		}
	})

	t.Run("comments: exactly one set (cluster_id) succeeds", func(t *testing.T) {
		var id string
		err := db.QueryRow(`INSERT INTO comments (account_id, cluster_id, body) VALUES ($1, $2, 'a perfectly valid review body') RETURNING id`,
			accountID, clusterID).Scan(&id)
		if err != nil {
			t.Errorf("expected success, got %v", err)
		}
	})

	t.Run("comments: exactly one set (post_id) succeeds", func(t *testing.T) {
		var id string
		err := db.QueryRow(`INSERT INTO comments (account_id, post_id, body) VALUES ($1, $2, 'a perfectly valid forum reply') RETURNING id`,
			accountID, postID).Scan(&id)
		if err != nil {
			t.Errorf("expected success, got %v", err)
		}
	})

	t.Run("comments: body length CHECK rejects too short", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO comments (account_id, cluster_id, body) VALUES ($1, $2, 'short')`, accountID, clusterID)
		if err == nil {
			t.Error("expected constraint violation for a 5-char body (CHECK is 10..1000), got none")
		}
	})

	t.Run("product_facets: invalid facet name rejected", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO product_facets (cluster_id, facet, value, source, confidence, evidence, classifier_version)
			VALUES ($1, 'not_a_real_facet', 'x', 'rule', 0.9, '{}', 1)`, clusterID)
		if err == nil {
			t.Error("expected constraint violation, got none")
		}
	})

	facetCases := []struct {
		facet string
		value string
	}{
		{"form", "capsule"}, {"route", "oral"}, {"extract", "full_spectrum"},
		{"profile", "cbd_dominant"}, {"carrier", "mct"}, {"purchasable", "true"},
	}
	for _, fc := range facetCases {
		t.Run("product_facets: valid facet "+fc.facet+" accepted", func(t *testing.T) {
			_, err := db.Exec(`INSERT INTO product_facets (cluster_id, facet, value, source, confidence, evidence, classifier_version)
				VALUES ($1, $2, $3, 'rule', 0.9, '{}', 1) ON CONFLICT (cluster_id, facet) DO UPDATE SET value = EXCLUDED.value`,
				clusterID, fc.facet, fc.value)
			if err != nil {
				t.Errorf("expected success for facet=%s value=%s, got %v", fc.facet, fc.value, err)
			}
		})
	}

	t.Run("product_clusters: invalid value_tier rejected", func(t *testing.T) {
		_, err := db.Exec(`UPDATE product_clusters SET value_tier = 'excellent' WHERE id = $1`, clusterID)
		if err == nil {
			t.Error("expected constraint violation for value_tier='excellent' (ADR-012 bands are good/mid/high), got none")
		}
	})

	t.Run("product_clusters: valid value_tier accepted", func(t *testing.T) {
		_, err := db.Exec(`UPDATE product_clusters SET value_tier = 'good' WHERE id = $1`, clusterID)
		if err != nil {
			t.Errorf("expected success, got %v", err)
		}
	})

	t.Run("product_listings: UNIQUE(source_slug, source_url) rejects a duplicate — the per-variant-URL bug fix", func(t *testing.T) {
		_, err := db.Exec(`INSERT INTO product_listings (source_slug, source_url, name_raw, price_paise) VALUES ('cbdstore', 'https://cbdstore.in/products/x?variant=1', 'A', 100)`)
		if err != nil {
			t.Fatalf("first insert should succeed: %v", err)
		}
		_, err = db.Exec(`INSERT INTO product_listings (source_slug, source_url, name_raw, price_paise) VALUES ('cbdstore', 'https://cbdstore.in/products/x?variant=1', 'A duplicate', 200)`)
		if err == nil {
			t.Error("expected UNIQUE violation on duplicate (source_slug, source_url) — this is what makes the harvested per-variant-URL fix meaningful, without it two variants collapse onto one row")
		}
	})

	t.Run("product_facets: ON DELETE CASCADE — deleting a cluster removes its facets", func(t *testing.T) {
		var cid string
		if err := db.QueryRow(`INSERT INTO product_clusters (name, concentration_type) VALUES ('Cascade Test', 'unknown') RETURNING id`).Scan(&cid); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO product_facets (cluster_id, facet, value, source, confidence, evidence, classifier_version)
			VALUES ($1, 'form', 'capsule', 'rule', 0.9, '{}', 1)`, cid); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`DELETE FROM product_clusters WHERE id = $1`, cid); err != nil {
			t.Fatalf("delete should succeed via cascade: %v", err)
		}
		var remaining int
		if err := db.QueryRow(`SELECT count(*) FROM product_facets WHERE cluster_id = $1`, cid).Scan(&remaining); err != nil {
			t.Fatal(err)
		}
		if remaining != 0 {
			t.Errorf("expected ON DELETE CASCADE to remove facets, %d rows remain", remaining)
		}
	})
}

func mustMigrateUp(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("SetDialect: %v", err)
	}
	if err := goose.Up(db, "."); err != nil {
		t.Fatalf("goose up: %v", err)
	}
}
