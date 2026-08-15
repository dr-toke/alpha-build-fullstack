# 10 — Decision Log

> Append-only. New decisions get a new ADR; superseded ones are marked, not
> deleted. History is useful.

---

## ADR-001 — Greenfield rewrite; harvest knowledge, discard code

**Status:** accepted (supersedes ADR-001a)

**Decision:** build a fresh backend repo. Before deleting anything, extract the
accumulated domain knowledge as **data files** — scraper selectors, cannabinoid
regexes, category keyword sets, compliance vocabulary, dedup fingerprint rules —
plus a golden fixture per known bug and a snapshot of the raw scraped data.
Procedure in `11-HARVEST.md`.

**Why:** the existing backend is alpha-quality throughout — structure, schema,
and pipeline reliability. Refactoring it would mean carrying prototype decisions
into the design in `01-ARCHITECTURE.md` while paying full rewrite cost anyway.

**What survives:** the knowledge, as configuration. Rules live in
`harvest/rules/*.json` and scrapers in `harvest/scrapers/*.yaml`, loaded at init.
A rule change becomes a reviewable data diff instead of a code change — which is
also what makes the review queue able to append to them automatically.

**What does not:** handlers, router, jobs, store layer, config, dashboard. All
regenerated from the specs in `docs/`.

**Cost control:** scraped data is re-scrapeable, so the DB is not a dependency —
but a snapshot is taken anyway as insurance and as a dev fixture. Three
selector-driven adapters replace ~14 hand-written scrapers.

### ADR-001a — Port the existing backend (superseded)

Originally accepted on the strength of `BACKEND_HANDOFF.md`, which described a
working, tested backend with 29 migrations and ~4,300 products. The project
owner's assessment is that the implementation is alpha across all layers, which
inverts the calculus: the tested-code argument for porting does not hold.

---

## ADR-002 — Facets replace the single category column

**Status:** accepted

The false-positive rate is a schema defect. One `category` column plus a
`categories[]` array forced form, route, and extract to be decided from the same
evidence pool; "no need to smoke or vape" therefore made a capsule vapeable.

**Decision:** orthogonal facets (`form`, `route`, `extract`, `profile`,
`carrier`, `purchasable`), each resolved independently, each carrying
provenance. Migration is additive with a dual-write window; the API keeps
emitting legacy `category` / `categories[]` so the frontend never breaks.

See `03-DOMAIN-MODEL.md §2`.

---

## ADR-003 — The model proposes; rules decide; humans overrule

**Status:** accepted

**Decision:** precedence is `override > rule > model > default`, enforced in one
function. No AI classification is ever written as an authoritative value; it
writes a proposal into the review queue. Human overrides are permanent and
auto-append a golden fixture that CI enforces.

**Consequence:** false positives become a queue length rather than a
data-quality problem, and error converges to zero over time.

---

## ADR-004 — Two content planes (build-time editorial, runtime data)

**Status:** accepted

The frontend is fully static with no SSR. Fetching editorial content at runtime
would mean no SEO, nothing readable with JS disabled (fatal on Tor at Safest), a
loading flash on every essay, and a reference site whose reference material
disappears when the API hiccups.

**Decision:** editorial content is pulled at build time and prerendered; only
genuinely live data (prices, comments, filters) is fetched at runtime. A
build-time catalogue snapshot gives the runtime pages a complete fallback.

See `01-ARCHITECTURE.md §3`.

---

## ADR-005 — CMS content is markdown in Postgres, generated into the frontend

**Status:** accepted

**Decision:** `content_docs` + append-only `content_revisions`; publish flips a
pointer. A Go CLI (`content:pull`) generates `content.generated.ts` / JSON into
the frontend checkout, and those generated files are **committed**.

**Why committed:** reproducible builds, diffable copy history, Netlify builds
that need no API access, and a working build when the backend is down.

**Open:** needs the frontend owner's buy-in on bot commits. Fallback is a
published package the frontend depends on.

---

## ADR-006 — Admin plane is HTMX + Go templates, not a second SPA

**Status:** accepted

**Decision:** server-rendered admin inside the backend binary. No build step, no
CORS, no token juggling, works over Tor, frontend repo untouched.

**Rejected:** a Svelte or React admin app. Admin work is queues and CRUD; an SPA
is the wrong weight and would couple two repos that should stay independent.

---

## ADR-007 — No cookies; token-in-memory auth

**Status:** accepted

The Constitution forbids cookies on the public site, so cookie sessions are out.

**Decision:** access JWT in memory only (15 min); refresh token in
`sessionStorage`, rotated every refresh (30 d). **Never `localStorage` for the
refresh token** — on a Tor-facing site, XSS persistence is unrecoverable.

The admin plane is the sole exception: a session cookie scoped to `/admin`, with
a separate signing key and users table, because it is not the public site.

---

## ADR-008 — Compliance splits into hard-block and review tiers

**Status:** accepted

Blocking any product whose description contains "marijuana" is wrong for a
cannabis-education catalogue. It hid legitimate products.

**Decision:** hard-block only genuine medical/illegal claims. Ordinary cannabis
vocabulary routes to the review queue with reason `terminology_review` and stays
visible. Word-boundary matching throughout.

---

## ADR-009 — ₹/mg is per-cannabinoid and basis-scoped

**Status:** accepted

Ranking per-mg-CBD against per-mg-THC against per-mg-total treats
non-comparable quantities as comparable.

**Decision:** store `cbd_price_per_mg` and `thc_price_per_mg` separately; add
`?basis=cbd|thc` which filters and scopes the value sort; default the catalogue
sort to `new` when no basis is chosen. `best_price_per_mg` stays for
back-compat. ₹/mg remains NULL for hemp seed, nutrition, topicals, unknowns.

---

## ADR-010 — Scrapes go to staging behind a promotion gate

**Status:** accepted

A store markup change that returns 40 products instead of 400 would otherwise
wipe 90% of a source silently.

**Decision:** scrapes write to staging; a gate compares against the last good
run and holds anything anomalous for human approval. Rejection alerts, never
overwrites.

---

## ADR-011 — No search server

**Status:** accepted

**Decision:** build-time JSON index plus client-side filtering for legality and
post search — hundreds of documents, works offline and on Tor, no extra origin.
Only product search touches the API.

---

## ADR-012 — `value_tier` bands are frontend-canonical

**Status:** accepted

`02-FRONTEND-CONTRACT.md` originally specified `<2 excellent / 2–5 good / 5–10
average / >10 premium`. The frontend's shipped `pmgColor()` already implements
different, live bands: `<3 green / 3–8 gold / >8 cream` — three tiers, not four.

**Decision:** the frontend's bands are canonical. The backend does not ship a
`value_tier` enum the frontend must adopt; instead `rank_score`'s `value_score`
component and any admin-side tier display are computed against the same
`<3 / 3–8 / >8` cutoffs, so "good value" means the same thing everywhere
without a frontend patch.

**Why:** the frontend is the already-designed, already-live surface. Changing
its bands is a design change; changing the backend's internal numbers to match
is not. Zero frontend risk beats a "more correct" four-tier scheme nobody asked
for.

**Consequence:** `02-FRONTEND-CONTRACT.md §9` and `03-DOMAIN-MODEL.md §5` are
corrected to the real numbers. Treat the old `<2/2–5/5–10/>10` table as dead.

---

## ADR-013 — Cannabinoid basis filter with `profile` sub-category; THC outranks CBD

**Status:** accepted

**Decision:** `/api/products` and `/api/compare` expose `?basis=cbd|thc`
(ADR-009) as the primary cannabinoid filter, with the `profile` facet
(`cbd_dominant | thc_dominant | balanced`, `03-DOMAIN-MODEL.md §2`) as its
sub-category. Wherever a single ₹/mg number must be chosen for a product
carrying both cannabinoids — `rank_score`'s dominant-basis fallback, the
compare table's `'best'` basis — the priority is **`THC > CBD > total`**,
reversing the previous `CBD > THC > total` order in `04-PIPELINE.md §6`.

**Why:** stakeholder direction — ₹/mg-THC is the most important value metric
for the target demographic.

**Scope, deliberately narrow:** this is a ranking preference within the
existing basis-filter machinery, not a change to the catalogue's default view.
ADR-009's reasoning (no `basis` chosen → default sort stays `new`, not
`value`) is unchanged, so the CBD/hemp majority of the actual catalogue is not
demoted or hidden by default. Widening this to a catalogue-wide default lens
needs its own ADR — see the compliance tension in `00-CONSTITUTION.md §1, §3`
(THC-bearing products are more often Rx-gated or grey-zone flagged, so
defaulting the whole site to a THC lens has legal-framing consequences beyond
ranking math).

**Consequence:** `04-PIPELINE.md §6` dominant-basis priority flips. No new
schema — `profile` and `basis` already existed; this wires them into the
public filter surface and states which wins.

---

## ADR-014 — Same apex domain, zero-trust internal segmentation

**Status:** accepted (closes `01-ARCHITECTURE.md §8` open item 1)

**Decision:** API and site share an apex domain (`drtoke.in/api/*`). Same-origin
removes most CORS complexity and lets `/media/*` serve same-origin.

Sharing a domain does **not** relax the internal trust boundary. The admin
plane stays fully segregated regardless of domain: its own `admin_users`
table, its own JWT/session signing key, TOTP, session cookie scoped to
`/admin` only, bound to localhost or a Tor hidden service, never reachable
from the public apex host (`06-ADMIN.md §3` is unchanged by this ADR, and is
now non-negotiable rather than a nice-to-have). Every internal service
credential (worker→DB, admin→DB, api→DB) is least-privilege and scoped to
what that service does — network adjacency (same domain, same box) is never
treated as authorization. CSP stays strict — `connect-src` still names exactly
one host, even though it's now the same host serving the site.

**Why:** convenience (one domain) and security (zero implicit trust from
co-location) are orthogonal; conflating them is how an admin panel ends up
reachable from a public subdomain by accident.

**Consequence:** `02-FRONTEND-CONTRACT.md §6` and `05-API-REFERENCE.md §6`
simplify (most of both sections describe cross-origin problems that no longer
exist), but `00-CONSTITUTION.md §4` and `06-ADMIN.md §3`'s isolation rules are
reaffirmed, not loosened.

---

## ADR-015 — Content and code licensing: creator-owned

**Status:** accepted

**Decision, two parts:**

1. **Content/IP.** Each piece of editorial content (`content_docs` /
   `content_revisions`, `03-DOMAIN-MODEL.md §11`) is copyright to its
   individual author, not assigned to Dr Toke. `content_revisions` gains a
   `license` field (free text or SPDX-style identifier — e.g.
   `all-rights-reserved` or `CC-BY-4.0`, author's choice) alongside the
   existing `author` column, and the frontend renders a byline + rights line
   on published posts.
2. **Repo code.** Both repos move off the current unset/all-rights-reserved
   default to an explicit `LICENSE` file naming their creators. This closes
   `10-DECISIONS.md` "Still open" item 4 as a *direction*; the actual license
   text/choice is stage-2 execution.

**Why:** stakeholder decision — the people who write the code and the copy
keep it.

**Consequence:** `07-CONTENT-CMS.md §3`'s export payload gains a `license`
field per doc. No change to the markdown/HTML safety discipline in that
section.

---

## ADR-016 — Confidence thresholds are AI-assisted, human-approved, versioned

**Status:** accepted

`03-DOMAIN-MODEL.md §2`'s confidence gate (`form.confidence >= 0.85`,
`route.confidence >= 0.90`) and the promotion gate's thresholds
(`04-PIPELINE.md §2`: 30% count drop, 15% null-field increase, 80%
selector-hit floor) were opening estimates, to be "tuned after the first
month, then frozen and versioned."

**Decision:** replace the one-time manual tuning pass with an ongoing
AI-assisted, human-in-the-loop process: threshold adjustments are proposed
from review-queue outcomes (false-positive/negative rates, queue-length
trends), a human approves, and the accepted set is versioned exactly like a
classifier version — never applied silently.

**Why:** consistent with ADR-003 (the model proposes, rules decide, humans
overrule) — this extends the same precedence to the thresholds themselves
rather than treating them as a separate, one-off calibration exercise.

**Consequence:** the AI proposing a threshold change is never authoritative,
same as the AI proposing a facet value. No new precedence rule needed — it's
the existing one, applied one level up.

---

## ADR-017 — Site imagery becomes CMS-managed content

**Status:** accepted

The frontend's hero/editorial imagery (`static/images/*.jpg` — sadhus,
shiva-bhang, shrine, thandai, etc.) currently ships as static files baked into
the frontend repo, with unverified rights (`skeleton-master/README.md`,
"Still open" item 6 below).

**Decision:** these images move into the same CMS plane as editorial copy
(`01-ARCHITECTURE.md §5`, `07-CONTENT-CMS.md`). An admin uploads/replaces them
through the admin panel; bytes live in MinIO behind `/media/*`; `content:pull`
writes the current image reference into the generated frontend files at build
time — same mechanism as `content_docs`, no runtime fetch, no new origin.

**Why:** stakeholder decision — imagery should be editable without a frontend
redeploy, same as copy. Rights-clearing itself is unchanged and deferred (see
"Still open" below); this ADR is about *where the asset lives and who can
change it*, not about clearing the current placeholders.

**Consequence:** `07-CONTENT-CMS.md` gains a media-asset section alongside its
content-doc pipeline. `06-ADMIN.md`'s content-editor surface (§1.1) extends to
image upload/replace, not just markdown.

---

## ADR-018 — Monorepo: `apps/web` + `apps/api` in one repository

**Status:** accepted (partially reverses the "separate repos" framing in
ADR-001 / `01-ARCHITECTURE.md §1` and the frontend-stack row of the
Superseded table below)

**Decision:** frontend (`skeleton`, brought in as-is) and backend live in one
repository as `apps/web` and `apps/api`. `docs/` stays at repo root, shared by
both. `harvest/`, `testdata/golden/`, `openapi/`, `SYMBOLS.md` move under
`apps/api/` — they're backend-only concerns, and each app in a monorepo should
own its own data rather than reach outside its directory tree.

**Why:** stakeholder decision — one repository, one clone, one PR history
across contract changes on both sides of the HTTP boundary.

**What does NOT change:** the architectural invariant in `00-CONSTITUTION.md
§5` — "frontend and backend are separate programs meeting only at HTTP/JSON
and generated types" — is about *coupling*, not *repository count*. `apps/web`
is still fully static, still builds independently, still renders with the API
down (`ADR-004` is untouched). `apps/api` still knows nothing about Svelte.
Co-locating the code does not mean co-locating the runtime — that boundary is
now enforced by discipline (no imports across the `apps/` boundary, contract
changes flow through `openapi/` codegen, not shared source) rather than by
physical repo separation. The frontend is still "already designed, not ours to
restyle casually" — that rule survives the repo merge unchanged; it was never
about which repo the code lived in.

**Consequence:** every doc reference to "the frontend is a separate
repository" (`README.md`, `01-ARCHITECTURE.md §1`, `02-FRONTEND-CONTRACT.md`
header) described the *pre-monorepo* state and has been corrected to match.

---

## ADR-019 — `internal/compliance` (M2) stays a placeholder until beta

**Status:** accepted — a scope note, not an implementation. No code written.
**Corrected 2026-08-11**, same day: the first version of this ADR only
deferred RBAC *depth* while still treating `filter.go`'s hard-block/review
logic as buildable now. That was a misreading — the placeholder covers the
whole package, not just an RBAC layer on top of it.

**Decision:** `internal/compliance` is **not built yet, at all** — not
`filter.go`'s `Evaluate` function, not `filter_test.go`, none of it. The
package stays a placeholder: at most a stub (a `role` field on
`admin_users`, M0, if a schema hook is needed elsewhere before M2 proper
starts) but not the actual hard-block/review-tier filter, even though its
rules are already harvested and sitting in `harvest/rules/compliance.json`
ready to use. Real compliance review — building `filter.go` for real, and
the RBAC/permissions work around who can act on its output — happens when a
beta build exists, not before.

**Why:** stakeholder direction — the project is pre-alpha. Compliance is
the one subsystem with real legal/reputational weight (it's what decides
whether a product description gets hidden or flagged), and building it
before there's a beta to review it against was explicitly called out as
premature.

**Consequence:** `08-BUILD-ORDERS.md §7`'s M2 is skipped for now.
`internal/ingest`'s pipeline stage list (`04-PIPELINE.md §1`) includes a
`compliance` step between `resolve` and `dedup` — when M4 is built before
M2 exists, that stage needs an explicit pass-through stub (everything
proceeds to `dedup` unfiltered) rather than silently omitting the stage,
so the gap is visible in the pipeline, not hidden by its absence.

---

## ADR-020 — Narrow exception to ADR-019: `service_listing` filtering built now

**Status:** accepted. Amends ADR-019, does not reverse it.

**Decision:** `internal/compliance` gets exactly one piece of the five
`harvest/rules/compliance.json` tiers implemented now — `service_listing`
(retreats/workshops/consultations/etc., word-boundary matched). Everything
else — `hard_block`, `terminology_review`, `price_anomaly`, `unknown_brand`
— stays exactly as deferred by ADR-019. This is not "start M2 early," it's
a scalpel cut of the single check needed to keep non-product listings out
of the pipeline.

**Why:** a real live scrape of `cbdstore.in` (M4) surfaced an actual
doctor-consultation booking sitting in the product catalogue — not a
hypothetical. Project owner's reaction was that leaving that visible in the
pipeline even pre-live was unacceptable, and the fix was small, already
harvested, and directly on point.

**A real finding changed the implementation from what the doc says:**
`harvest/rules/compliance.json` specifies `service_listing` as
`"matched_against": "product NAME only"`. Tested against the actual
motivating listing — its title contains none of the pattern's keywords at
all. What DOES carry the signal, confirmed by fetching the live listing
directly, is Shopify's `product_type` field ("Doctors Consultation"),
already flowing through the pipeline as `RawListing.CategoryRaw` /
`domain.RawProduct.CategoryRaw`. `compliance.Evaluate` checks both name and
category now, not name alone — a deliberate, evidence-based deviation from
the harvested doc, not a silent one. Whether name-only is right for other
stores' data is an open question for the full M2 build, not resolved here.

**Consequence:** `internal/compliance/filter.go` exists with a narrow
`Evaluate(rs *RuleSet, name, category string) Result` signature — not the
full `Evaluate(brandSlug, name, description string, priceINR, pricePMG
float64) Result` shape the eventual complete filter will need. Widening
that signature when the other four tiers get built is expected, normal
evolution, not a design mistake now. Wired into
`internal/ingest/classify_live_test.go`'s live demo, which now correctly
flags real service listings in real scraped data instead of using a
throwaway heuristic.

| Was | Now | Where |
|---|---|---|
| Next.js 15 + Turborepo monorepo frontend | SvelteKit 5 + Tailwind 4, `apps/web` in this monorepo (`ADR-018`) | `01-ARCHITECTURE.md §1` |
| Vercel hosting | Netlify (clearnet) | `09-OPS.md §5` |
| `packages/data/*.ts` as source of truth | Postgres; static TS was Phase-0 bootstrap | `03-DOMAIN-MODEL.md §6–7` |
| `SITE_TIER=clearnet\|onion` build flag | Same static bundle, different `VITE_API_URL` + feature gating | `09-OPS.md §5` |
| `next.config.ts` security headers | Backend-set headers + frontend CSP | `02-FRONTEND-CONTRACT.md §6` |
| Google Fonts | Self-hosted fonts (no external runtime requests) | `00-CONSTITUTION.md §4` |
| `next-intl` | `locale` column on `content_docs` | `07-CONTENT-CMS.md §7` |
| sqlc | raw SQL via pgx (already the implementation) | `00-CONSTITUTION.md §6` |
| `/products/[id]` dynamic route | `/product?id=` query param (static export requirement) | `02-FRONTEND-CONTRACT.md §1` |
| Single `category` column | Facets | ADR-002 |
| Single `best_price_per_mg` sort | Basis-scoped ₹/mg | ADR-009 |
| Hard-block on cannabis vocabulary | Two-tier compliance | ADR-008 |
| `forum_posts` table | `content_docs` with `kind='post'` | `03-DOMAIN-MODEL.md §11` |

---

## Still open

1. ~~Same apex domain for API and site?~~ **Resolved — ADR-014.**
2. **Generated files committed to the frontend repo** — needs the frontend
   owner's agreement. Now also covers generated image references (ADR-017).
3. **Onion/eepsite tier** — deferred to stage 2, per project owner
   (2026-08-11). Whether comments are clearnet-visible at all is unresolved.
4. ~~Licence on both repos.~~ **Direction resolved — ADR-015** (creator-owned).
   Actual license text/SPDX choice is stage-2 execution.
5. ~~Confidence thresholds~~ **Resolved — ADR-016** (ongoing AI-assisted,
   human-approved process replaces the one-time tuning pass).
6. **Image provenance** in the frontend's `static/images/`. Asset
   ownership/editability resolved (ADR-017); rights-clearing of the current
   placeholder photos deferred to stage 2, per project owner (2026-08-11).
7. **Zero-trust implementation detail.** ADR-014 sets the principle;
   per-service credential scoping, mutual TLS for any internal service-to-service
   traffic, and a threat-model pass are pre-launch execution, not yet done.
8. **`internal/compliance` (M2) unbuilt.** ADR-019: the whole package is a
   placeholder until a beta build exists, per project owner (2026-08-11) —
   not just RBAC depth. `filter.go` is skipped, not simplified. Any
   pipeline milestone built before it (M4's ingest stage list) needs an
   explicit pass-through stub where compliance would run, not a silent gap.
