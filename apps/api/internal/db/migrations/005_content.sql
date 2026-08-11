-- +goose Up
--
-- M0.6 — 03-DOMAIN-MODEL.md §11. Column list matches the doc's SQL block
-- exactly, including `license` (ADR-015, added to the doc during this same
-- build). `hero_image_id` is an addition beyond the doc's block: ADR-017
-- moved editorial imagery into the CMS plane ("these images move into the
-- same CMS plane as editorial copy... An admin uploads/replaces them through
-- the admin panel") but the doc doesn't give a column for it. A revision
-- needing at most one hero image is the simplest shape that satisfies ADR-017
-- without inventing a many-to-many gallery nobody asked for.

CREATE TABLE content_docs (
    id                  uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    kind                text        NOT NULL
                          CHECK (kind IN ('post','section_block','era','topic','state_note','concept','faq','glossary')),
    slug                text        NOT NULL,
    locale              text        NOT NULL DEFAULT 'en',
    status              text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published','archived')),
    current_revision_id uuid,       -- FK attached below, after content_revisions exists
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    UNIQUE (kind, slug, locale)
);

CREATE TABLE content_revisions (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    doc_id         uuid        NOT NULL REFERENCES content_docs(id) ON DELETE CASCADE,
    title          text        NOT NULL,
    body_md        text        NOT NULL DEFAULT '',
    frontmatter    jsonb       NOT NULL DEFAULT '{}',
    author         text        NOT NULL,
    license        text        NOT NULL DEFAULT 'all-rights-reserved',  -- ADR-015: content is copyright to its author
    hero_image_id  uuid        REFERENCES media_assets(id),             -- ADR-017
    created_at     timestamptz NOT NULL DEFAULT now(),
    published_at   timestamptz
);
CREATE INDEX content_revisions_doc_idx ON content_revisions (doc_id, created_at DESC);

ALTER TABLE content_docs
    ADD CONSTRAINT content_docs_current_revision_fk
    FOREIGN KEY (current_revision_id) REFERENCES content_revisions(id);

-- +goose Down
ALTER TABLE content_docs DROP CONSTRAINT content_docs_current_revision_fk;
DROP TABLE content_revisions;
DROP TABLE content_docs;
