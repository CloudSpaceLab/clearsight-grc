BEGIN;

ALTER TABLE principals
    ADD CONSTRAINT principals_tenant_identity_unique UNIQUE (tenant_id, id);

CREATE TABLE principal_identities (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    principal_id uuid NOT NULL,
    issuer text NOT NULL,
    subject text NOT NULL,
    status text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','REVOKED')),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (tenant_id, principal_id) REFERENCES principals(tenant_id, id),
    UNIQUE (tenant_id, issuer, subject),
    CHECK (length(issuer) BETWEEN 1 AND 2048),
    CHECK (length(subject) BETWEEN 1 AND 2048)
);

CREATE INDEX principal_identities_principal_idx
    ON principal_identities(tenant_id, principal_id)
    WHERE status='ACTIVE';

CREATE TABLE web_sessions (
    token text PRIMARY KEY,
    data bytea NOT NULL,
    expiry timestamptz NOT NULL
);

CREATE INDEX web_sessions_expiry_idx ON web_sessions(expiry);

COMMIT;
