# M3 decisions — internal reasoning, classified

**Date:** 2026-08-15. Same classification scheme as `M1-DECISIONS.md`:
**SPEC'D** (a doc states it exactly), **PORTED** (carried from the prior
alpha or an earlier milestone unchanged), **INFERRED** (a doc implies a
requirement without an exact shape, I chose one), **NEW** (no precedent
anywhere, my own call). All 11 files `08-BUILD-ORDERS.md §7`'s M3 table
names are done: `store.go`, `listings.go`, `clusters.go`, `facets.go`,
`overrides.go`, `brands.go`, `reference.go`, `queue.go`, `content.go`,
`community.go`, `golden.go`.

---

## Verified, not asserted

Same discipline as M0 and M1: a shared Postgres 16 test harness
(`main_test.go`) spins up once per `go test` run (not once per file — 11
files' worth of individual containers would make the suite slow enough
people stop running it), runs all 7 migrations, and every store method has
at least one test exercising it against the real schema. 81.8% coverage,
`staticcheck`/`go vet -all`/`gofmt` all clean. Zero containers left running
after any run — checked explicitly, every time, same as M0/M1.

**Four real bugs were caught by writing these tests, not by inspection:**

1. **`ClusterByID` returned the stale row instead of the moved error.**
   `Merge()` deliberately never deletes the old `product_clusters` row (a
   hard delete would cascade-destroy its facets/comments/click history via
   `ON DELETE CASCADE`, and `03-DOMAIN-MODEL.md §4` never asks for that) —
   so the old row is still directly findable after a merge. The first draft
   checked `cluster_merges` only as a not-found fallback, which meant a
   merged cluster's stale data was returned as if nothing had happened,
   silently defeating the entire point of `02-FRONTEND-CONTRACT.md §4`'s
   `moved_to` redirect. Fixed by checking `cluster_merges` FIRST,
   unconditionally. This is the most consequential bug this milestone
   found — it would have shipped a broken redirect for every merged
   product, and nothing short of an actual merge-then-fetch test would
   have caught it.
2. **Three separate `NOT NULL jsonb` columns crashed on a nil Go map.**
   `content_revisions.frontmatter`, `product_facets.evidence`,
   `review_queue.proposed_value` are all `NOT NULL DEFAULT '{}'` — but that
   default only applies when the column is *omitted* from an INSERT.
   Passing a nil `map[string]any` explicitly (the zero value of every
   `domain.X{...}` struct literal that doesn't set the field) encodes as
   SQL NULL, not `'{}'`, and violates the constraint. Found once
   (`content.go`'s `NewRevision`, via `TestContentPublishFlow` calling it
   with a bare `domain.ContentRevision{}`), then proactively checked and
   fixed in the other two places with the same shape
   (`facets.go`'s `UpsertFacets`, `queue.go`'s `Enqueue`) rather than
   waiting to hit the same crash three times.

Both bug classes share a lesson: **the store layer's job is partly to
absorb the gap between "what a caller naturally constructs" and "what the
schema strictly requires."** A caller building a `domain.ContentRevision{}`
without thinking about `Frontmatter` is not misusing the API — it's the
store layer's job to make that safe.

---

## Per-file notes

### `store.go` — SPEC'D shape

`New`/`Store`/`Close`, exactly as named in `08-BUILD-ORDERS.md §7`. Uses
`pgxpool.Pool` (native pgx, not `database/sql`) per `00-CONSTITUTION.md §6`'s
"no ORM" — `pgx/v5/stdlib`'s `database/sql` shim is used ONLY by
`main_test.go`, to hand a `*sql.DB` to `goose` (which doesn't speak native
pgx). Production code never touches that shim.

### Manual `Scan()` everywhere, not reflection-based struct mapping — INFERRED

pgx/v5 offers `pgx.RowToStructByName` (reflection + `db` struct tags) as a
faster-to-write alternative to what's here. Not used, deliberately: it would
mean retroactively adding `db:"..."` tags to every field of every struct in
M0's already-committed, already-tested `domain.go`, and it trades an
explicit, grep-able `Scan(&a, &b, &c)` call for a runtime reflection path
whose failure mode (a silently-unmapped field) is quieter than a compile-time
argument-count mismatch. `00-CONSTITUTION.md §6`'s "no ORM" reasoning extends
naturally to "prefer the more explicit of two pgx-native options," even
though `RowToStructByName` isn't technically an ORM.

### `clusters.go` — the check-order bug (above), plus one scope note

`ListClusters`'s `ClusterFilter` is deliberately narrower than
`05-API-REFERENCE.md §1`'s full `/api/products` param surface
(`?form=&route=&brand=&basis=...`). Building the complete filter-clause
constructor now, before M5 defines the actual handler that calls it, risks
guessing a shape that doesn't match what the handler needs. Covers what's
storage-layer-obvious today: the `publishable` gate, brand filtering, and
keyset pagination on `(rank_score, id)` — `02-FRONTEND-CONTRACT.md §5`'s
"never OFFSET" requirement, tested directly
(`TestListClusters`'s pagination case walks two pages and confirms no
duplicate or skipped row).

### `overrides.go`, `queue.go` — the fixture/audit split — INFERRED

`03-DOMAIN-MODEL.md §2` describes setting an override and appending a golden
fixture as one inseparable action ("every override auto-appends a
fixture"); `06-ADMIN.md §1.2` describes the same coupling for resolving a
review-queue item (writes an override AND appends a fixture). Neither
`SetOverride` nor `Resolve` does both here — each does only its own row.
Reason: `AppendFixture` needs the raw listing title/description, which
these two functions don't have and shouldn't need to fetch themselves (that
would mean a `SetOverride` call silently doing a second, unrelated query
against `product_listings`). The caller — an M9 admin handler, which
already has the full listing context on screen — is expected to call both.
**Flagged for a second look once M9 actually writes that caller**: if it
turns out awkward to keep these three writes (override row, fixture file,
audit log row) coordinated by every caller individually, a small
orchestration helper at that point would be a legitimate fix, not a sign
this decision was wrong now.

### `golden.go` — the one non-SQL file, placed here by the build order itself

Not a decision I made — `08-BUILD-ORDERS.md §7` puts `AppendFixture` at
`internal/store/golden.go` explicitly (3.11), despite it being pure file
I/O with no SQL in it at all. Honored as written. `AppendFixture` merges
into an existing fixture file rather than overwriting — **NEW**, reasoning:
a cluster can accumulate overrides on different facets over separate admin
actions (form corrected today, extract corrected next month), and each
should add to what's locked in, not discard the previous correction.

### `domain.State` / `domain.Aggregator` gained a `Stale bool` field

Not an M3 file, but a real gap M3 surfaced in M0's `domain.go`: the
self-correcting reference-content design (`03-DOMAIN-MODEL.md §7`, and the
harvested `reference.go` handler read earlier in this project) computes and
returns `stale` for both types, but M0's first pass at the struct omitted
the field. `reference.go`'s `ListStates`/`ListAggregators` now scan it
properly instead of discarding the computed value — confirmed by
`TestListStatesWithholdsExciseURLPastBrokenThreshold` and the seed-data
staleness check in `TestListStates` (the harvested seed rows are dated
2025-05-01, long past any reasonable `verify_interval_days`, so every one
should read `stale = true` right now — and does).

### `community.go` — a few naming calls, all **NEW**

Function names (`CreateAccount`, `AccountByHandle`, `CommentsForCluster`,
etc.) aren't given by the build order, which only describes this file's
scope as "accounts, refresh tokens, comments" in prose. Named to read
naturally as `Store` methods and to mirror the noun/verb pattern the build
order DOES specify elsewhere (`ClusterByID`, `ListClusters`). `DeleteComment`
takes a `*uuid.UUID` (nullable) for the requesting account specifically so
an admin-initiated delete (`byAdmin=true`) can pass `nil` without a
meaningless placeholder ID — tested directly.

---

## What's genuinely unverifiable without more context

- **`ClusterFilter`'s eventual full shape.** Noted above — this is
  intentionally incomplete pending M5's actual handler requirements, not an
  oversight.
- **The override/fixture/audit-log coordination.** Flagged above — whether
  three independent caller-orchestrated writes stays the right shape is a
  question for when M9 actually writes the caller, not answerable from
  here.
