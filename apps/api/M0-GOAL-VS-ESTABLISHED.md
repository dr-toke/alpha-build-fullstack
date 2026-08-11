# M0 — goal vs. established

**Date:** 2026-08-11. Companion to `M0-BUILD-LOG.md` (the how/why); this is
the shorter did-we-hit-the-target doc.

---

## 1. What M0's goal actually was

Not something I defined — it's specified in two places:

**`08-BUILD-ORDERS.md §7`, the M0 table** ("Spine (manual, you)"):

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

With one constraint attached directly under that table: **"Schema comes from
`03-DOMAIN-MODEL.md`. Do not improvise it."**

**`01-ARCHITECTURE.md §7`**, the top-level build order, describes what M0
is *for*: "Spine: migrations + `domain/types.go`, `go build` green |
**unblocks everything below**." M0's job isn't to be interesting — it's to
be a compiling, schema-correct floor that M1 (the rule engine) and every
milestone after it can stand on without having to revisit the foundation.

That's the whole goal: **ten specific files, schema traceable to one doc,
`go build` green, nothing more.**

---

## 2. What was established, item by item

| # | Goal | Established | Verified how |
|---|---|---|---|
| 0.1 | `go.mod` | `module github.com/dr-toke/api`, `go 1.25`. Only imports actually used are declared (`google/uuid`, `pressly/goose/v3`, `jackc/pgx/v5`) — rest of `00-CONSTITUTION.md §6`'s allowlist added when M2–M5 first import them, not pre-declared | `go build`, `go vet` clean |
| 0.2–0.8 | 7 migrations, schema from `03-DOMAIN-MODEL.md` | All 7 written, 24 tables total | **Run for real** against Postgres 16: up ✓, down ✓, back up ✓ (`migrations_test.go`) |
| 0.9 | `internal/domain/types.go` | One file, every struct + every enum (~28 types, ~30 enum constants) | `gofmt`, `go build`, `go vet` clean; enum string values pinned against migration `CHECK` literals in a unit test |
| 0.10 | `internal/domain/errors.go` | Sentinel errors, one per `02-FRONTEND-CONTRACT.md §3` machine code, plus `ClusterMovedError` | Unit-tested: distinctness, `%w`-wrap survival, `ClusterMovedError` correctly does NOT alias `ErrNotFound` |

**All ten items exist and are verified.** That's the goal, met.

---

## 3. What went beyond the stated goal

The M0 table doesn't ask for any of this — added because "done" should mean
*checkable*, not just *written*:

- **`internal/db/migrations/migrations_test.go`** — self-starting integration
  test (spins up its own Postgres container, tears it down). Re-runnable by
  anyone, not a one-time manual session.
- **`internal/domain/errors_test.go`** — unit tests for the error semantics
  the API layer will depend on later.
- **`M0-BUILD-LOG.md`** — full rationale, doc citations, and the two real
  bugs the test-writing process caught (`date_trunc()` in an index isn't
  IMMUTABLE; `goose.DownTo`'s real signature).

None of this was asked for by `08-BUILD-ORDERS.md`. It exists because
"schema comes from `03-DOMAIN-MODEL.md`, do not improvise it" is a much
easier rule to *claim* compliance with than to *prove* compliance with, and
the gap between those two is exactly what the tests close.

---

## 4. Where the goal itself was silent

"Do not improvise it" only works where the doc actually specifies something
to not-improvise. **`03-DOMAIN-MODEL.md` gives complete, literal SQL for
some tables and only prose behaviour for others** — this isn't a gap in what
I built, it's a gap in what the goal itself defined, and I made a call in
each case rather than stopping to ask. Five calls, most-consequential first
(full reasoning in `M0-BUILD-LOG.md §4`):

1. Legacy `category`/`categories[]`/`extract_type`/`carrier_oil` — computed
   at read time from facets, not stored as columns.
2. `admin_users`/`admin_audit_log` — placed in migration 006; the docs
   require these tables to exist but never assign them a migration file.
3. `scrape_sources`/`scrape_batches`/`raw_products` (the promotion-gate
   staging schema) — `ADR-010` describes behaviour, not SQL.
4. `media_assets` — inferred from the `/media/*` wire contract, not given a
   schema.
5. `review_queue` / `survey_counts` / the `publishable` cache column —
   same situation, less consequential.

**These five are the only sense in which M0 is not "fully specified and
mechanically transcribed."** They're not missing work, not bugs, and not
things I got wrong that I know of — they're places the goal itself required
a judgment call, flagged for someone (you, or Opus 5) to either confirm or
overrule.

---

## 5. Bottom line

| Question | Answer |
|---|---|
| Do all 10 files in `08-BUILD-ORDERS.md`'s M0 table exist? | Yes |
| Does `go build ./...` succeed? | Yes |
| Is the schema traceable to `03-DOMAIN-MODEL.md` everywhere the doc gives one? | Yes, verbatim where given |
| Where the doc doesn't give one, is a defensible schema there instead, clearly flagged? | Yes — 5 items, ranked, in `M0-BUILD-LOG.md §4` |
| Is any of it just asserted, or is it actually exercised against real Postgres? | Actually exercised — `go test ./...`, 7/7 pass, ~18-21s, re-runnable by anyone |
| Does M1 have a floor to build on? | Yes — that was the actual goal, and it's met |

M0 is done. The five flagged items are the only open questions, and they're
open because the spec was silent, not because the work is incomplete.
