# SYMBOLS

Running ledger of exported Go signatures. **Append after every merged file.**
This is the paste-source for the `AVAILABLE SYMBOLS` block in build orders
(`docs/08-BUILD-ORDERS.md §5`).

Signatures only — no bodies, no comments. Grouped by package, ordered by the
milestone that introduced them.

Regenerate a package's block with:

```bash
go doc -all ./internal/<pkg> | grep -E '^(func|type|const|var) '
```

---

## package domain

_(M0.9–0.10 — types.go, errors.go)_

```
var (
type Account struct {
type AdminAuditLog struct {
type AdminUser struct {
type Aggregator struct {
type Basis string
const (
type BatchStatus string
const (
type Brand struct {
type CarrierValue string
const (
type ClickEvent struct {
type ClusterMerge struct {
type ClusterMovedError struct {
func (e *ClusterMovedError) Error() string
func (e *ClusterMovedError) Unwrap() error
type Comment struct {
type ConcentrationType string
const (
type ContentDoc struct {
type ContentKind string
const (
type ContentRevision struct {
type ContentStatus string
const (
type ExtractValue string
const (
type Facet string
const (
type FacetSource string
const (
type FormValue string
const (
type LegalStatus string
const (
type LinkStatus string
const (
type MediaAsset struct {
type MediaKind string
const (
type ProductCluster struct {
type ProductFacet struct {
type ProductFacetOverride struct {
type ProductListing struct {
type ProfileValue string
const (
type PurchaseToken struct {
type ROAMethod struct {
type RawProduct struct {
type RefreshToken struct {
type ReviewQueueItem struct {
type ReviewReason string
const (
type ReviewStatus string
const (
type RouteValue string
const (
type ScrapeBatch struct {
type ScrapePlatform string
const (
type ScrapeSource struct {
type SourceRole string
const (
type State struct {
type SurveyCount struct {
type SurveyDimension string
const (
type SurveyMeta struct {
type ValueTier string
const (
```

Sentinel errors in `var (...)`: `ErrNotFound`, `ErrValidationFailed`,
`ErrInvalidFilter`, `ErrAuthRequired`, `ErrAuthInvalid`, `ErrBanned`,
`ErrRateLimited`, `ErrUnavailable`, `ErrDuplicateHandle`,
`ErrPurchaseTokenAlreadyClaimed` — one per stable machine error code in
`docs/02-FRONTEND-CONTRACT.md §3`. See `M0-BUILD-LOG.md §3`.

## package resolve

_(M1 — the rule engine: text.go, evidence.go, ruleset.go, match.go,
cannabinoids.go, facets.go, precedence.go, legacy.go, value.go)_

```
func DominantPerMg(cbdPerMg, thcPerMg, totalPerMg *float64) (perMg *float64, basis domain.Basis)
func LegacyCategories(category string, secondary []string) []string
func LegacyCategory(form domain.FormValue, route domain.RouteValue, concentrationType domain.ConcentrationType) (category string, secondary []string)
func Normalize(s string) string
func PerMg(pricePaise int64, mg float64) *float64
func Publishable(purchasable bool, formConfidence float32, route *domain.ProductFacet, pricePaise int64) bool
func RankScore(valueScore, facetConfidence, brandTrust, completeness float64) float64
func Resolve(clusterID uuid.UUID, facet domain.Facet, in FacetInputs, classifierVersion int) *domain.ProductFacet
func Tokens(s string) []string
func ValueScore(perMg float64) float64
func ValueTier(perMg *float64) *domain.ValueTier
type CannabinoidExtraction struct {
func ExtractCannabinoids(rs *CannabinoidRuleSet, name, description string) CannabinoidExtraction
func (r CannabinoidExtraction) BestMG() float64
type CannabinoidRuleSet struct {
type CategoryRuleSet struct {
type CoherenceMatrix struct {
type Evidence struct {
func Merge(a, b Evidence) Evidence
type FacetInputs struct {
type FacetResult struct {
func ResolveCarrier(description string) FacetResult
func ResolveExtract(name, description string) FacetResult
func ResolveForm(rs *CategoryRuleSet, name, description, rawCategory string) FacetResult
func ResolveProfile(cbdMg, thcMg float64) FacetResult
func ResolvePurchasable(rs *CategoryRuleSet, name, description, rawCategory string) FacetResult
func ResolveRoute(rs *CategoryRuleSet, name, description, rawCategory string) FacetResult
type RuleSet struct {
func LoadRuleSet(dir string) (*RuleSet, error)
type Span struct {
func ApplyNegation(s string, cat *CategoryRuleSet) (stripped string, negated []Span)
func MatchWordBoundary(re *regexp.Regexp, s string) (matched bool, spans []Span)
func NegationWindows(s string, negation *regexp.Regexp) (stripped string, windows []Span)
```

`ResolveProfile` is not in `08-BUILD-ORDERS.md §7`'s named export list for
facets.go — added because `03-DOMAIN-MODEL.md §2` lists `profile` as one of
six facets and the build order's list only named five. See
`M1-DECISIONS.md`.

## package store

_(M3 — pending)_

## package content

_(M6 — pending)_

## package ingest

_(M7 — pending)_

## package api

_(M8 — pending)_
