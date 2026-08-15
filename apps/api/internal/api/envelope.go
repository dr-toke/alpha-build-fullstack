// Package api is the public HTTP surface — 08-BUILD-ORDERS.md §5's M8/M5
// slice, deliberately scoped down for the first runnable binary
// (cmd/server): GET /api/products (list) and GET /api/products/{id}
// (detail, with the moved_to contract for a merged cluster). Everything
// else that section lists — compare.go, brands.go, reference.go, auth,
// community, rate limiting/request-ID middleware, OpenAPI codegen — is NOT
// built here; see API-DECISIONS.md.
//
// Response shapes match apps/web/src/lib/api/catalog.ts's ApiProduct /
// ProductListResponse types EXACTLY — those are the real, already-running
// frontend contract (ported verbatim from the trusted collaborator's
// design), not 05-API-REFERENCE.md's aspirational envelope
// ({ data, page, limit, total, has_more }), which was never reconciled
// against the frontend code already in the repo. See API-DECISIONS.md.
package api

import (
	"encoding/json"
	"net/http"
)

// WriteJSON writes v as the response body with the given status and the
// correct content type — the one place that decides those two things, so
// every handler produces byte-identical headers.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
