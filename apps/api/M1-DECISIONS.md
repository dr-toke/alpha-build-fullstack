# M1 decisions — internal reasoning, classified

**Date:** 2026-08-11. Companion to `SYMBOLS.md`'s resolve section. This is
not a rationale summary after the fact — it's the actual reasoning behind
each real decision made while writing `internal/resolve`, kept at the
granularity of "why this and not that," for a second model or a second pass
to interrogate.

---

## Classification legend

Every decision below is tagged with one of five classes. The class is the
important part — it tells you how much scrutiny the decision deserves and
what kind of scrutiny:

| Class | Meaning | How to review it |
|---|---|---|
| **SPEC'D** | A doc states this exactly; no judgment involved | Check I transcribed it correctly, nothing more |
| **PORTED-VERBATIM** | The prior alpha's real, working Go code; carried over unchanged | Check the port is faithful — diff against the harvested source |
| **PORTED-ADAPTED** | Same source logic, mechanically restructured to load from `RuleSet` instead of package-level `var` | Check the mechanical transform didn't change behavior (the golden/cannabinoid/facet tests do this) |
| **INFERRED** | A doc states a requirement or behavior but not an exact shape; I chose one reasonable shape | Check the reasoning, consider alternatives |
| **NEW** | No precedent anywhere — doc or harvested source. A decision I introduced because the system doesn't work without one | Highest scrutiny — these are opinions, not transcriptions |

---

## `evidence.go`, `text.go` — NEW, low stakes

`Evidence`/`Span`/`Merge` and `Normalize`/`Tokens`/`NegationWindows` have no
precedent — the harvested source never separated "what matched" from "what
the match implies" into a reusable type; it just returned final values.
**NEW**, but low-consequence: these are plumbing types with a small, obvious
surface (a match, a strip-and-report function), not design decisions with
alternatives worth debating. `Tokens` in particular is unused by anything
else in M1 (see its own doc comment) — included because `08-BUILD-ORDERS.md
§7` names it as an export, kept honest about not being load-bearing yet
rather than inventing a caller for it.

## `ruleset.go` — one real decision, one mechanical one

**Decision: `LoadRuleSet` reads `cannabinoids.json` and `categories.json`
only, not `compliance.json` or `dedup.md`.** — **INFERRED.**
`08-BUILD-ORDERS.md §7` literally says "reads `harvest/rules/*.json`"
(all of them). I scoped it down because `internal/compliance` (M2) is a
separate package per `01-ARCHITECTURE.md §6`'s stated package boundaries,
and having M1's loader own M2's rules would be a real coupling the repo
layout doesn't call for. Alternative not taken: a shared, package-agnostic
`internal/rules` loader both `resolve` and `compliance` import. Rejected for
now as unrequested scope — M2 doesn't exist yet, so there's nothing to prove
that shape is needed rather than assumed.

**Everything else in this file — compiling `harvest/rules/*.json`'s literal
patterns and word lists into `*regexp.Regexp`, `wordSet()`'s exact
compilation shape** — **PORTED-ADAPTED.** `wordSet()` is character-for-
character the harvested source's own helper (`dr-toke-init/.../categories.go`).

## `match.go` — PORTED-ADAPTED

`MatchWordBoundary` and `ApplyNegation` aren't new logic, they're the
harvested source's `detectForms()` and `stripNegations()` behavior
factored into named, testable primitives instead of inline calls. The
two-pattern order in `ApplyNegation` (primary negation pattern, then the
form-free pattern) is **PORTED-VERBATIM** — same order as
`stripNegations()`.

## `cannabinoids.go` — mostly PORTED-VERBATIM, one NEW addition

The control flow — branch order, ratio orientation, per-serving
reconciliation, name-first identity resolution, the `>=3x` dominance
threshold in `finalize()` — is **PORTED-VERBATIM**. I did not redesign any
of it; `harvest/NOTES.md` was explicit that this algorithm can't be
flattened into data-driven rules without losing behavior, and the golden/
cannabinoid test suite (17 ported cases, all passing unmodified) is the
evidence that the port preserved it exactly.

**Decision: `CannabinoidExtraction.Confidence` — NEW, flagged loudly in the
code itself.** The harvested source never computed a confidence number; it
always returned a definite type. `03-DOMAIN-MODEL.md §3` requires
`cannabinoid_confidence`, so something has to produce one. I assigned a
value per branch (0.95 explicit match, 0.9 labelled ratio, 0.85 equal bare
ratio, 0.75 oriented bare ratio, 0.6 generic-mg-with-identity, 0.5 genuinely
mixed, 0/1.0 for unknown/hemp-seed), loosely anchored to the illustrative
numbers in `11-HARVEST.md §2.2`'s example (0.95 explicit, 0.5 fallback) —
but the numbers *between* those two anchors are mine, not derived from
anything. **This is the single most consequential NEW decision in
`cannabinoids.go`** — it directly feeds `03-DOMAIN-MODEL.md §2`'s publish
gate (`form.confidence >= 0.85`) indirectly by setting expectations for what
"confident" cannabinoid data looks like, even though the gate itself checks
`form`/`route`, not cannabinoid confidence directly. Worth an explicit
sign-off before any threshold tuning happens against it.

## `facets.go` — the largest NEW surface in M1

**The classifier itself (`classify()`)** — **PORTED-VERBATIM.** Same three
failure modes, same coherence matrix, same pet/apparel exclusivity
ordering. `TestClassifyRealFailures` runs all 17 of the original
`categories_test.go` cases unmodified and all pass.

**Decision: legacy-bucket → new-facet mapping (`resolveForm`) — NEW, the
single largest judgment call in all of M1.** `03-DOMAIN-MODEL.md §2`'s form
vocabulary (`oil_tincture, capsule, edible, topical, flower, vape,
concentrate, beverage, pet, apparel, accessory`) is *finer* than the
harvested classifier's buckets (`topical, vapeable, smokable, edible_solid,
edible, tincture, extract, beverage, nutrition, pet, apparel, other`) in two
places, and I had to decide how to split them:

1. **`edible_solid` → `capsule` vs `edible`.** Sub-check capsule-specific
   words (capsule/softgel/tablet/pill/vati) against the name; everything
   else in the bucket (gummy/chocolate/candy) stays `edible`.
2. **`vapeable` → `vape` vs `concentrate`.** Sub-check concentrate-specific
   words (distillate/shatter/dab) OR whether the bucket was set via the
   `concentrate_markers` regex ("not diluted with any carrier oil") rather
   than an explicit vape word — the latter needed a new field on the
   internal `formDetection` type (`viaConcentrateMarker`) because the
   marker frequently fires on the *description*, and re-scanning just the
   name afterward silently missed it (**this was a real bug**, see below).
3. **`extract` (RSO/FECO/hash oil) → `concentrate` with `route=oral`**,
   distinct from the vapeable-bucket concentrate's `route=inhaled` — per
   the harvested source's own comment ("raw extract with no other form is
   most commonly ingested orally"), which gave me the route but not the
   form; concentrate was the closest honest fit among the 11 new values.
4. **`nutrition` → `edible`**, since no `nutrition` form value exists in
   the new vocabulary at all. This one had a real downstream consequence —
   see `legacy.go` below.

None of this is verifiable against a doc or the harvested source, because
neither one had to solve this problem (the new facet vocabulary didn't
exist yet when the harvested classifier was written). It's checked against
my own judgment of "what would a person mean by these facet names," made
concrete and falsifiable via `TestResolveFormMapping`'s 10 cases and the
golden fixtures, but it's still an opinion, not a transcription. **This is
what most deserves a second reviewer's eyes.**

**Decision: `ResolveExtract` doesn't take a `*CategoryRuleSet` parameter —
NEW, but reason given.** The harvested `ClassifyExtractType` was already
ruleset-independent (four hardcoded `strings.Contains` checks, unchanged
across the whole harvest). Rather than force it through the JSON-loading
machinery for consistency's sake, I kept it a plain two-argument function.
Flagged, not hidden — `ResolveCarrier` similarly takes no ruleset.

**Decision: `vijaya`-labelled products map to `full_spectrum`, flagged
ambiguous — NEW.** Vijaya isn't one of the three extract facet values
(`full_spectrum`/`broad_spectrum`/`isolate`) — it's an Indian Ayurvedic
formulation term the harvested source treated as its own `ClassifyExtractType`
category. I mapped it to the closest real value and flagged the result for
review rather than silently picking one. Alternative not taken: add
`vijaya` as a fourth extract value beyond what `03-DOMAIN-MODEL.md §2`
lists — rejected because extending an enum the doc gives explicitly is a
bigger, more consequential move than routing a handful of products to
review.

**Decision: `ResolveProfile` exists at all — NEW, filling a real gap.**
`03-DOMAIN-MODEL.md §2` lists `profile` as one of six facets;
`08-BUILD-ORDERS.md §7`'s facets.go export list names only five functions,
omitting a `ResolveProfile`. The doc's own facet table is the more
authoritative source here (a milestone table's export list is a work
breakdown, not a redefinition of the facet vocabulary), so I added it,
reusing `cannabinoids.go`'s `finalize()`'s exact `>=3x` dominance threshold
rather than inventing a second one for the same underlying question asked
from a different angle.

**Decision: `ResolvePurchasable`'s scope — NEW, deliberately narrow.**
`03-DOMAIN-MODEL.md §2` describes `purchasable` as killing "retreats,
courses, consultations, merch." Full retreat/workshop/consultation
detection is `compliance.json`'s `service_listing` pattern — a different
package's (M2's) concern. `resolve`'s `ResolvePurchasable` only catches the
`apparel` case, which is legitimately resolve's business (it's the same
`classify()` call everything else here uses). Everything else defaults
`true`, on the assumption that M2's compliance filter is the real gate for
service listings and `resolve` shouldn't duplicate it.

## `precedence.go` — SPEC'D shape, one bug caught by testing it

The precedence order (`override > rule > model > default`) and override's
fixed `1.0` confidence are **SPEC'D**, `03-DOMAIN-MODEL.md §2`, verbatim.

**Bug caught, not a decision:** the first draft's `Resolve` took a
`clusterID` parameter and never actually set it on any of the four returned
`domain.ProductFacet` structs — every persisted facet would have carried a
zero-value UUID. `08-BUILD-ORDERS.md §7` doesn't list a `precedence_test.go`
at all (only `ruleset_test.go`, `cannabinoids_test.go`, `facets_test.go`,
`golden_test.go` are named for M1) — this bug would not have been caught by
the minimum required test surface. Writing `precedence_test.go` anyway (an
addition beyond the build order, same as M0's pattern) is what caught it.

`Publishable()` mirrors the exact gate formula from `03-DOMAIN-MODEL.md §2`
— **SPEC'D**. Not in the build order's file list either; added as a pure
function because the formula exists in the doc and `product_clusters
.publishable`'s doc comment (M0) already promised something would compute
it — better a named, tested function than an inline recomputation wherever
it's needed later.

## `legacy.go` — NEW, forced by a self-inflicted problem

`LegacyCategory`'s mapping table is **NEW**, and it exists specifically to
undo a side effect of the `facets.go` mapping decisions above:

- **`form=concentrate` is ambiguous on its own** — it's the merge point of
  two different legacy buckets (vapeable's dab/distillate path, and
  extract's RSO/FECO path). Resolved using `route` as the disambiguator
  (oral → legacy `extract` + secondary `edible`; inhaled → legacy
  `vapeable`), which only works because `resolveForm` happened to compute
  route from the same original bucket information.
- **`form=edible` alone loses the legacy `nutrition` category** — the
  frontend's `catalog.ts` `CATEGORY_COLORS` map still has a `nutrition`
  entry, and `ADR-002` promises the legacy view never breaks. Recovered
  using `concentration_type` (hemp_seed/nutrition-typed edibles map back to
  legacy `nutrition`) — a genuinely different signal than `form`, reached
  for specifically because `form` alone had already thrown the information
  away.

**This is worth naming plainly: `legacy.go` is patching a hole `facets.go`'s
mapping decisions created.** It works, and it's tested (`TestLegacyCategory`,
13 cases), but the fact that two facts (a route, a concentration type)
had to be recruited to reconstruct one piece of information the harvested
source got directly from a single classification pass is a sign the new
facet vocabulary and the legacy vocabulary aren't a clean bijection.
Something to watch when M6 (content CMS) or M8 (contract hardening) next
touches this seam.

## `value.go` — SPEC'D composition, one NEW formula inside it

`RankScore`'s multiplicative shape and `ValueTier`'s bands
(`ADR-012`, `<3/3-8/>8`) are **SPEC'D**. `DominantPerMg`'s `THC > CBD >
total` order is **SPEC'D** (`ADR-013`).

**Decision: `ValueScore(perMg) = 1/perMg` — NEW.** `03-DOMAIN-MODEL.md §5`
gives `RankScore`'s composition but never `value_score`'s own formula.
Simple inverse is the most direct reading of "cheaper is better" that
composes cleanly with the other three multiplicative dampeners — and
critically, the doc's own stated safety mechanism ("pure ₹/mg crowns
whatever product had its mg misparsed high") is *those other three
factors* doing the correcting (a suspiciously good value_score gets killed
by low facet confidence), not a cap inside `value_score` itself. Tested
directly (`TestRankScore`'s zero-confidence case).

**Decision: `PerMg` divides paise by 100 before dividing by mg — INFERRED,
narrow.** `price_paise` is `int64` paise everywhere per the Constitution,
but `05-API-REFERENCE.md`'s example payload shows `"cbd_price_per_mg": 2.10`
as a plain rupee decimal, not a paise figure. Converting before dividing is
the only reading consistent with that example.

---

## Bugs caught while building this (not decisions — defects, found and fixed)

Four, all caught by tests that exist *because* this pass insisted on
writing them beyond the bare minimum the build order names:

1. **`resolveForm`'s `edible_solid` case was dead code.** `classify()`
   already collapses `edible_solid` → the public name `edible` before
   returning (matching the harvested source exactly), so `d.primary` is
   never literally `"edible_solid"` — the capsule/edible sub-split had to
   move into the `"edible"` case instead. Caught by `TestResolveFormMapping`.
2. **The concentrate-marker path was invisible to the mapping layer.** A
   product whose "vapeable" classification came from `concentrate_markers`
   firing on the *description* (not a name-level word) fell through to
   `FormVape` instead of `FormConcentrate`, because the mapping only
   re-scanned the *name*. Fixed by threading a `viaConcentrateMarker` flag
   through `formDetection`. Caught by the same test.
3. **`Resolve` never set `ClusterID`** on any returned `domain.ProductFacet`
   — see `precedence.go` above. Caught by a test the build order doesn't
   require.
4. **`ResolveProfile`'s zero-minority case fell through to "balanced."**
   `lo > 0 && hi/lo >= 3` requires the minority to be positive before
   checking dominance — but a minority of exactly zero (a THC-free CBD
   product) is the *strongest* possible dominance, not an undetermined
   case. Caught by `TestGoldenFixtures` failing against `starcbd-trace-thc
   .json`, a fixture that predates `ResolveProfile`'s existence by several
   turns and had nothing to do with why it was written.

All four are fixed, and all four now have a test that would fail again if
the fix regressed — three of the four exist in files the build order never
asked for.

---

## Recheck pass, 2026-08-11 (same day, after the initial build)

Asked to recursively recheck M1 and confirm best practices. Ran
`staticcheck` (installed fresh, wasn't available before) and `go test
-cover` across the whole module, then re-read every file line-by-line
rather than trusting the green test run. `staticcheck ./...` and `go vet
-all ./...`: zero findings, both passes. The manual re-read found five real
issues static analysis doesn't catch — logic and coverage gaps, not style:

1. **`ruleset.go` had no fail-fast validation.** `rs.Patterns["x"]` and
   `rs.FormWordLists["x"]` are indexed directly, with no per-call nil
   check, throughout `cannabinoids.go` and `facets.go` — correct and
   appropriately terse, but only as long as the loader guarantees every key
   those two files reference actually exists. It didn't guarantee that: a
   `harvest/rules/*.json` edit dropping a key would have loaded
   successfully and then panicked on a nil `*regexp.Regexp` the first time
   business logic happened to reach that specific code path — far from
   the actual cause, at runtime, in production. Fixed:
   `requiredCannabinoidPatterns` / `requiredFormWordLists` checked right
   after parsing, `LoadRuleSet` now errors immediately naming exactly which
   key is missing. `TestLoadRuleSetMissingRequiredKey` proves it.
2. **`Tokens` split domain vocabulary on underscores.** `full_spectrum`,
   `oil_tincture`, `cbd_dominant` — every enum value in `domain.go` — would
   have come back as two tokens, not one, the first time anything actually
   called `Tokens` (nothing does yet, which is exactly why this had zero
   coverage and zero chance of being noticed otherwise). Found by writing
   `text_test.go` — which itself exists because a coverage check
   (`go test -cover`) turned up `Normalize`/`Tokens` at a flat 0%: exported
   API, per `08-BUILD-ORDERS.md §7`'s file list, with no test at all.
3. **`resolveForm`'s concentrate-sub-word check had a stray `lowName+" "`**
   — harmless (a trailing space doesn't change a `\b`-anchored regex
   match), but dead, confusing code left over from an earlier draft.
   Removed.
4. **`ResolveRoute`'s pet/apparel case returned `Reason: ""`.** The
   `!hasRoute` branch conflated two different situations under one
   `Ambiguous: true` — "route genuinely doesn't apply" (pet/apparel) and
   "couldn't classify at all" (the true `other` case) — and only the
   second had a reason string. `resolveForm`'s pet/apparel returns now
   carry `"route not applicable to pet products"` / `"...to apparel"`
   instead of an empty string.

Coverage after fixes: **91.4%** (was 89.9% before adding `text_test.go`).
Remaining gaps are almost entirely defensive branches in ported code
(`orientRatio`'s `a<=0 && b<=0` guard, `BestMG`'s post-switch fallback ifs)
that real product data may or may not ever exercise — deliberately not
chased to 100%, since the ported algorithm's own shape, not test-writing
effort, is what determines whether those branches are reachable.

**On long functions** (`ExtractCannabinoids` 166 lines, `classify` 113):
flagged by a manual complexity check, not fixed. Both lengths track the
harvested source's own function lengths closely — breaking either into
smaller pieces would mean restructuring control flow that
`harvest/NOTES.md` explicitly warns against flattening. Long-but-faithful
beats short-but-restructured here.

## What's genuinely unverifiable without more context

Two things this document can flag but not resolve:

- **`CannabinoidExtraction.Confidence`'s exact numbers** (0.95/0.9/0.85/
  0.75/0.6/0.5) have no ground truth to check against — there's no labeled
  dataset saying "this extraction should be 73% confident." They're
  internally consistent (higher for more direct evidence) but not
  calibrated to anything. `ADR-016`'s AI-assisted threshold-tuning process
  is the eventual answer here, once there's a review queue generating
  outcomes to tune against.
- **The `resolveForm` mapping's coverage.** `TestResolveFormMapping` and
  the golden fixtures check 10 and 10 cases respectively — real catalogue
  data (M4's ingest pipeline, run against real scraped listings) will
  surface edge cases neither test set anticipated. This is expected, not a
  gap to close now — it's exactly what the review queue and golden-fixture
  auto-append mechanism (`03-DOMAIN-MODEL.md §2`) exist to converge on over
  time.
