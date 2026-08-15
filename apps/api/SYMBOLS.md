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
cannabinoids.go, facets.go, precedence.go, legacy.go, value.go. `ExtractSize`
and `ParsePriceINR` added later, M4/M5, when `internal/ingest/promote.go`
needed a pack-size input for `Fingerprint` and a raw-string-to-paise
conversion that didn't exist anywhere yet — see `M1-DECISIONS.md`'s scope vs.
these two. `EvidenceToMap` was `precedence.go`-private until the same
milestone, exported once `promote.go` needed the identical Evidence->jsonb
conversion for `ProductCluster.CannabinoidEvidence`.)_

```
func DominantPerMg(cbdPerMg, thcPerMg, totalPerMg *float64) (perMg *float64, basis domain.Basis)
func EvidenceToMap(ev Evidence) map[string]any
func ExtractSize(rs *CannabinoidRuleSet, name, description string) (volumeML, weightG float64)
func LegacyCategories(category string, secondary []string) []string
func LegacyCategory(form domain.FormValue, route domain.RouteValue, concentrationType domain.ConcentrationType) (category string, secondary []string)
func Normalize(s string) string
func ParsePriceINR(raw string) (int64, error)
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

## package compliance

_(M2 — NARROW EXCEPTION to the ADR-019 placeholder: only `service_listing`
is implemented, per ADR-020. `hard_block`, `terminology_review`,
`price_anomaly`, `unknown_brand` remain unbuilt.)_

```
type Result struct {
type RuleSet struct {
func LoadRuleSet(path string) (*RuleSet, error)
func Evaluate(rs *RuleSet, name, category string) Result
```

`Evaluate` checks NAME per `harvest/rules/compliance.json`'s own
documented scope, plus CATEGORY — a deviation from that doc found necessary
by testing against a real live listing whose title alone didn't carry the
service_listing signal. See `M2-DECISIONS.md` / `ADR-020`.

## package store

_(M3 — all 11 files: store.go, listings.go, clusters.go, facets.go,
overrides.go, brands.go, reference.go, queue.go, content.go, community.go,
golden.go. `staging.go` added M4 (`CreateBatch`, `StageRawProduct`,
`FinishBatch`, `RawProductsForBatch`, plus `BatchByID`/
`LastApprovedBatchCount`/`DecideBatch` added M5 for the promotion gate) —
see M4-DECISIONS.md for why batch/staging SQL lives here and not in
`internal/ingest` despite supporting a file of the same name there.
`CreateCluster`/`UpdateClusterDerived`/`ClusterByFingerprint` and
`ClusterFilter.Sort`/`CountClusters` added M4/M5 for dedup + the
`sort=new` default API list needs. `BrandByID` added M5 — `product_
clusters.brand_id` is a UUID FK, `BrandBySlug` alone can't serve it.)_

```
const (
func AppendFixture(dir string, clusterID uuid.UUID, source string, raw GoldenRaw, expect map[string]any, regressionNote string) error
type ClusterFilter struct {
type GoldenFixture struct {
type GoldenRaw struct {
type QueueFilter struct {
type Store struct {
func New(ctx context.Context, databaseURL string) (*Store, error)
func (s *Store) AccountByHandle(ctx context.Context, handle string) (*domain.Account, error)
func (s *Store) AccountByID(ctx context.Context, id uuid.UUID) (*domain.Account, error)
func (s *Store) Approve(ctx context.Context, slug string) error
func (s *Store) BatchByID(ctx context.Context, id uuid.UUID) (*domain.ScrapeBatch, error)
func (s *Store) BrandByID(ctx context.Context, id uuid.UUID) (*domain.Brand, error)
func (s *Store) BrandBySlug(ctx context.Context, slug string) (*domain.Brand, error)
func (s *Store) Close()
func (s *Store) ClusterByFingerprint(ctx context.Context, fingerprint string) (*domain.ProductCluster, error)
func (s *Store) ClusterByID(ctx context.Context, id uuid.UUID) (*domain.ProductCluster, error)
func (s *Store) CommentsForCluster(ctx context.Context, clusterID uuid.UUID, limit, offset int) ([]domain.Comment, error)
func (s *Store) CommentsForPost(ctx context.Context, postID uuid.UUID, limit, offset int) ([]domain.Comment, error)
func (s *Store) CountClusters(ctx context.Context, f ClusterFilter) (int, error)
func (s *Store) CreateAccount(ctx context.Context, handle, passwordHash string) (*domain.Account, error)
func (s *Store) CreateBatch(ctx context.Context, sourceSlug string) (uuid.UUID, error)
func (s *Store) CreateCluster(ctx context.Context, c domain.ProductCluster) (uuid.UUID, error)
func (s *Store) CreateComment(ctx context.Context, c domain.Comment) (*domain.Comment, error)
func (s *Store) CreateRefreshToken(ctx context.Context, t domain.RefreshToken) error
func (s *Store) DecideBatch(ctx context.Context, batchID uuid.UUID, status domain.BatchStatus, previousCount *int, reason *string, decidedBy string) error
func (s *Store) DeleteComment(ctx context.Context, id uuid.UUID, requestingAccountID *uuid.UUID, byAdmin bool) error
func (s *Store) DocBySlug(ctx context.Context, kind domain.ContentKind, slug, locale string) (*domain.ContentDoc, *domain.ContentRevision, error)
func (s *Store) Enqueue(ctx context.Context, item domain.ReviewQueueItem) (uuid.UUID, error)
func (s *Store) FacetsFor(ctx context.Context, clusterID uuid.UUID) ([]domain.ProductFacet, error)
func (s *Store) FinishBatch(ctx context.Context, batchID uuid.UUID, productCount int) error
func (s *Store) LastApprovedBatchCount(ctx context.Context, sourceSlug string) (*int, error)
func (s *Store) ListAggregators(ctx context.Context) ([]domain.Aggregator, error)
func (s *Store) ListBrands(ctx context.Context) ([]domain.Brand, error)
func (s *Store) ListClusters(ctx context.Context, f ClusterFilter) ([]domain.ProductCluster, error)
func (s *Store) ListQueue(ctx context.Context, f QueueFilter) ([]domain.ReviewQueueItem, error)
func (s *Store) ListROA(ctx context.Context) ([]domain.ROAMethod, error)
func (s *Store) ListStates(ctx context.Context) ([]domain.State, error)
func (s *Store) ListingsForCluster(ctx context.Context, clusterID uuid.UUID) ([]domain.ProductListing, error)
func (s *Store) Merge(ctx context.Context, oldID, newID uuid.UUID) error
func (s *Store) NewRevision(ctx context.Context, r domain.ContentRevision) (uuid.UUID, error)
func (s *Store) OverridesFor(ctx context.Context, clusterID uuid.UUID) ([]domain.ProductFacetOverride, error)
func (s *Store) Publish(ctx context.Context, docID, revisionID uuid.UUID) error
func (s *Store) PublishedDocs(ctx context.Context, kind domain.ContentKind, locale string) ([]domain.ContentDoc, error)
func (s *Store) RawProductsForBatch(ctx context.Context, batchID uuid.UUID) ([]domain.RawProduct, error)
func (s *Store) RefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
func (s *Store) Resolve(ctx context.Context, id uuid.UUID, status domain.ReviewStatus, resolvedBy string) error
func (s *Store) RevokeRefreshToken(ctx context.Context, tokenHash string) error
func (s *Store) SetOverride(ctx context.Context, o domain.ProductFacetOverride) error
func (s *Store) StageRawProduct(ctx context.Context, batchID uuid.UUID, p domain.RawProduct) (uuid.UUID, error)
func (s *Store) TouchLastSeen(ctx context.Context, accountID uuid.UUID) error
func (s *Store) UpdateClusterDerived(ctx context.Context, id uuid.UUID, c domain.ProductCluster) error
func (s *Store) UpsertFacets(ctx context.Context, facets []domain.ProductFacet) error
func (s *Store) UpsertListing(ctx context.Context, l domain.ProductListing) (uuid.UUID, error)
```

`ClusterFilter.Sort` is `SortNew` (default, `first_seen_at DESC`) or
`SortValue` (`rank_score DESC`, requires non-null rank_score) —
05-API-REFERENCE.md §1's documented default, added when `internal/api`
needed it; see `API-DECISIONS.md`.

Note: `domain.State`/`domain.Aggregator` gained a `Stale bool` field during
M3 — M0's first pass omitted it despite the self-correcting
reference-content design needing it. See `M3-DECISIONS.md`.

## package content

_(M6 — pending)_

## package ingest

_(M4 — PARTIAL: adapter.go, spec.go, shopify.go, staging.go. `gate.go`,
`dedup.go`, and `promote.go` added M5. NOT built: `woocommerce.go` — scope
deliberately held at cbdstore.in/Shopify only, per project owner
(2026-08-15). See M4-DECISIONS.md.)_

```
func AssignCluster(ctx context.Context, st *store.Store, fingerprint string, cluster domain.ProductCluster) (uuid.UUID, error)
func Fingerprint(brandSlug, rawName string, volumeML, concentrationMG float64) string
func LoadBatch(ctx context.Context, st *store.Store, batchID uuid.UUID) ([]domain.RawProduct, error)
func LoadScraperSpec(dir string) (map[string]*ScraperSpec, error)
func StageBatch(ctx context.Context, st *store.Store, adapter Adapter) (batchID uuid.UUID, count int, err error)
type Adapter interface {
type GateDecision struct {
func DecideGate(ctx context.Context, st *store.Store, batchID uuid.UUID) (GateDecision, error)
func EvaluateGate(currentCount int, previousCount *int) GateDecision
type PromoteResult struct {
func Promote(ctx context.Context, st *store.Store, rs *resolve.RuleSet, crs *compliance.RuleSet, batchID uuid.UUID) (PromoteResult, error)
type RawListing struct {
type ScraperSpec struct {
type Shopify struct {
func NewShopify(spec *ScraperSpec, userAgent string, rateLimitMS int) *Shopify
func (s *Shopify) ScrapeAll(ctx context.Context, onListing func(RawListing) error) error
func (s *Shopify) Source() string
```

`EvaluateGate` implements one of 04-PIPELINE.md §2's four auto-reject
thresholds (product-count drop >30%) — the other three need data nothing
built so far computes. `Promote` is the full staging->live orchestration:
compliance -> resolve -> dedup -> facets -> listing, in-process (no River
job queue wired up yet). See gate.go/dedup.go/promote.go's own doc comments
and this milestone's decisions notes.

## package api

_(M5/M8 — PARTIAL first slice: envelope.go, errors.go, params.go,
products.go, router.go, cmd/server/main.go. `GET /api/products` (list,
`sort=new|value`, `brand` filter, keyset cursor) and `GET /api/products/{id}`
(detail + `moved_to`) only — compare/brands/reference/community/admin/auth
not built. See API-DECISIONS.md.)_

```
const (
func NewRouter(st *store.Store, allowedOrigin string) http.Handler
func WriteError(w http.ResponseWriter, status int, code, message string)
func WriteJSON(w http.ResponseWriter, status int, v any)
type Envelope struct {
type Handlers struct {
func (h *Handlers) GetProduct(w http.ResponseWriter, r *http.Request)
func (h *Handlers) ListProducts(w http.ResponseWriter, r *http.Request)
```
