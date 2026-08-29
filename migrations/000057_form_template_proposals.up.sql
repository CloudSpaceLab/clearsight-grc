BEGIN;

ALTER TABLE document_imports
    ADD COLUMN extraction_details jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD CONSTRAINT document_imports_extraction_details_ck CHECK (
        jsonb_typeof(extraction_details)='object'
        AND octet_length(extraction_details::text) <= 4194304
    ),
    ADD CONSTRAINT document_imports_id_tenant_entity_key UNIQUE(id,tenant_id,legal_entity_id);

ALTER TABLE monitoring_form_templates
    ADD CONSTRAINT monitoring_form_templates_tenant_entity_revision_key
        UNIQUE(tenant_id,legal_entity_id,id,version);

CREATE TABLE form_template_proposals (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    source_kind text NOT NULL CHECK (source_kind IN ('DOCUMENT','AI')),
    source_document_id uuid,
    source_document_version bigint,
    source_sha256 text NOT NULL DEFAULT '',
    base_template_id uuid,
    base_template_version bigint,
    status text NOT NULL CHECK (status IN ('GENERATING','REVIEW_REQUIRED','ACCEPTED','REJECTED','FAILED')),
    proposed_contract jsonb NOT NULL DEFAULT '{}'::jsonb,
    field_changes jsonb NOT NULL DEFAULT '[]'::jsonb,
    unresolved_items jsonb NOT NULL DEFAULT '[]'::jsonb,
    provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
    failure_code text NOT NULL DEFAULT '',
    failure_message text NOT NULL DEFAULT '',
    created_by uuid NOT NULL,
    reviewed_by uuid,
    accepted_change_ids jsonb NOT NULL DEFAULT '[]'::jsonb,
    result_template_id uuid,
    result_template_version bigint,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    reviewed_at timestamptz,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    UNIQUE(id,tenant_id,legal_entity_id),
    CONSTRAINT form_template_proposals_entity_tenant_fk
        FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    CONSTRAINT form_template_proposals_source_document_fk
        FOREIGN KEY (source_document_id,tenant_id,legal_entity_id)
        REFERENCES document_imports(id,tenant_id,legal_entity_id),
    CONSTRAINT form_template_proposals_base_revision_fk
        FOREIGN KEY (tenant_id,legal_entity_id,base_template_id,base_template_version)
        REFERENCES monitoring_form_templates(tenant_id,legal_entity_id,id,version),
    CONSTRAINT form_template_proposals_result_revision_fk
        FOREIGN KEY (tenant_id,legal_entity_id,result_template_id,result_template_version)
        REFERENCES monitoring_form_templates(tenant_id,legal_entity_id,id,version),
    CONSTRAINT form_template_proposals_created_by_fk
        FOREIGN KEY (created_by,tenant_id) REFERENCES principals(id,tenant_id),
    CONSTRAINT form_template_proposals_reviewed_by_fk
        FOREIGN KEY (reviewed_by,tenant_id) REFERENCES principals(id,tenant_id),
    CONSTRAINT form_template_proposals_source_ck CHECK (
        (source_kind='DOCUMENT'
            AND source_document_id IS NOT NULL
            AND source_document_version > 0
            AND source_sha256 ~ '^[0-9a-f]{64}$')
        OR source_kind='AI'
    ),
    CONSTRAINT form_template_proposals_base_pair_ck CHECK (
        (base_template_id IS NULL AND base_template_version IS NULL)
        OR (base_template_id IS NOT NULL AND base_template_version > 0)
    ),
    CONSTRAINT form_template_proposals_result_pair_ck CHECK (
        (result_template_id IS NULL AND result_template_version IS NULL)
        OR (result_template_id IS NOT NULL AND result_template_version > 0)
    ),
    CONSTRAINT form_template_proposals_json_shape_ck CHECK (
        jsonb_typeof(proposed_contract)='object'
        AND jsonb_typeof(field_changes)='array'
        AND jsonb_typeof(unresolved_items)='array'
        AND jsonb_typeof(provenance)='object'
        AND jsonb_typeof(accepted_change_ids)='array'
    ),
    CONSTRAINT form_template_proposals_bounds_ck CHECK (
        octet_length(proposed_contract::text) <= 1048576
        AND jsonb_array_length(field_changes) <= 500
        AND octet_length(field_changes::text) <= 2097152
        AND jsonb_array_length(unresolved_items) <= 1000
        AND octet_length(unresolved_items::text) <= 1048576
        AND octet_length(provenance::text) <= 262144
        AND jsonb_array_length(accepted_change_ids) <= 500
        AND octet_length(accepted_change_ids::text) <= 65536
        AND char_length(failure_code) <= 128
        AND char_length(failure_message) <= 2000
    ),
    CONSTRAINT form_template_proposals_review_state_ck CHECK (
        (status IN ('GENERATING','REVIEW_REQUIRED','FAILED')
            AND reviewed_by IS NULL AND reviewed_at IS NULL
            AND accepted_change_ids='[]'::jsonb
            AND result_template_id IS NULL AND result_template_version IS NULL)
        OR (status='REJECTED'
            AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL
            AND accepted_change_ids='[]'::jsonb
            AND result_template_id IS NULL AND result_template_version IS NULL)
        OR (status='ACCEPTED'
            AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL
            AND jsonb_array_length(accepted_change_ids) > 0
            AND result_template_id IS NOT NULL AND result_template_version > 0)
    ),
    CHECK (updated_at >= created_at),
    CHECK (reviewed_at IS NULL OR reviewed_at >= created_at)
);

CREATE UNIQUE INDEX form_template_proposals_source_revision_uq
    ON form_template_proposals(
        tenant_id,legal_entity_id,source_kind,source_document_id,source_document_version,
        source_sha256,base_template_id,base_template_version
    ) NULLS NOT DISTINCT;
CREATE INDEX form_template_proposals_review_queue_idx
    ON form_template_proposals(tenant_id,legal_entity_id,status,updated_at DESC,id DESC);
CREATE INDEX form_template_proposals_source_idx
    ON form_template_proposals(tenant_id,legal_entity_id,source_document_id,created_at DESC,id DESC)
    WHERE source_document_id IS NOT NULL;
CREATE INDEX form_template_proposals_base_idx
    ON form_template_proposals(tenant_id,legal_entity_id,base_template_id,base_template_version,created_at DESC)
    WHERE base_template_id IS NOT NULL;

COMMIT;
