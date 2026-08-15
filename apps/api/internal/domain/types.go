// Package domain holds every struct and enum shared across the backend —
// the single vocabulary internal/resolve, internal/store, internal/api, and
// internal/jobs all speak. Nothing here touches SQL, HTTP, or business logic;
// see 03-DOMAIN-MODEL.md for the schema this mirrors and
// 08-BUILD-ORDERS.md §3 for why this file is written by hand, not delegated.
package domain

import (
	"time"

	"github.com/google/uuid"
)

// ── Enums ────────────────────────────────────────────────────────────────────
//
// String-backed, matching the CHECK constraints in internal/db/migrations
// exactly. Kept as distinct types (not bare string) so a typo'd literal is a
// compile error at the call site, not a runtime constraint violation.

// ConcentrationType is which cannabinoid basis a cluster's mg figures are
// dosed against. 03-DOMAIN-MODEL.md §3.
type ConcentrationType string

const (
	ConcentrationCBD       ConcentrationType = "cbd"
	ConcentrationTHC       ConcentrationType = "thc"
	ConcentrationTotal     ConcentrationType = "total"
	ConcentrationHempSeed  ConcentrationType = "hemp_seed"
	ConcentrationNutrition ConcentrationType = "nutrition"
	ConcentrationUnknown   ConcentrationType = "unknown"
)

// ValueTier is the frontend-canonical ₹/mg band. ADR-012: these bands
// (<3 / 3-8 / >8) are the frontend's pmgColor() bands, not the four-tier
// scheme 03-DOMAIN-MODEL.md originally specified — the backend aligns to the
// frontend, not the reverse.
type ValueTier string

const (
	ValueTierGood ValueTier = "good" // < 3
	ValueTierMid  ValueTier = "mid"  // 3-8
	ValueTierHigh ValueTier = "high" // > 8
)

// Basis selects which cannabinoid a ₹/mg figure or sort is scoped to.
// ADR-009 (basis-scoped value), ADR-013 (THC > CBD > total dominant-basis
// priority when no basis is specified and a single number is required).
type Basis string

const (
	BasisCBD   Basis = "cbd"
	BasisTHC   Basis = "thc"
	BasisTotal Basis = "total"
)

// Facet is one of the six orthogonal product facets. 03-DOMAIN-MODEL.md §2 —
// replaces the single category/categories[] column specifically so that
// form, route, and extract type are never decided from the same evidence.
type Facet string

const (
	FacetForm        Facet = "form"
	FacetRoute       Facet = "route"
	FacetExtract     Facet = "extract"
	FacetProfile     Facet = "profile"
	FacetCarrier     Facet = "carrier"
	FacetPurchasable Facet = "purchasable"
)

// FacetSource is where a facet value came from. Precedence is absolute:
// override > rule > model > default, enforced in one function
// (internal/resolve.Facet(), M1.10) and nowhere else.
type FacetSource string

const (
	FacetSourceOverride FacetSource = "override"
	FacetSourceRule     FacetSource = "rule"
	FacetSourceModel    FacetSource = "model"
	FacetSourceDefault  FacetSource = "default"
)

// FormValue enumerates product_facets rows where facet = form.
type FormValue string

const (
	FormOilTincture FormValue = "oil_tincture"
	FormCapsule     FormValue = "capsule"
	FormEdible      FormValue = "edible"
	FormTopical     FormValue = "topical"
	FormFlower      FormValue = "flower"
	FormVape        FormValue = "vape"
	FormConcentrate FormValue = "concentrate"
	FormBeverage    FormValue = "beverage"
	FormPet         FormValue = "pet"
	FormApparel     FormValue = "apparel"
	FormAccessory   FormValue = "accessory"
)

// RouteValue enumerates product_facets rows where facet = route.
type RouteValue string

const (
	RouteSublingual  RouteValue = "sublingual"
	RouteOral        RouteValue = "oral"
	RouteInhaled     RouteValue = "inhaled"
	RouteTopical     RouteValue = "topical"
	RouteTransdermal RouteValue = "transdermal"
)

// ExtractValue enumerates product_facets rows where facet = extract.
type ExtractValue string

const (
	ExtractFullSpectrum  ExtractValue = "full_spectrum"
	ExtractBroadSpectrum ExtractValue = "broad_spectrum"
	ExtractIsolate       ExtractValue = "isolate"
)

// ProfileValue enumerates product_facets rows where facet = profile.
// Derived from the cbd_mg/thc_mg ratio, NEVER from text — 03-DOMAIN-MODEL.md
// §2 is explicit about this; a profile is a computed fact, not a classified one.
type ProfileValue string

const (
	ProfileCBDDominant ProfileValue = "cbd_dominant"
	ProfileTHCDominant ProfileValue = "thc_dominant"
	ProfileBalanced    ProfileValue = "balanced"
)

// CarrierValue enumerates product_facets rows where facet = carrier.
type CarrierValue string

const (
	CarrierMCT      CarrierValue = "mct"
	CarrierHempSeed CarrierValue = "hemp_seed"
	CarrierOlive    CarrierValue = "olive"
	CarrierNone     CarrierValue = "none"
)

// ScrapePlatform is which generic adapter a source uses.
// 08-BUILD-ORDERS.md §1: three adapters, not fourteen scrapers.
type ScrapePlatform string

const (
	PlatformShopify     ScrapePlatform = "shopify"
	PlatformWooCommerce ScrapePlatform = "woocommerce"
	PlatformCustom      ScrapePlatform = "custom"
)

// SourceRole distinguishes a single-brand store from a multi-brand aggregator.
type SourceRole string

const (
	SourceRoleDirect     SourceRole = "direct"
	SourceRoleAggregator SourceRole = "aggregator"
)

// BatchStatus is a scrape_batches row's position in the promotion gate.
// ADR-010, 04-PIPELINE.md §2.
type BatchStatus string

const (
	BatchRunning       BatchStatus = "running"
	BatchPendingReview BatchStatus = "pending_review"
	BatchApproved      BatchStatus = "approved"
	BatchRejected      BatchStatus = "rejected"
)

// LegalStatus is a state's cannabis/bhang legal status. 03-DOMAIN-MODEL.md §7.
type LegalStatus string

const (
	LegalStatusLegal     LegalStatus = "legal"
	LegalStatusTolerated LegalStatus = "tolerated"
	LegalStatusGrey      LegalStatus = "grey"
	LegalStatusLimited   LegalStatus = "limited"
	LegalStatusIllegal   LegalStatus = "illegal"
)

// LinkStatus is the self-correction state of an external reference URL
// (states.excise_url, aggregators.url). 03-DOMAIN-MODEL.md §7,
// harvested verbatim from the prior alpha's migration 026.
type LinkStatus string

const (
	LinkUnknown  LinkStatus = "unknown"
	LinkOK       LinkStatus = "ok"
	LinkRedirect LinkStatus = "redirect"
	LinkBroken   LinkStatus = "broken"
	LinkNoURL    LinkStatus = "no_url"
)

// ContentKind is a content_docs row's editorial type. 03-DOMAIN-MODEL.md §11.
type ContentKind string

const (
	ContentPost         ContentKind = "post"
	ContentSectionBlock ContentKind = "section_block"
	ContentEra          ContentKind = "era"
	ContentTopic        ContentKind = "topic"
	ContentStateNote    ContentKind = "state_note"
	ContentConcept      ContentKind = "concept"
	ContentFAQ          ContentKind = "faq"
	ContentGlossary     ContentKind = "glossary"
)

// ContentStatus is a content_docs row's publish state.
type ContentStatus string

const (
	ContentDraft     ContentStatus = "draft"
	ContentPublished ContentStatus = "published"
	ContentArchived  ContentStatus = "archived"
)

// ReviewReason is why a review_queue row exists. 04-PIPELINE.md §5.
type ReviewReason string

const (
	ReviewUnknownBrand        ReviewReason = "unknown_brand"
	ReviewPriceAnomaly        ReviewReason = "price_anomaly"
	ReviewComplianceUncertain ReviewReason = "compliance_uncertain"
	ReviewTerminologyReview   ReviewReason = "terminology_review"
	ReviewCategoryUncertain   ReviewReason = "category_uncertain"
	ReviewLowConfidence       ReviewReason = "low_confidence"
)

// ReviewStatus is a review_queue row's resolution state.
type ReviewStatus string

const (
	ReviewPending  ReviewStatus = "pending"
	ReviewApproved ReviewStatus = "approved"
	ReviewRejected ReviewStatus = "rejected"
)

// MediaKind distinguishes a scraped product photo from an admin-uploaded
// editorial asset. ADR-017.
type MediaKind string

const (
	MediaProduct   MediaKind = "product"
	MediaEditorial MediaKind = "editorial"
)

// SurveyDimension is one of the four aggregate-only survey counters.
// 03-DOMAIN-MODEL.md §9 — individual responses are never stored, only counts.
type SurveyDimension string

const (
	SurveyExtractType SurveyDimension = "extract_type"
	SurveyUseCase     SurveyDimension = "use_case"
	SurveyPriceRange  SurveyDimension = "price_range"
	SurveyCarrierOil  SurveyDimension = "carrier_oil"
)

// ── Ingest: sources, batches, staging, live listings ────────────────────────

// ScrapeSource is one of the ~14 stores. internal/db/migrations/001.
type ScrapeSource struct {
	Slug              string
	Name              string
	Platform          ScrapePlatform
	BaseURL           string
	TrustedAggregator bool // harvest/rules/compliance.json: auto-passes unknown_brand
	Role              SourceRole
	Active            bool
	RateLimitMS       int
	LastSuccessAt     *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ScrapeBatch is one scrape run — the promotion gate's unit of decision.
// ADR-010: scrapes never write to live tables; a batch is approved or
// rejected as a whole before its raw_products rows ever reach product_listings.
type ScrapeBatch struct {
	ID                   uuid.UUID
	SourceSlug           string
	StartedAt            time.Time
	FinishedAt           *time.Time
	Status               BatchStatus
	ProductCount         *int
	PreviousProductCount *int
	NullFieldPct         *float64
	SelectorHitRate      *float64
	PriceMedianShift     *float64
	RejectionReason      *string
	DecidedBy            *string
	DecidedAt            *time.Time
	CreatedAt            time.Time
}

// RawProduct is one staged, unpromoted scrape result. Never read by the
// public API — see internal/api, which only ever queries ProductListing /
// ProductCluster.
type RawProduct struct {
	ID          uuid.UUID
	BatchID     uuid.UUID
	SourceSlug  string
	SourceURL   string
	SourceSKU   *string
	Name        string
	BrandRaw    string
	PriceRaw    string
	Description string
	ImageURL    *string
	CategoryRaw string
	RawData     map[string]any
	ScrapedAt   time.Time
}

// ProductListing is one row as a store presents a product — one variant, one
// URL, one price. 03-DOMAIN-MODEL.md §1. Carries its own raw text (not just
// staging) so POST /admin/reprocess can re-run resolve without re-scraping.
type ProductListing struct {
	ID                  uuid.UUID
	SourceSlug          string
	SourceURL           string
	SourceSKU           *string
	ClusterID           *uuid.UUID // nil until dedup assigns one
	NameRaw             string
	BrandRaw            string
	DescriptionRaw      string
	CategoryRaw         string
	ImageURLRaw         *string
	PricePaise          int64 // 00-CONSTITUTION.md §5: never float
	AffiliateURL        *string
	InStock             bool
	PromotedFromBatchID *uuid.UUID
	FirstSeenAt         time.Time
	LastSeenAt          time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// PurchaseToken backs the Verified Purchase flow.
// 02-FRONTEND-CONTRACT.md §10, 05-API-REFERENCE.md §3. The opaque token
// itself is never stored — only its SHA-256 hash — and it carries no account
// authority until claimed.
type PurchaseToken struct {
	ID        uuid.UUID
	ListingID uuid.UUID
	ClusterID *uuid.UUID
	TokenHash string
	IssuedAt  time.Time
	ClaimedBy *uuid.UUID
	ClaimedAt *time.Time
}

// ── Media ────────────────────────────────────────────────────────────────────

// MediaAsset is one stored image — product photo or, per ADR-017, an
// admin-uploaded editorial asset. Served via GET /media/{hash}/{size}.{ext}
// (05-API-REFERENCE.md §7); Width/Height/Blurhash ride in the product/content
// payload directly, never fetched from the image endpoint.
type MediaAsset struct {
	ID          uuid.UUID
	Hash        string
	Ext         string // avif | webp
	ContentType string
	Width       int
	Height      int
	Blurhash    string
	Kind        MediaKind
	SourceURL   *string // provenance — 01-ARCHITECTURE.md §8's open image-provenance item
	UploadedBy  *string
	CreatedAt   time.Time
}

// ── Catalogue: clusters, facets, merges ─────────────────────────────────────

// ProductCluster is the canonical product. Many listings merge into one;
// value, ranking, facets, comments, and the public URL all attach here, never
// to a listing. 03-DOMAIN-MODEL.md §1, §3, §5.
//
// Legacy `category` / `categories[]` / `extract_type` / `carrier_oil` are
// deliberately NOT fields here — they're derived at read time from
// ProductFacets by internal/resolve/legacy.go (M1.11, ADR-002's dual-write
// plan) so there is exactly one writer for facet-derived data.
type ProductCluster struct {
	ID               uuid.UUID
	BrandID          *uuid.UUID
	Name             string
	ShortDescription *string // capped ~160 chars server-side, 02-FRONTEND-CONTRACT.md §8
	// Fingerprint is harvest/rules/dedup.md's exact-match key
	// (brand|name|volume|concentration_mg, sha256, truncated). Added in M4
	// — internal/db/migrations/008, after M0's original schema pass.
	Fingerprint *string

	// Cannabinoid content — 03-DOMAIN-MODEL.md §3. All nullable, NEVER zero
	// for unknown (00-CONSTITUTION.md §5).
	CBDMg                 *float64
	THCMg                 *float64
	TotalCannabinoidsMg   *float64
	ConcentrationType     ConcentrationType
	CannabinoidConfidence *float64
	CannabinoidEvidence   map[string]any

	VolumeML *float64
	WeightG  *float64

	// Pricing — 03-DOMAIN-MODEL.md §5.
	BestPricePaise  *int64
	BestPricePerMg  *float64 // dominant basis (THC > CBD > total, ADR-013), back-compat
	CBDPricePerMg   *float64
	THCPricePerMg   *float64
	PricePerMgBasis *string
	ValueTier       *ValueTier // ADR-012
	RankScore       *float64

	ImageID              *uuid.UUID
	COAAvailable         bool
	PrescriptionRequired bool

	// Publishable is a maintained read cache of the confidence-gate formula
	// in 03-DOMAIN-MODEL.md §2 (facets.purchasable AND form.confidence>=0.85
	// AND (route IS NULL OR route.confidence>=0.90) AND price_paise>0).
	// internal/resolve/precedence.go is the only writer.
	Publishable bool

	FirstSeenAt time.Time
	UpdatedAt   time.Time
	CreatedAt   time.Time
}

// ClusterMerge records that OldID's identity now lives at NewID.
// GET /api/products/{old} returns {"moved_to": new_id} instead of 404ing —
// 03-DOMAIN-MODEL.md §4.
type ClusterMerge struct {
	OldID    uuid.UUID
	NewID    uuid.UUID
	MergedAt time.Time
}

// ProductFacet is one resolved facet value on a cluster, with full
// provenance. 03-DOMAIN-MODEL.md §2 — the exact schema from that doc.
type ProductFacet struct {
	ClusterID         uuid.UUID
	Facet             Facet
	Value             string
	Source            FacetSource
	Confidence        float32 // 0..1; override is always 1.0
	Evidence          map[string]any
	ClassifierVersion int
	DecidedAt         time.Time
}

// ProductFacetOverride is a permanent human correction. Applied AFTER every
// pipeline run, unconditionally — never recomputed, never expired.
// Auto-appends a testdata/golden fixture (03-DOMAIN-MODEL.md §2).
type ProductFacetOverride struct {
	ClusterID uuid.UUID
	Facet     Facet
	Value     string
	Reason    string
	SetBy     string
	SetAt     time.Time
}

// ── Brands & reference content ──────────────────────────────────────────────

// Brand is DB-backed ground truth. 03-DOMAIN-MODEL.md §6 — static TS files
// were Phase-0 bootstrap only.
type Brand struct {
	ID             uuid.UUID
	Slug           string
	Name           string
	FullName       *string
	Founded        *string // bare year string in source data, not a date
	City           *string
	State          *string
	URL            *string
	Description    *string
	Categories     []string
	Verified       bool // drives the pending-verification badge
	Ayush          bool
	FSSAI          bool
	AyushRegNumber *string
	FSSAILicence   *string
	COAAvailable   bool
	AffiliateURL   *string
	Highlight      *string
	LastVerified   time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// State is one row of the self-checking legal reference grid.
// 03-DOMAIN-MODEL.md §7; self-correction columns harvested verbatim from the
// prior alpha's migration 026_reference_content.sql.
type State struct {
	ID                 uuid.UUID
	Slug               string
	Name               string
	Status             LegalStatus
	BhangShops         string
	Detail             string
	ExciseURL          *string
	Notes              *string
	Featured           bool // Delhi NCR is pinned via this
	DisplayOrder       int
	LastVerified       time.Time
	VerifyIntervalDays int
	LinkStatus         LinkStatus
	LinkCheckedAt      *time.Time
	LinkFailures       int
	// Stale is computed at query time, not stored — (last_verified +
	// verify_interval_days) < now(). Added during M3's store layer: the
	// prior alpha's reference.go handler computes and returns this field
	// (03-DOMAIN-MODEL.md §7's "self-checking" reference content is what
	// this powers), but M0's first pass at this struct omitted it.
	Stale     bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ROAMethod is one route-of-administration guide entry.
// 03-DOMAIN-MODEL.md §7 — the edibles-delay golden rule and the bhang
// thandai festival warning are WarningNote content here, not frontend strings.
type ROAMethod struct {
	ID              uuid.UUID
	Slug            string
	Method          string
	Onset           string
	Duration        string
	Bioavailability string
	Pros            []string
	Cons            []string
	BestFor         []string
	WarningNote     *string
	DisplayOrder    int
	LastVerified    time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Aggregator is one directory entry (ItsHemp, CBD Store India, CannaMeds
// India). SourceSlug links it to a ScrapeSource so counts self-derive from
// our own catalogue when we scrape that aggregator.
type Aggregator struct {
	ID                  uuid.UUID
	Slug                string
	Name                string
	URL                 string
	Description         string
	SourceSlug          *string
	BrandCountLabel     *string
	ProductCountLabel   *string
	DerivedBrandCount   *int
	DerivedProductCount *int
	Featured            bool
	DisplayOrder        int
	Active              bool
	LastVerified        time.Time
	VerifyIntervalDays  int
	LinkStatus          LinkStatus
	LinkCheckedAt       *time.Time
	LinkFailures        int
	Stale               bool // computed at query time, same as State.Stale — see its comment
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// ── Content / CMS ────────────────────────────────────────────────────────────

// ContentDoc is one editorial document. Revisions are append-only; publishing
// flips CurrentRevisionID. 03-DOMAIN-MODEL.md §11.
type ContentDoc struct {
	ID                uuid.UUID
	Kind              ContentKind
	Slug              string
	Locale            string
	Status            ContentStatus
	CurrentRevisionID *uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ContentRevision is one immutable version of a ContentDoc's copy.
// License + Author: content is copyright to its individual author, not
// assigned to Dr Toke (ADR-015) — the frontend renders both as a byline and
// rights line. HeroImageID is ADR-017 (editorial images are CMS-managed).
type ContentRevision struct {
	ID          uuid.UUID
	DocID       uuid.UUID
	Title       string
	BodyMD      string
	Frontmatter map[string]any
	Author      string
	License     string
	HeroImageID *uuid.UUID
	CreatedAt   time.Time
	PublishedAt *time.Time
}

// ── Community (pseudonymous, no PII) ────────────────────────────────────────

// Account is a pseudonymous handle+password identity. No email, no phone, no
// name, no IP — 00-CONSTITUTION.md §1, 03-DOMAIN-MODEL.md §8.
type Account struct {
	ID           uuid.UUID
	Handle       string
	PasswordHash string // Argon2id
	CreatedAt    time.Time
	LastSeenAt   time.Time
	Banned       bool
	BanReason    *string
}

// RefreshToken is one rotatable session token. ADR-007: the access JWT is
// never persisted; only the refresh token has a server-side record, and only
// its hash.
type RefreshToken struct {
	ID        uuid.UUID
	AccountID uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
	Revoked   bool
}

// Comment is a review (ClusterID set) or a forum/post reply (PostID set) —
// exactly one of the two, enforced by a DB CHECK. Public responses return
// Handle only, never AccountID (00-CONSTITUTION.md §2, 05-API-REFERENCE.md §3).
type Comment struct {
	ID               uuid.UUID
	AccountID        uuid.UUID
	PostID           *uuid.UUID
	ClusterID        *uuid.UUID
	Body             string // 10..1000 chars, DB CHECK
	VerifiedPurchase bool
	PurchaseTokenID  *uuid.UUID
	Rating           *int // 1..5, product comments only
	Deleted          bool
	DeletedByAdmin   bool
	CreatedAt        time.Time
}

// ── Admin (isolated from the public tier — 00-CONSTITUTION.md §4) ──────────

// AdminUser is a distinct identity system from Account — separate table,
// separate signing key, TOTP required. 06-ADMIN.md §3.
type AdminUser struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string // Argon2id
	TOTPSecret   string
	CreatedAt    time.Time
	LastLoginAt  *time.Time
	Disabled     bool
}

// AdminAuditLog is an append-only record of every mutating admin action.
// "Non-negotiable" — 06-ADMIN.md §1.9.
type AdminAuditLog struct {
	ID         uuid.UUID
	AdminID    uuid.UUID
	Action     string
	TargetType string
	TargetID   string
	Before     map[string]any
	After      map[string]any
	CreatedAt  time.Time
}

// ── Analytics & survey (00-CONSTITUTION.md §2: no IP, no UA, no account ID) ─

// ClickEvent is one outbound-link click. 03-DOMAIN-MODEL.md §10.
type ClickEvent struct {
	ID            uuid.UUID
	ListingID     uuid.UUID
	ClusterID     *uuid.UUID
	BrandSlug     *string
	SourceSlug    string
	PagePath      string
	FilterContext map[string]any
	TokenIssued   bool
	ClickedAt     time.Time
}

// SurveyCount is one aggregate counter. Individual survey responses are
// NEVER stored — 03-DOMAIN-MODEL.md §9; an individual row would be a
// re-identification target (00-CONSTITUTION.md §2).
type SurveyCount struct {
	Dimension SurveyDimension
	Value     string
	Count     int64
}

// SurveyMeta is the singleton totals/freshness row alongside SurveyCount.
type SurveyMeta struct {
	TotalResponses int64
	LastUpdated    time.Time
}

// ── Review queue ─────────────────────────────────────────────────────────────

// ReviewQueueItem is one pending (or resolved) human-review decision.
// 04-PIPELINE.md §5 (reasons), 06-ADMIN.md §1.2 (workflow). Resolving one
// writes a ProductFacetOverride and appends a golden fixture — the review
// queue itself does not hold that outcome, it only tracks the flag lifecycle.
type ReviewQueueItem struct {
	ID            uuid.UUID
	ClusterID     uuid.UUID
	Reason        ReviewReason
	Detail        string
	ProposedValue map[string]any
	Status        ReviewStatus
	ResolvedBy    *string
	ResolvedAt    *time.Time
	CreatedAt     time.Time
}
