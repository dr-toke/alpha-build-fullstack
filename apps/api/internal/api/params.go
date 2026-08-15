package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// defaultLimit/maxLimit mirror store.ListClusters' own clamp
// (internal/store/clusters.go) — kept in sync by hand since the store
// package doesn't export its constant; both are 05-API-REFERENCE.md §1's
// `limit` param, just enforced at two boundaries (defence in depth, not
// duplication of authority — the store layer is what actually enforces
// this against a caller that isn't internal/api at all, e.g. a future
// worker).
const (
	defaultLimit = 24
	maxLimit     = 100
)

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

// encodeCursor packs a sort key (the raw string form of whatever field
// SortNew/SortValue orders by) and the tie-break id into one opaque token —
// 02-FRONTEND-CONTRACT.md §5's keyset pagination, the client only ever
// round-trips this value verbatim as `?cursor=`, never constructs or reads
// it.
func encodeCursor(key string, id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString([]byte(key + "|" + id.String()))
}

// decodeCursor reverses encodeCursor. A malformed cursor is the caller's
// problem to report as invalid_filter, not a panic — this never trusts
// client input.
func decodeCursor(raw string) (key string, id uuid.UUID, err error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("malformed cursor")
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return "", uuid.Nil, fmt.Errorf("malformed cursor")
	}
	id, err = uuid.Parse(parts[1])
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("malformed cursor")
	}
	return parts[0], id, nil
}
