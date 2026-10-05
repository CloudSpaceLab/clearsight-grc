BEGIN;

DROP INDEX IF EXISTS domain_metric_snapshot_outbox_uq;
DROP TABLE IF EXISTS attention_episodes;

COMMIT;
