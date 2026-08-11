# 06 — Admin Plane

> **Go templates + HTMX, server-rendered, inside the backend binary.** No second
> SPA: no build step, no CORS, no token juggling, works over Tor, and it keeps
> the frontend repo untouched. Admin work is queues and CRUD; HTMX is exactly
> the right weight.
>
> Bound to localhost or a Tor hidden service. **Never exposed on the clearnet
> API host.** Separate users table, separate signing key, TOTP required, session
> scoped to `/admin`. Once the API and public site share an apex domain
> (`ADR-014`), this isolation is non-negotiable, not a nice-to-have — same
> domain does not mean same trust boundary.

An existing self-contained `dashboard.html` served by the Go API is the
precedent; this formalises and extends it.

---

## 1. Surfaces, in build order

### 1. Content editor — **build first**

Markdown pane, live preview rendered by the *same* Go markdown pipeline that
validates the export, revision diff, publish / rollback, inline build status,
and a "publish triggers rebuild" button.

Image upload/replace lives here too (`ADR-017`, `07-CONTENT-CMS.md §6`) — same
revision/publish/rollback model as copy, not a separate surface.

First because it unblocks writers immediately and proves the two-plane split end
to end.

### 2. Review queue — the heart

Photo, raw scraped text with **matched evidence spans highlighted**, the
proposal, and the diff vs. current.

Keyboard-driven: `j`/`k` to move, number keys to assign, `a`/`r` to
approve/reject. Target **under 3 seconds per item.**

Every action writes a facet override **and** appends a golden fixture
(`04-PIPELINE.md §7`).

Filter by reason: `unknown_brand`, `price_anomaly`, `compliance_uncertain`,
`terminology_review`, `category_uncertain`, `low_confidence`.

Resolving must **stick through reprocess** — that is what the override table is
for. If a resolution can be undone by a re-scrape, the wiring is wrong.

### 3. Classifier dry-run diff

"v14 changes 143 products; here they are, grouped by facet."

This is what makes rule improvement fearless instead of terrifying. Build it
early — it pays for itself the first time.

### 4. Source health

Per store: last success, count delta, selector-hit rate, error rate, staging
batches awaiting promotion, one-click adapter re-test, manual promote/reject.

### 5. Value outliers

₹/mg below p1 or above p99, null mg where mg was expected, ₹0 prices. Auto-catches
the entire ₹0.28/mg bug class without anyone remembering to look.

### 6. Brands

Verify / unverify, alias merging, Ayush and FSSAI registration numbers with
links to the official portals, `last_verified` bump, affiliate configuration.

Approving a brand releases everything queued under `unknown_brand` for it.

### 7. Moderation

Comments, forum, reports, ban-by-handle, soft-delete with `deleted_by_admin`.

Pseudonymity means moderation is the **only** lever available. It cannot be
deferred to "later".

### 8. Click analytics

Totals, `with_token` rate (conversion intent), by brand, by source, by day.
No IP, no user agent, no account linkage — the privacy design is the schema
(`03-DOMAIN-MODEL.md §10`).

### 9. Audit log

Who, what, when, before, after. Append-only. **Non-negotiable.**

Every override, publish, brand approval, moderation action, and manual promotion
writes a row.

---

## 2. Operational loop

```
scrape lands → gate holds anything suspicious → source health shows it
     ↓
review queue drains (overrides + fixtures written)
     ↓
rules improved → dry-run diff inspected → version shipped
     ↓
CI golden set stays 100% green
     ↓
POST /admin/reprocess → queue drains → verify
```

Queue length is the honest measure of data quality. Watch it, not an error rate.

---

## 3. Security

- Own `admin_users` table. **Never** the public `accounts` table.
- Argon2id + TOTP. Session cookie scoped to `/admin` only — this is the one
  place cookies are permitted, because it is not the public site.
- Separate JWT signing key from the public tier.
- Rate-limited, bound to localhost or a Tor hidden service.
- Admin key in env, never committed, rotated on any suspicion.
- Every mutating action is audited before it is applied.
