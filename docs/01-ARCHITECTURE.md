# 01 — Architecture

---

## 0. The one principle

**Every decision the system makes must be reproducible, attributable, and
permanently correctable by a human.**

Reproducible: same raw input + same classifier version → same output.
Attributable: for any field on any product page we can say which rule or which
person put it there, and on what evidence.
Permanently correctable: a human override survives every re-scrape, every schema
change, and every classifier version, forever.

The facet model, the admin review queue, the golden-set CI gate, and the
promotion gate are all this one idea wearing different clothes.

---

## 1. Where we actually are

**A prior implementation exists and is being replaced.** It reached alpha —
Go/Chi/pgx/River/Goose, ~29 migrations, seven pipeline stages, ~4,300 scraped
products — but is alpha across all layers: structure, schema, and pipeline
reliability. It is not the foundation.

What we take from it is **knowledge, not code**: scraper selectors for ~14
stores, the cannabinoid extraction priority and its regexes, category keyword
sets and their hard-won exclusions, the compliance vocabulary tiers, the dedup
fingerprint, and a fixture for every bug already paid for. All of it is
extracted as data files before anything is deleted — see `11-HARVEST.md`, which
must be complete before M0 starts.

What has changed around it:

- The frontend is no longer Next.js. It is **SvelteKit 5 + Tailwind 4, fully
  static**, living as `apps/web` in this monorepo (`ADR-018`; originally its
  own repo, `dr-toke/skeleton`, brought in as-is), live at
  `toke-v02.netlify.app`. Design is done and is not ours to restyle casually.
- `packages/data/*.ts` (brands, states, roa) was Phase-0 bootstrap. The DB is
  the source of truth.
- Editorial copy currently lives in the frontend as `content.ts` files. It moves
  into the backend as a CMS (§5).

Scraped data is re-scrapeable, so the old database is not a migration
dependency — a snapshot is kept as insurance and as a dev fixture only.

See `10-DECISIONS.md#adr-001`.

---

## 2. System map

```
  ~14 STORES                    BACKEND (Go)                        DELIVERY
  boheco, itshemp,     ┌──────────────────────────────────┐
  cbdstore, cannameds… │                                  │
      │                │  INGEST                          │
      │  scrape        │  scrape → staging → diff → GATE  │
      └───────────────►│                    │             │
                       │                    ▼             │
                       │  NORMALISE / RESOLVE (per facet) │
                       │  rules → evidence → confidence   │
                       │  model ──► review queue          │
                       │  overrides ──► always win        │
                       │                    │             │
                       │  COMPLIANCE (hard-block | review)│
                       │  DEDUP (durable cluster UUID)    │
                       │  BESTDEAL (₹/mg per cannabinoid) │
                       │  IMAGES (MinIO)                  │
                       │                    │             │
                       │                    ▼             │
                       │       ┌────────────────────┐     │
                       │       │  PostgreSQL 16     │     │
                       │       │  catalogue +       │     │
                       │       │  content_docs      │     │
                       │       └─────────┬──────────┘     │
                       │                 │                │
        ADMIN          │     ┌───────────┴──────────┐     │
     (HTMX, Tor/      ◄┼─────┤ PLANE A    PLANE B   │     │
      localhost only)  │     │ content    live API  │     │
                       │     └────┬───────────┬─────┘     │
                       └──────────┼───────────┼───────────┘
                                  │           │
                    build time    │           │  runtime
                   (content:pull) │           │  (Remote<T>)
                                  ▼           ▼
                       ┌──────────────────────────────────┐
                       │   SvelteKit static build         │
                       │   prerendered HTML + /data/*.json│
                       │   → Netlify (clearnet)           │
                       │   → onion / eepsite (community)  │
                       └──────────────────────────────────┘
```

---

## 3. Two content planes

The frontend is **fully static**: no SSR, no server runtime, no API routes.
`Remote<T>` fetches client-side. That constraint forces a split.

### Plane A — build time (editorial)

`/`, `/history`, `/science`, `/legality`, `/parcha`, blog & forum post bodies,
state legal notes, ROA guide, aggregator directory, FAQ, glossary.

Backend is the CMS and source of truth; a sync step pulls it at build; it ships
as prerendered HTML inside the bundle.

Why not runtime: content-less HTML means no SEO, nothing readable with JS
disabled (fatal on Tor at Safest), a loading flash on every essay, and a
reference site whose *reference material* disappears when the API hiccups.

### Plane B — runtime (data)

`/products`, `/compare`, `/brands`, product detail, comments, survey, account.
Prices move; these pages should show today's number.

### The hybrid

| Page | Prerendered (A) | Hydrated (B) |
|---|---|---|
| Blog / forum post | title, body, metadata | comment thread |
| `/products` | build-time catalogue **snapshot** | live prices, filters, pagination |
| `/product?id=` | `/data/products/{id}.json` | live price, comments, stock |
| `/legality` | full state grid + NDPS text | — |
| `/brands` | verified brand list snapshot | live product counts |

The snapshot changes what "service offline" means: not a blank page but *stale
but complete* — catalogue still renders, ranked and readable, with a quiet
"prices last updated {date}" note. `Remote<T>` keeps all three branches; the
error branch just gets much better fallback content.

---

## 4. The facet refactor

The false-positive rate is a **schema** defect, not a model-quality defect. One
`category` column plus a `categories[]` array forces unrelated signals to
collide: "no need to smoke or vape" broke classification because *form* and
*route of administration* were decided from the same evidence pool.

Replace with orthogonal facets, each resolved independently. Full schema in
`03-DOMAIN-MODEL.md §2`. Migration path (additive, never a big-bang cutover):

1. Add `product_facets` + `facet_overrides` alongside the existing columns.
2. Backfill facets from the current `category` / `categories[]` / `extract_type`.
3. Dual-write: the pipeline writes both for one release.
4. API reads facets, keeps emitting the legacy `category` / `categories` fields
   so the frontend never breaks.
5. Drop the legacy write path once the golden set is green.

**Precedence is absolute:** `override > rule > model > default`, enforced in one
function (`resolve.Facet`) and nowhere else. If a second code path can write a
facet, this design is already broken.

---

## 5. Content system

Editorial copy moves from frontend `content.ts` files into `content_docs` /
`content_revisions` (append-only revisions, publish = pointer flip). A CLI pulls
published content at build time and writes generated `.ts` / `.json` into the
frontend checkout; those generated files are **committed**. Detail in
`07-CONTENT-CMS.md`.

This is the change that alters the project's nature: the moment copy lives in
the backend, Dr Toke stops being a designer's repo with an API bolted on and
becomes a publication with a CMS.

---

## 6. Repo layout

**Monorepo** (`ADR-018`): `docs/` at root, shared by both apps; everything
backend-specific nests under `apps/api/` so it stays self-contained.

```
drtoke/
├── docs/               this doc set — the shared contract, read by both apps
├── apps/
│   ├── web/            SvelteKit 5 + Tailwind 4, fully static — brought in
│   │                   as-is from the original `dr-toke/skeleton` repo
│   └── api/
│       ├── cmd/
│       │   ├── server/        API + admin binary
│       │   ├── worker/        River job runner
│       │   └── contentpull/   build-time export CLI
│       ├── internal/
│       │   ├── api/           handlers, envelope, errors, OpenAPI annotations
│       │   ├── admin/         HTMX handlers + templates
│       │   ├── content/       docs, revisions, markdown validate + render
│       │   ├── resolve/       rule engine: cannabinoids, facets, evidence,
│       │   │                  precedence  ← the single writer
│       │   ├── compliance/    hard-block vs review tiers
│       │   ├── ingest/        selector-driven adapters, staging, gate, dedup
│       │   ├── jobs/          River job definitions
│       │   ├── media/         fetch, transcode, blurhash, proxy
│       │   ├── auth/          public (pseudonymous) + admin (TOTP)
│       │   ├── db/migrations/ goose
│       │   └── config/        env-driven config
│       ├── harvest/
│       │   ├── rules/         cannabinoids, categories, compliance, dedup  (JSON)
│       │   ├── scrapers/      one YAML selector spec per store
│       │   └── reference/     states, roa, aggregators, brands
│       ├── testdata/golden/   auto-appended fixtures
│       ├── openapi/           generated spec → apps/web/src/lib/api/generated/
│       └── SYMBOLS.md
```

**Boundary discipline replaces repo separation.** Nothing in `apps/api` imports
from `apps/web` or vice versa — the only thing that crosses is generated
TypeScript (`apps/api/openapi` → `apps/web/src/lib/api/generated/`) and, at
build time, `content:pull`'s generated `.ts`/`.json` files (`07-CONTENT-CMS.md`).
A PR touching both directories in the same commit should still be reviewable
as two independent diffs.

**Modularity rules.** SQL lives only in the store layer. Classification lives
only in `resolve`. HTTP lives only in `api`. A file reaching across two of those
is a refactor, not a feature.

**And the rule that makes the rewrite worth it:** no classification pattern,
keyword list, or CSS selector is ever hardcoded in Go. They load from
`harvest/`. A rule change is a reviewable data diff with no recompile — and it
is what lets the review queue append to the rules automatically.

---

## 7. Build order

| # | Milestone | Unblocks |
|---|---|---|
| 0 | **Harvest** knowledge from the old repo (`11-HARVEST.md`) | everything; do this first |
| 1 | Spine: migrations + `domain/types.go`, `go build` green | everything below |
| 2 | Facet schema + provenance + overrides (additive migration) | the real fix |
| 3 | Rules engine with negation-aware evidence; model demoted to proposals | permanent end to false positives |
| 4 | Golden set + CI gate | fearless iteration |
| 5 | `content_docs` + `/api/content/export` + `content:pull` | **writers, immediately** |
| 6 | Admin: content editor → review queue → dry-run diff | data-quality workflow |
| 7 | Staging + promotion gate + source health | protects against silent data loss |
| 8 | OpenAPI → generated TS; envelope, errors, keyset pagination | contract safety |
| 9 | Image pipeline + `/media` proxy + catalogue snapshot | offline-capable frontend |
| 10 | Moderation, click analytics, audit log | launch readiness |

Milestones 2–4 are roughly a focused week. Because the rules arrive as
harvested data rather than retyped code, the category problem is retired at the
same time the engine is written — not patched a fourth time.

---

## 8. Open decisions

1. ~~Same apex domain for API and site?~~ **Resolved — `drtoke.in/api/*`,
   `ADR-014`.** Same domain does not imply shared trust; admin isolation is
   reaffirmed, not loosened.
2. **Generated files committed to the frontend repo?** Needs the frontend
   owner's buy-in. Fallback: publish as a package the frontend depends on.
   Now also covers generated image references (`ADR-017`).
3. **Onion/eepsite deploy target** — deferred to stage 2, per project owner
   (2026-08-11).
4. ~~Licence.~~ **Direction resolved — creator-owned, `ADR-015`.** Both repos
   and all editorial content are copyright to their individual creators, not
   assigned to Dr Toke. Actual license text/SPDX choice is stage-2 execution.
5. **Image provenance.** `static/images/` in the frontend holds placeholders
   with unverified provenance. Asset *ownership and editability* is resolved —
   images are CMS-managed (`ADR-017`) — but clearing the current placeholders'
   rights is deferred to stage 2, per project owner (2026-08-11).
