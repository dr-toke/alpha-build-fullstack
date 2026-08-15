# QA Report — drtoke-v03.netlify.app / Railway API

Full QA pass against the live production deployment: frontend
`https://drtoke-v03.netlify.app`, API `https://api-production-12683.up.railway.app`.
Everything below was checked against the real deployed system, not local —
API responses via direct HTTP, pages via Playwright, database state via
direct psql against Railway's Postgres.

## Fixed this pass

### 1. Brand linkage was broken catalog-wide (data bug, fixed)

**Symptom:** `?brand=<slug>` returned 0 products for every real brand,
including brands that clearly had products showing on the site. Every
product's brand badge showed unverified/no-Ayush/no-FSSAI, even for brands
`docs/09-OPS.md §6`'s ground-truth table confirms are verified (Cure By
Design, Noigra, BOHECO, etc).

**Root cause:** when the corrected local dataset was bulk-copied to Railway
via `pg_dump`/`psql`, the `brands` table was seeded independently and got
fresh UUIDs. `product_clusters.brand_id` still carried the old local UUIDs,
which don't exist in Railway's `brands` table — a silently orphaned foreign
key on 396 clusters across 10 brands (Cure By Design 96, Noigra 61, Health
Horizons 57, BOHECO 45, Ananta Hemp Works 32, The Trost 27, Hempbuti 23,
Wholeleaf 21, Indie Extracts 20, Hemp Tribe 14).

**Fix:** re-ran the real pipeline (`ingest.Promote`'s per-product path, not
hand-written SQL) against Railway's database, scoped to just the affected
raw products via a temporary `ingest.PromoteSubset` export (added, used,
reverted). This re-resolves `brand_id`, `brand_trust`, and `rank_score`
through the actual `resolveBrand`/`buildClusterShell` code path — the same
code that runs on every real scrape — rather than a one-off SQL patch that
could drift from the app's real matching semantics.

### 2. Nutrition category (and any pure-nutrition/no-cannabinoid category) showed 0 products by default

**Symptom:** `/products?category=nutrition` showed "0 products" under the
default sort. `sort=new` and `sort=price` correctly showed 64 real,
in-stock hemp protein/hemp-hearts products.

**Root cause:** `sort=value` is the API's default sort
(`internal/api/products.go`). Its query had `AND rank_score IS NOT NULL` —
deliberately excluding rows with no computable ₹/mg (hemp seed oil, protein
powder, anything with zero cannabinoid content has nothing to rank by
value). That's reasonable for the *sort*, but because it's also the
*default*, any category made up entirely of such products — Nutrition — was
completely invisible on first load, with no indication anything was wrong.

**Fix:** `internal/store/clusters.go` — dropped the hard `WHERE` exclusion,
changed the ORDER BY to `rank_score DESC NULLS LAST, id ASC`. Unranked
products now appear, sorted after every ranked product, instead of vanishing
entirely. `cbd`/`thc` basis-scoped value sort (an explicit user choice, not
the silent default) is untouched — excluding rows with no CBD/THC price
there is still correct. Existing test
`internal/store/clusters_test.go::TestListClusters` updated to assert the
new (correct) behavior; full suite passes.

### 3. Price/₹-per-mg block overflowed its card on mobile (frontend CSS bug, fixed)

**Symptom:** on the product detail page (and, same underlying pattern, the
product grid cards), the ₹/mg price breakdown on the right side of the price
block visually stuck out past the white card's edge and got clipped by the
viewport on narrow (390px) screens — confirmed with a real product ("The
Trost - Stress Buster..."), both via screenshot and by measuring the DOM:
the price text's right edge (x=385) sat outside `<main>`'s padded content
area (right edge x=358).

**Root cause:** classic flexbox overflow. `flex items-end justify-between`
with a left `<div>` holding unwrapped text (`"30ml · cheapest at
cbdstore"`) and no `min-width` constraint. Flex items default to `min-width:
auto`, meaning a text-holding flex child won't shrink below its content's
natural width unless told to — so on a narrow viewport the left block kept
its full intrinsic width and pushed the right (₹/mg) block outside the
card.

**Fix:** `apps/web/src/routes/product/+page.svelte` and
`apps/web/src/lib/sections/products/ProductCard.svelte` (identical pattern,
same fix, found the same shape by inspection after confirming the first one
— ProductCard's grid layout is narrower still, so it carried the same risk)
— added `min-w-0 flex-1` to the left block so it can shrink/wrap, `shrink-0`
to the right block so it holds its size, `gap-2` so they never touch when
both are at minimum width. No visual change on desktop/normal widths where
there was already room; only affects the cramped case. `pnpm check` clean,
0 errors.

## Investigated, not a bug

- **`/products?page=999` (out-of-range page):** returns 0 cards with a
  clean "No products" message. Handled correctly.
- **`/products?brand=not-a-real-slug`:** returns "0 products" — correct,
  same contract as an unknown category slug (valid filter, empty match, not
  an error).
- **Comments 404 on product detail pages:** `GET
  /api/products/{id}/comments` 404s in the browser console. Expected —
  community features (comments/survey/checkout) are explicitly not built
  per the README's "Current state" section. The frontend already degrades
  gracefully ("No reviews yet. Be the first.") — no visible breakage, just
  a benign console error from a documented gap.
- **CORS:** real frontend origin gets `Access-Control-Allow-Origin` back
  correctly; a fake origin gets none. Locked down as intended.
- **API edge params:** negative `limit`/`page`, huge `limit` (999999,
  clamped to 100), malformed UUID on `/api/products/{id}` (400, and the
  frontend shows "Product not found" rather than crashing) — all handled
  safely.
- **"Verified brands only" checkbox showing 0 results:** expected, not a
  bug — no admin panel exists yet to run brand verification workflow beyond
  the 12 ground-truth rows already seeded (and now correctly linked, see
  fix #1 above).

## Known limitation, not fixed (design/scope, not a bug)

- **Vendor-baked image badges collide with the "Out of stock" badge.** Some
  vendors (Hemplife, confirmed via direct image fetch) bake their own
  star-rating graphic into the top-right corner of their product photos.
  Dr Toke's own "Out of stock" badge is also positioned top-right
  (`ProductCard.svelte`), so for out-of-stock products from these vendors
  the two overlap and both become hard to read. Root cause is the
  already-documented hotlinked-image limitation (README: "raw scraped image
  URLs are hotlinked directly, not proxied through MinIO") — Dr Toke has no
  control over vendor image content. Didn't reposition the badge unilaterally
  since that's the frontend designer's placement choice, not a functional
  bug; flagging for a call on whether to move it (e.g., bottom-left) or
  leave it.

## Not yet checked

Nothing outstanding — homepage/nav, all product filters/sort/pagination,
product detail pages (including edge cases), all other routes, API-level
checks, and mobile/responsive were all covered this pass.

## Deploy status

- Backend fix (Nutrition/value-sort) needs a Railway redeploy to take
  effect in production (code change, not yet pushed at time of writing this
  report — see final summary message for current status).
- Brand-linkage repair was applied directly against Railway's live database
  (data fix, no redeploy needed for that half — it's already live).
