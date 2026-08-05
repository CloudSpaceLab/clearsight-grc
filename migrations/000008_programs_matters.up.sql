BEGIN;

CREATE TABLE programs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid,
    code text NOT NULL,
    name text NOT NULL,
    program_type text NOT NULL,
    status text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','ACTIVE','PAUSED','RETIRED')),
    owning_function text NOT NULL,
    owner_principal_id uuid,
    authority_principal_id uuid,
    jurisdiction text NOT NULL DEFAULT '',
    scope jsonb NOT NULL DEFAULT '{}'::jsonb,
    effective_from timestamptz NOT NULL,
    effective_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1,
    UNIQUE (id, tenant_id),
    CHECK (jsonb_typeof(scope)='object'),
    CHECK (effective_until IS NULL OR effective_from < effective_until),
    CONSTRAINT programs_legal_entity_tenant_fk FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    CONSTRAINT programs_owner_tenant_fk FOREIGN KEY (owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CONSTRAINT programs_authority_tenant_fk FOREIGN KEY (authority_principal_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE UNIQUE INDEX programs_active_code_idx ON programs(tenant_id,code) WHERE status<>'RETIRED';
CREATE INDEX programs_queue_idx ON programs(tenant_id,status,updated_at DESC,id);
CREATE INDEX programs_owner_idx ON programs(tenant_id,owner_principal_id,status) WHERE status IN ('ACTIVE','PAUSED');

CREATE TABLE program_requirements (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    program_id uuid NOT NULL,
    source_id uuid,
    code text NOT NULL,
    title text NOT NULL,
    statement text NOT NULL,
    source_anchor text NOT NULL DEFAULT '',
    modality text NOT NULL CHECK (modality IN ('MUST','MUST_NOT','MAY','SHOULD','EXPECTED')),
    actor text NOT NULL DEFAULT '',
    action text NOT NULL DEFAULT '',
    object_name text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','APPROVED','SUPERSEDED','RETIRED')),
    effective_from timestamptz NOT NULL,
    effective_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1,
    UNIQUE (id,tenant_id,program_id),
    UNIQUE (tenant_id,program_id,code,version),
    CHECK (effective_until IS NULL OR effective_from < effective_until),
    CONSTRAINT program_requirements_program_tenant_fk FOREIGN KEY (program_id,tenant_id) REFERENCES programs(id,tenant_id),
    CONSTRAINT program_requirements_source_tenant_fk FOREIGN KEY (source_id,tenant_id) REFERENCES evidence_sources(id,tenant_id)
);
CREATE INDEX program_requirements_current_idx ON program_requirements(tenant_id,program_id,status,effective_from DESC,id);

CREATE TABLE program_applicability (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    program_id uuid NOT NULL,
    requirement_id uuid NOT NULL,
    status text NOT NULL CHECK (status IN ('POTENTIALLY_APPLICABLE','APPLICABLE','PARTIALLY_APPLICABLE','NOT_APPLICABLE','APPLIES_LATER','SUPERSEDED')),
    scope jsonb NOT NULL DEFAULT '{}'::jsonb,
    rationale text NOT NULL,
    approved_by uuid NOT NULL,
    effective_from timestamptz NOT NULL,
    effective_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1,
    UNIQUE (id,tenant_id,program_id),
    CHECK (jsonb_typeof(scope)='object'),
    CHECK (effective_until IS NULL OR effective_from < effective_until),
    CONSTRAINT program_applicability_program_tenant_fk FOREIGN KEY (program_id,tenant_id) REFERENCES programs(id,tenant_id),
    CONSTRAINT program_applicability_requirement_scope_fk FOREIGN KEY (requirement_id,tenant_id,program_id) REFERENCES program_requirements(id,tenant_id,program_id),
    CONSTRAINT program_applicability_approver_tenant_fk FOREIGN KEY (approved_by,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX program_applicability_current_idx ON program_applicability(tenant_id,program_id,requirement_id,effective_from DESC,id);

CREATE TABLE control_objectives (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    program_id uuid NOT NULL,
    code text NOT NULL,
    name text NOT NULL,
    outcome text NOT NULL,
    status text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','ACTIVE','RETIRED')),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1,
    UNIQUE (id,tenant_id,program_id),
    UNIQUE (tenant_id,program_id,code),
    CONSTRAINT control_objectives_program_tenant_fk FOREIGN KEY (program_id,tenant_id) REFERENCES programs(id,tenant_id)
);
CREATE INDEX control_objectives_program_idx ON control_objectives(tenant_id,program_id,status,id);

CREATE TABLE control_implementations (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    program_id uuid NOT NULL,
    objective_id uuid NOT NULL,
    name text NOT NULL,
    description text NOT NULL,
    implementation_type text NOT NULL,
    owner_principal_id uuid,
    scope jsonb NOT NULL DEFAULT '{}'::jsonb,
    status text NOT NULL DEFAULT 'PLANNED' CHECK (status IN ('PLANNED','IN_PROGRESS','IMPLEMENTED','INACTIVE','RETIRED')),
    effective_from timestamptz NOT NULL,
    effective_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1,
    UNIQUE (id,tenant_id,program_id),
    CHECK (jsonb_typeof(scope)='object'),
    CHECK (effective_until IS NULL OR effective_from < effective_until),
    CONSTRAINT control_implementations_program_tenant_fk FOREIGN KEY (program_id,tenant_id) REFERENCES programs(id,tenant_id),
    CONSTRAINT control_implementations_objective_scope_fk FOREIGN KEY (objective_id,tenant_id,program_id) REFERENCES control_objectives(id,tenant_id,program_id),
    CONSTRAINT control_implementations_owner_tenant_fk FOREIGN KEY (owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX control_implementations_program_idx ON control_implementations(tenant_id,program_id,status,updated_at DESC,id);

CREATE TABLE requirement_control_links (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    program_id uuid NOT NULL,
    requirement_id uuid NOT NULL,
    implementation_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (tenant_id,program_id,requirement_id,implementation_id),
    CONSTRAINT requirement_control_links_program_tenant_fk FOREIGN KEY (program_id,tenant_id) REFERENCES programs(id,tenant_id),
    CONSTRAINT requirement_control_links_requirement_scope_fk FOREIGN KEY (requirement_id,tenant_id,program_id) REFERENCES program_requirements(id,tenant_id,program_id),
    CONSTRAINT requirement_control_links_implementation_scope_fk FOREIGN KEY (implementation_id,tenant_id,program_id) REFERENCES control_implementations(id,tenant_id,program_id)
);
CREATE INDEX requirement_control_links_program_idx ON requirement_control_links(tenant_id,program_id,requirement_id);

CREATE TABLE evidence_contracts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    program_id uuid NOT NULL,
    requirement_id uuid,
    control_implementation_id uuid,
    code text NOT NULL,
    name text NOT NULL,
    claim text NOT NULL,
    acceptable_source_ids jsonb NOT NULL DEFAULT '[]'::jsonb,
    population_scope jsonb NOT NULL DEFAULT '{}'::jsonb,
    freshness_minutes integer NOT NULL CHECK (freshness_minutes BETWEEN 1 AND 525600),
    minimum_coverage numeric(6,5) NOT NULL CHECK (minimum_coverage BETWEEN 0 AND 1),
    independence_required boolean NOT NULL DEFAULT false,
    contradiction_policy text NOT NULL CHECK (contradiction_policy IN ('HOLD','REVIEW','FAIL')),
    failure_action text NOT NULL CHECK (failure_action IN ('FLAG','REQUEST','MATTER','BLOCK')),
    status text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','ACTIVE','RETIRED')),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1,
    UNIQUE (id,tenant_id,program_id),
    UNIQUE (tenant_id,program_id,code),
    CHECK (jsonb_typeof(acceptable_source_ids)='array'),
    CHECK (jsonb_typeof(population_scope)='object'),
    CHECK ((requirement_id IS NULL) <> (control_implementation_id IS NULL)),
    CONSTRAINT evidence_contracts_program_tenant_fk FOREIGN KEY (program_id,tenant_id) REFERENCES programs(id,tenant_id),
    CONSTRAINT evidence_contracts_requirement_scope_fk FOREIGN KEY (requirement_id,tenant_id,program_id) REFERENCES program_requirements(id,tenant_id,program_id),
    CONSTRAINT evidence_contracts_implementation_scope_fk FOREIGN KEY (control_implementation_id,tenant_id,program_id) REFERENCES control_implementations(id,tenant_id,program_id)
);
CREATE INDEX evidence_contracts_program_idx ON evidence_contracts(tenant_id,program_id,status,id);

CREATE TABLE evidence_contract_sources (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    program_id uuid NOT NULL,
    contract_id uuid NOT NULL,
    source_id uuid NOT NULL,
    PRIMARY KEY (tenant_id,program_id,contract_id,source_id),
    CONSTRAINT evidence_contract_sources_contract_scope_fk FOREIGN KEY (contract_id,tenant_id,program_id) REFERENCES evidence_contracts(id,tenant_id,program_id),
    CONSTRAINT evidence_contract_sources_source_tenant_fk FOREIGN KEY (source_id,tenant_id) REFERENCES evidence_sources(id,tenant_id)
);
CREATE INDEX evidence_contract_sources_source_idx ON evidence_contract_sources(tenant_id,source_id,program_id);

CREATE TABLE evidence_assessments (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    program_id uuid NOT NULL,
    contract_id uuid NOT NULL,
    conclusion text NOT NULL CHECK (conclusion IN ('SUPPORTED','PARTIALLY_SUPPORTED','UNSUPPORTED','CONTRADICTED','INDETERMINATE','EXPIRED')),
    coverage numeric(6,5) NOT NULL CHECK (coverage BETWEEN 0 AND 1),
    basis jsonb NOT NULL DEFAULT '{}'::jsonb,
    valid_until timestamptz,
    assessed_by uuid,
    assessed_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (jsonb_typeof(basis)='object'),
    CONSTRAINT evidence_assessments_program_tenant_fk FOREIGN KEY (program_id,tenant_id) REFERENCES programs(id,tenant_id),
    CONSTRAINT evidence_assessments_contract_scope_fk FOREIGN KEY (contract_id,tenant_id,program_id) REFERENCES evidence_contracts(id,tenant_id,program_id),
    CONSTRAINT evidence_assessments_assessor_tenant_fk FOREIGN KEY (assessed_by,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX evidence_assessments_latest_idx ON evidence_assessments(tenant_id,program_id,contract_id,assessed_at DESC,id DESC);

CREATE TABLE program_state_snapshots (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    program_id uuid NOT NULL,
    overall_state text NOT NULL CHECK (overall_state IN ('CURRENT','AT_RISK','GAP_IDENTIFIED','EVIDENCE_INSUFFICIENT','IMPLEMENTATION_PENDING','OVERDUE','UNDER_REVIEW','NOT_APPLICABLE','UNKNOWN')),
    dimensions jsonb NOT NULL,
    reasons jsonb NOT NULL DEFAULT '[]'::jsonb,
    open_matter_count integer NOT NULL DEFAULT 0 CHECK (open_matter_count >= 0),
    trigger_type text NOT NULL DEFAULT '',
    trigger_id text NOT NULL DEFAULT '',
    generated_at timestamptz NOT NULL,
    program_version bigint NOT NULL,
    CHECK (jsonb_typeof(dimensions)='object'),
    CHECK (jsonb_typeof(reasons)='array'),
    CONSTRAINT program_state_snapshots_program_tenant_fk FOREIGN KEY (program_id,tenant_id) REFERENCES programs(id,tenant_id)
);
CREATE INDEX program_state_snapshots_latest_idx ON program_state_snapshots(tenant_id,program_id,generated_at DESC,id DESC);

CREATE TABLE program_trigger_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    program_id uuid NOT NULL,
    trigger_type text NOT NULL,
    subject_type text NOT NULL DEFAULT '',
    subject_id text NOT NULL DEFAULT '',
    dedupe_key text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    observed_at timestamptz NOT NULL,
    source text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (tenant_id,dedupe_key),
    CHECK (jsonb_typeof(payload)='object'),
    CONSTRAINT program_trigger_events_program_tenant_fk FOREIGN KEY (program_id,tenant_id) REFERENCES programs(id,tenant_id)
);
CREATE INDEX program_trigger_events_program_idx ON program_trigger_events(tenant_id,program_id,observed_at DESC,id DESC);

CREATE TABLE matters (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    reference text NOT NULL,
    matter_type text NOT NULL CHECK (matter_type IN ('REGULATORY_CHANGE','SUPERVISORY_FINDING','AUTHORITY_REQUEST','RISK_SITUATION','CONTROL_GAP','AUDIT_FINDING','EXCEPTION','INCIDENT','OPERATIONAL_LOSS','DATA_BREACH','VENDOR_DEFICIENCY','CUSTOMER_CONCERN','OVERDUE_OBLIGATION','FAILED_VERIFICATION','EVIDENCE_CONTRADICTION','KRI_BREACH')),
    status text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','TRIAGE','ASSESSMENT','DECISION_REQUIRED','ACTION_IN_PROGRESS','RESPONSE_PREPARATION','VERIFICATION','CLOSED','CANCELLED')),
    priority integer NOT NULL CHECK (priority BETWEEN 1 AND 5),
    title text NOT NULL,
    summary text NOT NULL,
    scope jsonb NOT NULL DEFAULT '{}'::jsonb,
    source_type text NOT NULL DEFAULT '',
    source_id text NOT NULL DEFAULT '',
    trigger_type text NOT NULL DEFAULT '',
    trigger_id text NOT NULL DEFAULT '',
    trigger_key text NOT NULL DEFAULT '',
    known_facts jsonb NOT NULL DEFAULT '{}'::jsonb,
    missing_facts jsonb NOT NULL DEFAULT '[]'::jsonb,
    contradictions jsonb NOT NULL DEFAULT '[]'::jsonb,
    owner_principal_id uuid,
    required_authority text NOT NULL DEFAULT '',
    due_at timestamptz,
    closed_at timestamptz,
    closure_reason text NOT NULL DEFAULT '',
    reopen_count integer NOT NULL DEFAULT 0 CHECK (reopen_count >= 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1,
    UNIQUE (id,tenant_id),
    UNIQUE (tenant_id,reference),
    CHECK (jsonb_typeof(scope)='object'),
    CHECK (jsonb_typeof(known_facts)='object'),
    CHECK (jsonb_typeof(missing_facts)='array'),
    CHECK (jsonb_typeof(contradictions)='array'),
    CHECK ((status='CLOSED') = (closed_at IS NOT NULL)),
    CONSTRAINT matters_owner_tenant_fk FOREIGN KEY (owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE UNIQUE INDEX matters_open_trigger_idx ON matters(tenant_id,trigger_key) WHERE trigger_key<>'' AND status NOT IN ('CLOSED','CANCELLED');
CREATE INDEX matters_queue_idx ON matters(tenant_id,status,priority DESC,due_at,updated_at DESC,id);

CREATE TABLE matter_links (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    matter_id uuid NOT NULL,
    program_id uuid,
    requirement_id uuid,
    control_id uuid,
    relationship text NOT NULL DEFAULT 'AFFECTS',
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (program_id IS NOT NULL),
    CHECK (requirement_id IS NULL OR program_id IS NOT NULL),
    CHECK (control_id IS NULL OR program_id IS NOT NULL),
    CONSTRAINT matter_links_matter_tenant_fk FOREIGN KEY (matter_id,tenant_id) REFERENCES matters(id,tenant_id),
    CONSTRAINT matter_links_program_tenant_fk FOREIGN KEY (program_id,tenant_id) REFERENCES programs(id,tenant_id),
    CONSTRAINT matter_links_requirement_scope_fk FOREIGN KEY (requirement_id,tenant_id,program_id) REFERENCES program_requirements(id,tenant_id,program_id),
    CONSTRAINT matter_links_control_scope_fk FOREIGN KEY (control_id,tenant_id,program_id) REFERENCES control_implementations(id,tenant_id,program_id)
);
CREATE UNIQUE INDEX matter_links_unique_idx ON matter_links(tenant_id,matter_id,COALESCE(program_id,'00000000-0000-0000-0000-000000000000'::uuid),COALESCE(requirement_id,'00000000-0000-0000-0000-000000000000'::uuid),COALESCE(control_id,'00000000-0000-0000-0000-000000000000'::uuid),relationship);
CREATE INDEX matter_links_program_idx ON matter_links(tenant_id,program_id,matter_id) WHERE program_id IS NOT NULL;

CREATE TABLE matter_decisions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    matter_id uuid NOT NULL,
    decision_type text NOT NULL,
    status text NOT NULL CHECK (status IN ('PROPOSED','APPROVED','CONDITIONALLY_APPROVED','REJECTED','RETURNED','EXPIRED','SUPERSEDED')),
    options jsonb NOT NULL DEFAULT '[]'::jsonb,
    selected_option text NOT NULL DEFAULT '',
    rationale text NOT NULL,
    conditions jsonb NOT NULL DEFAULT '[]'::jsonb,
    authority_principal_id uuid,
    decided_at timestamptz,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1,
    UNIQUE (id,tenant_id,matter_id),
    CHECK (jsonb_typeof(options)='array'),
    CHECK (jsonb_typeof(conditions) IN ('array','object')),
    CONSTRAINT matter_decisions_matter_tenant_fk FOREIGN KEY (matter_id,tenant_id) REFERENCES matters(id,tenant_id),
    CONSTRAINT matter_decisions_authority_tenant_fk FOREIGN KEY (authority_principal_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX matter_decisions_matter_idx ON matter_decisions(tenant_id,matter_id,created_at DESC,id);

CREATE TABLE matter_actions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    matter_id uuid NOT NULL,
    title text NOT NULL,
    description text NOT NULL,
    owner_principal_id uuid,
    status text NOT NULL DEFAULT 'PLANNED' CHECK (status IN ('PLANNED','IN_PROGRESS','IMPLEMENTED','BLOCKED','CANCELLED')),
    due_at timestamptz,
    implemented_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1,
    UNIQUE (id,tenant_id,matter_id),
    CHECK ((status='IMPLEMENTED') = (implemented_at IS NOT NULL)),
    CONSTRAINT matter_actions_matter_tenant_fk FOREIGN KEY (matter_id,tenant_id) REFERENCES matters(id,tenant_id),
    CONSTRAINT matter_actions_owner_tenant_fk FOREIGN KEY (owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX matter_actions_queue_idx ON matter_actions(tenant_id,status,due_at,matter_id,id);

CREATE TABLE verification_contracts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    matter_id uuid NOT NULL,
    action_id uuid,
    expected_outcome text NOT NULL,
    baseline jsonb NOT NULL DEFAULT '{}'::jsonb,
    scope jsonb NOT NULL DEFAULT '{}'::jsonb,
    measurement_source_id uuid,
    threshold jsonb NOT NULL DEFAULT '{}'::jsonb,
    observation_period_minutes integer NOT NULL DEFAULT 0 CHECK (observation_period_minutes BETWEEN 0 AND 525600),
    authority_principal_id uuid,
    failure_response text NOT NULL CHECK (failure_response IN ('REOPEN','CREATE_MATTER','ESCALATE','BLOCK_CLOSE')),
    status text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','RETIRED')),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1,
    UNIQUE (id,tenant_id,matter_id),
    CHECK (jsonb_typeof(baseline)='object'),
    CHECK (jsonb_typeof(scope)='object'),
    CHECK (jsonb_typeof(threshold)='object'),
    CONSTRAINT verification_contracts_matter_tenant_fk FOREIGN KEY (matter_id,tenant_id) REFERENCES matters(id,tenant_id),
    CONSTRAINT verification_contracts_action_scope_fk FOREIGN KEY (action_id,tenant_id,matter_id) REFERENCES matter_actions(id,tenant_id,matter_id),
    CONSTRAINT verification_contracts_source_tenant_fk FOREIGN KEY (measurement_source_id,tenant_id) REFERENCES evidence_sources(id,tenant_id),
    CONSTRAINT verification_contracts_authority_tenant_fk FOREIGN KEY (authority_principal_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX verification_contracts_matter_idx ON verification_contracts(tenant_id,matter_id,status,id);

CREATE TABLE verification_results (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    matter_id uuid NOT NULL,
    contract_id uuid NOT NULL,
    result text NOT NULL CHECK (result IN ('PASS','FAIL','INCONCLUSIVE')),
    observations jsonb NOT NULL DEFAULT '{}'::jsonb,
    evidence_references jsonb NOT NULL DEFAULT '[]'::jsonb,
    reviewer_principal_id uuid,
    rationale text NOT NULL,
    observed_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (jsonb_typeof(observations)='object'),
    CHECK (jsonb_typeof(evidence_references)='array'),
    CONSTRAINT verification_results_matter_tenant_fk FOREIGN KEY (matter_id,tenant_id) REFERENCES matters(id,tenant_id),
    CONSTRAINT verification_results_contract_scope_fk FOREIGN KEY (contract_id,tenant_id,matter_id) REFERENCES verification_contracts(id,tenant_id,matter_id),
    CONSTRAINT verification_results_reviewer_tenant_fk FOREIGN KEY (reviewer_principal_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX verification_results_latest_idx ON verification_results(tenant_id,matter_id,contract_id,observed_at DESC,id DESC);

CREATE TABLE response_packages (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    matter_id uuid NOT NULL,
    purpose text NOT NULL,
    audience text NOT NULL,
    status text NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT','IN_REVIEW','APPROVED','TRANSMITTED','ACKNOWLEDGED','REJECTED','WITHDRAWN')),
    manifest jsonb NOT NULL DEFAULT '[]'::jsonb,
    approved_by uuid,
    transmitted_at timestamptz,
    acknowledged_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1,
    UNIQUE (id,tenant_id,matter_id),
    CHECK (jsonb_typeof(manifest)='array'),
    CONSTRAINT response_packages_matter_tenant_fk FOREIGN KEY (matter_id,tenant_id) REFERENCES matters(id,tenant_id),
    CONSTRAINT response_packages_approver_tenant_fk FOREIGN KEY (approved_by,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX response_packages_matter_idx ON response_packages(tenant_id,matter_id,created_at DESC,id);

CREATE TABLE continuity_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    aggregate_type text NOT NULL CHECK (aggregate_type IN ('PROGRAM','MATTER')),
    aggregate_id uuid NOT NULL,
    aggregate_version bigint NOT NULL CHECK (aggregate_version > 0),
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    actor_type text NOT NULL CHECK (actor_type IN ('PERSON','SERVICE','SYSTEM')),
    actor_id uuid,
    occurred_at timestamptz NOT NULL,
    UNIQUE (tenant_id,aggregate_type,aggregate_id,aggregate_version),
    CHECK (jsonb_typeof(payload)='object'),
    CONSTRAINT continuity_events_actor_tenant_fk FOREIGN KEY (actor_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX continuity_events_replay_idx ON continuity_events(tenant_id,aggregate_type,aggregate_id,aggregate_version);
CREATE INDEX continuity_events_time_idx ON continuity_events(tenant_id,occurred_at DESC,id);

COMMIT;
