# 07 — Content, CMS, and Build Integration

> How editorial content gets from a writer's head into a prerendered SvelteKit
> page without the site ever depending on the API being awake.

---

## 1. Generate, don't replace

The frontend's `content.ts`-per-section convention is correct. Keep it.
**Generate it.**

```
pnpm content:pull
  → GET /api/content/export?since=<etag>
  → writes  src/lib/sections/<section>/content.generated.ts
            src/lib/content/posts/<slug>.json
            src/lib/content/index.json           (search index)
            static/data/products/*.json          (catalogue snapshot)
            static/data/catalog-index.json       (snapshot manifest)
```

Generated files are **committed**, not gitignored. That buys:

- reproducible builds — a tagged commit rebuilds byte-identically a year later
- a diffable, reviewable history of every copy change
- Netlify builds that need **no** network access to our API
- a working build when the backend is down

Hand-written `content.ts` keeps working for anything not yet migrated. Sections
move over one at a time; nothing breaks mid-migration.

`content:pull` is a small Go CLI in **this** repo (`cmd/contentpull`), invoked by
CI against a checkout of the frontend. The frontend repo gains no Go dependency.

---

## 2. Publish flow

```
admin: edit → preview → publish
  → backend fires the Netlify build hook
  → CI: pnpm content:pull
        git commit -m "content: <kind>/<slug> r<n>"
        pnpm check && pnpm build        (both must pass — existing PR gate)
  → deploy
```

Content changes become commits. A bad edit is `git revert`, not database
archaeology.

Drafts build to a **preview branch** with `noindex`, so writers see real pages
without touching production.

---

## 3. The export payload

```json
{
  "generated_at": "2026-08-09T12:00:00Z",
  "classifier_version": 14,
  "docs": [
    { "kind": "post", "slug": "ndps-section-20", "locale": "en",
      "title": "…", "body_md": "…", "frontmatter": { },
      "author": "…", "license": "all-rights-reserved",
      "revision": 7, "published_at": "…" }
  ]
}
```

One endpoint, ETag'd, published content only. Schema in
`03-DOMAIN-MODEL.md §11`.

`author` and `license` are rendered as a byline + rights line on published
posts — content is copyright to its author, not assigned to Dr Toke
(`ADR-015`).

### Markdown discipline

**Body is markdown, never HTML.** The frontend's `markdown.ts` is
dependency-free and escapes before rendering; sending HTML would defeat the one
thing that makes it safe.

Validate at publish time: render server-side with the same rules, and reject
anything that produces raw HTML or a link scheme outside `https:` / `mailto:`.

The `✎ to write:` chip in the frontend becomes automatic — a `section_block`
with no published revision. The frontend needs no change to get this.

---

## 4. Search without a search server

The legality state grid and post search are small — hundreds of documents.
Build-time JSON index plus client-side filtering: no server, no extra origin,
works offline and on Tor.

**Do not introduce Meilisearch or Elasticsearch for this.** It costs a service
and buys nothing at this scale. Only product search touches the API.

---

## 5. The catalogue snapshot

`content:pull` also writes a static snapshot of the catalogue:

- `static/data/catalog-index.json` — ranked list with the fields a card needs
- `static/data/products/{id}.json` — per-product detail

Effects:

- `/products` renders ranked and readable even with the API down, with a quiet
  "prices last updated {date}" line driven by `updated_at`
- `/product?id=` loads instantly from a same-origin file; `Remote<T>` then
  refreshes live price and comments in the background
- "service offline" stops meaning "blank page" and starts meaning "stale but
  complete"

Keep the query-param route shape (`/product?id=`) — it is what static export
requires and what the frontend already implements.

---

## 6. Media assets are content too

The frontend's hero/editorial imagery (`static/images/*.jpg`) moves into the
same CMS plane as copy (`ADR-017`), not just markdown:

- Admin uploads/replaces an image through the content-editor surface
  (`06-ADMIN.md §1.1`); bytes land in MinIO behind `/media/*`, same pipeline as
  scraped product images (`01-ARCHITECTURE.md §2`).
- `content:pull` writes the current content-hashed reference into the
  generated frontend files at build time — same mechanism as `content_docs`,
  no runtime fetch, no new origin, no frontend change to consume it.
- Rights-clearing of the *current* placeholder photos is unchanged by this —
  still open, still stage-2. This section is about where the asset lives and
  who can change it, not about clearing what's already there.

## 7. Migration order for existing copy

Move sections one at a time, verifying the built page is byte-comparable before
deleting the hand-written file:

1. `history` — pure prose, lowest risk, proves the loop
2. `science` / ROA — pulls in the `roa` reference table
3. `legality` — pulls in the `states` table; highest editorial value
4. `parcha`
5. Forum / blog posts — `forum_posts` folds into `content_docs` with
   `kind='post'`; comments keep pointing at the doc ID
6. Home page blocks last — most visually coupled

---

## 8. Localisation

`locale` ships now, populated only with `en`. Hindi later becomes a data task
rather than a migration.

**Never publish machine-translated Hindi without human review.** A wrong legal
or dosing statement in a second language is the same safety defect as in the
first.
