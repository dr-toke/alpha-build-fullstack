-- +goose Up
--
-- M0.8 — 03-DOMAIN-MODEL.md §9 (survey), §10 (click events); review_queue's
-- reason vocabulary from 04-PIPELINE.md §5's table and its workflow from
-- 06-ADMIN.md §1.2 (evidence spans, dry-run diff, keyboard-driven resolution)
-- — the doc describes review_queue's BEHAVIOUR in detail but, like
-- admin_users, never gives it a SQL block. Schema below is this migration's
-- own design to satisfy that behaviour, flagged for review same as the other
-- inferred tables.

-- ── Click events (00-CONSTITUTION.md §2: no IP, no UA, no account ID) ──────────

CREATE TABLE click_events (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id     uuid        NOT NULL REFERENCES product_listings(id),
    cluster_id     uuid        REFERENCES product_clusters(id),
    brand_slug     text,
    source_slug    text        NOT NULL REFERENCES scrape_sources(slug),
    page_path      text        NOT NULL,
    filter_context jsonb       NOT NULL DEFAULT '{}',
    token_issued   boolean     NOT NULL DEFAULT false,
    clicked_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX click_events_cluster_idx ON click_events (cluster_id, clicked_at);
CREATE INDEX click_events_brand_idx ON click_events (brand_slug, clicked_at);
CREATE INDEX click_events_source_idx ON click_events (source_slug, clicked_at);
-- Plain range index, not a date_trunc() functional index: date_trunc() on a
-- timestamptz is STABLE (timezone-dependent), not IMMUTABLE, and Postgres
-- rejects STABLE functions in an index expression. A range scan on
-- clicked_at itself is what /admin/analytics/clicks?days=30 needs anyway —
-- caught by actually running this migration against Postgres 16, not by
-- inspection.
CREATE INDEX click_events_clicked_at_idx ON click_events (clicked_at);

-- ── Survey (03-DOMAIN-MODEL.md §9: aggregate counts only, never individual rows) ─

CREATE TABLE survey_counts (
    dimension  text   NOT NULL CHECK (dimension IN ('extract_type','use_case','price_range','carrier_oil')),
    value      text   NOT NULL,
    count      bigint NOT NULL DEFAULT 0,
    PRIMARY KEY (dimension, value)
);

CREATE TABLE survey_meta (
    id              boolean     PRIMARY KEY DEFAULT true CHECK (id),  -- singleton row
    total_responses bigint      NOT NULL DEFAULT 0,
    last_updated    timestamptz NOT NULL DEFAULT now()
);
INSERT INTO survey_meta DEFAULT VALUES;

-- ── Review queue (04-PIPELINE.md §5 reasons; 06-ADMIN.md §1.2 workflow) ────────

CREATE TABLE review_queue (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id     uuid        NOT NULL REFERENCES product_clusters(id) ON DELETE CASCADE,
    reason         text        NOT NULL
                     CHECK (reason IN ('unknown_brand','price_anomaly','compliance_uncertain',
                                        'terminology_review','category_uncertain','low_confidence')),
    detail         text        NOT NULL DEFAULT '',
    proposed_value jsonb       NOT NULL DEFAULT '{}',   -- what the model/rule proposed, for the diff view
    status         text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
    resolved_by    text,
    resolved_at    timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX review_queue_pending_idx ON review_queue (reason, created_at) WHERE status = 'pending';
CREATE INDEX review_queue_cluster_idx ON review_queue (cluster_id);

-- +goose Down
DROP TABLE review_queue;
DROP TABLE survey_meta;
DROP TABLE survey_counts;
DROP TABLE click_events;
