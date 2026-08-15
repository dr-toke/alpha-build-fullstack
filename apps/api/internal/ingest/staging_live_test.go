package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/dr-toke/api/internal/store"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// cappedAdapter wraps a real Adapter and stops after n listings — so this
// demo proves the full scrape -> stage -> Postgres pipeline against LIVE
// data without pulling cbdstore.in's entire catalogue on every run of a
// test that exists to prove the *pipeline*, not to be the production full
// scrape (that's the uncapped adapter, used as-is once M5's job runner
// calls StageBatch for real).
type cappedAdapter struct {
	inner Adapter
	max   int
}

func (c *cappedAdapter) Source() string { return c.inner.Source() }
func (c *cappedAdapter) ScrapeAll(ctx context.Context, onListing func(RawListing) error) error {
	n := 0
	err := c.inner.ScrapeAll(ctx, func(l RawListing) error {
		if n >= c.max {
			return errStopEarly
		}
		n++
		return onListing(l)
	})
	if err == errStopEarly {
		return nil
	}
	return err
}

// TestStageBatchLiveCBDStore is the actual proof: real network scrape of
// cbdstore.in, staged into a real throwaway Postgres 16 container, read
// back and checked. Guarded the same way as shopify_live_test.go
// (INGEST_LIVE_TEST=1) plus needing docker — this is the single most
// expensive test in the repo (network + a fresh container) and stays
// deliberately opt-in.
func TestStageBatchLiveCBDStore(t *testing.T) {
	if os.Getenv("INGEST_LIVE_TEST") != "1" {
		t.Skip("set INGEST_LIVE_TEST=1 to run this against real cbdstore.in + a real Postgres container")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}

	st, cleanup := startTestStore(t)
	defer cleanup()

	specs, err := LoadScraperSpec("../../harvest/scrapers")
	if err != nil {
		t.Fatalf("LoadScraperSpec: %v", err)
	}
	spec := specs["cbdstore"]
	if spec == nil {
		t.Fatal("cbdstore spec not found")
	}

	real := NewShopify(spec, "Mozilla/5.0 (compatible; DrTokeBot/0.1; +https://drtoke.in)", 500)
	adapter := &cappedAdapter{inner: real, max: 40}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	batchID, count, err := StageBatch(ctx, st, adapter)
	if err != nil {
		t.Fatalf("StageBatch: %v", err)
	}
	t.Logf("staged batch %s: %d real listings from live cbdstore.in", batchID, count)
	if count == 0 {
		t.Fatal("staged zero listings from a live site with real products")
	}

	// Prove it actually landed in raw_products (staging), NOT
	// product_listings (live) — ADR-010's whole point.
	var liveCount int
	if err := st.Pool.QueryRow(ctx, `SELECT count(*) FROM product_listings WHERE source_slug = 'cbdstore'`).Scan(&liveCount); err != nil {
		t.Fatal(err)
	}
	if liveCount != 0 {
		t.Errorf("product_listings has %d cbdstore rows — scraped data must never reach the live table without going through the promotion gate", liveCount)
	}

	got, err := LoadBatch(ctx, st, batchID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != count {
		t.Errorf("LoadBatch returned %d rows, StageBatch reported staging %d", len(got), count)
	}

	t.Logf("real staged rows (first 5 of %d):", len(got))
	for i, p := range got {
		if i >= 5 {
			break
		}
		t.Logf("  %-40s | brand=%-20s | %s", truncate(p.Name, 40), p.BrandRaw, p.PriceRaw)
	}

	// Batch status and count should reflect what actually happened.
	var status string
	var productCount int
	if err := st.Pool.QueryRow(ctx, `SELECT status, product_count FROM scrape_batches WHERE id = $1`, batchID).Scan(&status, &productCount); err != nil {
		t.Fatal(err)
	}
	if status != "pending_review" {
		t.Errorf("batch status = %s, want pending_review", status)
	}
	if productCount != count {
		t.Errorf("scrape_batches.product_count = %d, want %d", productCount, count)
	}
}

// startTestStore spins up a throwaway postgres:16 container, migrates it,
// and returns a connected *store.Store plus a cleanup func — same pattern
// as internal/db/migrations/migrations_test.go and internal/store/
// main_test.go, duplicated here rather than shared across packages because
// this is the only test in internal/ingest that needs a real database (the
// rest are pure-logic or network-only) and a whole TestMain-per-package
// felt like the wrong weight for one test.
func startTestStore(t *testing.T) (*store.Store, func()) {
	t.Helper()
	name := fmt.Sprintf("drtoke-ingesttest-%d", time.Now().UnixNano())
	run := exec.Command("docker", "run", "-d", "--rm", "--name", name,
		"-e", "POSTGRES_PASSWORD=test", "-e", "POSTGRES_USER=drtoke", "-e", "POSTGRES_DB=drtoke_test",
		"-p", "127.0.0.1::5432", "postgres:16")
	if out, err := run.CombinedOutput(); err != nil {
		t.Fatalf("docker run: %v\n%s", err, out)
	}
	cleanup := func() { _ = exec.Command("docker", "rm", "-f", name).Run() }

	portOut, err := exec.Command("docker", "port", name, "5432/tcp").CombinedOutput()
	if err != nil {
		cleanup()
		t.Fatalf("docker port: %v\n%s", err, portOut)
	}
	addr := strings.TrimSpace(strings.Split(string(portOut), "\n")[0])
	port := addr[strings.LastIndex(addr, ":")+1:]
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
				break
			}
			_ = db.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
	if db == nil {
		cleanup()
		t.Fatalf("postgres did not become ready: %v", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		cleanup()
		t.Fatal(err)
	}
	if err := goose.Up(db, "../db/migrations"); err != nil {
		cleanup()
		t.Fatalf("goose.Up: %v", err)
	}
	db.Close()

	st, err := store.New(context.Background(), dsn)
	if err != nil {
		cleanup()
		t.Fatalf("store.New: %v", err)
	}
	return st, func() { st.Close(); cleanup() }
}
