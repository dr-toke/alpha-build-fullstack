# dr-toke-skeleton

The SvelteKit frontend for **Dr Toke** — a harm-reduction and reference site about
cannabis in India: its history, the pharmacology, the law (NDPS), and a price-per-mg
product catalogue. Svelte 5 (runes) + Tailwind 4, built to a **fully static** bundle
that any dumb file host can serve.

> **The backend is a separate repo and you don't need it.** `pnpm install && pnpm dev`
> runs standalone. Every page renders with clear "service offline" states until an API
> is pointed at it, so you can work on layout, copy, and components with nothing else
> running.

---

## Quick start

**Prerequisites:** Node **≥ 22** and pnpm **10.28**. The `packageManager` field pins
pnpm, so the easiest route is Corepack:

```bash
corepack enable          # once, gets you the pinned pnpm
```

```bash
git clone https://github.com/dr-toke/skeleton.git
cd skeleton
pnpm install
pnpm dev                 # → http://localhost:5173
```

| Script | What it does |
|---|---|
| `pnpm dev` | Dev server with HMR |
| `pnpm check` | `svelte-check` — TypeScript + a11y. **Run before every PR; must be 0 errors.** |
| `pnpm build` | Static site → `build/` |
| `pnpm preview` | Serve the production build locally |

Copy `.env.example` to `.env` if you want to point at a live API:

```bash
VITE_API_URL=http://localhost:8080   # the default when unset
```

---

## Routes

Two visual worlds live in this repo and **they are deliberately different.** Don't
unify them.

| Route | Look | What it is |
|---|---|---|
| `/` | pixel/white | Front page: wordmark, vision block, photo altarpiece, stacked section words. **No nav bar — the page is its own navigation.** |
| `/history` | pixel/white | Eras: roots → courts & mystics → the 1894 Commission → the ban → the world since → tools & rituals |
| `/science` | pixel/white | Plant, extraction, decarb, routes-of-administration primer + live ROA guide |
| `/legality` | pixel/white | NDPS explainers, timeline, penalties, live searchable state grid, FAQ, glossary |
| `/parcha` | pixel + iridescent | Our own thing — discreet paper concepts. Nothing for sale. |
| `/products` | **dark catalog** | The catalogue: filters held in the URL, ₹/mg ranking |
| `/product?id=` | **dark catalog** | Product detail, buy flow, comment thread |
| `/compare` | **dark catalog** | Dense sortable ₹/mg table |
| `/brands` | **dark catalog** | Verified brand directory + aggregators |
| `/forum`, `/forum/post?slug=` | **dark catalog** | Discussions, markdown posts, threaded comments |
| `/survey`, `/survey/results` | **dark catalog** | Anonymous survey flow and aggregate results |
| `/account` | **dark catalog** | Pseudonymous account, purchase-token claim, your contributions |

The dark pages are an **exact port of the original `apps/web`** — markup, classes, copy,
and behaviour, down to the Cormorant Garamond / DM Sans and gold-on-green tokens.
Don't restyle them casually.

---

## Structure

```
src/
├── app.css                    # fonts, BOTH theme worlds, catalog tokens (has a DO NOT TOUCH block)
├── routes/                    # thin pages — they compose sections, they don't hold logic
└── lib/
    ├── api/                   # all backend plumbing
    │   ├── client.ts          # apiFetch + ApiError, reads VITE_API_URL, injects the JWT
    │   ├── remote.svelte.ts   # Remote<T> — the data-fetching primitive (see below)
    │   ├── auth.svelte.ts     # pseudonymous session store (handle + password, no PII)
    │   ├── catalog.ts         # product/brand types + ₹/mg helpers
    │   ├── reference.ts       # states / roa / aggregators types
    │   ├── markdown.ts        # dependency-free markdown → safe HTML (escapes first)
    │   └── purchase.ts        # pending purchase-token stash
    ├── components/
    │   ├── layout/            # Disclaimer (every page), Nav, Footer
    │   └── ui/                # PageHead
    ├── ui/reveal.ts           # scroll-reveal
    └── sections/              # ONE FOLDER PER SECTION — this is where the work happens
        ├── home/              # Wordmark, Vision, SectionIndex, PhotoBand, Marks, AquaIcon
        ├── history/           # content.ts + EraBlock
        ├── science/           # content.ts + TopicBlock + RoaLive
        ├── legality/          # ndps.ts, extras.ts, LegalSubnav, StateGrid, StateCard, Archives
        ├── products/          # CatalogGrid, ProductCard, CompareTable, BuyButton,
        │                      #   BrandsGrid, BrandCard, Aggregators, Tag, CatalogShell
        ├── community/         # AuthModal, CommentThread, SurveyFlow, VerifiedBadge, survey.ts
        └── parcha/            # content.ts + ConceptCard (visuals drawn in CSS)
```

Import aliases: `$lib` → `src/lib`, `$sections` → `src/lib/sections`.

### Two things to understand before you write code

**1. Fetching data — use `Remote<T>`, never bare `fetch`.**

```svelte
<script lang="ts">
  import { Remote } from '$lib/api/remote.svelte';
  const remote = new Remote<ProductListResponse>();
  $effect(() => remote.load(`/api/products?${query}`));
</script>

{#if remote.loading}   …skeleton…
{:else if remote.error} …service-offline state…
{:else}                 …render remote.data…
{/if}
```

It handles stale responses, starts in `loading` so you never flash an empty state, and
routes errors through `ApiError`. **Always render all three branches** — the offline
state is a feature, not an edge case.

**2. Copy lives in `content.ts`, not in components.**

Every editorial section keeps its writing in a plain data file — `history/content.ts`,
`science/content.ts`, `parcha/content.ts`, `legality/ndps.ts` + `extras.ts`,
`community/survey.ts`. Components render that data. If you're writing prose, you should
be in a `.ts` file, not a `.svelte` one. Spots still needing real copy render a
`✎ to write:` chip in the UI.

---

## Conventions

- **The disclaimer strip renders on every page and is not dismissible.** Don't touch it.
- **The site never sells anything or takes a transaction** — `/parcha` included, concepts only.
- **No PII, no cookies, no tracking.** Accounts are a handle and a password, nothing else.
- External links always get `rel="noopener noreferrer"`.
- **Fully static.** No server-side runtime, no API routes. If it can't prerender, it
  ships as a client-rendered shell (`export const ssr = false`).
- TypeScript everywhere; `pnpm check` must pass clean.
- Tabs for indentation, single quotes — see `.editorconfig`.

## Fonts & images

| Face | Used for |
|---|---|
| Cyberwave 2000 | the `dr. toke` wordmark |
| Pixelta | headings and pixel accents |
| Ubuntu Sans | body text on the pixel/white pages |
| Cormorant Garamond + DM Sans | the dark catalog pages only |

All fonts are self-hosted in `static/fonts/` — **no external requests at runtime**
(CSP- and Tor-friendly). Keep it that way. Licences ship beside them as
`*-EULA.txt`; read those before any commercial launch.

Shipped images live in `static/images/` and are referenced as `/images/…`. Uncropped
originals and working files go in `.drafts/`, which is gitignored and never published.

> **Note:** the photographs in `static/images/` are working placeholders with unverified
> provenance. Clear their rights, or replace them, before this site goes public.

## Contributing

```bash
git switch -c your-branch
pnpm check && pnpm build     # both must pass
git commit && git push -u origin your-branch
```

Then open a PR against `master`. Keep each section's work inside its own
`src/lib/sections/<section>/` folder where you can — that's the whole point of the layout.

## Not built yet

- Static pages: about, privacy, terms, content policy
- i18n (EN/HI)
- No automated tests yet — `pnpm check` is the only gate yet
- No `LICENSE` file — the repo is currently all-rights-reserved by default
