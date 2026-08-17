BEGIN;

CREATE TABLE monitoring_form_templates (
    revision_id uuid PRIMARY KEY DEFAULT uuidv7(),
    id uuid NOT NULL DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code text NOT NULL CHECK (code=btrim(code) AND char_length(code) BETWEEN 1 AND 128),
    name text NOT NULL CHECK (name=btrim(name) AND char_length(name) BETWEEN 1 AND 512),
    purpose text NOT NULL CHECK (purpose=btrim(purpose) AND char_length(purpose) BETWEEN 1 AND 2000),
    fields jsonb NOT NULL CHECK (jsonb_typeof(fields)='array' AND jsonb_array_length(fields) BETWEEN 1 AND 50),
    status text NOT NULL CHECK (status IN ('DRAFT','PENDING_APPROVAL','ACTIVE','REJECTED','PAUSED','RETIRED')),
    is_current boolean NOT NULL DEFAULT false,
    effective_from timestamptz,
    effective_until timestamptz,
    version bigint NOT NULL CHECK (version > 0),
    created_by uuid,
    submitted_by uuid,
    approved_by uuid,
    rejected_by uuid,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK (updated_at >= created_at),
    UNIQUE(tenant_id, id, version),
    FOREIGN KEY (created_by,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (submitted_by,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (approved_by,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (rejected_by,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK ((status IN ('ACTIVE','PAUSED') AND is_current AND effective_from IS NOT NULL AND effective_until IS NULL)
        OR (status='RETIRED' AND NOT is_current AND effective_from IS NOT NULL AND effective_until IS NOT NULL)
        OR (status IN ('DRAFT','PENDING_APPROVAL','REJECTED') AND NOT is_current AND effective_from IS NULL AND effective_until IS NULL)),
    CHECK (effective_until IS NULL OR effective_from <= effective_until)
);
CREATE UNIQUE INDEX monitoring_form_templates_current_id_idx ON monitoring_form_templates(tenant_id,id) WHERE is_current;
CREATE UNIQUE INDEX monitoring_form_templates_current_code_idx ON monitoring_form_templates(tenant_id,code) WHERE is_current;
CREATE INDEX monitoring_form_templates_history_idx ON monitoring_form_templates(tenant_id,id,version DESC);

CREATE TABLE monitoring_checks (
    revision_id uuid PRIMARY KEY DEFAULT uuidv7(),
    id uuid NOT NULL DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    program_id uuid NOT NULL,
    requirement_id uuid,
    control_implementation_id uuid,
    evidence_contract_id uuid,
    code text NOT NULL CHECK (code=btrim(code) AND char_length(code) BETWEEN 1 AND 128),
    name text NOT NULL CHECK (name=btrim(name) AND char_length(name) BETWEEN 1 AND 512),
    claim text NOT NULL CHECK (claim=btrim(claim) AND char_length(claim) BETWEEN 1 AND 2000),
    input_kind text NOT NULL CHECK (input_kind IN ('FORM','SOURCE')),
    form_template_id uuid,
    form_template_version bigint,
    binding_id uuid,
    binding_version bigint,
    source_rules jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(source_rules)='array' AND jsonb_array_length(source_rules) <= 100),
    thresholds jsonb NOT NULL CHECK (jsonb_typeof(thresholds)='object'),
    freshness_minutes integer NOT NULL CHECK (freshness_minutes BETWEEN 1 AND 525600),
    minimum_coverage numeric(6,5) NOT NULL CHECK (minimum_coverage BETWEEN 0 AND 1),
    owner_principal_id uuid,
    reviewer_principal_id uuid,
    failure_action text NOT NULL CHECK (failure_action IN ('REVIEW','RECOMMEND_MATTER')),
    status text NOT NULL CHECK (status IN ('DRAFT','PENDING_APPROVAL','ACTIVE','REJECTED','PAUSED','RETIRED')),
    is_current boolean NOT NULL DEFAULT false,
    effective_from timestamptz,
    effective_until timestamptz,
    version bigint NOT NULL CHECK (version > 0),
    created_by uuid,
    submitted_by uuid,
    approved_by uuid,
    rejected_by uuid,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp() CHECK (updated_at >= created_at),
    UNIQUE(tenant_id, id, version),
    FOREIGN KEY (program_id,tenant_id) REFERENCES programs(id,tenant_id),
    FOREIGN KEY (requirement_id,tenant_id,program_id) REFERENCES program_requirements(id,tenant_id,program_id),
    FOREIGN KEY (control_implementation_id,tenant_id,program_id) REFERENCES control_implementations(id,tenant_id,program_id),
    FOREIGN KEY (evidence_contract_id,tenant_id,program_id) REFERENCES evidence_contracts(id,tenant_id,program_id),
    FOREIGN KEY (tenant_id,form_template_id,form_template_version) REFERENCES monitoring_form_templates(tenant_id,id,version),
    FOREIGN KEY (owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (reviewer_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (created_by,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (submitted_by,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (approved_by,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (rejected_by,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK ((input_kind='FORM' AND form_template_id IS NOT NULL AND form_template_version > 0 AND binding_id IS NULL AND binding_version IS NULL AND source_rules='[]'::jsonb)
        OR (input_kind='SOURCE' AND form_template_id IS NULL AND form_template_version IS NULL AND binding_id IS NOT NULL AND binding_version > 0 AND jsonb_array_length(source_rules) > 0)),
    CHECK ((status IN ('ACTIVE','PAUSED') AND is_current AND effective_from IS NOT NULL AND effective_until IS NULL)
        OR (status='RETIRED' AND NOT is_current AND effective_from IS NOT NULL AND effective_until IS NOT NULL)
        OR (status IN ('DRAFT','PENDING_APPROVAL','REJECTED') AND NOT is_current AND effective_from IS NULL AND effective_until IS NULL)),
    CHECK (effective_until IS NULL OR effective_from <= effective_until)
);
CREATE UNIQUE INDEX monitoring_checks_current_id_idx ON monitoring_checks(tenant_id,id) WHERE is_current;
CREATE UNIQUE INDEX monitoring_checks_current_code_idx ON monitoring_checks(tenant_id,program_id,code) WHERE is_current;
CREATE INDEX monitoring_checks_program_idx ON monitoring_checks(tenant_id,program_id,is_current,status,code,id);
CREATE INDEX monitoring_checks_history_idx ON monitoring_checks(tenant_id,id,version DESC);

CREATE TABLE monitoring_results (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    program_id uuid NOT NULL,
    monitoring_check_id uuid NOT NULL,
    monitoring_check_version bigint NOT NULL CHECK (monitoring_check_version > 0),
    input_kind text NOT NULL CHECK (input_kind IN ('FORM','SOURCE')),
    input_reference_id text NOT NULL CHECK (btrim(input_reference_id)<>'' AND char_length(input_reference_id) <= 512),
    input_reference_version bigint NOT NULL CHECK (input_reference_version > 0),
    evaluation jsonb NOT NULL CHECK (jsonb_typeof(evaluation)='object'),
    source_receipt jsonb CHECK (source_receipt IS NULL OR jsonb_typeof(source_receipt)='object'),
    submission_provenance jsonb CHECK (submission_provenance IS NULL OR jsonb_typeof(submission_provenance)='object'),
    evaluated_at timestamptz NOT NULL,
    evaluator_version text NOT NULL CHECK (evaluator_version=btrim(evaluator_version) AND char_length(evaluator_version) BETWEEN 1 AND 128),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(tenant_id, monitoring_check_id, input_reference_id, evaluator_version),
    FOREIGN KEY (program_id,tenant_id) REFERENCES programs(id,tenant_id),
    FOREIGN KEY (tenant_id,monitoring_check_id,monitoring_check_version) REFERENCES monitoring_checks(tenant_id,id,version)
);
CREATE INDEX monitoring_results_latest_idx ON monitoring_results(tenant_id,monitoring_check_id,evaluated_at DESC,id DESC);
CREATE INDEX monitoring_results_program_idx ON monitoring_results(tenant_id,program_id,evaluated_at DESC,id DESC);

ALTER TABLE capture_requests
    ADD COLUMN form_template_id uuid,
    ADD COLUMN form_template_version bigint,
    ADD COLUMN collection_period_start timestamptz,
    ADD COLUMN collection_period_end timestamptz,
    ADD CONSTRAINT capture_requests_form_template_pair CHECK ((form_template_id IS NULL AND form_template_version IS NULL) OR (form_template_id IS NOT NULL AND form_template_version > 0)),
    ADD CONSTRAINT capture_requests_collection_period_pair CHECK ((collection_period_start IS NULL AND collection_period_end IS NULL) OR (collection_period_start IS NOT NULL AND collection_period_end IS NOT NULL AND collection_period_start <= collection_period_end)),
    ADD CONSTRAINT capture_requests_form_template_fk FOREIGN KEY (tenant_id,form_template_id,form_template_version) REFERENCES monitoring_form_templates(tenant_id,id,version);

CREATE INDEX capture_requests_form_template_idx ON capture_requests(tenant_id,form_template_id,form_template_version,created_at DESC) WHERE form_template_id IS NOT NULL;

COMMIT;
