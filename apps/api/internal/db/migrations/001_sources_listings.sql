-- +goose Up
--
-- M0.2 — 03-DOMAIN-MODEL.md §1 (entity map), 04-PIPELINE.md §1-2 (stages,
-- promotion gate), ADR-010 (staging behind a gate, never write live).
--
-- Three layers, in promotion order:
--   scrape_sources  — the ~14 stores + their scraper config pointer
--   scrape_batches  — one row per scrape run; the promotion gate's unit of decision
--   raw_products    — STAGING. What a scraper saw. Never read by the public API.
--   product_listings — LIVE. One row as a store presents a product (04-PIPELINE.md
--                       §1: "listing = one variant, one URL, one price").
--   purchase_tokens — child of product_listings per the entity map; checkout flow
--                       (02-FRONTEND-CONTRACT.md §10, 05-API-REFERENCE.md §3).
--
-- product_listings.cluster_id is added WITHOUT its foreign key here (the
-- referenced table, product_clusters, doesn't exist until migration 002) and
-- the FK constraint is attached at the end of 002. This is deliberate
-- migration ordering, not an oversight — see harvest/NOTES.md-style reasoning
-- in the M0 build log.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ── Sources ──────────────────────────────────────────────────────────────────

CREATE TABLE scrape_sources (
    slug                text        PRIMARY KEY,
    name                text        NOT NULL,
    platform            text        NOT NULL CHECK (platform IN ('shopify','woocommerce','custom')),
    base_url            text        NOT NULL,
    trusted_aggregator  boolean     NOT NULL DEFAULT false,   -- harvest/rules/compliance.json: auto-passes unknown_brand
    role                text        NOT NULL DEFAULT 'direct' CHECK (role IN ('direct','aggregator')),
    active              boolean     NOT NULL DEFAULT true,
    rate_limit_ms       int         NOT NULL DEFAULT 2000,
    last_success_at     timestamptz,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

-- Seed: PoC scope is cbdstore.in only (harvest/scrapers/cbdstore.yaml,
-- harvest/NOTES.md). The other 13 stores are catalogued there, not seeded
-- here, so a later harvest pass adds rows instead of guessing config now.
INSERT INTO scrape_sources (slug, name, platform, base_url, trusted_aggregator, role, rate_limit_ms) VALUES
    ('cbdstore', 'CBD Store India', 'shopify', 'https://cbdstore.in', true, 'aggregator', 2000);

-- ── Promotion gate (ADR-010, 04-PIPELINE.md §2) ────────────────────────────────

CREATE TABLE scrape_batches (
    id                    uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    source_slug           text        NOT NULL REFERENCES scrape_sources(slug),
    started_at            timestamptz NOT NULL DEFAULT now(),
    finished_at           timestamptz,
    status                text        NOT NULL DEFAULT 'running'
                            CHECK (status IN ('running','pending_review','approved','rejected')),
    product_count         int,
    previous_product_count int,
    null_field_pct        numeric,        -- % of previously-populated fields that came back null
    selector_hit_rate     numeric,        -- 0..1
    price_median_shift    numeric,        -- multiplier vs. last approved batch's ₹/mg median
    rejection_reason      text,           -- free text: which of the four ADR-010 thresholds tripped
    decided_by            text,           -- 'auto' | admin identifier
    decided_at            timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX scrape_batches_source_idx ON scrape_batches (source_slug, started_at DESC);
CREATE INDEX scrape_batches_pending_idx ON scrape_batches (status) WHERE status = 'pending_review';

-- ── Staging (raw scrape output — never live) ───────────────────────────────────

CREATE TABLE raw_products (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id        uuid        NOT NULL REFERENCES scrape_batches(id) ON DELETE CASCADE,
    source_slug     text        NOT NULL REFERENCES scrape_sources(slug),
    source_url      text        NOT NULL,
    source_sku      text,
    name            text        NOT NULL,
    brand_raw       text        NOT NULL DEFAULT '',   -- vendor name / slugified fallback, pre-compliance-check
    price_raw       text        NOT NULL DEFAULT '',   -- e.g. "₹2,499.00" — parsed by normaliser.ExtractPriceINR equivalent
    description     text        NOT NULL DEFAULT '',
    image_url       text,
    category_raw    text        NOT NULL DEFAULT '',
    raw_data        jsonb       NOT NULL DEFAULT '{}', -- platform-specific extras (shopify vendor/tags/handle, woo attrs/coa_url)
    scraped_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX raw_products_batch_idx ON raw_products (batch_id);

-- ── Live listings ────────────────────────────────────────────────────────────

CREATE TABLE product_listings (
    id                    uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    source_slug           text        NOT NULL REFERENCES scrape_sources(slug),
    source_url            text        NOT NULL,
    source_sku            text,
    cluster_id            uuid,                          -- FK attached in 002; NULL until dedup assigns one
    -- Raw text is carried onto the live row (not just staging) so
    -- POST /admin/reprocess can re-run resolve/normalise without re-scraping —
    -- 04-PIPELINE.md §7's "rebuild, restart, POST /admin/reprocess" only works
    -- if the evidence text is still here.
    name_raw              text        NOT NULL,
    brand_raw             text        NOT NULL DEFAULT '',
    description_raw       text        NOT NULL DEFAULT '',
    category_raw          text        NOT NULL DEFAULT '',
    image_url_raw         text,
    price_paise           bigint      NOT NULL CHECK (price_paise >= 0),   -- 00-CONSTITUTION.md §5: int64 paise, never float
    affiliate_url         text,
    in_stock              boolean     NOT NULL DEFAULT true,
    promoted_from_batch_id uuid       REFERENCES scrape_batches(id),
    first_seen_at         timestamptz NOT NULL DEFAULT now(),
    last_seen_at          timestamptz NOT NULL DEFAULT now(),
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source_slug, source_url)   -- harvest/scrapers/cbdstore.yaml quirk: per-variant URL is what makes this safe
);
CREATE INDEX product_listings_cluster_idx ON product_listings (cluster_id) WHERE cluster_id IS NOT NULL;
CREATE INDEX product_listings_source_idx ON product_listings (source_slug);

-- ── Purchase tokens (checkout flow) ─────────────────────────────────────────

CREATE TABLE purchase_tokens (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id    uuid        NOT NULL REFERENCES product_listings(id) ON DELETE CASCADE,
    cluster_id    uuid,                    -- FK attached in 002; denormalised for verified_purchase lookups by cluster
    token_hash    text        NOT NULL UNIQUE,   -- SHA-256 of the opaque token; raw token never stored (05-API-REFERENCE.md §3)
    issued_at     timestamptz NOT NULL DEFAULT now(),
    claimed_by    uuid,                    -- FK to accounts(id) attached in 006
    claimed_at    timestamptz
);
CREATE INDEX purchase_tokens_claimed_by_idx ON purchase_tokens (claimed_by) WHERE claimed_by IS NOT NULL;

-- +goose Down
DROP TABLE purchase_tokens;
DROP TABLE product_listings;
DROP TABLE raw_products;
DROP TABLE scrape_batches;
DROP TABLE scrape_sources;
