-- +goose Up
--
-- M0.3 — 03-DOMAIN-MODEL.md §3 (cannabinoids), §4 (cluster identity), §5
-- (pricing). "cluster = the canonical product... Value, ranking, facets,
-- comments, and public URLs all attach to the cluster" (§1).
--
-- Deliberately NOT stored as columns here: legacy `category` / `categories[]`
-- / `extract_type` / `carrier_oil`. Those are derived at read time from
-- product_facets (internal/resolve/legacy.go, M1.11) so there is exactly one
-- writer for facet-derived data — a stored+derived pair would drift the moment
-- someone updates one and forgets the other. See the M0 build log for the
-- full reasoning; this is the single riskiest interpretation call in M0 and
-- is worth Opus 5's scrutiny.
--
-- media_assets has no explicit schema in the docs (01-ARCHITECTURE.md §6 only
-- names the `internal/media` package: "fetch, transcode, blurhash, proxy").
-- Designed here to match 05-API-REFERENCE.md §7's media proxy contract
-- exactly (content-hashed filename, dimensions + blurhash returned in the
-- product payload, sizes thumb|card|full derived by the proxy from one
-- stored original) and ADR-017 (editorial images are now CMS-managed too,
-- hence `kind` and a nullable `source_url` for provenance tracking —
-- 01-ARCHITECTURE.md §8's still-open image-provenance item wants exactly
-- this column to exist).

-- ── Media (product images now; editorial images per ADR-017 later) ─────────────

CREATE TABLE media_assets (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    hash          text        NOT NULL UNIQUE,   -- content hash; filename base for /media/{hash}/{size}.{ext}
    ext           text        NOT NULL,          -- avif | webp
    content_type  text        NOT NULL,
    width         int         NOT NULL,
    height        int         NOT NULL,
    blurhash      text        NOT NULL,
    kind          text        NOT NULL DEFAULT 'product' CHECK (kind IN ('product','editorial')),
    source_url    text,                          -- original scraped/uploaded URL — provenance tracking (01-ARCHITECTURE.md §8)
    uploaded_by   text,                           -- admin identifier, set only for kind='editorial' uploads (ADR-017)
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- ── Clusters ─────────────────────────────────────────────────────────────────

CREATE TABLE product_clusters (
    id                      uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id                uuid,                 -- FK attached in 004 (brands table doesn't exist yet)
    name                    text        NOT NULL,
    short_description       text,                 -- capped ~160 chars server-side (02-FRONTEND-CONTRACT.md §8)

    -- Cannabinoid content (03-DOMAIN-MODEL.md §3) — all nullable, NEVER zero for unknown.
    cbd_mg                  numeric,
    thc_mg                  numeric,
    total_cannabinoids_mg   numeric,
    concentration_type      text        NOT NULL DEFAULT 'unknown'
                              CHECK (concentration_type IN ('cbd','thc','total','hemp_seed','nutrition','unknown')),
    cannabinoid_confidence  real,
    cannabinoid_evidence    jsonb       NOT NULL DEFAULT '{}',

    -- Size. One cluster = one size — a different size is a different dedup
    -- fingerprint (harvest/rules/dedup.md) and therefore a different cluster.
    volume_ml               numeric,
    weight_g                numeric,

    -- Pricing (03-DOMAIN-MODEL.md §5). paise, int64, never float (00-CONSTITUTION.md §5).
    best_price_paise        bigint,
    best_price_per_mg       numeric,              -- dominant basis (THC > CBD > total, ADR-013), back-compat
    cbd_price_per_mg        numeric,
    thc_price_per_mg        numeric,
    price_per_mg_basis      text,
    value_tier              text        CHECK (value_tier IN ('good','mid','high')),  -- ADR-012: frontend-canonical <3/3-8/>8 bands
    rank_score              numeric,

    -- Cross-cutting product facts not modelled as a facet.
    image_id                 uuid        REFERENCES media_assets(id),
    coa_available             boolean     NOT NULL DEFAULT false,
    prescription_required     boolean     NOT NULL DEFAULT false,

    -- Denormalised, maintained gate — 03-DOMAIN-MODEL.md §2's confidence-gate
    -- formula (facets.purchasable AND form.confidence>=0.85 AND (route IS NULL
    -- OR route.confidence>=0.90) AND price_paise>0) is a join across
    -- product_facets on every evaluation; this column caches its outcome so
    -- /api/products can filter with a plain WHERE instead of a live join on
    -- every catalogue request. Recomputed by internal/resolve/precedence.go
    -- (M1.10) whenever any input changes — this is the single source of
    -- computation, this column is a read cache, never written elsewhere.
    publishable               boolean     NOT NULL DEFAULT false,

    first_seen_at            timestamptz NOT NULL DEFAULT now(),
    updated_at                timestamptz NOT NULL DEFAULT now(),
    created_at                timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX product_clusters_publishable_idx ON product_clusters (rank_score DESC, id ASC) WHERE publishable = true;
CREATE INDEX product_clusters_cbd_pmg_idx ON product_clusters (cbd_price_per_mg ASC NULLS LAST) WHERE publishable = true;
CREATE INDEX product_clusters_thc_pmg_idx ON product_clusters (thc_price_per_mg ASC NULLS LAST) WHERE publishable = true;
CREATE INDEX product_clusters_brand_idx ON product_clusters (brand_id);

-- ── Merges (03-DOMAIN-MODEL.md §4) ──────────────────────────────────────────

CREATE TABLE cluster_merges (
    old_id     uuid        PRIMARY KEY,             -- a cluster can be merged away exactly once
    new_id     uuid        NOT NULL REFERENCES product_clusters(id) ON DELETE CASCADE,
    merged_at  timestamptz NOT NULL DEFAULT now(),
    CHECK (old_id <> new_id)
);

-- ── Attach the FKs deferred from 001 ─────────────────────────────────────────

ALTER TABLE product_listings
    ADD CONSTRAINT product_listings_cluster_fk
    FOREIGN KEY (cluster_id) REFERENCES product_clusters(id);

ALTER TABLE purchase_tokens
    ADD CONSTRAINT purchase_tokens_cluster_fk
    FOREIGN KEY (cluster_id) REFERENCES product_clusters(id);

-- +goose Down
ALTER TABLE purchase_tokens DROP CONSTRAINT purchase_tokens_cluster_fk;
ALTER TABLE product_listings DROP CONSTRAINT product_listings_cluster_fk;
DROP TABLE cluster_merges;
DROP TABLE product_clusters;
DROP TABLE media_assets;
