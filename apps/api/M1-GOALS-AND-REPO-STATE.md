# Next milestone (M1) and current repo state

**Date:** 2026-08-11. Note on numbering: what's been called "M0"/"M1" in this
build follows `08-BUILD-ORDERS.md §7`'s file-level milestones, not
`01-ARCHITECTURE.md §7`'s higher-level 0–10 build-order table — the two use
the same numbers for different things. M0 (just finished) = the spine.
**M1 is next.**

---

## 1. M1's goal

`08-BUILD-ORDERS.md §7`, "M1 — Rule engine (pure logic, no DB, no HTTP —
best odds for a weak model)":

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
| 1.11 | `internal/resolve/legacy.go` | facets → `category`/`categories[]` |
| 1.12 | `internal/resolve/value.go` | `PerMg`, `ValueTier`, `RankScore` |
| 1.13 | `internal/resolve/golden_test.go` | loads `testdata/golden/*.json` |

**What it's actually for**, in the doc's own words: *"M1 is where the
false-positive problem actually dies, and it's the safest phase to
delegate: pure functions, no I/O, every file testable in isolation. Do it
first and do it carefully."*

Three things worth knowing going in, all established earlier this session:

1. **`harvest/rules/*.json` already exists** (`1.3`'s input) — the real
   regexes and word lists extracted verbatim from the prior alpha. `1.6` and
   `1.8` are ports of real, working Go logic (`ExtractCannabinoids`,
   `ClassifyWithConfidence`), not new designs — see `harvest/NOTES.md`'s
   "algorithms are not flat data" section for why `1.3`'s loader can't be a
   generic rule-interpreter; the control flow has to be real Go, loading
   these files for its literal patterns.
2. **`testdata/golden/*.json` already has 9 fixtures** (`1.13`'s input),
   converted from the prior alpha's real, passing test cases — not invented.
3. **`1.10`'s precedence** (`override > rule > model > default`) and `1.12`'s
   `ValueTier` bands (`<3`/`3–8`/`>8`, `ADR-012`) and `RankScore`'s
   dominant-basis order (`THC > CBD > total`, `ADR-013`) are already decided
   — M1 implements decisions made earlier in this session, it doesn't make
   new ones.

`internal/domain` (M0) is what M1 imports for its types — `resolve.go`'s
functions will take/return `domain.CannabinoidResult`-shaped data, `Facet`,
`FormValue`, etc. That's the "unblocks everything below" M0 was for.

---

## 2. What's in the repo right now, before M1 starts

```
drtoke/                                    5 commits
├── README.md
├── docs/                                  12 files — 00 through 11
│
├── apps/web/                              SvelteKit frontend, brought in as-is
│   ├── package.json, svelte.config.js, tsconfig.json, vite.config.ts
│   ├── src/  (routes, lib/{api,components,sections,ui})
│   └── static/  (fonts, images)
│   — untouched since the monorepo restructure; nothing built against
│     apps/api yet, still points at VITE_API_URL with nothing live there
│
└── apps/api/
    ├── go.mod, go.sum                     module github.com/dr-toke/api, go 1.25
    ├── SYMBOLS.md                          package domain filled in; resolve/store/content/ingest/api still "pending"
    ├── M0-BUILD-LOG.md                     M0 rationale + 5 flagged design calls
    ├── M0-GOAL-VS-ESTABLISHED.md            M0 goal-vs-delivered audit
    │
    ├── harvest/                            10 files — from the harvest session
    │   ├── rules/{cannabinoids,categories,compliance}.json, dedup.md
    │   ├── scrapers/cbdstore.yaml          PoC scope; 13 stores catalogued, not built
    │   ├── reference/{brands,states,roa,aggregators}.json
    │   └── NOTES.md
    │
    ├── testdata/golden/                    9 fixtures, from real prior-alpha test cases
    │
    └── internal/
        ├── domain/                         M0 — DONE
        │   ├── types.go                    ~28 structs, ~30 enum consts
        │   ├── errors.go                   9 sentinels + ClusterMovedError
        │   ├── types_test.go               (none needed — no logic to test)
        │   └── errors_test.go              4 tests, all passing
        │
        └── db/migrations/                  M0 — DONE, empirically verified
            ├── 001–007_*.sql               24 tables, 742 lines total
            └── migrations_test.go          3 tests: up/down, seed data, 16 constraint cases
```

**Not yet created — everything from M1 onward:** `internal/resolve/`,
`internal/compliance/`, `internal/store/`, `internal/ingest/`,
`internal/jobs/`, `internal/api/`, `internal/admin/`, `internal/content/`,
`internal/auth/`, `internal/media/`, `internal/config/`, all of `cmd/`, all
of `openapi/`. These directories don't exist on disk yet — M0 only scaffolded
empty placeholders for some of them during the harvest-session restructure,
and none have real files.

**State of the two apps relative to each other:** `apps/web` still has
nothing to talk to — `apps/api` has a schema and domain types but no
`cmd/server` binary, no HTTP layer, nothing that listens on `:8080` yet. That
first becomes possible at M5 (`08-BUILD-ORDERS.md §7`: jobs + API — "ship
after this"). M1–M4 are pure backend plumbing the frontend never touches
directly.

**Everything claimed above as "done" is verified, not asserted** — `cd
apps/api && go test ./...` reproduces it: 7/7 tests pass, ~18–21s, against a
throwaway Postgres container it starts and cleans up itself.
