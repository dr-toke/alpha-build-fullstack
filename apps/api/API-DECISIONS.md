# API decisions — internal reasoning, classified

This is `internal/api` + `cmd/server`: the first runnable binary in the
repo. Scoped hard to "get real, correctly-classified cbdstore.in products
onto a live HTTP endpoint" — everything else 08-BUILD-ORDERS.md §5 lists
under M5/M8 is deferred, not silently dropped.

---

## What's built

- `GET /api/products` — list, `brand` filter, `sort=new|value`
  (SPEC'D — 05-API-REFERENCE.md §1), keyset `cursor`/`limit`.
- `GET /api/products/{id}` — detail, `{ moved_to }` for a merged cluster
  (SPEC'D — 02-FRONTEND-CONTRACT.md §4).
- `GET /healthz` — pings the pool, not a static 200 (NEW — not in the
  docs, but 02-FRONTEND-CONTRACT.md §4's "backend not ready -> 503" only
  means something if a health check can actually fail).
- Envelope, error codes, CORS (single allowed origin, hand-written —
  00-CONSTITUTION.md §6's dependency discipline forbids a third-party CORS
  package without an ADR), RequestID/Recoverer/Timeout middleware.

## What's NOT built (deferred, not forgotten)

- `category`/`form`/`route`/`extract`/`profile`/`carrier` filters on
  `/api/products` — `store.ClusterFilter` doesn't carry them yet.
- `?basis=cbd|thc` value-scoped sorting (05-API-REFERENCE.md §1's third
  sort mode) — `ListClusters`' `SortValue` mode uses the cluster's single
  dominant `rank_score`, not a basis-scoped alternate ordering.
- `/api/products/new`, `/api/compare`, `/api/brands`, `/api/brands/{slug}`,
  `/api/states`, `/api/roa`, `/api/aggregators`, `/api/forum/*`,
  `/api/content/export` — none of 05-API-REFERENCE.md §2's endpoints.
- Everything in §3 (community: comments, survey, checkout) and the whole
  admin surface (`06-ADMIN.md`) — needs `internal/auth` (M7), not started.
- Rate limiting, request-ID propagated into response headers/logs,
  `ETag`/`Cache-Control` on catalogue reads (02-FRONTEND-CONTRACT.md §5).
- OpenAPI generation / generated TypeScript (02-FRONTEND-CONTRACT.md §2 —
  "the highest-leverage single item in this repo," still not this one).
- `image` is always `null` in every product payload — no image pipeline
  exists (M6, pending). This is the *correct* value per the nullable-
  never-fabricated convention, not a stub to apologize for.

## Envelope's `page` field — INFERRED deviation

02-FRONTEND-CONTRACT.md §3 shows `{ data, page, limit, total, has_more }`,
but §5 is explicit pagination is keyset-only ("never OFFSET"), which has no
page-number concept. `page` is hardcoded to `1` here purely for wire
compatibility with the documented shape; `next_cursor` (additive, not in
the doc) is what a client actually needs to walk forward. Flagging this
rather than silently inventing page-number semantics that don't exist
underneath.

## `sort=new` default — SPEC'D, but required extending the store layer

`ListClusters` only supported `rank_score`-ordered pagination before this
milestone. 05-API-REFERENCE.md §1 is explicit that `new` is the default
specifically *because* `sort=value` without a `basis` is misleading — "₹/mg-
CBD and ₹/mg-THC are not comparable." Implementing this properly (not
deviating to rank_score-only for expedience) meant adding a second keyset
cursor mode (`first_seen_at`) to `ClusterFilter`/`ListClusters` — done, and
unlike `SortValue`, `SortNew` does not require `rank_score IS NOT NULL`, so
a publishable product with no computable ₹/mg (hemp-seed oil, for example)
still appears in the default catalogue view.

## Compliance wiring — answered a direct question, then acted on it

A prior turn asked "has this filtering implemented or something better
implemented" re: compliance's reason-code logging. At the time,
`compliance.Evaluate` was only exercised by a demo test, never by the real
pipeline. `internal/ingest/promote.go` (built the same session, before this
API layer) now actually calls it: a service listing is filtered before a
cluster is ever created, and never reaches `/api/products`. No
`review_queue` entry is written for the block specifically because that
table's `cluster_id` is `NOT NULL` — a blocked listing never gets a
cluster, so `raw_products` (permanent, undeleted staging) is itself the
record, matching the "a filtered-out row should still exist" principle
without a schema mismatch.
