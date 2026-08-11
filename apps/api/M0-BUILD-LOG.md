# M0 build log — for review

**Date:** 2026-08-11
**Scope:** `08-BUILD-ORDERS.md`'s M0 file list (0.1–0.10): `go.mod`, seven
migrations, `internal/domain/types.go`, `internal/domain/errors.go`.
**Written by hand**, not delegated — `08-BUILD-ORDERS.md §3` names exactly
these files as never-delegate.

This doc exists so a second model (or a second pass by me) can review the
work without re-deriving it from scratch. Every non-obvious decision below
says which doc section it came from, and — where I made something up because
the docs describe *behaviour* without giving *schema* — says so explicitly.
**Those flagged spots are the ones worth the most scrutiny.** Everything else
is close to mechanical transcription of `03-DOMAIN-MODEL.md`.

---

## 1. What was actually verified, not just written

I had a real Go 1.26 toolchain and Docker in this environment, so I didn't
stop at "the SQL looks right":

- `go mod init`. Tried pre-fetching every package in `00-CONSTITUTION.md §6`'s
  allowlist (chi, pgx/v5, river + riverpgxv5, goose, colly, minio-go,
  golang.org/x/crypto) up front, then ran `go mod tidy` — which correctly
  pruned all of them back out, because M0's actual Go code (`domain`) only
  imports `github.com/google/uuid`. Left it tidied rather than fighting the
  module graph: `go.mod` should reflect what's imported, not what's
  eventually allowed to be imported. The rest of the allowlist gets added via
  plain `go get` the moment M2/M3/M4/M5's code first imports each one — that
  is the normal, expected shape of an incrementally-built Go module, not a
  gap. `go.mod`'s `go` directive is `1.25` to match `08-BUILD-ORDERS.md §4`'s
  stated target (the local toolchain is 1.26.5, which builds a `go 1.25`
  module fine).
- Ran all **7 migrations up** against a real `postgres:16` Docker container.
- Ran all **7 migrations down**, then back **up** again — the rollback path
  is real, not just written and hoped-for.
- Ran positive AND negative INSERT tests against live constraints:
  `comments`' exactly-one-of-`(post_id, cluster_id)` CHECK correctly
  rejected a row with neither set; `product_facets`' facet-name CHECK
  correctly rejected `'not_a_real_facet'`; a valid comment insert succeeded.
- `gofmt -l .`, `go build ./...`, `go vet ./...`, `go test ./...` all clean
  (`go test` reports "no test files" for `internal/domain` — expected, M0's
  file list has no `_test.go` for this package).
- **One real bug was caught by this**, not by inspection: a `CREATE INDEX`
  on `date_trunc('day', clicked_at)` failed with *"functions in index
  expression must be marked IMMUTABLE"* — `date_trunc` on a `timestamptz` is
  timezone-dependent (STABLE), not IMMUTABLE, so Postgres rejects it in an
  index. Fixed by indexing `clicked_at` directly (a plain range index is what
  `/admin/analytics/clicks?days=30` needs anyway). This is exactly the kind
  of error that reads as correct SQL and silently isn't — worth remembering
  next time a migration reaches for a functional index.

The throwaway container (`drtoke-m0-check`) was removed after validation. No
`docker-compose.yml` / persistent dev DB was created as part of M0 — that's
`09-OPS.md §1`'s job, not this one.

---

## 2. Table-by-table rationale

### `001_sources_listings.sql`

| Table | Source | Notes |
|---|---|---|
| `scrape_sources` | Inferred (see below) | Registry of stores; seeded with **only `cbdstore`** — PoC scope per the harvest session (`harvest/NOTES.md`). |
| `scrape_batches` | `ADR-010`, `04-PIPELINE.md §2` | The promotion gate's unit of decision. Columns map 1:1 to the four ADR-010 thresholds (30% count drop, 15% null-field increase, 80% selector-hit floor, 2x ₹/mg median shift). |
| `raw_products` | `ADR-010` | Staging. Never read by the public API. |
| `product_listings` | `03-DOMAIN-MODEL.md §1` | Live. Carries its own raw text (name/description/category), not just staging — needed so `POST /admin/reprocess` (`04-PIPELINE.md §7`) can re-run `resolve` without re-scraping. |
| `purchase_tokens` | Entity map (`03-DOMAIN-MODEL.md §1`) + `02-FRONTEND-CONTRACT.md §10` | Only the SHA-256 hash is stored, never the raw token. |

**`scrape_sources`/`scrape_batches`/`raw_products` have no schema given in
the docs at all** — `04-PIPELINE.md` describes the promotion gate's
*behaviour* (thresholds, alert-and-hold, never overwrite) in prose, not SQL.
This is the first flagged design call: I inferred a batch/staging split from
that prose. An alternative shape (e.g. gate decisions living as a status
column directly on `raw_products` rows rather than a separate `scrape_batches`
row) is defensible too — I chose the batch-as-first-class-row version because
`06-ADMIN.md §1.4`'s "Source health" admin surface wants "staging batches
awaiting promotion" as a list, which reads more naturally as querying a
`scrape_batches` table than aggregating `raw_products` on the fly.

**Deliberate migration-ordering trick, not an oversight:**
`product_listings.cluster_id` is created as a bare nullable `uuid` column
here (no FK — `product_clusters` doesn't exist until migration 002), and the
FK constraint is attached via `ALTER TABLE` at the end of 002. Same pattern
for `purchase_tokens.cluster_id` (FK attached in 002) and
`purchase_tokens.claimed_by` (FK attached in 006, once `accounts` exists).

### `002_clusters_merges.sql`

| Table | Source |
|---|---|
| `media_assets` | Inferred — see below |
| `product_clusters` | `03-DOMAIN-MODEL.md §3, §5` |
| `cluster_merges` | `03-DOMAIN-MODEL.md §4`, exact column list given in prose |

**The single riskiest call in all of M0:** legacy `category` / `categories[]`
/ `extract_type` / `carrier_oil` are **not columns** on `product_clusters`.
`ADR-002`'s migration plan says the API "keeps emitting legacy `category` /
`categories[]`" during the transition — I read that as a *read-time*
obligation (derive from `product_facets` via `internal/resolve/legacy.go`,
M1.11), not a *write-time* one (a stored, separately-maintained column pair).
The frontend's `catalog.ts` does read `extract_type` and `carrier_oil` as flat
strings directly on `ApiProduct` (confirmed by reading the actual frontend
code earlier in this session), so *something* has to serve those fields — the
question is only whether it's a stored column kept in sync by hand, or a
computed value with one source of truth. I went with computed, because a
stored-and-derived pair is exactly the kind of dual-write the facet refactor
(`01-ARCHITECTURE.md §4`) exists to get away from. **If M1's `legacy.go`
turns out to need these as columns for query performance (e.g. filtering
`/api/products?category=` without a join), that's a straightforward additive
migration later — but it should be a deliberate call once query patterns are
known, not a default taken now.**

**`media_assets` has no schema in the docs** — only the package name
(`internal/media`, `01-ARCHITECTURE.md §6`) and the wire contract it must
satisfy (`05-API-REFERENCE.md §7`: content-hashed filename, dimensions +
blurhash in the payload, sizes `thumb|card|full` derived by the proxy). I
designed one table storing one *original* per asset (hash, ext, width,
height, blurhash) with `thumb/card/full` treated as proxy-computed resize
variants of that one original, not three separate stored rows — this matches
"the proxy... streams from MinIO" language in `05-API-REFERENCE.md §7`
reasonably closely but is an inference. `kind` (`product`/`editorial`) and
`source_url`/`uploaded_by` exist specifically to satisfy `ADR-017` (editorial
imagery becomes CMS-managed) and the still-open image-provenance item in
`01-ARCHITECTURE.md §8` — worth checking those two concerns actually got
served by this shape once M6 (content CMS) is built against it.

**`publishable boolean` on `product_clusters`** is a maintained cache of
`03-DOMAIN-MODEL.md §2`'s confidence-gate formula, not something the doc asks
for directly. Reasoning: that formula is a join across `product_facets`
(`form.confidence`, `route.confidence`) plus a `price_paise > 0` check: doing
that join on every `/api/products` request seemed like exactly the kind of
thing the doc's own performance-consciousness elsewhere (partial indexes,
keyset pagination) would want cached instead. `internal/resolve/precedence.go`
is intended as the sole writer. **Flag this one too** — if it turns out
wrong, the fix is deleting the column and joining live, not a schema
disaster, but it's a real design opinion I inserted.

### `003_facets_overrides.sql`

Copied **verbatim** from `03-DOMAIN-MODEL.md §2`'s SQL block — this one the
doc gives in full, so there was nothing to infer. The only addition is a
`CHECK (facet IN (...))` constraint restricting the six facet names, which
the doc's block doesn't include; added so a typo'd facet name fails at
`INSERT` instead of becoming a silently-orphaned, unqueryable row (the index
`product_facets_lookup_idx` is on `(facet, value)`, so a misspelled facet
would just never be found by anything, not error). Facet *value* vocabularies
(e.g. `form` must be one of the 11 named form values) are **not** enforced at
the DB layer — that's `internal/resolve`'s job in Go, per
`00-CONSTITUTION.md §6`'s preference for avoiding triggers where application
code can do the check.

### `004_brands_reference.sql`

Brands/states/ROA/aggregators schema and seed data both come straight from
the harvest pass (`harvest/reference/*.json`), which in turn came from
`packages/data/brands.ts` and the prior alpha's migration
`026_reference_content.sql` (states/ROA/aggregators' self-correction columns
— `link_status`, `link_failures`, `verify_interval_days` — are that
migration's design, not newly invented here; `03-DOMAIN-MODEL.md §7`
describes reference content as "self-checking" without repeating the schema
that makes it so).

One seed-data wrinkle: `aggregators.source_slug` is a FK to
`scrape_sources.slug`, but only `cbdstore` exists this pass (PoC scope). The
`itshemp` and `cannameds-india` aggregator rows are seeded with
`source_slug = NULL` (the column is nullable) rather than blocked or faked —
re-point them once those two scrapers are actually built.

### `005_content.sql`

Matches `03-DOMAIN-MODEL.md §11`'s SQL block, plus `license` (already added
to that doc during the ADR-015 pass earlier this session) and one addition
beyond the doc: `hero_image_id` on `content_revisions`, `FK -> media_assets`.
`ADR-017` says editorial images move into the CMS plane but never gives a
column for it; "one revision, at most one hero image" is the simplest shape
that satisfies the ADR without inventing an unrequested many-to-many gallery.

### `006_community.sql`

`accounts` / `refresh_tokens` / `comments` match `03-DOMAIN-MODEL.md §8`'s
block exactly, including the handle format CHECK (`^[A-Za-z0-9_]{3,24}$`,
`05-API-REFERENCE.md §3`) and the exactly-one-of `(post_id, cluster_id)` CHECK
on comments (verified live against Postgres — see §1 above).

`admin_users` / `admin_audit_log` are the **second-most placement-uncertain
call** in M0. They're real requirements — `00-CONSTITUTION.md §4` ("shares NO
signing key, session, or users table with the public tier") and
`06-ADMIN.md §3, §1.9` ("Own `admin_users` table... Audit log...
Non-negotiable") — but `08-BUILD-ORDERS.md`'s M0 file list never names a
migration file for them; `06-ADMIN.md` is scoped to M9 in the build order
(`01-ARCHITECTURE.md §7`). I judged that a Constitution-level hard
requirement (auth isolation) shouldn't wait until M9 to exist at the schema
level even if the *handlers* using it are built later, and put it here since
it's auth-adjacent to `accounts`. **This is the one placement decision most
worth a second opinion** — an argument exists for a dedicated
`008_admin.sql` instead, built alongside M9, so M0 stays strictly scoped to
what `08-BUILD-ORDERS.md` actually lists.

### `007_clicks_survey_queue.sql`

`click_events` matches `03-DOMAIN-MODEL.md §10` exactly (verified no IP, no
user agent, no account ID column exists anywhere in it). `survey_counts` /
`survey_meta` implement `03-DOMAIN-MODEL.md §9`'s "aggregate counters only"
requirement as a `(dimension, value) -> count` table plus a singleton totals
row — the doc names the four dimensions and says "individual responses are
never stored" but doesn't give a table shape, so this is inferred too, though
lower-stakes than the others (a counter table has few ways to be wrong).

`review_queue` — same situation as `admin_users`: extensively described by
*behaviour* (`04-PIPELINE.md §5`'s six reason codes, `06-ADMIN.md §1.2`'s
workflow: evidence spans, keyboard-driven approve/reject, dry-run diff) but
never given SQL. `proposed_value jsonb` exists specifically to support the
"diff vs. current" requirement in `06-ADMIN.md §1.2` — the queue item needs
to remember what the model/rule *proposed* so an admin approving it can see
current-vs-proposed before it becomes a `product_facet_overrides` row.

---

## 3. `internal/domain/types.go` and `errors.go`

Mechanical: one Go struct per table, one typed string enum per `CHECK (...
IN (...))` constraint (so a typo'd value is a compile error, not a runtime
constraint violation caught only in production). Nullable SQL columns are
`*T` pointer fields, never zero values — this is
`00-CONSTITUTION.md §5`'s "unknown numerics are NULL/nil pointers, never
zero" applied to the whole struct set, not just cannabinoid mg fields.

The one thing added beyond a direct schema mirror: `ClusterMovedError`, a
typed error carrying `OldID`/`NewID` so `internal/api`'s product-detail
handler can build the `{"moved_to": "<uuid>"}` response
(`02-FRONTEND-CONTRACT.md §4`, `03-DOMAIN-MODEL.md §4`) without a second
query. It deliberately does **not** implement `Is(ErrNotFound)` — a moved
cluster and a genuinely missing one must stay distinguishable by
`errors.As`, since `02-FRONTEND-CONTRACT.md §4` treats "moved" and "not
found" as different status-code outcomes (200-with-a-body vs. 404), not
variations on the same failure.

Sentinel errors in `errors.go` map 1:1 to the nine stable machine codes in
`02-FRONTEND-CONTRACT.md §3`. Two extras beyond that list
(`ErrDuplicateHandle`, `ErrPurchaseTokenAlreadyClaimed`) both still map to
the `validation_failed` HTTP code — they're separate Go sentinels only so the
handlers that raise them can give a specific message without string-matching
error text, not because the wire contract gains new codes.

---

## 4. Punch list for review

Ranked by how much I'd want a second opinion on each, most first:

1. **Legacy `category`/`categories[]`/`extract_type`/`carrier_oil` computed
   at read time, not stored.** Correct per `ADR-002`'s spirit, but unverified
   against real query-performance needs — could be wrong once M1's `resolve`
   package and M5's `/api/products` handler exist for real.
2. **`admin_users`/`admin_audit_log` placed in `006_community.sql`.** A
   Constitution-required table with no assigned migration file in
   `08-BUILD-ORDERS.md`. Defensible alternative: its own file, built at M9.
3. **`scrape_sources`/`scrape_batches`/`raw_products` shape** — the promotion
   gate has real behavioural spec (`ADR-010`) but no schema; this is one
   reasonable shape among a few.
4. **`media_assets` schema** — satisfies the wire contract
   (`05-API-REFERENCE.md §7`) as far as I can tell, but is a guess at the
   underlying storage model.
5. **`review_queue.proposed_value` / `publishable` materialized column** —
   both are reasonable engineering, both are additions beyond what any doc
   explicitly specifies.

Everything not on this list — `product_facets`/`product_facet_overrides`
(verbatim from the doc), `accounts`/`refresh_tokens`/`comments` (verbatim),
`content_docs`/`content_revisions` (verbatim + one ADR-mandated column),
`states`/`roa_methods`/`aggregators`/`brands` (harvested verbatim from a real
prior migration + a real TS file) — is about as low-risk as a hand-written
migration gets, and all of it is now proven to actually apply and roll back
against real Postgres 16, not just parse.
