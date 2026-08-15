# Dr Toke

India's consumer guide to the legal cannabis economy. Not a dispensary, not a
brand — a comparison catalogue. Think PC Mag for Vijaya, not a storefront.

## What this is

A monorepo with two apps that only talk over HTTP/JSON:

- **`apps/api`** — Go backend. Scrapes hemp/CBD stores, classifies and prices
  products, serves the catalogue API.
- **`apps/web`** — SvelteKit 5 + Tailwind 4 frontend. Built by someone else,
  brought in as-is. Not restyled here beyond matching the site's own color
  tokens where it was outright wrong. See `apps/web/README.md` for
  frontend-only dev commands.

Each app builds and runs independently. Nothing shares code across the
HTTP boundary except generated types (not yet wired up — see below).

## Why it's built this way

**Facets, not a category column.** A product's form (oil/capsule/vape/...),
route of administration, extract type, cannabinoid profile, carrier, and
purchasability are five independently-resolved fields, not one string. A
capsule mentioning vaping in a negated sentence ("no need to smoke or vape")
can't get re-categorized by unrelated evidence if the two facets are computed
separately. This is the actual fix for the false-positive rate a single
category column produces — see `docs/10-DECISIONS.md` ADR-002.

**Every facet carries provenance.** `value, source, confidence, evidence,
classifier_version` — not just a bare string. `source` is one of
`override > rule > model > default`, in that precedence, absolute. The rule
engine (currently the only "model" — there's no ML yet) never has final say;
a human override always wins and survives every future re-scrape. This
mechanism sat unused in the schema for a while — see the commit history
around `internal/ingest/promote.go` for what happened when it wasn't
actually wired into the pipeline.

**Money is `int64` paise, never `float64`.** Everywhere in the backend. The
API converts to a plain rupee number at the JSON boundary for the frontend's
benefit — that's a presentation-layer decision, not a backend one.

**Dedup is exact-fingerprint, not fuzzy matching.** `sha256(brand|name|
volume_ml|dominant_cannabinoid_mg)[:16 bytes]`, ported byte-for-byte from a
prior implementation. Fuzzy string matching (Jaro-Winkler) was scaffolded in
that prior version and never actually wired in — dead code, not carried
forward. Two listings of the same product with slightly different scraped
text (extra whitespace, a "-" vs "–") will not merge under this scheme. Known
limitation, not a bug.

**Scrapes never write live.** `scrape → staging → gate → promote → live`.
The gate auto-rejects a batch if the product count drops more than 30% from
the last approved run — one store's markup change should never silently wipe
90% of a source while the site looks fine. Currently the only gate check
implemented; the harvested spec also names a null-field-percentage check, a
₹/mg-distribution-shift check, and a selector-hit-rate check, none built yet.

**Publishing is confidence-gated.** A product needs `purchasable == true`,
form confidence ≥ 0.85, route confidence ≥ 0.90 (when a route applies), and a
real price before it's shown. Below that, it's not hidden data-loss — it just
doesn't publish. "Publish less rather than publish wrong."

**Compliance is a placeholder with two narrow, evidence-based exceptions.**
The full compliance filter (five tiers: hard-block claims, terminology
review, service-listing detection, price-anomaly, unknown-brand) isn't built
— it's explicitly deferred to beta. Two tiers got pulled forward early,
each because a real scrape of the live catalogue surfaced a real problem a
placeholder couldn't leave sitting there: a doctor-consultation booking in
the product catalogue (`service_listing`), and prohibited-claim wording like
"cures cancer" with nothing catching it at all (`hard_block`). See
`apps/api/M2-DECISIONS.md` and ADR-019/ADR-020 in `docs/10-DECISIONS.md` for
why those two and not the rest.

## Current state

The backend pipeline is real and has been run against the actual live
catalogue, not fixtures: scrape → stage → gate → classify → dedup → price →
publish, end to end, ~10,900 real products from cbdstore.in. `GET
/api/products` and `GET /api/products/{id}` work and match the frontend's
actual TypeScript contract (`apps/web/src/lib/api/catalog.ts`), not an
aspirational spec doc that was never reconciled against the real frontend
code — that mismatch was found and fixed once already; see
`apps/api/API-DECISIONS.md` for the full story and don't repeat it.

Not built: auth, community (comments/survey/checkout), the admin panel,
brand/reference endpoints beyond products, image processing (raw scraped
image URLs are hotlinked directly, not proxied through MinIO), OpenAPI/TS
codegen, and a job queue (River is a listed dependency, nothing uses it —
`Promote` runs in-process, synchronously, from whatever calls it). None of
this is silently missing; every gap is named in the relevant
`*-DECISIONS.md` file next to the code it's a gap in.

`cmd/worker` and `cmd/contentpull` are empty directories. `cmd/server` is
the only real binary.

## Setup

Backend needs Postgres 16. No `docker-compose.yml` in this repo (yet) —
run one directly:

```bash
docker run -d --name drtoke-pg -e POSTGRES_PASSWORD=dev -e POSTGRES_DB=drtoke \
  -p 5432:5432 postgres:16

cd apps/api
export DATABASE_URL="postgres://postgres:dev@localhost:5432/drtoke?sslmode=disable"
goose -dir internal/db/migrations postgres "$DATABASE_URL" up
DATABASE_URL="$DATABASE_URL" go run ./cmd/server   # API on :8080
```

The database starts empty. Nothing scrapes on a schedule — there's no job
runner. To populate it, write a small throwaway `cmd/` program that calls
`ingest.StageBatch` → `ingest.DecideGate` → `ingest.Promote` against a real
`internal/store.Store`, same as every integration test in
`internal/ingest/*_test.go` does. `internal/ingest/shopify_live_test.go` and
`internal/ingest/promote_test.go` are the reference for how the whole chain
fits together.

```bash
cd apps/web
pnpm install
VITE_API_URL=http://localhost:8080 pnpm dev   # frontend on :5173
```

Frontend has no backend dependency to start — every page renders "offline"
states with nothing running behind it.

## Testing

```bash
cd apps/api
go vet -all ./...
staticcheck ./...        # go install honnef.co/go/tools/cmd/staticcheck@latest
go test ./...
```

Most of the store/ingest/api test suites spin up a real throwaway Postgres
container per run (`docker` required, tests skip cleanly if it's not
available) — not mocks. `internal/ingest/*_live_test.go` hit the real
cbdstore.in network and are gated behind `INGEST_LIVE_TEST=1` specifically
so CI doesn't depend on a third party's uptime.

`testdata/golden/*.json` is the classifier's regression suite — one fixture
per real bug already found and fixed, run by
`internal/resolve/golden_test.go`. Every classifier change should pass the
existing set before adding a new one; every new one should be a real case,
not an invented edge case (check the `source` field in the existing
fixtures — most cite the exact live listing that broke).

## Deploy

`docs/09-OPS.md §5a` has the actual steps (Railway, Render, or Fly.io). No
Dockerfile — the backend is one static Go binary, which is most of what
Docker would buy you anyway; `apps/api/nixpacks.toml` gives the platform's
native Go buildpack an explicit build target since this is a monorepo.

## Docs

Read `docs/00-CONSTITUTION.md` first — it's short, and everything else
assumes you have. After that, read whichever of these applies to what
you're touching, not all of them in order:

| Doc | Covers |
|---|---|
| `01-ARCHITECTURE.md` | System design and the reasoning behind it |
| `02-FRONTEND-CONTRACT.md` | What the backend owes the frontend, and why |
| `03-DOMAIN-MODEL.md` | Schema, facets, precedence, pricing |
| `04-PIPELINE.md` | scrape → normalise → compliance → dedup → bestdeal |
| `05-API-REFERENCE.md` | The aspirational API spec — **cross-check against `apps/api/API-DECISIONS.md` before trusting it**, it drifted from the real frontend once already |
| `06-ADMIN.md` | Admin panel design (not built yet) |
| `07-CONTENT-CMS.md` | Editorial content pipeline |
| `08-BUILD-ORDERS.md` | File-by-file build plan, useful for delegating work |
| `09-OPS.md` | Running, deploying, debugging |
| `10-DECISIONS.md` | Every ADR. "Why is it like this" starts here |
| `11-HARVEST.md` | What got ported from the prior implementation and why |

`apps/api/SYMBOLS.md` is a hand-maintained ledger of every exported Go
signature, grouped by package. `apps/api/*-DECISIONS.md` (one per
milestone) is where you actually find out what's real versus what the docs
above describe as intent — read the milestone's decisions file before
assuming a doc describes shipped behavior.

## Legal

Educational content only. Not a licensed medical or legal service. Private
repo — see `docs/00-CONSTITUTION.md` before publishing anything from it
anywhere.
