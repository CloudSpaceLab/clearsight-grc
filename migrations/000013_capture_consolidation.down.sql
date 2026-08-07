BEGIN;

CREATE TABLE evidence_requests (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    workflow_id uuid REFERENCES workflow_instances(id),
    purpose text NOT NULL,
    audience jsonb NOT NULL,
    sensitivity text NOT NULL,
    state text NOT NULL,
    schema_version text NOT NULL,
    request_schema jsonb NOT NULL,
    deadline timestamptz,
    submitted_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1
);
CREATE INDEX evidence_requests_queue_idx ON evidence_requests (tenant_id, state, deadline);

CREATE TABLE invitation_grants (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    request_id uuid NOT NULL REFERENCES evidence_requests(id),
    token_hash bytea NOT NULL UNIQUE,
    audience_hash bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    redeemed_at timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX invitation_active_idx ON invitation_grants (request_id, expires_at) WHERE redeemed_at IS NULL AND revoked_at IS NULL;

COMMIT;
