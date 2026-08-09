BEGIN;
CREATE TABLE program_review_checkpoints (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    program_id uuid NOT NULL,
    principal_id uuid NOT NULL,
    program_version bigint NOT NULL CHECK (program_version > 0),
    projection_version bigint NOT NULL CHECK (projection_version > 0),
    accepted_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (id, tenant_id),
    UNIQUE (tenant_id, program_id, principal_id, program_version, projection_version),
    CONSTRAINT program_review_program_tenant_fk FOREIGN KEY (program_id, tenant_id) REFERENCES programs(id, tenant_id),
    CONSTRAINT program_review_principal_tenant_fk FOREIGN KEY (principal_id, tenant_id) REFERENCES principals(id, tenant_id),
    CONSTRAINT program_review_projection_fk FOREIGN KEY (tenant_id, program_id, projection_version)
        REFERENCES program_state_snapshots(tenant_id, program_id, projection_version)
);
CREATE INDEX program_review_actor_latest_idx ON program_review_checkpoints(tenant_id, principal_id, program_id, accepted_at DESC, id DESC);
COMMIT;
