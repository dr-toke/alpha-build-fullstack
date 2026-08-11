# 00 — Constitution

> Hard rules. Nothing in this repository may violate any of them. If a feature
> requires breaking one, the feature is wrong, not the rule. Changes here need an
> ADR in `10-DECISIONS.md`, never a quiet edit.

---

## 1. Legal

- The clearnet site must **never** facilitate a direct product transaction. We
  link out; we do not sell, cart, or checkout.
- **No user PII, ever.** No email, no phone, no name, no Aadhaar, no address.
  Not "hashed", not "temporarily" — not collected.
- All affiliate links open in a new tab with `rel="noopener noreferrer"`, and
  affiliate relationships are disclosed per ASCI guidelines.
- Sponsored listings are labelled as sponsored, visibly, always.
- The legal disclaimer appears on **every** page, above the fold on mobile, and
  cannot be dismissed. It is not a cookie banner.

  > Educational resource only. Dr Toke does not sell products. Content is for
  > harm reduction and legal awareness. Know your state laws. Not a licensed
  > medical or legal service.

- No content that could be construed as encouraging activity illegal under the
  **NDPS Act 1985**. Bhang and cannabis content is framed as cultural,
  Ayurvedic, and harm-reduction only.
- Never claim something is legal without a verifiable source. Every state legal
  claim carries a `last_verified` date and, where available, an official excise
  department link.
- Ayush / FSSAI badges appear **only** when registration is confirmed against
  `ayush.gov.in` / `foscos.fssai.gov.in`.
- Editorial content is copyright to its individual author, not assigned to Dr
  Toke (`10-DECISIONS.md#adr-015`). Published posts carry a byline and a
  rights line.

## 2. Privacy

- No tracking pixels. No Meta anything. No third-party analytics on the onion tier.
- **No cookies.** This is why auth is token-in-memory (see §4 and `05-API-REFERENCE.md`).
- Survey responses are stored as **aggregate counts only** — never individual
  rows, not even hashed. An individual row is a re-identification target.
- Click events store listing, cluster, brand, source, timestamp. **Never** IP,
  user agent, or account ID. Affiliate clicks are counts, not sessions.
- Onion tier: zero analytics, zero IP logging, zero user-agent logging.
- Purchase tokens are opaque and must not be linkable to the outbound click.

## 3. Safety

**This is a harm-reduction site.** A wrong route of administration or a wrong mg
figure is a safety defect, not a cosmetic one.

- **We publish less rather than publish wrong.** Low-confidence data goes to
  review, not to `/api`.
- ₹/mg is shown **only** when the API returns a non-null value. Never fabricate,
  never default to zero, never infer from a product name.
- Smokable products always carry the legal grey-zone warning.
- Prescription-required products always carry the Rx badge.
- Unverified brands are shown **with** a pending-verification badge, not hidden.
  Transparent, not paternalistic.

## 4. Security

- No single point of failure: the static frontend must render fully without the
  backend and without JavaScript.
- All secrets in environment variables. Never hardcoded, never committed.
- Parameterised SQL only (`$1`, `$2`). String-concatenated SQL is a merge blocker.
- Rate limiting on every public endpoint.
- Input sanitisation on all user-generated content. Markdown in, escaped HTML
  out — never raw HTML from any user or any CMS field.
- Admin plane is bound to localhost or a Tor hidden service. It shares **no**
  signing key, session, or users table with the public tier — even once the
  API and public site share an apex domain (`10-DECISIONS.md#adr-014`). Same
  domain never means same trust boundary.
- Every internal service credential (worker↔DB, admin↔DB, api↔DB) is
  least-privilege and scoped to what that service does. Network adjacency —
  same domain, same host — is never treated as authorization.
- CSP on every response. Exactly **one** extra origin beyond self: the API.
  Images proxy through it. No third origin, ever.

## 5. Architecture invariants

- Frontend and backend are separate programs meeting only at HTTP/JSON and
  generated types.
- Editorial content is prerendered at build time and must survive the API being
  down (`01-ARCHITECTURE.md §3`).
- Money is `int64` paise. Never float.
- Unknown numerics are `NULL` / nil pointers. **Never zero.**
- The model proposes, rules decide, humans overrule. No AI classification is
  ever authoritative.
- Human overrides are permanent and survive every re-scrape and every classifier
  version.
- Every compliance decision is logged with a reason code. "We filter this" is
  only defensible if we can produce the record.

## 6. Dependency discipline

Allowed in the backend: Go standard library, Chi, pgx, River, Goose, Colly,
MinIO client, `golang.org/x/crypto` (argon2), `google/uuid`.

Anything else requires an ADR. Specifically **do not** add: an ORM (raw SQL via
pgx only), a search server (build-time JSON index — see `07-CONTENT-CMS.md §4`),
a message broker (River is enough), or a second web framework.
