# 02 — Frontend Contract

> The frontend (`apps/web` in this monorepo — brought in as-is from the
> original `dr-toke/skeleton` repo, `ADR-018` — SvelteKit 5 + Tailwind 4,
> static, live at `toke-v02.netlify.app`) is **already designed**. It is not
> ours to restyle casually just because it now shares a repo with the backend.
> This document is what the backend owes it — functionally and aesthetically.
>
> Rule of thumb: the frontend does **no** data cleaning, no price maths, no
> categorisation. It receives finished, ranked, priced products and draws them.

---

## 1. Frontend facts the backend must respect

| Fact | Consequence for us |
|---|---|
| Fully static build, no SSR, no API routes | Editorial content must be pushed at build time, not fetched at runtime (`01-ARCHITECTURE.md §3`) |
| `Remote<T>` client-side fetch, three branches (loading / error / data) | Status codes are semantic, not decorative (§4) |
| Detail routes use query params: `/product?id=`, `/forum/post?slug=` | Cluster IDs are permanent public identifiers (`03-DOMAIN-MODEL.md §4`) |
| Copy lives in `content.ts` per section | We **generate** those files, not replace the convention (`07-CONTENT-CMS.md`) |
| Dependency-free `markdown.ts` that escapes first | Send markdown, never HTML |
| Self-hosted fonts, no external runtime requests | Exactly one extra origin: the API. Images proxy through it |
| No cookies anywhere | Token-in-memory auth (§7) |
| `pnpm check` is the PR gate | Generated TypeScript types make contract drift a build failure |
| Disclaimer strip on every page | Never returned by the API; it is frontend chrome |
| Two visual worlds: pixel/white editorial + dark catalogue | Same API, two renderings — no per-theme fields |

---

## 2. Type generation

Emit OpenAPI from Go annotations → generate TypeScript into
`src/lib/api/generated/`. Contract drift then fails `pnpm check`.

This is the **highest-leverage single item in this repo**. Hand-mirroring Go
structs against `catalog.ts` fails silently on a casing change and costs an
afternoon to find.

---

## 3. Envelope and errors

```json
{ "data": [ ... ], "page": 1, "limit": 24, "total": 3812, "has_more": true }
```

```json
{ "error": { "code": "rate_limited", "message": "Too many requests." } }
```

Stable machine codes so the frontend's `ApiError` can branch:
`not_found`, `moved`, `rate_limited`, `unavailable`, `invalid_filter`,
`auth_required`, `auth_invalid`, `banned`, `validation_failed`.

---

## 4. Status semantics are load-bearing

| Situation | Response |
|---|---|
| No results for this filter | `200` + `{ "data": [], "total": 0 }` |
| Backend not ready / degraded | `503` + `Retry-After` |
| Rate limited | `429` + `Retry-After` + code `rate_limited` |
| Merged product | `200` + `{ "moved_to": "<uuid>" }` |
| Unknown product | `404` + code `not_found` |

Empty-array and 503 are **different states**. Conflating them turns "nothing
matches your filters" into a scary offline banner.

---

## 5. Pagination and caching

- **Keyset pagination** on `(rank_score, id)`, never `OFFSET`. A scrape landing
  mid-browse must not show the user duplicates.
- **Always tie-break with `, id ASC`** so the grid doesn't reshuffle between loads.
- `ETag` + `Cache-Control: public, max-age=60, stale-while-revalidate=600` on
  catalogue reads. Free 304s matter enormously over Tor.

## 6. CORS and CSP

- `Access-Control-Expose-Headers: ETag, Retry-After, X-Total-Count` — without
  this the client cannot see them cross-origin.
- CSP: `connect-src` lists exactly one API host; `img-src` the same host via the
  `/media/*` proxy. Raw MinIO hostnames would widen the policy and break the
  no-external-requests promise the frontend is built around.
- If API and site can share an apex domain (`drtoke.in/api/*`), most of this
  section disappears. Prefer that.

## 7. Auth without cookies

- **Access JWT in memory only** (15 min expiry). Never persisted.
- **Refresh token in `sessionStorage`**, rotated on every refresh, 30-day TTL.
- **Never `localStorage` for the refresh token.** On a Tor-facing site, XSS
  persistence is the one failure you cannot walk back.
- Purchase tokens in `localStorage.dt_tokens` are fine — they are opaque,
  single-purpose, and carry no account authority.
- Public responses return `handle` only. **Never** `account_id`.

---

## 8. What the backend owes the design

Each of these is a contract term, not a nicety. Most backends sabotage good
frontends by getting exactly these wrong.

| Rule | Why |
|---|---|
| `null`, never `0`, for unknown mg or ₹/mg | `0` renders "₹0.00/mg" and looks broken; `null` lets them render an em-dash |
| Match `pmgColor()`'s live bands (`<3 / 3–8 / >8`) in every server-side ranking computation | `ADR-012` — the frontend's bands are canonical; the backend aligns to them, not the reverse. No `value_tier` enum is required of the frontend |
| Images as objects: `{ thumb, card, full, width, height, blurhash }` | Dimensions kill layout shift; blurhash gives the dark catalogue a graceful load at zero JS weight. AVIF/WebP, content-hashed, immutable |
| Cap `short_description` at ~160 chars server-side | Truncation belongs where the canonical text lives, not in a card component |
| Markdown, never HTML | Their renderer escapes first. Don't undermine it |
| Stable sort tie-break | No reshuffling between loads |
| `price_paise` as `int64` | Never float currency. Format client-side |
| ISO 8601 UTC timestamps always | No local time, no epoch ints |
| `updated_at` on every catalogue payload | Powers "prices last updated" on the stale-snapshot path |
| Emit both facets **and** legacy `category` / `categories[]` during migration | The frontend never breaks mid-refactor |

---

## 9. Display intelligence the API must enable

These are frontend rules, but they only work if the API supplies the right
fields. Carried forward from the previous implementation brief — they are
correct and must not be lost.

### ₹/mg

Show **only** when non-null. The API returns `null` for hemp seed, nutrition,
topicals, and unknowns — `concentration_type` outside `('cbd','thc','total')`.
When null the frontend shows "n/a", never a number.

Colour bands — **frontend-canonical** (`ADR-012`). `pmgColor()` already ships
these; the backend does not hand the frontend a `value_tier` to render, it
computes its own ranking against the same numbers:

| Band | ₹/mg | Colour (`pmgColor()`) |
|---|---|---|
| best | < 3 | `--green2` |
| mid | 3–8 | `--gold` |
| high | > 8 | `--cream2` (no special treatment) |

When a product has both cannabinoids, the API returns **both**
`cbd_price_per_mg` and `thc_price_per_mg` so the card can show
`₹2.10/mg CBD · ₹6.40/mg THC`. `price_per_mg_basis` remains the
single-cannabinoid fallback label.

### Basis-aware comparison

`?basis=cbd|thc` filters to products having that cannabinoid and sorts
`value` by `{cbd|thc}_price_per_mg ASC NULLS LAST`. **Cross-basis value ranking
is misleading** — ₹/mg-CBD and ₹/mg-THC are not comparable quantities. With no
basis chosen, the catalogue default sort is `new`, not `value`.

### Categories

Multi-category products: **first category is primary** (large coloured badge),
the rest are small grey tags. Filter by primary only, or the same product
appears several times.

Colour map (frontend token references, kept here as the canonical mapping):

| Category | Border token |
|---|---|
| `tincture` | `--gold` |
| `smokable` | `--red2` |
| `vapeable` | `--green2` |
| `edible` | `--gold3` |
| `topical` | `--cream2` |
| `extract` | `--gold` |
| `nutrition` | `--green3` |
| `beverage` | `--green2` |
| `pet` | `--cream2` |

### Mandatory badges

- **Smokable warning** — if primary *or* secondary category is `smokable`:
  amber border, non-dismissable, on both card and detail page. Legal grey zone
  in India; Ayush-licensed extracts are protected, raw flower is not.
- **Rx badge** — `prescription_required === true`: red border, shown above the
  buy button. The buy button still works (links to a licensed platform).
- **Pending verification** — brand scraped but not editorially verified: amber
  badge plus "Compliance not yet confirmed by Dr Toke editorial team". Product
  still shown, buy button still works. Transparent, not paternalistic.
- **Ayush / FSSAI badges** — only when confirmed. Each carries a "how to verify"
  link to the official portal.

### Default catalogue exclusions

`/api/products` with no `category` param excludes `pet` and `apparel` from both
the primary column and the `categories` array, giving a clean human catalogue.

---

## 10. Purchase token flow (end to end)

1. User clicks Buy on a listing.
2. Frontend `POST /api/checkout/initiate` with `listing_id`.
3. API records a click event, issues a token, returns `{ token, redirect_url }`.
4. Frontend appends the token to `localStorage.dt_tokens[]`.
5. Frontend opens `redirect_url` in a new tab (`noopener,noreferrer`).
6. User later registers or logs in.
7. On login, the frontend replays stored tokens to `POST /api/auth/claim-token`.
8. Claimed tokens mark that account's comments on that product
   `verified_purchase = true`.

**On API error the buy button falls back to the raw `source_url`.** The user can
always reach the store; verification is the bonus, not the gate.

---

## 11. Design tokens (canonical)

The frontend owns its styling, but these are the brand tokens of record. If the
admin plane or any generated content needs colour, it uses these names.

```
bg #080f08   bg2 #0c1a0d   bg3 #132015   bg4 #1a2e1c
gold #c8a84b gold2 #e0c878 gold3 #a07828
green #3a7a2f green2 #5aaa4a green3 #2a5a20
cream #ede4d0 cream2 #b0a080 red2 #e07a5a

display: Cormorant Garamond   body: DM Sans
spring: cubic-bezier(0.22, 1, 0.36, 1)
```
