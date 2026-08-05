BEGIN;

DROP TABLE IF EXISTS continuity_projection_jobs;
DROP INDEX IF EXISTS program_state_asof_idx;
DROP INDEX IF EXISTS program_state_projection_version_uq;
ALTER TABLE program_state_snapshots DROP COLUMN IF EXISTS projection_version;

COMMIT;
