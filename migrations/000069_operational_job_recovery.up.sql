BEGIN;

CREATE TABLE operational_recovery_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    queue text NOT NULL CHECK (queue IN ('outbox-delivery','workflow-timers')),
    job_id uuid NOT NULL,
    decision text NOT NULL CHECK (decision='RETRY'),
    previous_attempts integer NOT NULL CHECK (previous_attempts > 0),
    terminal_at timestamptz NOT NULL,
    actor_principal_id uuid NOT NULL,
    rationale text NOT NULL CHECK (char_length(rationale) BETWEEN 20 AND 2000),
    recovered_at timestamptz NOT NULL,
    CONSTRAINT operational_recovery_actor_tenant_fk FOREIGN KEY (actor_principal_id,tenant_id) REFERENCES principals(id,tenant_id)
);

CREATE INDEX operational_recovery_events_job_idx
    ON operational_recovery_events(tenant_id,queue,job_id,recovered_at DESC,id DESC);

COMMIT;
