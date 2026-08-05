BEGIN;

DROP TABLE IF EXISTS continuity_projection_jobs;
DROP INDEX IF EXISTS program_state_asof_idx;
DROP INDEX IF EXISTS program_state_projection_version_uq;
ALTER TABLE program_state_snapshots DROP COLUMN IF EXISTS projection_version;

ALTER TABLE continuity_events
  DROP CONSTRAINT continuity_events_aggregate_type_check;
ALTER TABLE continuity_events
  ADD CONSTRAINT continuity_events_aggregate_type_check
  CHECK (aggregate_type IN ('PROGRAM','MATTER'));

COMMIT;
