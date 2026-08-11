# 05 — API Reference

> Base URL is one env var on the frontend: `VITE_API_URL`.
> Everything returns JSON in the envelope from `02-FRONTEND-CONTRACT.md §3`.
> Public `/api/*` GET endpoints need no auth.

---

## 1. Catalogue

| Method | Path | Notes |
|---|---|---|
| GET | `/api/products` | The catalogue. Params below. |
| GET | `/api/products/new` | Recently added, for the new-launches feed. |
| GET | `/api/products/{id}` | Full detail. Returns `{ moved_to }` if merged. |
| GET | `/api/compare` | Dense value-comparison payload, sorted client-side. |
| GET | `/api/brands` | All brands with verified flag + categories. |
| GET | `/api/brands/{slug}` | One brand. |

### `/api/products` parameters

| Param | Values |
|---|---|
| `category` | primary facet/category. Omitted → **excludes `pet` and `apparel`** |
| `form`, `route`, `extract`, `profile`, `carrier` | facet filters |
| `brand` | brand slug |
| `basis` | `cbd` \| `thc` — filters to products having it, scopes the value sort |
| `sort` | `value` \| `price` \| `new`. **Default `new`** when no `basis` given |
| `verified` | `true` → verified brands only |
| `cursor`, `limit` | keyset pagination on `(rank_score, id)` |

`sort=value` without `basis` is misleading — ₹/mg-CBD and ₹/mg-THC are not
comparable. Hence the `new` default.

### Product payload (essentials)

```json
{
  "id": "uuid",
  "name": "string",
  "short_description": "≤160 chars",
  "brand": { "slug": "…", "name": "…", "verified": true,
             "ayush": true, "fssai": false },
  "category": "tincture",
  "categories": ["tincture", "extract"],
  "facets": { "form": "oil_tincture", "route": "sublingual",
              "extract": "full_spectrum", "profile": "cbd_dominant" },
  "cbd_mg": 909.0,
  "thc_mg": null,
  "concentration_type": "cbd",
  "best_price_paise": 249900,
  "cbd_price_per_mg": 2.10,
  "thc_price_per_mg": null,
  "best_price_per_mg": 2.10,
  "price_per_mg_basis": "CBD",
  "value_tier": "good",
  "prescription_required": false,
  "in_stock": true,
  "image": { "thumb": "…", "card": "…", "full": "…",
             "width": 800, "height": 800, "blurhash": "…" },
  "listings": [ { "id": "uuid", "source_slug": "itshemp",
                  "price_paise": 249900, "url": "…", "in_stock": true } ],
  "updated_at": "2026-08-09T12:00:00Z"
}
```

Nullability is the contract. `null` means unknown; the frontend renders an
em-dash. **Never send `0` for unknown.**

---

## 2. Reference & editorial

| Method | Path | Notes |
|---|---|---|
| GET | `/api/states` | Legal status per state, DB-backed |
| GET | `/api/roa` | Routes-of-administration guide |
| GET | `/api/aggregators` | Aggregator directory |
| GET | `/api/forum/posts` | Published posts, paginated |
| GET | `/api/forum/posts/{slug}` | One post + metadata |
| GET | `/api/content/export` | **Build-time only.** Everything published, ETag'd |

`/api/content/export` is what `content:pull` consumes (`07-CONTENT-CMS.md`).

---

## 3. Community (auth required where noted)

| Method | Path | Auth |
|---|---|---|
| POST | `/api/auth/register` | — handle 3–24 chars, alphanumeric + `_` |
| POST | `/api/auth/login` | — returns JWT (15 min) + refresh token (30 d) |
| POST | `/api/auth/refresh` | — rotates: revoke old, issue new pair |
| POST | `/api/auth/logout` | — revokes the refresh token |
| GET | `/api/auth/me` | JWT |
| POST | `/api/auth/claim-token` | JWT — SHA-256 the purchase token, match, claim |
| GET | `/api/products/{id}/comments` | — paginated, newest first |
| POST | `/api/products/{id}/comments` | JWT — optional `rating` 1–5 |
| GET | `/api/forum/posts/{slug}/comments` | — |
| POST | `/api/forum/posts/{slug}/comments` | JWT |
| DELETE | `/api/comments/{id}` | JWT, own comments only |
| GET | `/api/survey/results` | — aggregate counts |
| POST | `/api/survey/response` | — rate-limited, stored as counts |
| POST | `/api/checkout/initiate` | — records click, issues token, returns redirect |

**Never return `account_id` in a public response.** Only `handle`.

After inserting a product comment, check whether the account holds a valid
purchase token for that cluster; if so set `verified_purchase = true`.

### Comment shape

```json
{ "id": "uuid", "handle": "someone", "body": "…",
  "verified_purchase": true, "rating": 4,
  "created_at": "ISO8601", "is_own": true }
```

---

## 4. Admin (never called by the public frontend)

Everything under `/admin/*`, behind the admin key plus TOTP. See `06-ADMIN.md`.

| Method | Path |
|---|---|
| POST | `/admin/reprocess` | re-run the pipeline over all raw products |
| GET/POST | `/admin/review-queue` | list, approve, reject, reject-brand |
| POST | `/admin/facets/override` | set an override + append a golden fixture |
| GET | `/admin/classifier/dryrun?version=N` | diff without shipping |
| GET | `/admin/sources/health` | per-store status, staging batches |
| GET | `/admin/analytics/clicks` | `?brand_slug=&source_slug=&days=30` |
| POST | `/admin/brands/{slug}/approve` | |
| CRUD | `/admin/content/*` | editor, revisions, publish, rollback |
| DELETE | `/admin/comments/{id}` | moderation |

`/admin/analytics/clicks` returns totals, `with_token` (conversion intent),
and breakdowns by brand, source, and day.

---

## 5. Cross-cutting rules

- **Envelope, error codes, status semantics** — `02-FRONTEND-CONTRACT.md §3–4`.
- **Keyset pagination**, always tie-broken by `, id ASC`.
- `ETag` + `Cache-Control: public, max-age=60, stale-while-revalidate=600` on
  catalogue reads.
- Rate limiting on every public endpoint; `429` carries `Retry-After`.
- Request ID and real-IP middleware for logs only — **IP is never persisted**.
- Correct status codes: 200 / 201 / 400 / 401 / 403 / 404 / 429 / 500 / 503.

## 6. CORS & origins

`Access-Control-Expose-Headers: ETag, Retry-After, X-Total-Count`.
`connect-src` on the frontend lists exactly one API host. Images go through
`/media/*` on that same host — **never a raw MinIO hostname**, which would widen
the CSP and break the frontend's no-external-requests promise.

If API and site can share an apex domain (`drtoke.in/api/*`), prefer it: most of
this section then disappears and `/media` becomes same-origin.

## 7. Media proxy

`GET /media/{hash}/{size}.{ext}` → streams from MinIO with
`Cache-Control: public, max-age=31536000, immutable`. Content-hashed filenames,
AVIF/WebP, sizes `thumb | card | full`. Dimensions and blurhash come back in the
product payload, not from the image endpoint.
