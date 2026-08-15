package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/dr-toke/api/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const brandSelectColumns = `
	SELECT id, slug, name, full_name, founded, city, state, url, description,
	       categories, verified, ayush, fssai, ayush_reg_number, fssai_licence,
	       coa_available, affiliate_url, highlight, last_verified,
	       created_at, updated_at`

func scanBrand(row pgx.Row) (*domain.Brand, error) {
	var b domain.Brand
	err := row.Scan(&b.ID, &b.Slug, &b.Name, &b.FullName, &b.Founded, &b.City,
		&b.State, &b.URL, &b.Description, &b.Categories, &b.Verified, &b.Ayush,
		&b.FSSAI, &b.AyushRegNumber, &b.FSSAILicence, &b.COAAvailable,
		&b.AffiliateURL, &b.Highlight, &b.LastVerified, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// ListBrands returns every brand, verified first then alphabetical — the
// ordering 02-FRONTEND-CONTRACT.md §9's "pending-verification badge" design
// wants surfaced consistently rather than left to whatever order Postgres
// happens to return rows in.
func (s *Store) ListBrands(ctx context.Context) ([]domain.Brand, error) {
	q := brandSelectColumns + ` FROM brands ORDER BY verified DESC, name ASC`
	rows, err := s.Pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("store.ListBrands: %w", err)
	}
	defer rows.Close()

	var out []domain.Brand
	for rows.Next() {
		b, err := scanBrand(rows)
		if err != nil {
			return nil, fmt.Errorf("store.ListBrands: scan: %w", err)
		}
		out = append(out, *b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store.ListBrands: %w", err)
	}
	return out, nil
}

// BrandBySlug returns domain.ErrNotFound (wrapped) when no such brand
// exists — never a bare pgx.ErrNoRows leaking past the store boundary; every
// caller above this package should only ever need to know about
// internal/domain's sentinel errors, not pgx's.
func (s *Store) BrandBySlug(ctx context.Context, slug string) (*domain.Brand, error) {
	q := brandSelectColumns + ` FROM brands WHERE slug = $1`
	b, err := scanBrand(s.Pool.QueryRow(ctx, q, slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store.BrandBySlug(%s): %w", slug, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("store.BrandBySlug: %w", err)
	}
	return b, nil
}

// BrandByID looks up a brand by its UUID — product_clusters.brand_id is a
// FK on id, not slug, so internal/api's product payload assembly (which
// only has a cluster's BrandID) needs this rather than BrandBySlug.
func (s *Store) BrandByID(ctx context.Context, id uuid.UUID) (*domain.Brand, error) {
	q := brandSelectColumns + ` FROM brands WHERE id = $1`
	b, err := scanBrand(s.Pool.QueryRow(ctx, q, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("store.BrandByID(%s): %w", id, domain.ErrNotFound)
		}
		return nil, fmt.Errorf("store.BrandByID: %w", err)
	}
	return b, nil
}

// Approve marks a brand verified and bumps last_verified — 06-ADMIN.md §1.6:
// "Approving a brand releases everything queued under unknown_brand for it."
// This function only flips the brand row; releasing the queued review_queue
// rows is a separate, explicit step (ReviewQueue.Resolve, queue.go) — kept
// separate rather than cascaded automatically so an admin approving a brand
// can still see and individually confirm what was queued under it, per
// 06-ADMIN.md §1.2's keyboard-driven review workflow, rather than having
// them silently mass-approved as a side effect.
func (s *Store) Approve(ctx context.Context, slug string) error {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE brands SET verified = true, last_verified = CURRENT_DATE, updated_at = now() WHERE slug = $1`,
		slug)
	if err != nil {
		return fmt.Errorf("store.Approve: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("store.Approve(%s): %w", slug, domain.ErrNotFound)
	}
	return nil
}
