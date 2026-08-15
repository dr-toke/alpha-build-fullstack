// Package store is the sole SQL layer — 01-ARCHITECTURE.md §6's modularity
// rule: "SQL lives only in the store layer." Every query in this package is
// hand-written and parameterised ($1, $2, ...) per 00-CONSTITUTION.md §4;
// 08-BUILD-ORDERS.md §3 lists this package among the never-delegate files
// for exactly that reason. No ORM (00-CONSTITUTION.md §6) — pgx/v5 native,
// not database/sql, so query results map onto internal/domain structs
// directly via pgx.RowToStructByName without a reflection-heavy ORM layer
// in between.
package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store wraps a pgx connection pool. One Store per process — internal/api
// and internal/jobs both hold one, constructed once in cmd/server and
// cmd/worker respectively (M5).
type Store struct {
	Pool *pgxpool.Pool
}

// New opens a pool against databaseURL and verifies connectivity with a
// ping before returning — a Store that can't reach Postgres should fail at
// startup, not on the first request.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store.New: ping: %w", err)
	}
	return &Store{Pool: pool}, nil
}

// Close releases the pool. Safe to call once at process shutdown.
func (s *Store) Close() {
	s.Pool.Close()
}
