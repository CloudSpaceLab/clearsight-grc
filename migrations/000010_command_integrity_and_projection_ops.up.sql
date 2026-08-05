BEGIN;

ALTER TABLE continuity_events
  DROP CONSTRAINT continuity_events_aggregate_type_check;
ALTER TABLE continuity_events
  ADD CONSTRAINT continuity_events_aggregate_type_check
  CHECK (aggregate_type IN ('PROGRAM','MATTER','PROGRAM_STATE'));

ALTER TABLE program_state_snapshots
  ADD COLUMN projection_version BIGINT;

WITH ranked AS (
  SELECT id, row_number() OVER (PARTITION BY tenant_id, program_id ORDER BY generated_at, id) AS version
  FROM program_state_snapshots
)
UPDATE program_state_snapshots pss
SET projection_version = ranked.version
FROM ranked
WHERE ranked.id = pss.id;

ALTER TABLE program_state_snapshots
  ALTER COLUMN projection_version SET NOT NULL;

CREATE UNIQUE INDEX program_state_projection_version_uq
  ON program_state_snapshots(tenant_id, program_id, projection_version);
CREATE INDEX program_state_asof_idx
  ON program_state_snapshots(tenant_id, program_id, generated_at DESC, projection_version DESC);

CREATE TABLE continuity_projection_jobs (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  projection_name TEXT NOT NULL,
  aggregate_type TEXT NOT NULL,
  aggregate_id UUID NOT NULL,
  source_aggregate_version BIGINT NOT NULL,
  reason TEXT NOT NULL,
  trigger_id TEXT NOT NULL DEFAULT '',
  requested_by TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('READY','CLAIMED','COMPLETED','FAILED')),
  attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
  available_at TIMESTAMPTZ NOT NULL,
  claimed_at TIMESTAMPTZ,
  claimed_by TEXT NOT NULL DEFAULT '',
  completed_at TIMESTAMPTZ,
  last_error TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CHECK (projection_name = 'PROGRAM_STATE'),
  CHECK (aggregate_type = 'PROGRAM')
);

CREATE UNIQUE INDEX continuity_projection_active_job_uq
  ON continuity_projection_jobs(tenant_id, projection_name, aggregate_id)
  WHERE status IN ('READY','CLAIMED');
CREATE INDEX continuity_projection_claim_idx
  ON continuity_projection_jobs(status, available_at, created_at)
  WHERE status IN ('READY','CLAIMED');
CREATE INDEX continuity_projection_health_idx
  ON continuity_projection_jobs(tenant_id, projection_name, status, created_at DESC);

COMMIT;
