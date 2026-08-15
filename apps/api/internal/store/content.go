package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PublishedDocs lists published docs of one kind/locale — the
// /api/content/export build-time feed (07-CONTENT-CMS.md §3) and
// /api/forum/posts both read through this.
func (s *Store) PublishedDocs(ctx context.Context, kind domain.ContentKind, locale string) ([]domain.ContentDoc, error) {
	const q = `
		SELECT id, kind, slug, locale, status, current_revision_id, created_at, updated_at
		FROM content_docs
		WHERE kind = $1 AND locale = $2 AND status = 'published'
		ORDER BY updated_at DESC`

	rows, err := s.Pool.Query(ctx, q, kind, locale)
	if err != nil {
		return nil, fmt.Errorf("store.PublishedDocs: %w", err)
	}
	defer rows.Close()

	var out []domain.ContentDoc
	for rows.Next() {
		var d domain.ContentDoc
		if err := rows.Scan(&d.ID, &d.Kind, &d.Slug, &d.Locale, &d.Status,
			&d.CurrentRevisionID, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("store.PublishedDocs: scan: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.PublishedDocs: %w", err)
	}
	return out, nil
}

// DocBySlug returns a doc and its CURRENT revision (which may be an
// unpublished draft — 06-ADMIN.md's content editor needs to preview drafts;
// callers wanting published-only should check doc.Status themselves).
// UNIQUE (kind, slug, locale) — internal/db/migrations/005 — makes this a
// safe single-row lookup.
func (s *Store) DocBySlug(ctx context.Context, kind domain.ContentKind, slug, locale string) (*domain.ContentDoc, *domain.ContentRevision, error) {
	const docQ = `
		SELECT id, kind, slug, locale, status, current_revision_id, created_at, updated_at
		FROM content_docs WHERE kind = $1 AND slug = $2 AND locale = $3`

	var d domain.ContentDoc
	err := s.Pool.QueryRow(ctx, docQ, kind, slug, locale).Scan(
		&d.ID, &d.Kind, &d.Slug, &d.Locale, &d.Status, &d.CurrentRevisionID, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, fmt.Errorf("store.DocBySlug(%s/%s): %w", kind, slug, domain.ErrNotFound)
		}
		return nil, nil, fmt.Errorf("store.DocBySlug: %w", err)
	}
	if d.CurrentRevisionID == nil {
		return &d, nil, nil // doc exists, nothing published/drafted onto it yet
	}

	rev, err := s.revisionByID(ctx, *d.CurrentRevisionID)
	if err != nil {
		return nil, nil, fmt.Errorf("store.DocBySlug: loading current revision: %w", err)
	}
	return &d, rev, nil
}

func (s *Store) revisionByID(ctx context.Context, id uuid.UUID) (*domain.ContentRevision, error) {
	const q = `
		SELECT id, doc_id, title, body_md, frontmatter, author, license,
		       hero_image_id, created_at, published_at
		FROM content_revisions WHERE id = $1`
	var r domain.ContentRevision
	err := s.Pool.QueryRow(ctx, q, id).Scan(&r.ID, &r.DocID, &r.Title, &r.BodyMD,
		&r.Frontmatter, &r.Author, &r.License, &r.HeroImageID, &r.CreatedAt, &r.PublishedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// NewRevision appends an immutable revision — content_revisions is
// append-only (03-DOMAIN-MODEL.md §11); this never updates an existing row.
// Does NOT flip content_docs.current_revision_id — that's Publish's job,
// and a revision existing without being current is exactly how a draft
// preview works (06-ADMIN.md's "drafts build to a preview branch").
func (s *Store) NewRevision(ctx context.Context, r domain.ContentRevision) (uuid.UUID, error) {
	const q = `
		INSERT INTO content_revisions (doc_id, title, body_md, frontmatter, author, license, hero_image_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		RETURNING id`
	// frontmatter is NOT NULL DEFAULT '{}' — but that default only applies
	// when the column is OMITTED from the INSERT. Explicitly passing a nil
	// Go map encodes as SQL NULL, not JSON '{}', and violates the
	// constraint. Caught by TestContentPublishFlow calling NewRevision with
	// a zero-value ContentRevision (Frontmatter left unset) — exactly the
	// call shape a caller who doesn't need frontmatter would naturally make.
	frontmatter := r.Frontmatter
	if frontmatter == nil {
		frontmatter = map[string]any{}
	}
	var id uuid.UUID
	err := s.Pool.QueryRow(ctx, q, r.DocID, r.Title, r.BodyMD, frontmatter, r.Author, r.License, r.HeroImageID).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("store.NewRevision: %w", err)
	}
	return id, nil
}

// Publish flips a doc's pointer to a revision and sets that revision's
// published_at — "publish = pointer flip", 01-ARCHITECTURE.md §5.
// Rollback is the same operation run against an older revision ID, not a
// separate code path — 03-DOMAIN-MODEL.md §11: "rollback flips it back."
// Both statements run in one transaction: a doc pointing at a revision with
// no published_at (or vice versa) is an inconsistent state this function
// must never produce even under a partial failure.
func (s *Store) Publish(ctx context.Context, docID, revisionID uuid.UUID) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("store.Publish: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx,
		`UPDATE content_docs SET current_revision_id = $2, status = 'published', updated_at = now()
		 WHERE id = $1`, docID, revisionID)
	if err != nil {
		return fmt.Errorf("store.Publish: update doc: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store.Publish(doc=%s): %w", docID, domain.ErrNotFound)
	}

	tag, err = tx.Exec(ctx,
		`UPDATE content_revisions SET published_at = now() WHERE id = $1 AND doc_id = $2`,
		revisionID, docID)
	if err != nil {
		return fmt.Errorf("store.Publish: update revision: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store.Publish: revision %s does not belong to doc %s", revisionID, docID)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("store.Publish: commit: %w", err)
	}
	return nil
}
