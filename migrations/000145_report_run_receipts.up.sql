BEGIN;
CREATE UNIQUE INDEX report_runs_receipt_identity_idx ON report_runs(id,tenant_id,legal_entity_id,requested_by_ref);
CREATE TABLE report_run_receipts (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid NOT NULL,
    principal_id text NOT NULL CHECK (length(btrim(principal_id)) > 0),
    idempotency_key uuid NOT NULL,
    request_fingerprint text NOT NULL CHECK (request_fingerprint ~ '^[0-9a-f]{64}$'),
    run_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (tenant_id, legal_entity_id, principal_id, idempotency_key),
    FOREIGN KEY (legal_entity_id, tenant_id) REFERENCES legal_entities(id, tenant_id),
    FOREIGN KEY (run_id, tenant_id, legal_entity_id, principal_id) REFERENCES report_runs(id, tenant_id, legal_entity_id, requested_by_ref)
);
COMMIT;
