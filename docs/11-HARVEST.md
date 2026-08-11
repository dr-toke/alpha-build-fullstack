# 11 — Harvest

> **Do this before deleting anything.** The old backend's code is disposable.
> The knowledge inside it is not — it is the accumulated result of scraping ~14
> real stores and debugging their output against live data.
>
> Rule: **extract knowledge as data, not as code.** A regex in a JSON file
> survives a rewrite. A regex inside a handler does not.
>
> Target: one focused day. Nothing gets deleted until every box here is ticked
> and committed to the new repo.

---

## 1. What is actually valuable

| Asset | Why it can't be regenerated |
|---|---|
| Scraper selectors per store | Found by hand against 14 live sites, each with its own quirks |
| Cannabinoid regexes + priority order | Seven-step order was derived from real failures |
| Category keyword sets & exclusions | `"joint"` → joint pain, pet/apparel exclusivity, form-over-extract |
| Negation windows | The "no need to smoke or vape" class of bug |
| Compliance vocabulary lists | Hard-block vs terminology tiers, split after real over-blocking |
| Dedup fingerprint | Whatever normalisation actually merges duplicates across stores |
| Known-bug fixtures | Every bug already paid for once |
| Brand → source mapping | Which aggregator carries which brand |
| Human review decisions | If any exist in `review_queue` / approvals |

Everything else — handlers, router, jobs, store layer, config, dashboard — is
regenerated from the specs in `docs/`. Do not carry it across.

---

## 2. Harvest checklist

### 2.1 Scrapers → `harvest/scrapers/<source>.yaml`

For each of the ~14 sources, one file:

```yaml
slug: itshemp
base_url: https://itshemp.in
platform: woocommerce          # shopify | woocommerce | custom
trusted_aggregator: true       # auto-passes the brand check
listing:
  index_url: /shop/?paged={page}
  item: "li.product"
  next_page: "a.next"
detail:
  name: "h1.product_title"
  price: "p.price .amount"
  description: ".woocommerce-Tabs-panel--description"
  image: ".woocommerce-product-gallery img"
  variants: "form.variations_form"     # per-variant URL, see quirks
  in_stock: ".stock"
quirks:
  - "each size is a separate row sharing one URL — must build per-variant URLs"
  - "price element appears twice; take the last"
notes: "rate limit ~1 req/2s, blocks on faster"
```

**The `quirks` list is the single most valuable field in this entire harvest.**
Write down everything you remember, even if it sounds trivial.

### 2.2 Cannabinoid rules → `harvest/rules/cannabinoids.json`

```json
{
  "priority": [
    { "id": "explicit_cbd_mg", "pattern": "…", "confidence": 0.95 },
    { "id": "explicit_thc_mg", "pattern": "…", "skip_if": "thc_free" },
    { "id": "ratio_with_total", "pattern": "…" },
    { "id": "percent_plus_volume", "pattern": "…" },
    { "id": "percent_thc_volume", "pattern": "…" },
    { "id": "hemp_seed_nutrition", "pattern": "…" },
    { "id": "generic_mg_fallback", "pattern": "…", "confidence": 0.5 }
  ],
  "thc_free_markers": ["…"],
  "thc_dominant_markers": ["THC-dominant", "high THC", "THC tolerance"],
  "negation_markers": ["no need to", "without", "not for", "alternative to"]
}
```

Copy the regexes **verbatim**, including the escaping. Do not "clean them up"
while transcribing — that is how a fix gets silently undone.

### 2.3 Category rules → `harvest/rules/categories.json`

```json
{
  "forms": {
    "smokable": { "include": ["flower","bud","pre-roll","hash"],
                  "exclude": ["joint"] },
    "vapeable": { "include": ["vape","cart","distillate","dab"] },
    "tincture": { "include": ["oil","drops","tincture"],
                  "unless_any": ["massage","topical","balm"] }
  },
  "exclusive": ["pet","apparel"],
  "secondary_implications": { "extract": ["edible"] },
  "concentrate_markers": ["not diluted with carrier oil","distillate","dab"],
  "precedence": "form over extract"
}
```

### 2.4 Compliance lists → `harvest/rules/compliance.json`

```json
{
  "hard_block": ["cures cancer","cures diabetes","reverses","guaranteed high"],
  "terminology_review": ["marijuana","ganja","charas","weed",
                         "psychoactive","get high"],
  "matching": "word_boundary",
  "trusted_sources": ["itshemp","cbdstore","cannameds"]
}
```

### 2.5 Dedup fingerprint → `harvest/rules/dedup.md`

Plain prose is fine. What gets normalised (case, punctuation, unit spellings,
brand prefix stripping), what is compared, what the match threshold is, and any
known false-merge or false-split cases.

### 2.6 Golden fixtures → `testdata/golden/*.json`

Every bug already fixed becomes a fixture. Start with these, from
`04-PIPELINE.md`:

- [ ] **CannEx Strong Plus** — `(10:1 THC:CBD) 1000mg | 1ml` → `thc_mg ≈ 909`,
      `cbd_mg ≈ 91`, THC-dominant, vapeable/concentrate, visible,
      `terminology_review`
- [ ] A capsule whose copy says "no need to smoke or vape" → form `capsule`,
      route `oral`, **not** vapeable
- [ ] A `50% THC` product → parses as 50, not 0
- [ ] A `0.3% THC` product → parses as 0.3, not 0
- [ ] A Shopify multi-size listing → per-variant URLs, no ₹0.28/mg artefact
- [ ] A "joint pain" balm → topical, **not** smokable
- [ ] A hemp-seed nutrition product → `concentration_type=hemp_seed`, ₹/mg NULL
- [ ] A pet CBD oil → `pet` only, excluded from the default catalogue
- [ ] A retreat / course / consultation listing → `purchasable=false`
- [ ] A product blocked purely for containing "marijuana" → visible, flagged

Format:

```json
{ "source": "itshemp",
  "raw": { "title": "…", "description": "…", "price": "…" },
  "expect": { "form": "concentrate", "route": "inhaled",
              "profile": "thc_dominant", "thc_mg": 909, "cbd_mg": 91,
              "review_reason": "terminology_review" },
  "regression_note": "ratio parsed backwards before 2026-06" }
```

Each fixture gets a `regression_note` naming the bug it locks. Six months from
now that sentence is why nobody "simplifies" the rule.

### 2.7 Data snapshot

```bash
docker compose exec -T postgres pg_dump -U drtoke -d drtoke \
  --data-only --table=product_listings --table=brands \
  --table=review_queue > harvest/snapshot.sql
```

Data is re-scrapeable, so this is insurance, not a dependency. But take it
anyway — it costs a minute and it's your only record of what the sites looked
like on this date.

**Do dump:** raw listings, brands, review-queue decisions, brand approvals.
**Do not carry across:** clusters, facets, prices, images. Those are derived and
the new pipeline recomputes them from scratch — that's the point.

### 2.8 Reference content → `harvest/reference/`

- `states.json` — the state legal grid with every `last_verified` date and
  excise URL
- `roa.json` — the five routes with onset, duration, bioavailability, pros,
  cons, best-for; the edibles-delay golden rule; the bhang thandai festival
  warning
- `aggregators.json` — the directory
- `brands.json` — the twelve verified brands with Ayush/FSSAI status and source

Some of this is in the old `packages/data/*.ts`, some in the DB. Take whichever
is more current and note which.

### 2.9 Anything else you remember

`harvest/NOTES.md`, free-form. Rate limits, sites that block, seasonal stock
weirdness, brands that renamed, a store that changed platform mid-year. Nobody
will remember this in three months, and none of it is written down anywhere else.

---

## 3. Verification before deletion

- [ ] Every one of the ~14 sources has a `harvest/scrapers/*.yaml`
- [ ] All four rule files transcribed **verbatim**, regexes unmodified
- [ ] At least the ten fixtures above exist in `testdata/golden/`
- [ ] `harvest/snapshot.sql` taken and restorable
- [ ] Reference content exported
- [ ] `harvest/NOTES.md` written while it's still in your head
- [ ] The old repo is archived (not deleted) on a branch or a tag

Only then start M1.

---

## 4. How harvest feeds the build

`harvest/` is **input data**, not code. The new implementation loads these files
rather than hardcoding their contents:

- `internal/resolve` loads `rules/*.json` at init — rule changes become data
  changes, reviewable in a diff, with no recompile
- `internal/ingest/scrapers` loads `scrapers/*.yaml` — a selector fix is a
  one-line PR, and a new store is a new file, not new code
- `testdata/golden/` is enforced by CI from day one

This is why the rewrite is worth doing. In the old design these lived inside Go
files and every change was a code change. Here they're configuration, and the
review queue appends to them automatically (`03-DOMAIN-MODEL.md §2`).
