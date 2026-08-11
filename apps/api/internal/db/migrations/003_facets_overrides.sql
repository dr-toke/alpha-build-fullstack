-- +goose Up
--
-- M0.4 — 03-DOMAIN-MODEL.md §2. This SQL is copied VERBATIM from the doc —
-- it is given in full there, not summarised, so it is not this migration's
-- place to improvise it. The only addition is the two comments explaining
-- WHY, which the doc states in prose around the block.
--
-- Precedence is absolute: override > rule > model > default. Enforced in
-- ONE function, resolve.Facet() (M1.10, internal/resolve/precedence.go), and
-- nowhere else — if a second code path can write a facet, that is already a
-- bug, not a design choice.

CREATE TABLE product_facets (
  cluster_id         uuid  NOT NULL REFERENCES product_clusters(id) ON DELETE CASCADE,
  facet              text  NOT NULL,
  value              text  NOT NULL,
  source             text  NOT NULL,   -- override | rule | model | default
  confidence         real  NOT NULL,   -- 0..1; override is always 1.0
  evidence           jsonb NOT NULL,   -- matched spans, rule ids, negation windows
  classifier_version int   NOT NULL,
  decided_at         timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (cluster_id, facet)
);
CREATE INDEX product_facets_lookup_idx
  ON product_facets (facet, value) WHERE source <> 'model';

CREATE TABLE product_facet_overrides (
  cluster_id uuid NOT NULL REFERENCES product_clusters(id) ON DELETE CASCADE,
  facet      text NOT NULL,
  value      text NOT NULL,
  reason     text NOT NULL,
  set_by     text NOT NULL,
  set_at     timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (cluster_id, facet)
);

-- Not in the doc's SQL block, added here: a CHECK constraining `facet` to the
-- six named facets in 03-DOMAIN-MODEL.md §2's table, so a typo'd facet name
-- fails at INSERT time instead of silently becoming an unqueryable orphan
-- value. All six facet value vocabularies are enforced in Go
-- (internal/resolve), not here — a Postgres CHECK can't express "value must
-- be valid for THIS row's facet" without a trigger, and the doc's own
-- dependency discipline (00-CONSTITUTION.md §6) prefers not reaching for
-- triggers where a Go-layer check does the job.
ALTER TABLE product_facets
  ADD CONSTRAINT product_facets_facet_check
  CHECK (facet IN ('form','route','extract','profile','carrier','purchasable'));
ALTER TABLE product_facet_overrides
  ADD CONSTRAINT product_facet_overrides_facet_check
  CHECK (facet IN ('form','route','extract','profile','carrier','purchasable'));

-- +goose Down
DROP TABLE product_facet_overrides;
DROP TABLE product_facets;
