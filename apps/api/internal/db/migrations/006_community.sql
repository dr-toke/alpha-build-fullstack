-- +goose Up
--
-- M0.7 — 03-DOMAIN-MODEL.md §8 (community: accounts, refresh_tokens,
-- comments — column list matches the doc's SQL block exactly). Also attaches
-- the purchase_tokens.claimed_by FK deferred from 001 (accounts didn't exist
-- yet there).
--
-- admin_users / admin_audit_log are NOT in 03-DOMAIN-MODEL.md's §8 block —
-- that section is titled "Community" and is about the public pseudonymous
-- tier. They're required by 00-CONSTITUTION.md §4 ("Admin plane... shares NO
-- signing key, session, or users table with the public tier") and
-- 06-ADMIN.md §3 ("Own admin_users table. Never the public accounts table")
-- and §1.9 ("Audit log... Non-negotiable. Every override, publish, brand
-- approval, moderation action, and manual promotion writes a row"), but
-- 08-BUILD-ORDERS.md's M0 file list doesn't name a home for them. Placed here
-- because it's the nearest thematic fit (auth-adjacent), kept in a clearly
-- separated block, and never sharing a table with `accounts` above — flagged
-- in the M0 build log as a placement call worth a second opinion.

-- ── Public, pseudonymous tier ────────────────────────────────────────────────

CREATE TABLE accounts (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    handle        text        NOT NULL UNIQUE CHECK (handle ~ '^[A-Za-z0-9_]{3,24}$'),
    handle_lower  text        GENERATED ALWAYS AS (lower(handle)) STORED,
    password_hash text        NOT NULL,        -- Argon2id
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_seen_at  timestamptz NOT NULL DEFAULT now(),
    banned        boolean     NOT NULL DEFAULT false,
    ban_reason    text
    -- NO email, NO phone, NO name, NO IP. (00-CONSTITUTION.md §1)
);
CREATE UNIQUE INDEX accounts_handle_lower_idx ON accounts (handle_lower);

CREATE TABLE refresh_tokens (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id  uuid        NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    token_hash  text        NOT NULL UNIQUE,
    expires_at  timestamptz NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    revoked     boolean     NOT NULL DEFAULT false
);
CREATE INDEX refresh_tokens_account_idx ON refresh_tokens (account_id) WHERE revoked = false;

CREATE TABLE comments (
    id                 uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id         uuid        NOT NULL REFERENCES accounts(id),
    post_id            uuid        REFERENCES content_docs(id),
    cluster_id         uuid        REFERENCES product_clusters(id),
    body               text        NOT NULL CHECK (char_length(body) BETWEEN 10 AND 1000),
    verified_purchase  boolean     NOT NULL DEFAULT false,
    purchase_token_id  uuid        REFERENCES purchase_tokens(id),
    rating             int         CHECK (rating BETWEEN 1 AND 5),
    deleted            boolean     NOT NULL DEFAULT false,
    deleted_by_admin   boolean     NOT NULL DEFAULT false,
    created_at         timestamptz NOT NULL DEFAULT now(),
    CHECK ((post_id IS NOT NULL) <> (cluster_id IS NOT NULL))   -- exactly one of the two
);
CREATE INDEX comments_post_idx ON comments (post_id, created_at DESC) WHERE post_id IS NOT NULL AND deleted = false;
CREATE INDEX comments_cluster_idx ON comments (cluster_id, created_at DESC) WHERE cluster_id IS NOT NULL AND deleted = false;
CREATE INDEX comments_account_idx ON comments (account_id);

ALTER TABLE purchase_tokens
    ADD CONSTRAINT purchase_tokens_claimed_by_fk
    FOREIGN KEY (claimed_by) REFERENCES accounts(id);

-- ── Admin tier — deliberately isolated, see header note ─────────────────────

CREATE TABLE admin_users (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    username      text        NOT NULL UNIQUE,
    password_hash text        NOT NULL,        -- Argon2id
    totp_secret   text        NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_login_at timestamptz,
    disabled      boolean     NOT NULL DEFAULT false
);

CREATE TABLE admin_audit_log (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id    uuid        NOT NULL REFERENCES admin_users(id),
    action      text        NOT NULL,     -- e.g. 'facet_override', 'content_publish', 'brand_approve', 'comment_delete'
    target_type text        NOT NULL,     -- 'cluster' | 'content_doc' | 'brand' | 'comment' | ...
    target_id   text        NOT NULL,
    before      jsonb,
    after       jsonb,
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX admin_audit_log_target_idx ON admin_audit_log (target_type, target_id, created_at DESC);

-- +goose Down
DROP TABLE admin_audit_log;
DROP TABLE admin_users;
ALTER TABLE purchase_tokens DROP CONSTRAINT purchase_tokens_claimed_by_fk;
DROP TABLE comments;
DROP TABLE refresh_tokens;
DROP TABLE accounts;
