# Dr Toke

India's consumer guide to the legal cannabis economy.
**Not a dispensary. Not a brand. PC Mag for Vijaya.**

**Monorepo** (`docs/10-DECISIONS.md#adr-018`):

- `apps/api/` — the **Go API, data pipeline, admin plane, and CMS**.
- `apps/web/` — the **SvelteKit 5 + Tailwind 4 frontend**, deployed at
  `toke-v02.netlify.app`. Brought in **as-is** from its original standalone
  repo (`dr-toke/skeleton`) and **already designed and built** — this is not
  ours to restyle casually just because it now lives in the same repo as the
  backend. The backend is built *around* it, not the other way round.

One repo, two independently-buildable apps meeting only at HTTP/JSON and
generated types (`00-CONSTITUTION.md §5`). See `apps/web/README.md` for
frontend-specific dev commands (`pnpm dev`, `pnpm check`, etc.) — they're
unchanged by the move into this repo.

---

## Read in this order

| # | Doc | Read it when |
|---|---|---|
| 00 | [`docs/00-CONSTITUTION.md`](docs/00-CONSTITUTION.md) | **First, always.** Non-negotiable legal, privacy, and security rules. |
| 01 | [`docs/01-ARCHITECTURE.md`](docs/01-ARCHITECTURE.md) | Before any structural decision. The system design and its reasoning. |
| 02 | [`docs/02-FRONTEND-CONTRACT.md`](docs/02-FRONTEND-CONTRACT.md) | Before touching anything the frontend consumes. |
| 03 | [`docs/03-DOMAIN-MODEL.md`](docs/03-DOMAIN-MODEL.md) | Before writing a migration or a struct. |
| 04 | [`docs/04-PIPELINE.md`](docs/04-PIPELINE.md) | Before touching scrape / normalise / compliance / dedup / bestdeal. |
| 05 | [`docs/05-API-REFERENCE.md`](docs/05-API-REFERENCE.md) | Writing or changing a handler. |
| 06 | [`docs/06-ADMIN.md`](docs/06-ADMIN.md) | Working on the admin plane. |
| 07 | [`docs/07-CONTENT-CMS.md`](docs/07-CONTENT-CMS.md) | Working on editorial content or the build pipeline. |
| 08 | [`docs/08-BUILD-ORDERS.md`](docs/08-BUILD-ORDERS.md) | Delegating file-by-file work to a model. |
| 09 | [`docs/09-OPS.md`](docs/09-OPS.md) | Running, deploying, or debugging locally. |
| 10 | [`docs/10-DECISIONS.md`](docs/10-DECISIONS.md) | "Why is it like this?" / "Can I change X?" |
| 11 | [`docs/11-HARVEST.md`](docs/11-HARVEST.md) | **Before deleting the old repo.** Knowledge extraction checklist. |

`apps/api/SYMBOLS.md` is a running ledger of exported Go signatures. Append
after every merged file; it is the paste-source for build orders.

---

## The one-paragraph version

Dr Toke aggregates products from ~14 Indian hemp/cannabis stores, cleans the
messy data into one consistent catalogue, works out the **true value (₹ per mg
of cannabinoid)** of each product, and ranks them. It also publishes editorial
and legal-reference content. The backend does all data work and all publishing;
the frontend only draws JSON and prerendered content. The two are fully
decoupled and can be redesigned or replaced independently.

---

## Current state

**Greenfield rewrite, informed by a prior alpha.** An earlier implementation
reached alpha (Go/Chi/pgx/River, ~29 migrations, ~4,300 scraped products) but is
being replaced rather than refactored — it is alpha across structure, schema, and
pipeline reliability.

What carries over is **knowledge as data**: scraper selectors for ~14 stores,
cannabinoid extraction rules, category keyword sets, compliance vocabulary, the
dedup fingerprint, and a golden fixture for every bug already fixed once.

**Start here:** [`docs/11-HARVEST.md`](docs/11-HARVEST.md). Nothing from the old
repo gets deleted until that checklist is complete.

See [`docs/10-DECISIONS.md#adr-001`](docs/10-DECISIONS.md).

## Quickstart

```bash
docker compose up -d                                   # Postgres + MinIO
cd apps/api
goose -dir internal/db/migrations postgres "$DATABASE_URL" up
go run ./cmd/server              # API on :8080
go run ./cmd/worker              # River job runner
```

```bash
cd apps/web
pnpm install
VITE_API_URL=http://localhost:8080 pnpm dev             # frontend on :5173
```

Full detail in [`docs/09-OPS.md`](docs/09-OPS.md).

---

## Legal

Educational content only. Not a licensed medical or legal service.
Private repo. See `docs/00-CONSTITUTION.md` before publishing anything.
