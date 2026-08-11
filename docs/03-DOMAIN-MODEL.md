# 03 — Domain Model

> Read before writing a migration or a Go struct. Existing tables (29
> migrations) are described where they matter; new tables are given in full.

---

## 1. Entity map

```
sources ──< product_listings >── product_clusters ──< product_facets
                    │                    │           product_facet_overrides
                    │                    ├──< comments
                    │                    ├──< click_events
                    │                    └──< review_queue
                    └──< purchase_tokens

brands ──< product_clusters
accounts ──< comments, refresh_tokens, purchase_tokens(claimed)
content_docs ──< content_revisions
states / roa / aggregators   (reference, DB-backed)
```

**listing** = one row as a store presents it (one variant, one URL, one price).
**cluster** = the canonical product; many listings merge into one.
Value, ranking, facets, comments, and public URLs all attach to the **cluster**.

---

## 2. Facets — the fix for the category problem

One `category` column plus a `categories[]` array made unrelated signals collide.
Replace with orthogonal facets, each resolved independently.

| Facet | Values |
|---|---|
| `form` | `oil_tincture`, `capsule`, `edible`, `topical`, `flower`, `vape`, `concentrate`, `beverage`, `pet`, `apparel`, `accessory` |
| `route` | `sublingual`, `oral`, `inhaled`, `topical`, `transdermal` |
| `extract` | `full_spectrum`, `broad_spectrum`, `isolate` |
| `profile` | `cbd_dominant`, `thc_dominant`, `balanced` — **derived from mg ratio, never from text** |
| `carrier` | `mct`, `hemp_seed`, `olive`, `none` |
| `purchasable` | `true` / `false` — kills retreats, courses, consultations, merch |

A capsule that mentions vaping in negation can no longer be re-formed by ROA
evidence, because route no longer votes on form. That is the whole point.

```sql
CREATE TABLE product_facets (
  cluster_id         uuid  NOT NULL REFERENCES product_clusters(id) ON DELETE CASCADE,
  facet              text  NOT NULL,
  value              text  NOT NULL,
  source             text  NOT NULL,   -- override | rule | model | default
  confidence         real  NOT NULL,   -- 0..1; override is always 1.0
  evidence           jsonb NOT NULL,   -- matched spans, rule ids, negation windows
  classifier_version int   NOT NULL,
  decided_at         timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (cluster_id, facet)
);
CREATE INDEX product_facets_lookup_idx
  ON product_facets (facet, value) WHERE source <> 'model';

CREATE TABLE product_facet_overrides (
  cluster_id uuid NOT NULL REFERENCES product_clusters(id) ON DELETE CASCADE,
  facet      text NOT NULL,
  value      text NOT NULL,
  reason     text NOT NULL,
  set_by     text NOT NULL,
  set_at     timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (cluster_id, facet)
);
```

**Precedence is absolute:** `override > rule > model > default`. Enforced in one
function, `resolve.Facet()`, and nowhere else.

`evidence` stores matched span offsets, the rule IDs that fired, and any
negation windows applied. It is what makes the review queue possible and a
six-month-old bug debuggable.

### Overrides are permanent and become tests

Overrides apply **after** every pipeline run, unconditionally. Never recomputed,
never expired, never "refreshed by a better model."

Every override **auto-appends a fixture** to `testdata/golden/{cluster_id}.json`
— raw scraped title and description plus expected facet values.
`categories_test.go` runs the whole golden set in CI; a classifier version cannot
ship unless it passes 100%. This is how error converges to zero.

### Legacy compatibility

During migration the API keeps emitting `category` (primary) and `categories[]`
derived from facets:

```
category      := form value, mapped through the legacy name table
categories[]  := [form, …secondary forms, extract-derived tags]
```

Pet and apparel remain **exclusive** — if `form ∈ {pet, apparel}` that is the
only category returned. `extract` (RSO/FECO/hash oil) also emits `edible` as a
secondary, as today.

### Confidence gate

```
publishable := facets.purchasable
            AND form.confidence  >= 0.85
            AND (route IS NULL OR route.confidence >= 0.90)
            AND price_paise > 0
```

Route gets the higher bar because it is the safety-relevant field. Anything
below the gate goes to review, not to `/api/products`. Queue length, not error
rate, is the honest measure of data quality.

---

## 3. Cannabinoid content

Stored on the cluster. All nullable — **never zero for unknown**.

| Column | Notes |
|---|---|
| `cbd_mg` | numeric, null if unknown |
| `thc_mg` | numeric, null if unknown |
| `total_cannabinoids_mg` | numeric, null if unknown |
| `concentration_type` | `cbd` \| `thc` \| `total` \| `hemp_seed` \| `nutrition` \| `unknown` |
| `cannabinoid_confidence` | real |
| `cannabinoid_evidence` | jsonb |

**₹/mg is computed only for `concentration_type IN ('cbd','thc','total')`.** For
hemp seed, nutrition, topicals, and unknowns it stays NULL and the frontend
shows "n/a". This is a Constitution rule, not a preference.

---

## 4. Cluster identity is a public URL

The frontend uses `/product?id=` and comments attach to clusters, so cluster IDs
are load-bearing infrastructure.

- **Durable UUID assigned on first sight and stored.** Never a recomputed hash
  of the dedup fingerprint — one normalisation tweak would silently break every
  inbound link and orphan every comment thread.
- `cluster_merges(old_id, new_id, merged_at)`. `GET /api/products/{old}` returns
  `{ "moved_to": "<new_id>" }` so the frontend rewrites instead of 404ing.
- **Splits**: comments stay with the original ID; an admin reassigns explicitly.
  Never guess — a misattributed review is worse than a stale one.

---

## 5. Pricing

| Column | Notes |
|---|---|
| `best_price_paise` | int64, cheapest in-stock listing |
| `best_price_per_mg` | dominant basis, kept for back-compat |
| `cbd_price_per_mg` | `best_price / cbd_mg` when `cbd_mg > 0`, else NULL |
| `thc_price_per_mg` | `best_price / thc_mg` when `thc_mg > 0`, else NULL |
| `price_per_mg_basis` | label for the single-cannabinoid fallback |
| `value_tier` | computed server-side against the frontend's own bands — `<3` good, `3–8` mid, `>8` high (`ADR-012`) |
| `rank_score` | see below |

Partial indexes on both new ₹/mg columns for `sort=value&basis=…`.

### Ranking is editorial, not arithmetic

Pure ₹/mg crowns whatever product had its mg misparsed *high*.

```
rank_score = value_score
           × facet_confidence
           × brand_trust      (verified / Ayush-registered lift)
           × completeness     (has image, has COA, has full dosage)
```

When a product carries both cannabinoids and `value_score` needs one number
(no `basis` param given), the dominant-basis priority is **`THC > CBD > total`**
(`ADR-013` — stakeholder priority for the target demographic; reverses the
`CBD > THC > total` order this doc originally specified).

---

## 6. Brands

Carried from `packages/data/brands.ts`, now DB-backed. Static TS files are
**Phase-0 bootstrap only and are not the source of truth.**

| Field | Notes |
|---|---|
| `slug`, `name`, `full_name`, `founded`, `city`, `state`, `url` | |
| `description` | 2–3 sentences, consumer perspective |
| `verified` | editorial verification, drives the pending-verification badge |
| `ayush`, `fssai` | booleans; `ayush_reg_number`, `fssai_licence` |
| `coa_available` | do they provide a COA on request |
| `affiliate_url` | null if no programme |
| `last_verified` | date; shown to users so they know data freshness |

**Never show Ayush/FSSAI badges unless confirmed.** Ground-truth seed list and
trusted aggregators are in `09-OPS.md §6`.

---

## 7. Reference content (DB-backed, self-checking)

**States** — `name`, `slug`, `status ∈ (legal|tolerated|grey|limited|illegal)`,
`bhang_shops`, `detail`, `excise_url`, `notes`, `featured`, `last_verified`.
Delhi NCR is pinned via `featured`.

**ROA** — five routes with onset, duration, bioavailability, pros, cons,
best-for. The "edibles delay" golden-rule callout and the bhang thandai festival
dosing warning are content fields, not hardcoded frontend strings.

**Aggregators** — directory of ItsHemp, CBD Store India, CannaMeds India, etc.

---

## 8. Community

```sql
accounts(id, handle, handle_lower GENERATED, password_hash, created_at,
         last_seen_at, banned, ban_reason)
-- NO email, NO phone, NO name, NO IP.

refresh_tokens(id, account_id, token_hash, expires_at, created_at, revoked)

comments(id, account_id, post_id?, cluster_id?, body CHECK 10..1000,
         verified_purchase, purchase_token_id, rating CHECK 1..5,
         deleted, deleted_by_admin, created_at)
-- exactly one of (post_id, cluster_id) is non-null
```

Handles are 3–24 chars, alphanumeric plus underscore. Argon2id hashing.
No password reset — there is no email to reset to, and that is fine.

**Public comment shape returns `handle`, never `account_id`.**

## 9. Survey

Aggregate counters only: `extract_type`, `use_case`, `price_range`,
`carrier_oil`, `total_responses`, `last_updated`. Individual responses are never
stored. Re-taking is allowed.

## 10. Click events

`listing_id`, `cluster_id`, `brand_slug`, `source_slug`, `page_path`,
`filter_context`, `token_issued`, `clicked_at`. **No IP, no user agent, no
account ID.** Indexed for GROUP BY on cluster, brand, source, and day.

## 11. Content documents

```sql
content_docs(id, kind, slug, locale DEFAULT 'en', status, current_revision_id,
             UNIQUE (kind, slug, locale))
content_revisions(id, doc_id, title, body_md, frontmatter jsonb, author,
                  license, created_at, published_at)
```

`license` is free text or an SPDX-style identifier (e.g. `all-rights-reserved`,
`CC-BY-4.0`), author's choice. Content is copyright to its individual author,
not assigned to Dr Toke (`ADR-015`) — the frontend renders a byline + rights
line on published posts from these two columns.

`kind ∈ post | section_block | era | topic | state_note | concept | faq |
glossary`. Revisions are append-only; publish flips the pointer, rollback flips
it back. `locale` ships now populated only with `en`, so Hindi later is a data
task rather than a migration.

Legacy `forum_posts` folds into `content_docs` with `kind='post'`; comments keep
pointing at the doc ID.
