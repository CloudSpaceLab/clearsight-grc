BEGIN;

CREATE TABLE document_imports (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid REFERENCES legal_entities(id),
    file_name text NOT NULL,
    media_type text NOT NULL DEFAULT '',
    purpose text NOT NULL,
    source_type text NOT NULL DEFAULT 'DOCUMENT',
    size_bytes bigint NOT NULL CHECK (size_bytes > 0),
    sha256 text NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    storage_key text NOT NULL,
    artifact_status text NOT NULL CHECK (artifact_status IN ('STORED_UNSCANNED','AVAILABLE','QUARANTINED','DELETED')),
    extraction_status text NOT NULL CHECK (extraction_status IN ('EXTRACTED','UNSUPPORTED','FAILED')),
    extraction_method text NOT NULL,
    analysis_status text NOT NULL CHECK (analysis_status IN ('REVIEW_REQUIRED','NO_PROPOSALS','UNAVAILABLE')),
    analysis_method text NOT NULL,
    limitations jsonb NOT NULL DEFAULT '[]'::jsonb,
    sections jsonb NOT NULL DEFAULT '[]'::jsonb,
    proposals jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_by uuid NOT NULL REFERENCES principals(id),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1,
    CHECK (jsonb_typeof(limitations)='array'),
    CHECK (jsonb_typeof(sections)='array'),
    CHECK (jsonb_typeof(proposals)='array'),
    UNIQUE (tenant_id, storage_key)
);

CREATE INDEX document_imports_list_idx ON document_imports(tenant_id,created_at DESC,id DESC);
CREATE INDEX document_imports_review_idx ON document_imports(tenant_id,analysis_status,updated_at DESC,id DESC)
    WHERE analysis_status='REVIEW_REQUIRED';
CREATE INDEX document_imports_hash_idx ON document_imports(tenant_id,sha256,created_at DESC);

COMMIT;
