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

_(M2 — pending)_

## package store

_(M3 — pending)_

## package content

_(M6 — pending)_

## package ingest

_(M7 — pending)_

## package api

_(M8 — pending)_
