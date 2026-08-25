BEGIN;

ALTER TABLE monitoring_form_templates
    DROP CONSTRAINT monitoring_form_templates_fields_check,
    ADD COLUMN presentation jsonb NOT NULL DEFAULT '{"default_mode":"AUTOMATIC","allow_mode_switch":false}'::jsonb,
    ADD COLUMN sections jsonb NOT NULL DEFAULT '[{"id":"general","title":"General"}]'::jsonb,
    ADD CONSTRAINT monitoring_form_templates_fields_check CHECK (jsonb_typeof(fields)='array' AND jsonb_array_length(fields) BETWEEN 1 AND 200),
    ADD CONSTRAINT monitoring_form_templates_presentation_check CHECK (
        jsonb_typeof(presentation)='object'
        AND presentation ? 'default_mode'
        AND presentation ? 'allow_mode_switch'
        AND COALESCE(presentation->>'default_mode','') IN ('CLASSIC','WIZARD','AUTOMATIC')
        AND COALESCE(jsonb_typeof(presentation->'allow_mode_switch'),'')='boolean'
        AND octet_length(presentation::text) <= 1024
    ),
    ADD CONSTRAINT monitoring_form_templates_sections_check CHECK (
        jsonb_typeof(sections)='array'
        AND jsonb_array_length(sections) BETWEEN 1 AND 20
        AND octet_length(sections::text) <= 65536
    );

ALTER TABLE capture_requests
    DROP CONSTRAINT capture_requests_fields_check,
    ADD COLUMN presentation jsonb NOT NULL DEFAULT '{"default_mode":"AUTOMATIC","allow_mode_switch":false}'::jsonb,
    ADD COLUMN sections jsonb NOT NULL DEFAULT '[{"id":"general","title":"General"}]'::jsonb,
    ADD COLUMN origin_type text,
    ADD COLUMN origin_id text,
    ADD COLUMN origin_version bigint,
    ADD CONSTRAINT capture_requests_fields_check CHECK (jsonb_typeof(fields)='array' AND jsonb_array_length(fields) BETWEEN 1 AND 200),
    ADD CONSTRAINT capture_requests_presentation_check CHECK (
        jsonb_typeof(presentation)='object'
        AND presentation ? 'default_mode'
        AND presentation ? 'allow_mode_switch'
        AND COALESCE(presentation->>'default_mode','') IN ('CLASSIC','WIZARD','AUTOMATIC')
        AND COALESCE(jsonb_typeof(presentation->'allow_mode_switch'),'')='boolean'
        AND octet_length(presentation::text) <= 1024
    ),
    ADD CONSTRAINT capture_requests_sections_check CHECK (
        jsonb_typeof(sections)='array'
        AND jsonb_array_length(sections) BETWEEN 1 AND 20
        AND octet_length(sections::text) <= 65536
    ),
    ADD CONSTRAINT capture_requests_origin_check CHECK (
        (origin_type IS NULL AND origin_id IS NULL AND origin_version IS NULL)
        OR (
            origin_type IS NOT NULL AND origin_id IS NOT NULL AND origin_version IS NOT NULL
            AND origin_type=btrim(origin_type) AND char_length(origin_type) BETWEEN 1 AND 128
            AND origin_id=btrim(origin_id) AND char_length(origin_id) BETWEEN 1 AND 512
            AND origin_version > 0
        )
    );

CREATE UNIQUE INDEX capture_requests_origin_idx
    ON capture_requests(tenant_id,origin_type,origin_id,origin_version)
    WHERE origin_type IS NOT NULL;

CREATE TABLE capture_response_drafts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    request_id uuid NOT NULL,
    session_id uuid NOT NULL,
    answers jsonb NOT NULL DEFAULT '{}'::jsonb,
    presentation_mode text NOT NULL CHECK (presentation_mode IN ('CLASSIC','WIZARD','AUTOMATIC')),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(tenant_id,request_id,session_id),
    FOREIGN KEY (request_id,tenant_id) REFERENCES capture_requests(id,tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (session_id,tenant_id,request_id) REFERENCES capture_sessions(id,tenant_id,request_id) ON DELETE CASCADE,
    CHECK (jsonb_typeof(answers)='object' AND octet_length(answers::text) <= 1048576),
    CHECK (updated_at >= created_at)
);

COMMIT;
