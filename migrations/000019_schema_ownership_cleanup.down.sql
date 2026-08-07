BEGIN;

CREATE TABLE audit_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    actor_id uuid,
    event_type text NOT NULL,
    subject_type text NOT NULL,
    subject_id uuid,
    purpose text NOT NULL,
    safe_metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX audit_subject_time_idx ON audit_events (tenant_id, subject_type, subject_id, occurred_at DESC);

CREATE TABLE readiness_snapshots (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    program_id uuid,
    status text NOT NULL,
    dimensions jsonb NOT NULL,
    recommended_actions jsonb NOT NULL DEFAULT '[]'::jsonb,
    generated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX readiness_latest_idx ON readiness_snapshots(tenant_id, program_id, generated_at DESC);

COMMIT;
