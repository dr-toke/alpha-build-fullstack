# Harvest notes — 2026-08-11

Source repo: `/home/ago/Downloads/dr-toke-init/apps/api` (Go/Chi/pgx, 29
migrations — matches ADR-001's description of the prior alpha exactly).
**Not** `/home/ago/Downloads/d-p` — that's an unrelated Python ransomware/
dark-web threat-intel project that happens to share a name. Checked file
contents before harvesting from either; worth a second look if anyone else
picks this up and finds two "d-p"-adjacent repos in `~/Downloads`.

## Scope decision: cbdstore.in only for this pass

Project owner's call: get a basic PoC running before harvesting all ~14
stores. Only `harvest/scrapers/cbdstore.yaml` was written. The rule engines
(cannabinoids/categories/compliance/dedup) are **not** store-specific, so
those were harvested in full regardless of scraper scope — deferring them
would have meant re-doing this same reading pass later for no reason.

### Deferred stores (registry.go, not yet written as harvest/scrapers/*.yaml)

Direct-brand Shopify stores (same generic adapter as cbdstore, config-only differences):
| slug | domain | brand_slug |
|---|---|---|
| boheco | boheco.com | boheco |
| indie-extracts | indieextracts.com | indie-extracts |
| the-trost | thetrost.com | the-trost |
| india-hemp-organics | indiahemporganics.com | india-hemp-organics |
| vedi-herbals | vediherbals.com | vedi-herbals |
| india-hemp-and-co | indiahempandco.com | india-hemp-and-co |
| hempstrol | hempstrol.com | hempstrol |

WooCommerce stores (generic WooCommerce adapter, different from Shopify):
| slug | domain | shop_url | brand_slug | notes |
|---|---|---|---|---|
| magiccann | magiccann.in | /shop/ | magiccann | |
| cure-by-design | curebydesign.in | /shop/ | cure-by-design | |
| itshemp | itshemp.in | /shop/ | (multi-brand, extracted from `.product_meta a[href*='/brand/']`) | **non-standard product URL path `/shop-its/` instead of `/product/`** — thin wrapper `NewItsHemp()` exists for this one config override |

Shopify aggregators (same adapter as cbdstore, different vendor_map):
| slug | domain | vendor_map |
|---|---|---|
| cbdshopofindia | cbdshopofindia.com | `cbdShopVendorMap` (different from cbdstore's `cbdStoreVendorMap` — Cannabryl, Polyherbs, Namaste Organics, Hampa, Calmosis appear here but not in cbdstore's map) |
| cannameds | cannameds.in | `cbdStoreVendorMap` (same map as cbdstore) |

Custom-platform (one bespoke scraper each, not selector-driven):
| slug | domain | notes |
|---|---|---|
| cannabryl | cannabryl.com (Indogenix Biosciences LLP) | All products flagged `prescription_required: true` unconditionally — THC-dominant/prescription line. Listing discovery via `a[href^='/products/']` excluding category/disease-index links. Price found by scanning `p, span, div` text nodes for a short string containing '₹'. Images from `indogenix.b-cdn.net` CDN. |

**When resuming the wider harvest**: the two generic adapters (Shopify,
WooCommerce) cover 11 of the 14 stores — porting those two adapters once
covers cbdstore now and unlocks 10 more stores for the cost of a YAML file
each, not new code. Only `cannabryl` needs bespoke scraper code, matching
`08-BUILD-ORDERS.md §1`'s "three adapters, not fourteen scrapers" framing
exactly.

## Important: the rule "algorithms" are not flat data

`11-HARVEST.md §2.2`'s example `cannabinoids.json` format (a flat prioritized
pattern list) undersells what `ExtractCannabinoids()` and
`ClassifyWithConfidence()` actually do. Both are genuinely procedural:

- **Cannabinoids**: ordered branches (explicit mg -> %+volume -> ratio+total
  -> bare ratio -> generic mg fallback), ratio-orientation logic, per-serving-
  to-pack-total reconciliation, and a name-first identity resolution step that
  decides which cannabinoid a lone number belongs to. None of this is
  "check patterns in priority order and take the first match" — it's closer to
  a small decision procedure with several interacting special cases (see the
  `notes` block in `harvest/rules/cannabinoids.json` for each one, with the
  bug each one fixes).
- **Categories**: name-first-then-description-fallback, a pet/apparel
  exclusivity pre-check that runs BEFORE form detection, a post-detection
  coherence matrix that actively DELETES incompatible forms (order-dependent
  — topical's deletions must run before edible_solid's, see
  `lip-balm-topical-not-edible.json`), and negation-stripping that runs before
  any of it.

**Recommendation for M1**: don't try to force these into a declarative rule
list the engine "just interprets" — port the control flow as Go code in
`internal/resolve/cannabinoids.go` / `facets.go`, faithfully, using the
harvested regexes/word-lists as the named constants it loads from
`harvest/rules/*.json`. This is still "rules live in data, not hardcoded" in
the sense that matters (a keyword or regex change is a data diff, not a
recompile) — it just means the *shape* of the loader is "a Go function that
reads named patterns from JSON," not "a generic rule-list interpreter." Flag
this explicitly in the M1 build orders (`08-BUILD-ORDERS.md §5`'s `BEHAVIOUR`
sections for `cannabinoids.go`/`facets.go`) so whoever writes those orders
doesn't assume a shape the source algorithm was never in.

## Two packages carry zero test coverage forward

`internal/catalog/compliance/filter.go` and `internal/catalog/dedup/dedup.go`
have **no `_test.go` file** in the prior alpha. `testdata/golden/` fixtures
for compliance (hard-block, terminology-review, service-listing, price-anomaly
cases) and dedup need to be **authored fresh** during M2/M4 — there was
nothing to harvest here, only the rule/algorithm source itself. Flagged so
nobody assumes golden fixtures exist for these two the way they do for
cannabinoids/categories.

## Dedup fuzzy-matching is dead code

`dedup.JaroWinkler` and `dedup.WithinPct` have **no callers anywhere** in
`apps/api`. The only thing actually wired into the pipeline is
`dedup.Fingerprint()`, called with the **raw** scraped name (no separate
normalisation pass exists despite the function's doc comment implying one).
Dedup in the prior alpha is exact-fingerprint-only: two listings of the same
product with slightly different scraped name text do not merge. See
`harvest/rules/dedup.md` for the full finding — this changes what "harvest the
dedup fingerprint" actually means: there's a primitive to carry forward, not a
tuned, working fuzzy-match system.

## Not done in this pass

- **`harvest/snapshot.sql`** — no dev DB is currently running (`docker compose`
  wasn't stood up as part of this session). Deferred until infra is up;
  `11-HARVEST.md §2.7` explicitly frames this as insurance, not a dependency,
  so it isn't blocking M0.
- **`review_queue` decisions** — same reason, nothing to export from a DB
  that isn't running. If the prior alpha's dev DB (or a backup of it) exists
  anywhere, it's worth a second pass specifically for this — human review
  decisions are the one category of knowledge that genuinely can't be
  re-derived from source code at all.
- **Reference content beyond states/roa/aggregators/brands** — the four files
  in `harvest/reference/` cover everything `packages/data/` + migration
  `026_reference_content.sql` had. No gaps found there.

## Verification against `11-HARVEST.md §3`

- [x] cbdstore.yaml written (scope: 1 of ~14 sources, by design — see above)
- [x] All four rule files transcribed verbatim (cannabinoids, categories,
      compliance, dedup) — regexes copied character-for-character from the Go
      source, not retyped from memory
- [x] 9 golden fixtures in `testdata/golden/` (target was 10; the Shopify
      variant-URL artifact and the literal "joint pain balm" cases are
      scraper/ingest-level behaviours, not facet-extraction fixtures in this
      shape — captured instead as a `quirks` entry in `cbdstore.yaml` and as
      the existing smokable word-list design already excluding bare "joint",
      respectively)
- [x] `harvest/snapshot.sql` — **not done**, no running dev DB this pass
- [x] Reference content exported (states, roa, aggregators, brands)
- [x] This file
- [ ] Old repo archived on a tag — do this in `dr-toke-init` itself before
      anyone treats it as disposable; not done as part of this pass (out of
      scope for a docs/harvest session in a different directory)

**Given the two unchecked boxes, this is a partial harvest by
`11-HARVEST.md`'s own bar — sufficient to start M0/M1 against, not sufficient
to consider `dr-toke-init` safe to delete.** Full harvest (remaining 13
stores, DB snapshot, repo tag) is follow-up work, not blocked by anything
here.
