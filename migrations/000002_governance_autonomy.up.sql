BEGIN;

CREATE TABLE org_positions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    legal_entity_id uuid REFERENCES legal_entities(id),
    code text NOT NULL,
    title text NOT NULL,
    function_name text,
    parent_position_id uuid REFERENCES org_positions(id),
    occupant_principal_id uuid REFERENCES principals(id),
    valid_from timestamptz NOT NULL DEFAULT clock_timestamp(),
    valid_until timestamptz,
    recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX org_positions_active_code_idx ON org_positions(tenant_id, legal_entity_id, code) WHERE valid_until IS NULL;
CREATE INDEX org_positions_occupant_idx ON org_positions(tenant_id, occupant_principal_id) WHERE valid_until IS NULL;

CREATE TABLE role_templates (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    code text NOT NULL,
    name text NOT NULL,
    description text NOT NULL DEFAULT '',
    responsibilities text[] NOT NULL DEFAULT '{}',
    valid_from timestamptz NOT NULL DEFAULT clock_timestamp(),
    valid_until timestamptz,
    recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX role_templates_active_code_idx ON role_templates(tenant_id, code) WHERE valid_until IS NULL;

CREATE TABLE position_role_bindings (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    position_id uuid NOT NULL REFERENCES org_positions(id),
    role_template_id uuid NOT NULL REFERENCES role_templates(id),
    scope jsonb NOT NULL DEFAULT '{}'::jsonb,
    priority integer NOT NULL DEFAULT 0,
    valid_from timestamptz NOT NULL DEFAULT clock_timestamp(),
    valid_until timestamptz,
    recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX position_role_active_idx ON position_role_bindings(tenant_id, role_template_id, priority DESC) WHERE valid_until IS NULL;

CREATE TABLE delegations (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    from_principal_id uuid NOT NULL REFERENCES principals(id),
    to_principal_id uuid NOT NULL REFERENCES principals(id),
    responsibility text NOT NULL,
    scope jsonb NOT NULL DEFAULT '{}'::jsonb,
    starts_at timestamptz NOT NULL,
    ends_at timestamptz NOT NULL,
    status text NOT NULL DEFAULT 'ACTIVE' CHECK(status IN ('DRAFT','ACTIVE','REVOKED','EXPIRED')),
    reason text NOT NULL DEFAULT '',
    created_by uuid REFERENCES principals(id),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (from_principal_id <> to_principal_id),
    CHECK (starts_at < ends_at)
);
CREATE INDEX delegations_resolution_idx ON delegations(tenant_id, from_principal_id, responsibility, starts_at, ends_at) WHERE status='ACTIVE';

CREATE TABLE routing_policies (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    code text NOT NULL,
    name text NOT NULL,
    status text NOT NULL DEFAULT 'DRAFT' CHECK(status IN ('DRAFT','ACTIVE','RETIRED')),
    current_version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(tenant_id, code)
);

CREATE TABLE routing_policy_versions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    policy_id uuid NOT NULL REFERENCES routing_policies(id),
    version integer NOT NULL,
    definition jsonb NOT NULL,
    checksum text NOT NULL,
    effective_from timestamptz,
    effective_until timestamptz,
    created_by uuid REFERENCES principals(id),
    approved_by uuid REFERENCES principals(id),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    approved_at timestamptz,
    UNIQUE(policy_id, version),
    CHECK (effective_until IS NULL OR effective_from IS NULL OR effective_from < effective_until)
);
CREATE INDEX routing_policy_effective_idx ON routing_policy_versions(policy_id, effective_from, effective_until);

ALTER TABLE responsibility_assignments ALTER COLUMN principal_id DROP NOT NULL;
ALTER TABLE responsibility_assignments ADD COLUMN position_id uuid REFERENCES org_positions(id);
ALTER TABLE responsibility_assignments ADD COLUMN role_template_id uuid REFERENCES role_templates(id);
ALTER TABLE responsibility_assignments ADD COLUMN decision_type text;
ALTER TABLE responsibility_assignments ADD COLUMN resolution_strategy text NOT NULL DEFAULT 'DIRECT';
ALTER TABLE responsibility_assignments ADD CONSTRAINT responsibility_target_check CHECK (num_nonnulls(principal_id, position_id, role_template_id) = 1);

ALTER TABLE authority_grants ALTER COLUMN principal_id DROP NOT NULL;
ALTER TABLE authority_grants ADD COLUMN position_id uuid REFERENCES org_positions(id);
ALTER TABLE authority_grants ADD COLUMN role_template_id uuid REFERENCES role_templates(id);
ALTER TABLE authority_grants ADD CONSTRAINT authority_target_check CHECK (num_nonnulls(principal_id, position_id, role_template_id) = 1);

CREATE TABLE workflow_tasks (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    workflow_id uuid NOT NULL REFERENCES workflow_instances(id),
    step_key text NOT NULL,
    responsibility text NOT NULL,
    principal_id uuid REFERENCES principals(id),
    title text NOT NULL,
    status text NOT NULL CHECK(status IN ('READY','IN_PROGRESS','BLOCKED','ESCALATED','COMPLETED','CANCELLED')),
    due_at timestamptz,
    claimed_at timestamptz,
    completed_at timestamptz,
    context jsonb NOT NULL DEFAULT '{}'::jsonb,
    version bigint NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(workflow_id, step_key)
);
CREATE INDEX workflow_tasks_queue_idx ON workflow_tasks(tenant_id, principal_id, status, due_at, updated_at DESC);
CREATE INDEX workflow_tasks_unassigned_idx ON workflow_tasks(tenant_id, responsibility, due_at) WHERE principal_id IS NULL AND status IN ('READY','BLOCKED','ESCALATED');

CREATE TABLE workflow_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    workflow_id uuid NOT NULL REFERENCES workflow_instances(id),
    task_id uuid REFERENCES workflow_tasks(id),
    event_type text NOT NULL,
    actor_id uuid REFERENCES principals(id),
    safe_metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX workflow_events_history_idx ON workflow_events(tenant_id, workflow_id, occurred_at, id);

CREATE TABLE user_onboarding_state (
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    principal_id uuid NOT NULL REFERENCES principals(id),
    guide_code text NOT NULL,
    guide_version integer NOT NULL,
    current_step integer NOT NULL DEFAULT 0,
    completed_at timestamptz,
    dismissed_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    version bigint NOT NULL DEFAULT 1,
    PRIMARY KEY(tenant_id, principal_id, guide_code)
);

CREATE TABLE compliance_signals (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    signal_type text NOT NULL,
    subject_type text NOT NULL,
    subject_id text NOT NULL,
    dedupe_key text NOT NULL,
    source text NOT NULL,
    observed_at timestamptz NOT NULL,
    effective_at timestamptz NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(tenant_id, dedupe_key)
);
CREATE INDEX compliance_signals_subject_idx ON compliance_signals(tenant_id, subject_type, subject_id, effective_at DESC);
CREATE INDEX compliance_signals_type_time_idx ON compliance_signals(tenant_id, signal_type, observed_at DESC);

CREATE TABLE drift_assessments (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    subject_type text NOT NULL,
    subject_id text NOT NULL,
    dimension text NOT NULL,
    severity integer NOT NULL CHECK(severity BETWEEN 1 AND 5),
    state text NOT NULL CHECK(state IN ('ACTIVE','RESOLVED','SUPERSEDED')),
    summary text NOT NULL,
    required_action text NOT NULL,
    signal_id uuid REFERENCES compliance_signals(id),
    detected_at timestamptz NOT NULL,
    resolved_at timestamptz
);
CREATE UNIQUE INDEX drift_active_subject_idx ON drift_assessments(tenant_id, subject_type, subject_id, dimension) WHERE state='ACTIVE';
CREATE INDEX drift_priority_idx ON drift_assessments(tenant_id, state, severity DESC, detected_at DESC);

CREATE TABLE readiness_snapshots (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    program_id uuid,
    status text NOT NULL,
    dimensions jsonb NOT NULL,
    recommended_actions jsonb NOT NULL DEFAULT '[]'::jsonb,
    generated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX readiness_latest_idx ON readiness_snapshots(tenant_id, program_id, generated_at DESC);

CREATE TABLE automation_policies (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    code text NOT NULL,
    name text NOT NULL,
    action_class text NOT NULL,
    eligibility jsonb NOT NULL,
    blast_radius_limit jsonb NOT NULL,
    verification_contract jsonb NOT NULL,
    status text NOT NULL DEFAULT 'DRAFT' CHECK(status IN ('DRAFT','ACTIVE','SUSPENDED','EXPIRED','RETIRED')),
    effective_from timestamptz,
    effective_until timestamptz,
    version bigint NOT NULL DEFAULT 1,
    UNIQUE(tenant_id, code, version)
);

COMMIT;
