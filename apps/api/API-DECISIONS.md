# API decisions — internal reasoning, classified

This is `internal/api` + `cmd/server`: the first runnable binary in the
repo. Scoped hard to "get real, correctly-classified cbdstore.in products
onto a live HTTP endpoint" — everything else 08-BUILD-ORDERS.md §5 lists
under M5/M8 is deferred, not silently dropped.

---

## The reconciliation this doc exists to record

The first version of this milestone was built strictly from
`05-API-REFERENCE.md` / `02-FRONTEND-CONTRACT.md §3`'s documented envelope
(`{ data, page, limit, total, has_more }`, cursor pagination, paise prices,
`{ error: { code, message } }`). It built cleanly, tested green, and
rendered an **empty product grid** the first time it was actually run
against the real frontend — `apps/web/src/lib/api/catalog.ts`
(`ApiProduct`/`ProductListResponse`), ported verbatim from the trusted
collaborator's real design, was never reconciled against those docs. The
docs described an aspirational contract; the frontend code already in the
repo is the real one. This file records the second pass, built against
`catalog.ts` + `CatalogGrid.svelte` + `client.ts` + `product/+page.svelte`
directly, verified by actually running both servers and looking at the
rendered page (`docs/09-OPS.md`-style local dev, plus a live-scraped
cbdstore.in dataset).

**Lesson for anything built from these docs going forward:** treat
`0X-*.md` as the design intent, but when frontend code already exists for a
surface, that code is the ground truth for wire shape. Reconcile before
building, not after.

## What's built

- `GET /api/products` — `{ products, total, page, per_page }`, matching
  `ProductListResponse` exactly. Filters: `category` (reverse-mapped from
  `resolve.LegacyCategory`'s exact form/route/concentration_type branches
  into SQL), `extract` (direct facet join), `brand` (slug), `basis`
  (`cbd`/`thc`, scopes `sort=value` to that cannabinoid's own ₹/mg column),
  `verified` (brand join), `sort` (`value` default — matching
  `CatalogGrid.svelte`'s own client-side default, not `05-API-REFERENCE.md`'s
  documented `new` default; see below), `page`/`limit` (1-based OFFSET).
- `GET /api/products/{id}` — `{ product: {...} }`, `{ moved_to }` for a
  merged cluster (still SPEC'D and honored, even though the shipped
  frontend doesn't currently branch on it — see products.go's doc comment).
- `GET /healthz` — pings the pool, not a static 200.
- CORS (single allowed origin, hand-written — no third-party CORS package
  per `00-CONSTITUTION.md §6`), RequestID/Recoverer/Timeout middleware.

## Deviations from the docs — each one judged, not defaulted

Presented to the project owner as a comparison before implementing (not
just picked silently): three real technical judgment calls, decided as
follows.

**Pagination: page-number/OFFSET, not keyset cursor.** Keyset is the
objectively more correct design at scale and is what `02-FRONTEND-CONTRACT.md
§5` mandates ("never OFFSET" — avoids duplicate/skipped rows when a scrape
lands mid-browse). But `CatalogGrid.svelte` only ever moves the page number
±1, never jumps arbitrarily, and the catalogue is ~10k rows — OFFSET cost
is irrelevant at that size. Decision: OFFSET now, matching what's shipped;
revisit if the catalogue reaches hundreds of thousands of rows or
concurrent scrape-during-browse traffic becomes real.

**Errors: `{ error: "<string>" }`, not `{ error: { code, message } }`.**
The structured form is better practice long-term (stable machine codes vs.
message-string matching) and is what the docs specify. But the shipped
`ApiError` class (`client.ts`) only reads `body?.error` as a string — the
object form would set the error message to a stringified object and
visibly break the UI. Decision: string now (what won't break); the doc's
richer form is a cheap two-line addition to `client.ts` whenever someone
wants stable-code branching, not urgent.

**Money: paise internally, rupees at the API boundary.** No real conflict —
`00-CONSTITUTION.md §5`'s "money is int64 paise, never float" governs
internal computation (and still does, everywhere in `internal/resolve`/
`internal/store`/`internal/ingest`); `best_price_inr`/`price_inr` are a
single, one-way `paise / 100.0` conversion at JSON serialization, with
nothing downstream doing further arithmetic on the result. Not a
Constitution violation, a presentation-layer decision at the wire boundary
for this specific consumer.

Everything else that differs from the docs (`products` vs `data`, nested
`cannabinoids{}` vs a flat/facets-map shape, `best_listing`/
`other_listings` vs one flat `listings[]`) is pure naming/shape with no
technical tradeoff — matched to the frontend because that's what's real,
not worth re-litigating.

## Two correctness fixes made along the way

- **`brand` is never `null`.** `ProductCard.svelte`/`product/+page.svelte`
  read `p.brand.verified` with no null guard — `ApiBrandSummary` is
  non-optional in `catalog.ts`. A cluster with no matched `brands` row (an
  unmapped vendor, `internal/ingest/shopify.go`'s `slugifyVendor` fallback)
  now synthesizes a brand summary from the listing's raw scraped brand
  text instead of sending `null`, which would have thrown at render time.
- **`best_price_inr` computed from the actual listing rows at read time**,
  not trusted from `cluster.best_price_paise` — the latter is a known
  imprecision (`M5`'s ingest decisions: it reflects whichever listing was
  *last processed* into a cluster, not necessarily the cheapest, when
  multiple listings share a fingerprint). `buildApiProduct` now derives it
  from `ListingsForCluster`'s own `in_stock DESC, price_paise ASC` order,
  which is actually correct.
- **`listing_id` is the listing's own UUID**, not its SKU —
  `BuyButton.svelte` POSTs `{ listing_id }` to the not-yet-built
  `/api/checkout/initiate`, which needs `product_listings.id` to satisfy
  `click_events.listing_id`'s FK. Checkout isn't built this milestone, but
  the frontend needs zero changes when it is.

## What's NOT built (deferred, not forgotten)

- `/api/products/new`, `/api/compare`, `/api/brands`, `/api/brands/{slug}`,
  `/api/states`, `/api/roa`, `/api/aggregators`, `/api/forum/*`,
  `/api/content/export`, `/api/checkout/*`.
- Everything in community (comments, survey) and the whole admin surface
  (`06-ADMIN.md`) — needs `internal/auth` (M7), not started.
- Rate limiting, request-ID propagated into response headers/logs,
  `ETag`/`Cache-Control` on catalogue reads.
- OpenAPI generation / generated TypeScript (`02-FRONTEND-CONTRACT.md §2` —
  "the highest-leverage single item in this repo," still not this one; now
  more valuable than ever given this milestone's whole lesson was a
  hand-maintained contract silently drifting).
- `image_url` is always `null` — no image pipeline exists (M6, pending).
  Correct per the nullable-never-fabricated convention, not a stub.
- `slug` on `ApiProduct` is populated (kebab-cased from the name) but not
  load-bearing — nothing in the shipped frontend actually routes on it
  (routing is by `id`).

## Compliance wiring — answered a direct question, then acted on it

A prior turn asked "has this filtering implemented or something better
implemented" re: compliance's reason-code logging. At the time,
`compliance.Evaluate` was only exercised by a demo test, never by the real
pipeline. `internal/ingest/promote.go` now actually calls it: a service
listing is filtered before a cluster is ever created, and never reaches
`/api/products`. No `review_queue` entry is written for the block
specifically because that table's `cluster_id` is `NOT NULL` — a blocked
listing never gets a cluster, so `raw_products` (permanent, undeleted
staging) is itself the record, matching the "a filtered-out row should
still exist" principle without a schema mismatch.
