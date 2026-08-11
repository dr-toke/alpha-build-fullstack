# 09 — Operations

---

## 1. Local development

```bash
docker compose up -d          # Postgres 16 + MinIO
cp .env.example .env          # fill it in — see §3
goose -dir internal/db/migrations postgres "$DATABASE_URL" up
go run ./cmd/server           # API      :8080
go run ./cmd/worker           # River job runner
# Admin: http://localhost:8080/admin
```

Seed a dev database from the harvest snapshot so you're working against real
messy listings rather than invented fixtures:

```bash
psql "$DATABASE_URL" -f harvest/snapshot.sql
go run ./cmd/reprocess        # run the pipeline over the restored raw listings
```

Direct DB inspection:

```bash
docker compose exec -T postgres psql -U drtoke -d drtoke -c "<sql>"
```

Frontend, separately (`ADR-018` — `apps/web` in this same repo now, not a
sibling checkout):

```bash
cd apps/web
VITE_API_URL=http://localhost:8080 pnpm dev
pnpm check && pnpm build      # the PR gate
```

---

## 2. After a classifier or pipeline change

```
go build ./... && go vet ./... && go test ./...
restart the API
GET  /admin/classifier/dryrun?version=N      # inspect BEFORE shipping
POST /admin/reprocess                        # admin key required
# wait ~1–2 min for the River default queue to drain
re-query and diff
```

API-only changes need no frontend rebuild — it fetches live. Content changes do
(`07-CONTENT-CMS.md §2`).

---

## 3. Environment variables

```bash
# Database & storage
DATABASE_URL=
MINIO_ENDPOINT=
MINIO_ACCESS_KEY=
MINIO_SECRET_KEY=

# Auth (public tier)
JWT_SECRET=
JWT_ACCESS_TTL=15m
REFRESH_TTL=720h

# Admin plane — separate key, separate secret
ADMIN_KEY=
ADMIN_JWT_SECRET=
ADMIN_TOTP_ISSUER=drtoke

# Publishing
NETLIFY_BUILD_HOOK=
FRONTEND_CHECKOUT=../web       # apps/api -> apps/web, same repo (ADR-018)

# Serving
ALLOWED_ORIGINS=
PUBLIC_MEDIA_BASE=
```

Never committed. `.env.example` carries keys with empty values only.

---

## 4. Production checklist

**Before launch:**

- [ ] Postgres backups configured **and a restore tested**. An untested backup
      is not a backup.
- [ ] Monitoring + alerting on: pipeline job failures, promotion-gate
      rejections, review-queue length, API 5xx rate.
- [ ] Rate limiting verified on every public endpoint. This touches a legally
      sensitive topic in India — expect scraping and probing beyond the usual.
- [ ] Secrets in a real secret store, not a `.env` on the box.
- [ ] Admin plane unreachable from the clearnet API host. Verify from outside.
- [ ] TLS + HSTS on the API domain; CSP verified end-to-end against the
      deployed frontend.
- [ ] **Legal review of the compliance rules themselves** — not just the
      engineering around them. Someone with actual NDPS knowledge signs off on
      the hard-block list.
- [ ] Image provenance cleared. Frontend `static/images/` placeholders replaced;
      provenance tracked per asset from day one.
- [ ] Every `CREATE TABLE` re-read to confirm no PII column crept in.
- [ ] Old repo archived on a tag, and `11-HARVEST.md §3` verified complete.
- [ ] `LICENSE` decided on both repos.

**Ongoing:**

- Watch review-queue length, not error rate. It is the honest data-quality metric.
- Re-verify brand Ayush/FSSAI registrations on a schedule; bump `last_verified`.
- Re-verify state legal status on a schedule; laws change and the site displays
  the date.
- Rebuild cadence: every publish triggers a full static rebuild. Fine at dozens
  of posts. Revisit if publishing many times a day or if the catalogue snapshot
  starts slowing builds.

---

## 5. Deploy topology

| Component | Where |
|---|---|
| Frontend (clearnet) | Netlify, static |
| Frontend (community) | Tor hidden service / I2P eepsite, same bundle, different `VITE_API_URL` |
| API | VPS, single Go binary + River worker |
| Postgres 16 + MinIO | Same VPS or managed |
| Admin | Same binary, bound to localhost / Tor only |

---

## 6. Ground truth: verified brands

Do not hallucinate brands or add unverified data. Every row was manually
verified; re-verify on schedule and update `last_verified`.

| Brand | URL | Ayush | FSSAI | Source |
|---|---|---|---|---|
| BOHECO | boheco.com | ✅ MP | ✅ | boheco.com/about |
| Cannazo India | cannazoindia.com | ✅ | ❌ | cannazoindia.com |
| Cure By Design | curebydesign.in | ✅ | ✅ | curebydesign.in |
| Indie Extracts | indieextracts.com | ✅ | ❌ | itshemp.in |
| Ananta Hemp Works | itshemp.in/brand/ananta-hemp-works/ | ❌ | ✅ | itshemp.in |
| Awshad | itshemp.in/brand/awshad/ | ✅ | ❌ | itshemp.in |
| Health Horizons | itshemp.in/brand/health-horizons/ | ❌ | ✅ | itshemp.in |
| The Trost | thetrost.com | ✅ | ❌ | thetrost.com |
| Hemp Tribe | itshemp.in/brand/hemp-tribe/ | ❌ | ✅ | itshemp.in |
| Noigra | itshemp.in/brand/noigra/ | ✅ | ❌ | itshemp.in |
| Wholeleaf | itshemp.in/brand/wholeleaf/ | ✅ | ❌ | itshemp.in |
| Hempbuti | itshemp.in/brand/hempbuti/ | ❌ | ✅ | itshemp.in |

**Trusted aggregators** (auto-pass the brand check in compliance):
ItsHemp (itshemp.in), CBD Store India (cbdstore.in), CannaMeds India (cannameds.in).

Verification portals: `ayush.gov.in`, `foscos.fssai.gov.in`, NABL database for COAs.
