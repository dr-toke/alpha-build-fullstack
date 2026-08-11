# Dedup fingerprint — harvested from `dr-toke-init/apps/api/internal/catalog/dedup/dedup.go`

> No `_test.go` file exists for this package in the prior alpha — zero automated
> test coverage carried over, same gap as compliance. Fixtures need authoring
> fresh during M1/M4.

## Fingerprint (exact-match clustering)

```go
func Fingerprint(brandSlug, normName string, volumeML, concentrationMG float64) string {
	key := fmt.Sprintf("%s|%s|%.1f|%.1f",
		brandSlug,
		strings.ToLower(strings.TrimSpace(normName)),
		volumeML,
		concentrationMG,
	)
	h := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", h[:16])
}
```

- Key = `brand_slug | lowercased+trimmed normalised name | volume_ml (1 decimal) | concentration_mg (1 decimal)`.
- SHA-256 of that key, **truncated to the first 16 bytes** (32 hex chars), not
  the full 32-byte digest.
- Deterministic: same brand + same normalised name + same volume + same
  concentration → same fingerprint, always. This is what `dedup` job's
  existing-cluster branch matches against to decide "same product, refresh it"
  vs. "new product, assign a new cluster UUID."
- **What "normalised name" means here is NOT specified in this file** — it's
  whatever `normName` the caller passes in. Check `internal/catalog` callers of
  `Fingerprint(...)` (not read as part of this harvest pass) for the actual
  normalisation applied before the string reaches this function — casing,
  punctuation stripping, unit-spelling normalisation, brand-prefix stripping.
  **Flag for a second harvest pass** if that caller code still exists and
  wasn't captured here.

## Fuzzy matching (near-duplicate detection, not exact fingerprint)

`JaroWinkler(s1, s2) float64` — standard Jaro-Winkler string similarity
(0–1), lower-cased both sides first, with the standard Winkler prefix bonus
(up to a 4-char common prefix, weight 0.1). Used for fuzzy product-name
matching where the exact fingerprint doesn't hit — e.g. the same product
listed with slightly different punctuation/spacing across two stores.

`WithinPct(a, b, pct float64) bool` — true if `a` and `b` are within `pct`
percent of each other (`|a-b| / max(a,b) <= pct/100`), with `a==0 && b==0`
treated as true (both unknown counts as "close enough") and either-but-not-both
zero as false. Used for numeric near-match checks (price, volume) alongside
the name similarity score.

## Correction after checking the actual caller

The only caller of `Fingerprint` is `internal/jobs/normalise.go:65`:

```go
fp := dedup.Fingerprint(brandSlug, rawName, volML, cb.BestMG())
```

`rawName` is passed **directly** — there is no separate name-normalisation
step (no punctuation stripping, no unit-spelling normalisation, no
brand-prefix stripping) anywhere in the prior alpha. "normalised name" in the
function's doc comment is aspirational; in practice it's whatever
`strings.ToLower(strings.TrimSpace(...))` does inside `Fingerprint` itself to
the raw scraped name.

**`JaroWinkler` and `WithinPct` have no callers anywhere in `apps/api`.** They
are dead code — fuzzy/near-duplicate matching was scaffolded but never wired
into the pipeline. Dedup in the prior alpha is **exact-fingerprint-only**:
`brand_slug|lowercased_raw_name|volume_ml|dominant_cannabinoid_mg`. Two
listings of the same product with even slightly different scraped name text
(extra whitespace beyond what trim catches, a "-" vs "–", a trailing size
suffix one store adds and another doesn't) do **not** merge under this scheme
— they'd create two clusters.

## Known false-merge / false-split cases

None recorded — no comments, no test fixtures, and no `review_queue` merge
decisions were available to inspect in this harvest pass (no running dev DB,
see `harvest/NOTES.md`). Given the finding above, treat "false-split from name
variance" as an **expected, currently-unhandled failure mode** to design for
in the new `internal/resolve` dedup, not a solved problem being carried
forward. Whether to actually wire up fuzzy matching (real work: name
normalisation + a tuned similarity threshold) or accept exact-match-only for
the PoC is a decision for M4, not something this harvest resolves.
