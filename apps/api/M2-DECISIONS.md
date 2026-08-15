# M2 decisions — internal reasoning, classified (NARROW EXCEPTION, not the milestone)

**Date:** 2026-08-15. This is not M2. `08-BUILD-ORDERS.md §7`'s M2 is
`internal/compliance/filter.go` + `filter_test.go` implementing all five
tiers `harvest/rules/compliance.json` describes: `hard_block`,
`terminology_review`, `service_listing`, `price_anomaly`, `unknown_brand`.
This is **one of five**, built as a scoped exception to ADR-019 — see
`ADR-020` in `10-DECISIONS.md` for the full reasoning on why and how narrow.

---

## The finding that changed the implementation from what the doc says

`harvest/rules/compliance.json`'s `service_listing` entry says
`"matched_against": "product NAME only"`. That's what the prior alpha did,
and it's what got harvested faithfully.

The actual trigger for building this — a live scrape of `cbdstore.in`
during M4 — surfaced a real doctor-consultation listing:

```
title: "Dr. Harshal Sawarkar – BAMS Ayurvedic Physician | Vijaya-Based
        Medicine, Chronic Pain, Skin & Hair, Digestive & Lifestyle Disorders"
product_type: "Doctors Consultation"
tags: ["#AyurvedicDoctor", "#DoctorConsultation", "doctors",
       "doctors consultation", "medical consultation", ...]
```

Tested the harvested pattern against the real title before writing any
Go code — **it does not match.** None of `service_listing`'s keywords
(retreat, workshop, consultation, etc.) appear in the title text at all.
Name-only, faithfully as harvested, would have silently let this exact
motivating case through.

What DOES carry the signal, confirmed the same way: Shopify's
`product_type` field, which is `"Doctors Consultation"` — literally
contains "Consultation," and flows through the pipeline already as
`RawListing.CategoryRaw` (`internal/ingest`) →
`domain.RawProduct.CategoryRaw` (staged). No new field needed, just a
second check.

**Decision — INFERRED, evidence-based:** `Evaluate(rs, name, category)`
checks both. This is a real deviation from what the harvested doc
specifies, made because the doc's specification, taken literally, provably
fails the case that motivated building this at all. Flagged loudly in
`filter.go`'s doc comment and locked in by
`TestEvaluateRealServiceListing`, which asserts BOTH halves explicitly:
name-only does NOT catch it (proving the deviation was necessary, not
cosmetic), name+category DOES.

---

## Signature scoped to what's actually implemented — NEW

`Evaluate(rs *RuleSet, name, category string) Result` — not the full
`Evaluate(brandSlug, name, description string, priceINR, pricePMG float64)
Result` shape the harvested source's original `Filter.Check` had, and that
the complete M2 build will eventually need (four more tiers read brand,
description, and price). Building the wide signature now with four
parameters silently ignored would misrepresent how much of compliance
exists. Widening it when `hard_block`/`terminology_review`/
`price_anomaly`/`unknown_brand` get built is normal, expected evolution —
not something this narrow pass tries to pre-solve.

---

## What's still not built

Everything except `service_listing`. Concretely, right now:

- **No hard-block** — "cures cancer"-style prohibited claims are not
  filtered.
- **No terminology review queue** — ordinary cannabis vocabulary
  (marijuana, ganja, etc.) isn't flagged for review; nothing about this
  narrow pass touches `review_queue` at all, in either direction.
- **No price-anomaly detection.**
- **No unknown-brand queueing** — brands not yet in the `brands` table
  aren't caught.
- **No wiring into `internal/ingest/staging.go`'s actual `StageBatch`
  flow.** `compliance.Evaluate` is called by the *test* demo
  (`classify_live_test.go`), proving it works against real data — it is
  NOT called by `StageBatch` itself, so staging still stages everything,
  service listings included. That's deliberate: `04-PIPELINE.md §1` places
  `compliance` as a stage AFTER `resolve`, operating on already-staged
  data, not as a pre-filter blocking the stage — matching that shape (and
  the Constitution's "we publish less rather than publish wrong" +
  "every compliance decision is logged with a reason code" — a filtered-out
  row should still exist as a record) is real work for the actual M2 build,
  not something to shortcut here.
