package api

import (
	"context"
	"net/http"
	"time"

	"github.com/dr-toke/api/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter wires the routes this build actually serves —
// GET /api/products, GET /api/products/{id}, GET /healthz. 08-BUILD-ORDERS.md
// §5's full M8 route list (compare, brands, reference, community, admin) is
// not built; see API-DECISIONS.md. allowedOrigin is
// 02-FRONTEND-CONTRACT.md §6's "connect-src lists exactly one API host" from
// the OTHER direction — the API, symmetrically, should only ever answer CORS
// preflight for the one frontend origin it's actually serving, not "*".
func NewRouter(st *store.Store, allowedOrigin string) http.Handler {
	h := &Handlers{Store: st}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(corsMiddleware(allowedOrigin))

	r.Get("/healthz", healthHandler(st))

	r.Route("/api/products", func(r chi.Router) {
		r.Get("/", h.ListProducts)
		r.Get("/{id}", h.GetProduct)
	})

	return r
}

// corsMiddleware is hand-written, not github.com/go-chi/cors — 00-CONSTITUTION.md
// §6's dependency discipline: "Chi... anything else requires an ADR," and a
// single-origin allow/deny check plus a fixed header set doesn't need a
// third-party library.
func corsMiddleware(allowedOrigin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowedOrigin != "" && origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				// 02-FRONTEND-CONTRACT.md §6: without exposing these, the
				// client's fetch() cannot read them cross-origin even though
				// the server sent them.
				w.Header().Set("Access-Control-Expose-Headers", "ETag, Retry-After, X-Total-Count")
			}
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// healthHandler is a real readiness check, not a static 200 — 02-FRONTEND-CONTRACT.md
// §4: "Backend not ready / degraded -> 503 + Retry-After" is meant to be
// meaningfully true, and a health endpoint that always says yes can't back
// that up. Pings the pool with a short timeout rather than trusting that a
// live process means a live database connection.
func healthHandler(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := st.Pool.Ping(ctx); err != nil {
			writeUnavailable(w, "database unreachable")
			return
		}
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
