-- +goose Up
--
-- M4 addition, not in M0's original 7 migrations — dedup (harvest/rules/
-- dedup.md) needs somewhere to look up "does a cluster with this
-- fingerprint already exist" before deciding new-vs-existing, and
-- product_clusters had no such column. Written by hand, same discipline as
-- 001-007 — 08-BUILD-ORDERS.md §3's never-delegate list.
--
-- Deliberately a plain index, NOT a UNIQUE constraint: after a merge
-- (cluster_merges, M3), the OLD cluster row keeps its fingerprint forever
-- (Merge never deletes it — see clusters.go's own comment on why). A
-- UNIQUE constraint would then permanently block any future product from
-- ever being assigned that same fingerprint again, even onto the NEW
-- (merged-into) cluster. Uniqueness among LIVE clusters is an
-- application-level invariant AssignCluster maintains (check-then-create),
-- not a DB-enforced one — a deliberate trade-off for the merge case, not
-- an oversight.

ALTER TABLE product_clusters ADD COLUMN fingerprint text;
CREATE INDEX product_clusters_fingerprint_idx ON product_clusters (fingerprint) WHERE fingerprint IS NOT NULL;

-- +goose Down
DROP INDEX product_clusters_fingerprint_idx;
ALTER TABLE product_clusters DROP COLUMN fingerprint;
