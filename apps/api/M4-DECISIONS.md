# M4 decisions — internal reasoning, classified (PARTIAL milestone)

**Date:** 2026-08-15. Same classification scheme as M1/M3. **This is not a
complete M4** — `08-BUILD-ORDERS.md §7`'s M4 table names 9 files
(`spec.go`, `adapter.go`, `shopify.go`, `woocommerce.go`, `staging.go`,
`gate.go`, `gate_test.go`, `dedup.go`, `dedup_test.go`); this pass built 4
(`spec.go`, `adapter.go`, `shopify.go`, `staging.go`) plus a necessary
addition to `internal/store`. Scope was deliberately held at cbdstore.in /
Shopify only — project owner's explicit call, after a WooCommerce
investigation was started and then pulled back to stay inside the original
PoC scope (`harvest/NOTES.md`'s "cbdstore.in only for this pass"). No
`gate.go` or `dedup.go` either — those decide whether staged data becomes
live/clustered, and nothing yet reads what's staged except the tests below.

---

## Verified against the REAL live site, not fixtures — the point of this milestone

Every previous milestone verified against a self-controlled environment
(a throwaway Postgres container, harvested JSON). This one is different in
kind: `cbdstore.in` is a real, live, third-party website not under this
project's control, and "properly scrape it" specifically meant proving the
adapter against what that site *actually returns right now*, not what a
prior alpha's scraper assumed it would return.

**Two real, live-data findings came out of that, both fixed:**

1. **Shopify's public `/products.json` doesn't expose
   `inventory_quantity`/`inventory_policy` for this store.** Confirmed by
   fetching a real variant object directly: both fields are simply absent
   from the JSON. What IS present and reliable is `available` — a
   Shopify-computed boolean. The harvested field mapping
   (`harvest/scrapers/cbdstore.yaml`'s `in_stock` description) assumed the
   older fields would always be there; they're commonly hidden by Shopify's
   current defaults. Fixed: `shopifyVariant.Available` is now the primary
   signal, with `InventoryQuantity`/`InventoryPolicy` kept as a fallback for
   stores that still expose them. Every real sample after the fix correctly
   showed in-stock items — before the fix, every one of 250 live listings
   read `in_stock: false`, silently.
2. **The three harvested WooCommerce stores have all moved to a different
   theme ("Woodmart") since the prior alpha was built** — `li.product` and
   `a.woocommerce-LoopProduct-link` match nothing on any of the three live
   sites checked (`curebydesign.in`, `magiccann.in`, `itshemp.in`).
   `a.product-image-link` works on all three now; the underlying
   `/product/{slug}/` URL convention is unchanged. **Not fixed** — this
   surfaced while scoping whether to build `woocommerce.go` this pass, and
   the decision was to defer WooCommerce entirely rather than build it
   against a moving target without a firm ask. Recorded in full in
   `harvest/NOTES.md`'s 2026-08-15 update so the next pass doesn't have to
   re-discover it.

Both findings share a lesson the harvest-and-port model doesn't fully
capture on its own: **harvested knowledge has a shelf life.** A scraper
selector or field mapping is a claim about what a live site looked like on
the day it was written, and live sites change. `04-PIPELINE.md §2`'s
promotion gate (not built yet) is partly a defense against exactly this —
a store's field shape shifting out from under a scraper should show up as
an anomalous batch, not a silent bad scrape.

---

## Per-file notes

### `adapter.go`, `spec.go` — SPEC'D shape, one scoping call

`Adapter` (interface) and `RawListing` (type) match `08-BUILD-ORDERS.md §7`'s
naming exactly. `ScrapeAll` streams via a callback rather than returning a
slice — **INFERRED**, reasoning: an aggregator can have thousands of
variants, and holding them all in memory before any of them reach staging
serves no one; `StageBatch` writes each one as it arrives.

`LoadScraperSpec`'s "missing spec is not an error" design — **INFERRED**,
and a deliberate contrast with `resolve.LoadRuleSet`'s fail-fast validation
(M1 recheck). The difference: a missing *pattern* in `cannabinoids.json`
means the classifier is broken for every product. A missing *scraper spec*
for a store nobody's asked to scrape yet (13 of them, right now) is the
expected, normal state — `harvest/NOTES.md`'s whole deferred-stores list.
Treating that as load-time-fatal would make the loader unusable until all
14 stores exist.

### `shopify.go` — PORTED-VERBATIM logic, two live-data fixes (above)

The control flow — one `RawListing` per variant, the two-tier vendor
lookup, the per-variant URL construction — is unchanged from the harvested
source. `TestToRawListingsPerVariantURL` and `TestVendorMapTwoTier` port
the exact scenarios `harvest/scrapers/cbdstore.yaml`'s `quirks` list
describes. The `Available` field addition and its fallback ordering are
**NEW**, forced by live data the harvest didn't anticipate.

### `internal/store/staging.go` — placement call, not `internal/ingest`

**Decision: the actual `INSERT`s for `scrape_batches`/`raw_products` live
in `internal/store`, not `internal/ingest/staging.go` (despite that file's
name).** — **INFERRED**, directly from `01-ARCHITECTURE.md §6`'s "SQL lives
only in the store layer" — a rule M3 already established and this pass
respects rather than quietly bends for convenience. `internal/ingest`'s
`staging.go` is orchestration only (create a batch, run the adapter, call
the store, close the batch) — zero SQL in it, checked. This mirrors M3's
own `golden.go` situation in reverse: there, the build order placed a
non-SQL file inside `store`; here, staying consistent meant a *new*
store-layer file rather than letting SQL leak into `ingest` just because
the build order's file name suggested it might belong there.

### The classification demo — proves M1 against real text, not test fixtures

Not a build-order file — a test
(`internal/ingest/classify_live_test.go`) added specifically because
scraping and staging real data proves data *movement*, not that the
*filtering* (the actual point of this whole project) works against
anything beyond hand-written fixtures. Run against 60 real live listings:
25 had real cannabinoid content correctly extracted, 57 resolved a form
facet, 22 fell below the 0.85 publish-gate confidence threshold and would
correctly route to human review rather than being silently guessed at, and
the one clear non-product in the sample ("Dr. Harshal Sawarkar — BAMS
Ayurvedic Physician," a doctor consultation, not a cannabis product)
resolved to a null form with zero confidence rather than a false-positive
guess. `looksLikeServiceListing` in that test is explicitly NOT the real
compliance check — a narrow local approximation so the demo doesn't reach
into `internal/compliance`, which per ADR-019 doesn't exist yet.

---

## What's still missing before M4 is actually done

- **`woocommerce.go`** — deferred, live-data investigation recorded in
  `harvest/NOTES.md`, not started.
- **`gate.go`** (the promotion gate, ADR-010) — nothing currently reads a
  finished batch and decides pass/hold. Every batch this pass created sits
  at `status='pending_review'` forever; that's correct (never silently
  promoted) but means a human — or M9's admin plane, not built — has to
  look at it manually right now.
- **`dedup.go`** — nothing assigns a `cluster_id` to a promoted listing.
  Staged data has nowhere to go even once a gate exists to approve it.

**The "daily refresh" / admin trigger ask is real and not addressed by this
pass** — it's `cmd/worker` (M5, River job scheduling) and an admin HTTP
handler (M9), neither of which exists as a runnable program yet. What this
pass guarantees for when that layer arrives: `StageBatch` has no
"already ran today" state anywhere — calling it again always creates a
fresh `scrape_batches` row via `CreateBatch`. Whatever schedules it (a cron
trigger, an admin button) just needs to call it; nothing here needs to
change to support that.
