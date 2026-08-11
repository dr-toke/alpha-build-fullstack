# 04 — Pipeline

> The data from these stores is genuinely messy: wrong categories, missing
> dosages, duplicate listings, variant-price mixups. The pipeline is where
> almost all the engineering value lives.
>
> The implementation is a fresh rewrite, but the **rules below are not
> re-derived** — they are harvested verbatim from the prior alpha into
> `harvest/rules/*.json` and loaded at init (`11-HARVEST.md`). The engine is new
> code; the patterns are preserved data. Never retype a regex from memory.

---

## 1. Stages

Each stage is one River job, one file.

| Stage | Responsibility |
|---|---|
| `scrape` | Fetch raw listings per store into **staging**. Never writes live. |
| `normalise` | Extract mg CBD/THC, size, per-variant price, carrier. Emits evidence. |
| `resolve` | Apply rules per facet; low confidence → queue. Overrides applied last. |
| `compliance` | Hard-block vs review tiers. **Logs a reason code, always.** |
| `dedup` | Fingerprint → cluster. Assign or reuse durable UUID. |
| `bestdeal` | ₹/mg per cannabinoid, cheapest listing, `value_tier`, `rank_score`. |
| `images` | Fetch, transcode, hash, store in MinIO. Never hotlink a store. |

Job enqueue rules that already work and must be preserved:

- `compliance`: `pass`, trusted-`unknown_brand`, and `queue` all enqueue `dedup`.
  `blocked` does not; it hides any existing cluster via `in_stock = false`.
- `dedup`: the existing-cluster (fingerprint match) branch must **refresh all
  derived fields** — category/facets, cannabinoids, everything — not just bump
  `last_seen_at`. Without this, reprocessing never applies a classifier change.
  **This is critical. Keep it.**

---

## 2. The promotion gate

**Scrapes never write to live tables.** They write to staging, then a gate
decides. This is the single most important safety valve in the system: without
it, one store's markup change silently deletes 90% of a source and the site
looks fine while being wrong.

Auto-reject a source's batch when:

- product count drops more than **30%** vs. the last successful run
- more than **15%** of previously-populated fields come back null
- the ₹/mg distribution shifts beyond band (median moves > 2×)
- the scraper's selector-hit rate falls below **80%**

Rejection **alerts and holds**. It never overwrites. A human approves or fixes
the adapter. Thresholds are opening estimates, tuned by an ongoing AI-assisted,
human-approved process rather than a one-time freeze (`ADR-016`) — proposals
come from observed queue/rejection outcomes, a human accepts, and the accepted
set is versioned.

---

## 3. Cannabinoid extraction

`internal/resolve/cannabinoids.go`, driven by
`harvest/rules/cannabinoids.json`. Seven-step priority:

1. Explicit `500mg CBD` / `CBD 500mg`
2. Explicit THC mg — skipped if the product is labelled THC-free (`reTHCFree`)
3. THC:CBD ratios + total mg
4. % concentration + volume → computed mg (`5% CBD 30ml` → ~1350mg)
5. % THC + volume
6. Hemp seed / nutrition detection → `concentration_type = hemp_seed`, ₹/mg null
7. Generic mg fallback with context inference (CBD-labelled vs Vijaya vs unknown)

### Required behaviours (each one is a shipped bug)

- **Parse ratios in both orders.** Real copy says `(10:1 THC:CBD)` and
  `THC:CBD ratio of 10:1` — numbers before *and* after the label. Determine
  which cannabinoid is dominant and split the total accordingly.
- **Do not early-return after the first match.** Capture `cbd_mg` and `thc_mg`
  independently when both are present. Set `concentration_type` to the dominant
  basis but keep both numbers.
- **Detect THC-dominant wording** ("THC-dominant", "high THC", "THC tolerance")
  to disambiguate when only a total mg + ratio exist.
- **Percentages are full decimals.** `0.3% THC` is a valid small value, not
  zero. A regex reading `50% THC` as `0% THC` shipped once and made a 1:1
  product show as CBD-only.
- **Negation-stripping.** "No need to smoke or vape" must not make a capsule
  vapeable. Matches inside a negation window are discarded and recorded with
  rule ID `negated`.
- **Per-variant URLs.** Shopify lists each size as a separate row sharing one
  URL, which produced fake ₹0.28/mg prices. Each variant carries its own URL.

**Canonical test product:** *Cannavedic – CannEx Strong Plus THC+CBD Extract
(10:1 THC:CBD) 1000mg | 1ml*. It exhibits every bug above. Expected:
`thc_mg ≈ 909`, `cbd_mg ≈ 91`, THC-dominant, classified vapeable/concentrate,
visible in the catalogue, flagged for terminology review.

---

## 4. Classification

`internal/resolve/facets.go`, driven by `harvest/rules/categories.json`.

Harvested invariants — **must not regress**:

- **Word-boundary regexes** (`\b…\b`), never `strings.Contains`. `"joint"` was
  removed from smokable because it meant "joint pain".
- **Pet and apparel are exclusive** categories.
- **Consumption form takes precedence over extract type.**
- `reConcentrate`: "not diluted with carrier oil" / distillate / dab → vapeable.
- `tincture`: oils and drops **unless** the product is a topical or massage oil.
- `extract` (RSO/FECO/hash oil) also emits `edible` as secondary.

### Confidence and ambiguity

`ClassifyCategories` returns a confidence signal. Low confidence when:

- multiple mutually exclusive forms match (smokable + vapeable, tincture +
  vapeable via concentrate)
- only the catch-all `other` matches
- a high-potency low-volume liquid cannot be pinned to vape vs tincture

Low confidence → `review_queue` row with reason `category_uncertain`. The
product still shows with its best-guess facet meanwhile; an analyst sets the
definitive value and it **sticks through reprocess** because it becomes an
override.

Keyword rules cannot cleanly separate a pure vape concentrate from a sublingual
tincture, or a legitimately-both "smoke or vaporize" herbal blend. The point of
the queue is that the system stops guessing silently.

---

## 5. Compliance — two tiers

`internal/compliance/filter.go`, driven by `harvest/rules/compliance.json`.
Word-boundary matching, not substring.

### Hard block (hidden, `in_stock = false`)

Genuine non-compliance: "cures cancer", "cures diabetes", "reverses <disease>",
"guaranteed high", explicit illegal-sale language.

### Soft review (visible, flagged)

Ordinary cannabis vocabulary: `marijuana`, `ganja`, `charas`, `weed`,
`psychoactive`, `get high`. These route to the review queue with reason
`terminology_review` and **still appear** in the catalogue.

> Hiding a product because its description contains the word "marijuana" is
> wrong for a cannabis-education catalogue. This exact bug hid CannEx Strong Plus.

### Review queue reasons

| Reason | Trigger |
|---|---|
| `unknown_brand` | brand not in the approved list — human verifies at ayush.gov.in |
| `price_anomaly` | ₹/mg < 0.10 or > 100 |
| `compliance_uncertain` | genuine medical/legal ambiguity |
| `terminology_review` | soft-tier vocabulary |
| `category_uncertain` | ambiguous form (§4) |
| `low_confidence` | below the publish gate (`03-DOMAIN-MODEL.md §2`) |

**Trusted aggregators** (ItsHemp, CBD Store India) auto-pass the brand check even
when the brand is not yet in our DB.

**The review queue is the most important operational tool.** Without humans
resolving it, new brands never enter the catalogue.

---

## 6. Best deal

Runs after every new listing. Finds the cheapest in-stock listing per cluster,
then computes:

- `best_price_paise`
- `cbd_price_per_mg` when `cbd_mg > 0`, else NULL
- `thc_price_per_mg` when `thc_mg > 0`, else NULL
- `best_price_per_mg` on the dominant basis (THC > CBD > total priority,
  `ADR-013`), for back-compat
- `value_tier` and `rank_score` (`03-DOMAIN-MODEL.md §5`)

Hemp seed, nutrition, topicals, and unknowns get NULL. Always.

---

## 7. Regression protection

Every fix above is locked by a test. Every human override auto-appends a golden
fixture. CI runs the whole set; a classifier version cannot ship unless it is
100% green.

**Every bug you hit becomes a numbered behaviour line in a future spec.** That
is how the rules get stronger as the project proceeds.

After any classifier change: rebuild, restart, `POST /admin/reprocess`, wait for
the River queue to drain (~1–2 min), then re-query and diff. The admin dry-run
diff (`06-ADMIN.md §2`) shows what a version would change *before* you ship it.
