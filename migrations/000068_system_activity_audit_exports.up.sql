BEGIN;

CREATE TABLE audit_export_receipts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid REFERENCES legal_entities(id),
    requested_by_ref text NOT NULL,
    format text NOT NULL CHECK (format IN ('CSV','NDJSON')),
    filter jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(filter)='object' AND octet_length(filter::text)<=16384),
    as_of timestamptz NOT NULL,
    status text NOT NULL CHECK (status IN ('GENERATING','READY','FAILED')),
    row_count integer NOT NULL DEFAULT 0 CHECK (row_count>=0),
    data_object_key text,
    data_sha256 text,
    manifest_object_key text,
    manifest_sha256 text,
    failure_code text,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    completed_at timestamptz,
    expires_at timestamptz NOT NULL,
    CHECK ((status='READY') = (data_object_key IS NOT NULL AND data_sha256 IS NOT NULL AND manifest_object_key IS NOT NULL AND manifest_sha256 IS NOT NULL AND completed_at IS NOT NULL)),
    CHECK (status<>'FAILED' OR failure_code IS NOT NULL)
);

CREATE INDEX audit_export_receipts_tenant_time_idx
    ON audit_export_receipts(tenant_id, created_at DESC, id DESC);
CREATE INDEX audit_export_receipts_expiry_idx
    ON audit_export_receipts(expires_at, id);

COMMIT;
