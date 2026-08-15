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

This is the target topology. Until a VPS is bought, §5a below is the interim
path — same binary, same env-var contract, no rework when moving to a VPS.

---

## 5a. Interim beta hosting (pre-VPS)

For showing shareholders a working beta before buying a VPS. No Docker: Go
compiles to one static binary, so a platform's native buildpack (Nixpacks)
builds it the same way `go build` does locally — `apps/api/nixpacks.toml`
tells it which of the monorepo's three `cmd/` binaries to build. This is the
same bare-binary shape §5 already commits to for the VPS, not a different
architecture to migrate away from later.

**Railway** (recommended — simplest Go + managed Postgres combination):

1. Create a project, add a **PostgreSQL** plugin — Railway provisions it and
   exposes `DATABASE_URL` as a service variable automatically.
2. Add a service → **Deploy from GitHub repo** → `dr-toke/alpha-build-fullstack`.
   It's private, so Railway needs its GitHub App granted access to it — if
   the `dr-toke` org isn't already authorized, Railway's repo picker prompts
   for that during this step (GitHub org owner has to approve it; a repo
   collaborator alone usually can't). Set **Root Directory** to `apps/api`
   (monorepo — Railway/Nixpacks otherwise has no way to know which `cmd/`
   package is the API). It will find `nixpacks.toml` there and use its build/
   start commands.
3. Service variables to set:
   - `DATABASE_URL` — reference the Postgres plugin's variable (don't
     hand-copy it; referencing keeps it in sync if it ever rotates).
   - `ALLOWED_ORIGIN` — the deployed Netlify URL: `https://toke-v02.netlify.app`.
     CORS rejects everything else (`internal/api/router.go`'s single-origin
     check).
   - `PORT` — Railway injects this itself; the server already reads it
     (`cmd/server/main.go`), no action needed.
4. Run migrations against the new database — from a local machine, not
   inside the container (Railway's Postgres plugin exposes a public
   connection string in its dashboard, under the plugin's **Connect** tab,
   for exactly this):
   ```bash
   goose -dir apps/api/internal/db/migrations postgres "<railway-database-url>" up
   ```
5. Deploy. `GET https://<service>.up.railway.app/healthz` should return
   `{"status":"ok"}` once the build finishes. That URL is what goes into
   Netlify's `VITE_API_URL` below.
6. Nothing scrapes on a schedule (no job runner deployed — see the main
   README's "Current state"). The database is empty until something calls
   `ingest.StageBatch` → `DecideGate` → `Promote` against it, same as local
   dev. Cheapest path for a first beta: run that from a local machine
   pointed at the Railway `DATABASE_URL`, same shape as the local Setup
   section, just with the Railway connection string instead of a local
   container.

**Netlify — repointing the existing `toke-v02.netlify.app` site:**

That site currently builds from the old standalone `dr-toke/skeleton` repo.
The frontend now lives at `apps/web` inside `dr-toke/alpha-build-fullstack`
instead — the site's build source needs to move with it:

1. Netlify dashboard → the `toke-v02` site → **Site configuration** →
   **Build & deploy** → **Continuous deployment** → **Link a different
   repository** (or **Change repository**, wording varies by Netlify UI
   version). Pick `dr-toke/alpha-build-fullstack`. Netlify's GitHub App
   needs access to that repo the same way Railway's does — grant it if
   prompted.
2. **Base directory**: `apps/web`. **Build command**: `pnpm install && pnpm build`.
   **Publish directory**: `apps/web/build` — confirmed against
   `apps/web/svelte.config.js`'s `adapter-static` config (`pages: 'build',
   assets: 'build'`), not a guess. No `netlify.toml` in this repo, so these
   three settings live entirely in the Netlify dashboard.
3. Environment variable: `VITE_API_URL` = the Railway `https://....up.railway.app`
   URL from step 5 above.
4. Trigger a deploy. Once it's live, go back to Railway and set
   `ALLOWED_ORIGIN` to `https://toke-v02.netlify.app` if it isn't already
   (step 3 above) — CORS is bidirectional, both sides need the other's
   real URL, not a placeholder.

**Render** (alternative to Railway): same shape — a **PostgreSQL** instance
(free tier available) for `DATABASE_URL`, a **Web Service** pointed at
`dr-toke/alpha-build-fullstack` with **Root Directory** `apps/api`, Nixpacks
auto-detected the same way, same three env vars, same `goose ... up`
migration step run from a local machine against Render's external database
URL.

**Fly.io** (alternative): `fly launch` from `apps/api/` auto-detects Go via
its own buildpacks (no Dockerfile needed here either); attach a Postgres
app with `fly postgres create` and `fly postgres attach`; same env vars,
same migration step.

Whichever platform: the promotion pipeline (`internal/ingest.Promote`) isn't
wired to a scheduler yet (no River job runner deployed this pass — see
`SYMBOLS.md`'s ingest section) — running a scrape+promote cycle against the
deployed database is a manual step for now, from a local machine pointed at
the deployed `DATABASE_URL`, same as the migration command above.

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
