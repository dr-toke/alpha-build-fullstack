// Package api is the public HTTP surface — 08-BUILD-ORDERS.md §5's M8/M5
// slice, deliberately scoped down for the first runnable binary
// (cmd/server): GET /api/products (list, new-sort default per
// 05-API-REFERENCE.md §1) and GET /api/products/{id} (detail, with the
// moved_to contract for a merged cluster). Everything else that section
// lists — compare.go, brands.go, reference.go, auth, community, rate
// limiting/request-ID middleware, OpenAPI codegen — is NOT built here; see
// API-DECISIONS.md.
package api

import (
	"encoding/json"
	"net/http"
)

// Envelope is 02-FRONTEND-CONTRACT.md §3's list-response shape exactly:
// `{ data, page, limit, total, has_more }`. Page is always 1 here — that
// field predates this build's keyset-only pagination (05-API-REFERENCE.md
// §1: "cursor, limit" are the real list params, no page number exists in a
// keyset scheme) and is kept only for wire compatibility with the
// documented shape; NextCursor is what a client actually needs to walk
// forward and is additive, not a doc-specified field.
type Envelope struct {
	Data       any    `json:"data"`
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	Total      int    `json:"total"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// WriteJSON writes v as the response body with the given status and the
// correct content type — the one place that decides those two things, so
// every handler produces byte-identical headers.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
