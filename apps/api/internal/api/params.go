package api

import (
	"net/http"
	"strconv"
)

// defaultLimit/maxLimit mirror store.ListClusters' own clamp
// (internal/store/clusters.go) — kept in sync by hand since the store
// package doesn't export its constant; both are defence in depth against
// a caller that isn't internal/api at all (e.g. a future worker).
const (
	defaultLimit = 24
	maxLimit     = 100
)

// parseLimit reads CatalogGrid.svelte's `limit` param (it always sends 24,
// PER_PAGE, but nothing stops a direct API caller from asking for less/more
// within the clamp).
func parseLimit(r *http.Request) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return defaultLimit
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return defaultLimit
	}
	if n > maxLimit {
		return maxLimit
	}
	return n
}

// parsePage reads the real frontend's 1-based `page` param
// (CatalogGrid.svelte: `Math.max(1, parseInt(... ?? '1', 10) || 1)`) —
// mirrored here exactly so a malformed or missing value degrades to page 1
// the same way on both sides.
func parsePage(r *http.Request) int {
	raw := r.URL.Query().Get("page")
	if raw == "" {
		return 1
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 1
	}
	return n
}
