# 08 — Build Orders

> How to build this with a weak model (Haiku, Qwen, DeepSeek, GLM), **one file
> per chat window**, no agentic tooling, no cross-file reasoning required from
> the model.
>
> The model is a typist with good syntax, not an architect. Every decision is
> made in the other documents. The model turns a precise spec into valid Go.
>
> **Prerequisite: `11-HARVEST.md` is complete.** Do not start M1 until every box
> in its §3 is ticked.
>
> **All file paths below are relative to `apps/api/`** (`ADR-018` — this repo
> is now a monorepo with the Go backend at `apps/api/` rather than the repo
> root). `go.mod`, `internal/`, `cmd/`, `harvest/`, `testdata/` all nest under
> there.

---

## 1. Why greenfield is the easier mode here

Counter-intuitively, a clean rewrite is *better* for weak-model delegation than
a refactor. Refactoring requires the model to understand code it can't see;
greenfield only requires it to follow a spec.

Two design choices make this work:

- **Rules live in `harvest/rules/*.json`, not in Go.** The model writes a *rule
  engine*, not the rules. It never has to reproduce a regex it can't verify — it
  writes the loader and the matcher, and the hard-won patterns come from data
  files you transcribed by hand.
- **Scrapers are `scrapers/*.yaml` + one generic adapter per platform.** Instead
  of 14 hand-written scrapers, there are three adapters (Shopify, WooCommerce,
  custom) driven by selector files. That's ~11 fewer files a model can get wrong.

---

## 2. Rules of engagement

Each rule maps to a specific way weak models fail.

| Failure | Rule |
|---|---|
| Invents types that don't exist | Never order a file whose dependencies aren't written. Build leaf-first. |
| Silently edits other files | Every prompt ends with "Output ONLY the contents of this one file." |
| Drifts across a long chat | **One file = one fresh chat.** Never reuse a window. Never ask for two files. |
| Hallucinates a signature | Paste *signatures* of dependencies, never whole files (§5). |
| Adds libraries | The context block lists allowed imports. Anything else is a rejection. |
| Explains instead of coding | "No commentary, no markdown fences, no explanation." |
| Guesses at ambiguity | "If ambiguous or a symbol is missing, output `NEED: <what>` and nothing else." |
| Retypes a regex from memory | **Never put a harvested pattern in a prompt.** The code loads it from JSON. |

---

## 3. Never delegate these

| Never | Why |
|---|---|
| `go.mod` | Everything depends on the pinned set |
| `internal/db/migrations/*.sql` | A wrong migration is a data incident |
| `internal/domain/types.go` | Every downstream order references it by symbol |
| `internal/api/router.go` | Wiring errors are silent and expensive |
| `internal/admin/templates/*.html` | Broken markup produces no compile error |
| `harvest/**` | Transcribed by hand in `11-HARVEST.md`; a model paraphrasing a regex is a silent regression |

Write these yourself (or from a strong model, once), commit, treat as read-only.

---

## 4. The universal context block

Paste at the top of **every** chat, unchanged. Short on purpose — a weak model
with 400 words of context follows instructions better than one with 4,000.

```
You are writing one file of a Go 1.25 backend. Follow the spec exactly.

PROJECT: Dr Toke — a cannabis product catalogue + editorial API for India.
Go 1.25, PostgreSQL 16, Chi router, pgx/v5, River (jobs), Goose (migrations).

ALLOWED IMPORTS: standard library, github.com/go-chi/chi/v5,
github.com/jackc/pgx/v5, github.com/jackc/pgx/v5/pgxpool,
github.com/google/uuid, github.com/riverqueue/river,
golang.org/x/crypto/argon2, gopkg.in/yaml.v3.
Anything else is forbidden. Do not add dependencies.

CONVENTIONS:
- Package comment on every file. Exported symbols have doc comments.
- Errors wrapped with fmt.Errorf("<context>: %w", err). Never panic outside main.
- No global mutable state. Dependencies passed in via struct fields.
- Money is int64 paise. Never float.
- Timestamps are time.Time, serialised RFC3339 UTC.
- Unknown numeric values are nil pointers, NEVER zero.
- Parameterised SQL only ($1, $2). Never string concatenation.
- SQL lives only in the store layer. No SQL anywhere else.
- Classification rules are LOADED FROM DATA FILES, never hardcoded.
- Table-driven tests.

OUTPUT RULES:
- Output ONLY the complete contents of the one file requested.
- No markdown code fences. No commentary before or after. No explanation.
- Do not create, modify, or mention any other file.
- If the spec is ambiguous, or needs a symbol not listed in AVAILABLE SYMBOLS,
  output exactly one line: NEED: <what is missing> — and nothing else.
```

---

## 5. The work-order template

```
FILE: <exact path>
PACKAGE: <package name>

PURPOSE:
<2–3 sentences. What this file is responsible for, and what it is NOT.>

AVAILABLE SYMBOLS (already exist, do not redefine):
<signatures only — from SYMBOLS.md>

MUST EXPORT:
<exact signatures>

BEHAVIOUR:
1. <numbered, imperative, one behaviour per line>
2. <exact error cases and what to return>
3. <exact SQL, if this is a store file>

DO NOT:
- <the specific wrong thing a model would do here>

ACCEPTANCE:
  go build ./... && go vet ./... && go test ./...
```

`AVAILABLE SYMBOLS` is the whole trick. Signatures only — never whole dependency
files. Keeps context small, makes forward references impossible, stops the model
"helpfully" rewriting something it can see.

**Keep `SYMBOLS.md` current.** After each merged file, append its exported
signatures. It is the paste-source for every later order.

---

## 6. The verification loop

Run after **every** file, before opening the next chat.

```bash
gofmt -l .                 # must print nothing
go build ./...             # must be silent
go vet ./...               # must be silent
go test ./...              # green
```

On failure: **do not** paste the error into the same chat and iterate — that is
how a window drifts. Open a **fresh** chat, paste the context block, paste the
original order, and append:

```
The previous attempt produced this compile error:
<error>
Output the corrected complete file.
```

Two failures on the same file means the **spec** is wrong, not the model.

Commit per file: `git commit -m "feat(resolve): rules_form.go"`. One file per
commit, so a bad file is one `git revert`.

---

## 7. File manifest

Strict dependency order. Never build a file before everything above it exists.
**M1–M5 is the launchable API.** Everything after is improvement.

### M0 — Spine (manual, you)

| # | File |
|---|---|
| 0.1 | `go.mod` |
| 0.2 | `internal/db/migrations/001_sources_listings.sql` |
| 0.3 | `internal/db/migrations/002_clusters_merges.sql` |
| 0.4 | `internal/db/migrations/003_facets_overrides.sql` |
| 0.5 | `internal/db/migrations/004_brands_reference.sql` |
| 0.6 | `internal/db/migrations/005_content.sql` |
| 0.7 | `internal/db/migrations/006_community.sql` |
| 0.8 | `internal/db/migrations/007_clicks_survey_queue.sql` |
| 0.9 | `internal/domain/types.go` — every struct + enum, one file |
| 0.10 | `internal/domain/errors.go` — sentinel errors |

Schema comes from `03-DOMAIN-MODEL.md`. Do not improvise it.

### M1 — Rule engine (pure logic, no DB, no HTTP — best odds for a weak model)

| # | File | Exports |
|---|---|---|
| 1.1 | `internal/resolve/text.go` | `Normalize`, `Tokens`, `NegationWindows` |
| 1.2 | `internal/resolve/evidence.go` | `Evidence`, `Span`, `Merge` |
| 1.3 | `internal/resolve/ruleset.go` | `LoadRuleSet(dir)` — reads `harvest/rules/*.json` |
| 1.4 | `internal/resolve/ruleset_test.go` | — |
| 1.5 | `internal/resolve/match.go` | `MatchWordBoundary`, `ApplyNegation` |
| 1.6 | `internal/resolve/cannabinoids.go` | `ExtractCannabinoids` — priority-ordered, no early return |
| 1.7 | `internal/resolve/cannabinoids_test.go` | CannEx Strong Plus fixture |
| 1.8 | `internal/resolve/facets.go` | `ResolveForm`, `ResolveRoute`, `ResolveExtract`, `ResolveCarrier`, `ResolvePurchasable` |
| 1.9 | `internal/resolve/facets_test.go` | — |
| 1.10 | `internal/resolve/precedence.go` | `Resolve` — **the single writer** |
| 1.11 | `internal/resolve/legacy.go` | facets → `category` / `categories[]` |
| 1.12 | `internal/resolve/value.go` | `PerMg`, `ValueTier`, `RankScore` |
| 1.13 | `internal/resolve/golden_test.go` | loads `testdata/golden/*.json` |

Every order in M1 carries the invariants from `04-PIPELINE.md §3–4` in its
`BEHAVIOUR` and `DO NOT` sections — word-boundary matching, pet/apparel
exclusive, form over extract, negation windows, percentages as full decimals,
never zero for unknown.

> M1 is where the false-positive problem actually dies, and it's the safest
> phase to delegate: pure functions, no I/O, every file testable in isolation.
> Do it first and do it carefully.

### M2 — Compliance

| # | File | Exports |
|---|---|---|
| 2.1 | `internal/compliance/filter.go` | `Evaluate` → `pass \| review(reason) \| block(reason)` |
| 2.2 | `internal/compliance/filter_test.go` | — |

Two tiers, loaded from `harvest/rules/compliance.json`. See `04-PIPELINE.md §5`.

### M3 — Storage

| # | File | Exports |
|---|---|---|
| 3.1 | `internal/store/store.go` | `New`, `Store`, `Close` |
| 3.2 | `internal/store/listings.go` | `UpsertListing`, `ListingsForCluster` |
| 3.3 | `internal/store/clusters.go` | `ClusterByID`, `ListClusters`, `Merge` |
| 3.4 | `internal/store/facets.go` | `FacetsFor`, `UpsertFacets` |
| 3.5 | `internal/store/overrides.go` | `OverridesFor`, `SetOverride` |
| 3.6 | `internal/store/brands.go` | `ListBrands`, `BrandBySlug`, `Approve` |
| 3.7 | `internal/store/reference.go` | `ListStates`, `ListROA`, `ListAggregators` |
| 3.8 | `internal/store/queue.go` | `Enqueue`, `ListQueue`, `Resolve` |
| 3.9 | `internal/store/content.go` | `PublishedDocs`, `DocBySlug`, `NewRevision`, `Publish` |
| 3.10 | `internal/store/community.go` | accounts, refresh tokens, comments |
| 3.11 | `internal/store/golden.go` | `AppendFixture` |

Give each store file the **exact SQL** in the order. Do not let a model write
queries — the single highest-risk thing you could delegate.

### M4 — Ingest

| # | File | Exports |
|---|---|---|
| 4.1 | `internal/ingest/spec.go` | `LoadScraperSpec` — reads `harvest/scrapers/*.yaml` |
| 4.2 | `internal/ingest/adapter.go` | `Adapter` interface, `RawListing` |
| 4.3 | `internal/ingest/shopify.go` | selector-driven |
| 4.4 | `internal/ingest/woocommerce.go` | selector-driven |
| 4.5 | `internal/ingest/staging.go` | `StageBatch`, `LoadBatch` |
| 4.6 | `internal/ingest/gate.go` | `Evaluate` — the promotion gate |
| 4.7 | `internal/ingest/gate_test.go` | — |
| 4.8 | `internal/ingest/dedup.go` | `Fingerprint`, `AssignCluster` |
| 4.9 | `internal/ingest/dedup_test.go` | — |

Three adapters, not fourteen scrapers. A new store is a YAML file.

### M5 — Jobs + API (ship after this)

| # | File | Exports |
|---|---|---|
| 5.1 | `internal/jobs/jobs.go` | River registration |
| 5.2 | `internal/jobs/scrape.go` | → staging |
| 5.3 | `internal/jobs/normalise.go` | → resolve |
| 5.4 | `internal/jobs/compliance.go` | enqueue rules from `04-PIPELINE.md §1` |
| 5.5 | `internal/jobs/dedup.go` | **refresh all derived fields on fingerprint match** |
| 5.6 | `internal/jobs/bestdeal.go` | per-cannabinoid ₹/mg, tier, rank |
| 5.7 | `internal/jobs/images.go` | MinIO |
| 5.8 | `internal/api/envelope.go` | `Envelope`, `WriteJSON` |
| 5.9 | `internal/api/errors.go` | `WriteError`, codes |
| 5.10 | `internal/api/middleware.go` | request ID, rate limit, CORS, recoverer |
| 5.11 | `internal/api/params.go` | filter + keyset parsing |
| 5.12 | `internal/api/products.go` | list, detail, `?basis=` |
| 5.13 | `internal/api/compare.go` | |
| 5.14 | `internal/api/brands.go` | |
| 5.15 | `internal/api/reference.go` | states, ROA, aggregators |
| 5.16 | `internal/api/router.go` | **manual** |
| 5.17 | `cmd/server/main.go` | config, graceful shutdown |
| 5.18 | `cmd/worker/main.go` | River runner |

### M6 — Content CMS

`internal/content/markdown.go` (`Validate`, `Render`) →
`internal/content/export.go` (`BuildExport`) → `internal/api/content.go` →
`cmd/contentpull/main.go`. See `07-CONTENT-CMS.md`.

### M7 — Community

`internal/auth/argon.go` → `internal/auth/jwt.go` →
`internal/api/middleware/jwt.go` → `internal/api/auth.go` →
`internal/api/comments.go` → `internal/api/survey.go` →
`internal/api/checkout.go` (click event + purchase token).

### M8 — Contract hardening

`openapi/gen.go` → generated TS into the frontend; `internal/media/proxy.go`;
catalogue snapshot writer in `cmd/contentpull`.

### M9 — Admin

Templates **manual**, handlers delegated. Order per `06-ADMIN.md §1`.

---

## 8. Practical notes

- **Batch by milestone, not by day.** M1 is 13 files — a solid weekend at ~10
  minutes each.
- **Write each test file immediately after the file it tests**, in its own
  window, while the spec is still in your clipboard. With a weak model the tests
  are your only evidence the code does what you asked.
- **Cheap model:** M1, M2, M3, M4, M6, M7. **Better model:** M5.16–5.18
  (wiring), M8 (codegen), M9 handlers.
- **Restore the harvest snapshot into a dev DB early** — around M4 — so you're
  testing against real messy listings, not fixtures you invented.
- The thing that actually costs you is **writing the orders**, 5–15 minutes
  each. That is the real price of this approach, and it's still cheaper than
  debugging what an agent wrote against a vague prompt.

---

## 9. Realistic scope

M0–M5 is roughly 60 files and gets you a production API serving the live
frontend. M6–M9 is another ~30. The rules and selectors — the part that took a
year to learn — arrive as harvested data on day one, which is the whole reason
this rewrite is affordable.
