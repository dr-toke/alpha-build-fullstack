package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/dr-toke/api/internal/compliance"
	"github.com/dr-toke/api/internal/domain"
	"github.com/dr-toke/api/internal/ingest"
	"github.com/dr-toke/api/internal/resolve"
	"github.com/dr-toke/api/internal/store"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// startTestStore spins up a throwaway postgres:16 container, same pattern
// as internal/ingest's own helper of the same name (duplicated rather than
// shared across packages — internal/api is the only package here that
// needs a real DB end-to-end and pulling in a cross-package test-only
// dependency felt like the wrong weight for one test file).
func startTestStore(t *testing.T) (*store.Store, func()) {
	t.Helper()
	name := fmt.Sprintf("drtoke-apitest-%d", time.Now().UnixNano())
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

// stageAndPromote is the test's own miniature version of the scrape step —
// stages one raw product directly (no network), gates it (bootstrap
// approves), and promotes it through the real internal/ingest pipeline, so
// this test exercises the actual product a real request would see, not a
// hand-inserted row that skips resolve/dedup entirely.
func stageAndPromote(t *testing.T, ctx context.Context, st *store.Store, rs *resolve.RuleSet, crs *compliance.RuleSet, sourceSlug string, p domain.RawProduct) uuid.UUID {
	t.Helper()
	batchID, err := st.CreateBatch(ctx, sourceSlug)
	if err != nil {
		t.Fatal(err)
	}
	p.SourceSlug = sourceSlug
	if _, err := st.StageRawProduct(ctx, batchID, p); err != nil {
		t.Fatal(err)
	}
	if err := st.FinishBatch(ctx, batchID, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := ingest.DecideGate(ctx, st, batchID); err != nil {
		t.Fatal(err)
	}
	result, err := ingest.Promote(ctx, st, rs, crs, batchID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Promoted != 1 {
		t.Fatalf("stageAndPromote: got Promoted=%d, want 1 (errors: %v)", result.Promoted, result.Errors)
	}

	var clusterID uuid.UUID
	if err := st.Pool.QueryRow(ctx,
		`SELECT cluster_id FROM product_listings WHERE source_slug = $1 AND source_url = $2`,
		sourceSlug, p.SourceURL,
	).Scan(&clusterID); err != nil {
		t.Fatal(err)
	}
	return clusterID
}

func TestListAndGetProducts(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}

	st, cleanup := startTestStore(t)
	defer cleanup()
	ctx := context.Background()

	rs, err := resolve.LoadRuleSet("../../harvest/rules")
	if err != nil {
		t.Fatalf("resolve.LoadRuleSet: %v", err)
	}
	crs, err := compliance.LoadRuleSet("../../harvest/rules/compliance.json")
	if err != nil {
		t.Fatalf("compliance.LoadRuleSet: %v", err)
	}

	slug := "api-test-source"
	if _, err := st.Pool.Exec(ctx,
		`INSERT INTO scrape_sources (slug, name, platform, base_url) VALUES ($1,$2,'shopify','https://example.com')`,
		slug, "API Test Source"); err != nil {
		t.Fatal(err)
	}

	id1 := stageAndPromote(t, ctx, st, rs, crs, slug, domain.RawProduct{
		SourceURL:   "https://example.com/products/cbd-oil?variant=1",
		Name:        "BOHECO CBD Oil 500mg - 30ml",
		BrandRaw:    "boheco",
		PriceRaw:    "₹1999.00",
		Description: "Full spectrum CBD oil, MCT carrier, sublingual drops, 500mg CBD.",
		CategoryRaw: "Tinctures",
		RawData:     map[string]any{"in_stock": true},
	})
	id2 := stageAndPromote(t, ctx, st, rs, crs, slug, domain.RawProduct{
		SourceURL:   "https://example.com/products/cbd-oil?variant=2",
		Name:        "BOHECO CBD Oil 1000mg - 30ml",
		BrandRaw:    "boheco",
		PriceRaw:    "₹2999.00",
		Description: "Full spectrum CBD oil, MCT carrier, sublingual drops, 1000mg CBD.",
		CategoryRaw: "Tinctures",
		RawData:     map[string]any{"in_stock": true},
	})

	router := NewRouter(st, "https://drtoke.in")

	t.Run("GET /api/products returns the envelope with real promoted products", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/products?limit=10", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, body %s", w.Code, w.Body.String())
		}
		var env Envelope
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatalf("decode envelope: %v\nbody: %s", err, w.Body.String())
		}
		if env.Total < 2 {
			t.Errorf("got total=%d, want at least 2", env.Total)
		}

		var products []productPayload
		raw, _ := json.Marshal(env.Data)
		if err := json.Unmarshal(raw, &products); err != nil {
			t.Fatalf("decode products: %v", err)
		}
		if len(products) < 2 {
			t.Fatalf("got %d products, want at least 2", len(products))
		}
		found1, found2 := false, false
		for _, p := range products {
			if p.ID == id1 {
				found1 = true
				if p.Brand == nil || p.Brand.Slug != "boheco" {
					t.Errorf("product %s: brand not populated correctly: %+v", id1, p.Brand)
				}
				if p.CBDMg == nil || *p.CBDMg != 500 {
					t.Errorf("product %s: cbd_mg = %v, want 500", id1, p.CBDMg)
				}
				if p.BestPricePaise == nil || *p.BestPricePaise != 199900 {
					t.Errorf("product %s: best_price_paise = %v, want 199900", id1, p.BestPricePaise)
				}
				if len(p.Listings) != 1 {
					t.Errorf("product %s: got %d listings, want 1", id1, len(p.Listings))
				}
			}
			if p.ID == id2 {
				found2 = true
			}
		}
		if !found1 || !found2 {
			t.Errorf("expected both promoted products in the listing (found1=%v found2=%v)", found1, found2)
		}
	})

	t.Run("cursor pagination walks forward without repeating rows", func(t *testing.T) {
		w1 := httptest.NewRecorder()
		router.ServeHTTP(w1, httptest.NewRequest(http.MethodGet, "/api/products?limit=1", nil))
		var env1 Envelope
		if err := json.Unmarshal(w1.Body.Bytes(), &env1); err != nil {
			t.Fatal(err)
		}
		if !env1.HasMore || env1.NextCursor == "" {
			t.Fatalf("page 1: has_more=%v next_cursor=%q, want has_more=true and a cursor (2 products, limit 1)", env1.HasMore, env1.NextCursor)
		}
		var page1 []productPayload
		raw, _ := json.Marshal(env1.Data)
		_ = json.Unmarshal(raw, &page1)

		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/api/products?limit=1&cursor="+env1.NextCursor, nil))
		var env2 Envelope
		if err := json.Unmarshal(w2.Body.Bytes(), &env2); err != nil {
			t.Fatal(err)
		}
		var page2 []productPayload
		raw2, _ := json.Marshal(env2.Data)
		_ = json.Unmarshal(raw2, &page2)

		if len(page1) != 1 || len(page2) != 1 {
			t.Fatalf("got page1=%d page2=%d products, want 1 and 1", len(page1), len(page2))
		}
		if page1[0].ID == page2[0].ID {
			t.Error("same product appeared on both pages")
		}
	})

	t.Run("GET /api/products/{id} returns the full product", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/products/"+id1.String(), nil))
		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, body %s", w.Code, w.Body.String())
		}
		var p productPayload
		if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
			t.Fatal(err)
		}
		if p.ID != id1 {
			t.Errorf("got id=%s, want %s", p.ID, id1)
		}
	})

	t.Run("GET /api/products/{id} on an unknown id is 404 not_found", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/products/"+uuid.New().String(), nil))
		if w.Code != http.StatusNotFound {
			t.Fatalf("got status %d, want 404", w.Code)
		}
		var body errorBody
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Error.Code != CodeNotFound {
			t.Errorf("got code=%q, want %q", body.Error.Code, CodeNotFound)
		}
	})

	t.Run("GET /api/products/{id} on a merged cluster returns 200 with moved_to, never 404", func(t *testing.T) {
		newID := id2
		if err := st.Merge(ctx, id1, newID); err != nil {
			t.Fatal(err)
		}
		defer func() {
			// Undo the merge for any later subtests in this run that expect
			// id1 to still resolve directly.
			if _, err := st.Pool.Exec(ctx, `DELETE FROM cluster_merges WHERE old_id = $1`, id1); err != nil {
				t.Fatal(err)
			}
		}()

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/products/"+id1.String(), nil))
		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want 200 for a merged cluster", w.Code)
		}
		var body map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["moved_to"] != newID.String() {
			t.Errorf("got moved_to=%q, want %q", body["moved_to"], newID.String())
		}
	})

	t.Run("GET /healthz reports ok against a live database", func(t *testing.T) {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("got status %d, want 200", w.Code)
		}
	})

	t.Run("CORS headers only appear for the allowed origin", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/products", nil)
		req.Header.Set("Origin", "https://evil.example")
		router.ServeHTTP(w, req)
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("CORS header set for a non-allowed origin")
		}

		w2 := httptest.NewRecorder()
		req2 := httptest.NewRequest(http.MethodGet, "/api/products", nil)
		req2.Header.Set("Origin", "https://drtoke.in")
		router.ServeHTTP(w2, req2)
		if w2.Header().Get("Access-Control-Allow-Origin") != "https://drtoke.in" {
			t.Errorf("got Access-Control-Allow-Origin=%q, want https://drtoke.in", w2.Header().Get("Access-Control-Allow-Origin"))
		}
	})
}
