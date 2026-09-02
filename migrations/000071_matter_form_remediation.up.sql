CREATE TABLE matter_form_remediation_bindings (
    id uuid PRIMARY KEY,
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid NOT NULL,
    program_id uuid NOT NULL,
    matter_id uuid NOT NULL,
    subject_type text NOT NULL CHECK (subject_type = 'MATTER'),
    subject_id uuid NOT NULL CHECK (subject_id = matter_id),
    matter_version_at_binding bigint NOT NULL CHECK (matter_version_at_binding > 0),
    form_template_id uuid NOT NULL,
    form_template_version bigint NOT NULL CHECK (form_template_version > 0),
    mappings jsonb NOT NULL CHECK (jsonb_typeof(mappings) = 'array' AND jsonb_array_length(mappings) > 0),
    action_id uuid,
    verification_contract_id uuid NOT NULL,
    minimum_score numeric CHECK (minimum_score BETWEEN 0 AND 100),
    maximum_adverse_score numeric CHECK (maximum_adverse_score BETWEEN 0 AND 100),
    purpose text NOT NULL CHECK (length(btrim(purpose)) > 0),
    audience_class text NOT NULL CHECK (audience_class IN ('EXTERNAL')),
    responder_class text NOT NULL CHECK (length(btrim(responder_class)) > 0),
    status text NOT NULL CHECK (status = 'ACTIVE'),
    effective_from timestamptz NOT NULL,
    created_by uuid NOT NULL,
    created_at timestamptz NOT NULL,
    version bigint NOT NULL DEFAULT 1 CHECK (version = 1),
    FOREIGN KEY (tenant_id, legal_entity_id) REFERENCES legal_entities(tenant_id, id),
    FOREIGN KEY (tenant_id, program_id) REFERENCES programs(tenant_id, id),
    FOREIGN KEY (tenant_id, matter_id) REFERENCES matters(tenant_id, id),
    FOREIGN KEY (tenant_id, subject_id) REFERENCES matters(tenant_id, id),
    FOREIGN KEY (tenant_id, form_template_id, form_template_version) REFERENCES monitoring_form_templates(tenant_id, id, version),
    FOREIGN KEY (tenant_id, matter_id, action_id) REFERENCES matter_actions(tenant_id, matter_id, id),
    FOREIGN KEY (tenant_id, matter_id, verification_contract_id) REFERENCES verification_contracts(tenant_id, matter_id, id),
    FOREIGN KEY (tenant_id, created_by) REFERENCES principals(tenant_id, id),
    UNIQUE (tenant_id, id, version)
);

CREATE INDEX matter_form_remediation_bindings_matter_idx
    ON matter_form_remediation_bindings(tenant_id, matter_id, created_at DESC, id DESC);

CREATE INDEX matter_form_remediation_bindings_active_idx
    ON matter_form_remediation_bindings(tenant_id, legal_entity_id, matter_id, effective_from DESC, id DESC)
    WHERE status = 'ACTIVE';

CREATE TABLE matter_form_remediation_events (
    id bigserial PRIMARY KEY,
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    binding_id uuid NOT NULL,
    binding_version bigint NOT NULL,
    event_type text NOT NULL CHECK (event_type = 'MATTER_FORM_REMEDIATION_BOUND'),
    actor_principal_id uuid NOT NULL,
    occurred_at timestamptz NOT NULL,
    payload jsonb NOT NULL,
    FOREIGN KEY (tenant_id, binding_id, binding_version) REFERENCES matter_form_remediation_bindings(tenant_id, id, version),
    FOREIGN KEY (tenant_id, actor_principal_id) REFERENCES principals(tenant_id, id)
);

CREATE INDEX matter_form_remediation_events_binding_idx
    ON matter_form_remediation_events(tenant_id, binding_id, occurred_at, id);

CREATE TABLE matter_form_remediation_applications (
    id uuid PRIMARY KEY,
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid NOT NULL,
    binding_id uuid NOT NULL,
    binding_version bigint NOT NULL,
    matter_id uuid NOT NULL,
    matter_version bigint NOT NULL CHECK (matter_version > 0),
    distribution_id uuid NOT NULL,
    response_revision_id uuid NOT NULL,
    response_revision bigint NOT NULL CHECK (response_revision > 0),
    submission_id uuid NOT NULL,
    verification_contract_id uuid NOT NULL,
    applied_field_ids jsonb NOT NULL CHECK (jsonb_typeof(applied_field_ids) = 'array'),
    applied_by uuid NOT NULL,
    applied_at timestamptz NOT NULL,
    FOREIGN KEY (tenant_id, legal_entity_id) REFERENCES legal_entities(tenant_id, id),
    FOREIGN KEY (tenant_id, binding_id, binding_version) REFERENCES matter_form_remediation_bindings(tenant_id, id, version),
    FOREIGN KEY (tenant_id, matter_id) REFERENCES matters(tenant_id, id),
    FOREIGN KEY (distribution_id, tenant_id) REFERENCES capture_form_distributions(id, tenant_id),
    FOREIGN KEY (response_revision_id, tenant_id) REFERENCES capture_response_revisions(id, tenant_id),
    FOREIGN KEY (submission_id, tenant_id, distribution_id) REFERENCES capture_submissions(id, tenant_id, distribution_id),
    FOREIGN KEY (tenant_id, matter_id, verification_contract_id) REFERENCES verification_contracts(tenant_id, matter_id, id),
    FOREIGN KEY (tenant_id, applied_by) REFERENCES principals(tenant_id, id),
    UNIQUE (tenant_id, binding_id, response_revision_id)
);

CREATE INDEX matter_form_remediation_applications_binding_idx
    ON matter_form_remediation_applications(tenant_id, binding_id, applied_at DESC, id DESC);
