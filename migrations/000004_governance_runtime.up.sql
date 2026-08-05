BEGIN;

ALTER TABLE routing_policies DROP CONSTRAINT IF EXISTS routing_policies_status_check;
ALTER TABLE routing_policies
    ADD COLUMN IF NOT EXISTS maker_id uuid REFERENCES principals(id),
    ADD COLUMN IF NOT EXISTS checker_id uuid REFERENCES principals(id),
    ADD COLUMN IF NOT EXISTS submitted_at timestamptz,
    ADD COLUMN IF NOT EXISTS approved_at timestamptz,
    ADD COLUMN IF NOT EXISTS retired_at timestamptz,
    ADD COLUMN IF NOT EXISTS version bigint NOT NULL DEFAULT 1;
ALTER TABLE routing_policies
    ADD CONSTRAINT routing_policies_status_check
    CHECK (status IN ('DRAFT','PENDING_APPROVAL','ACTIVE','RETIRED')),
    ADD CONSTRAINT routing_policy_maker_checker_check
    CHECK (checker_id IS NULL OR maker_id IS NULL OR checker_id <> maker_id);
UPDATE routing_policies rp
SET maker_id = source.created_by
FROM (
    SELECT DISTINCT ON (policy_id) policy_id, created_by
    FROM routing_policy_versions
    WHERE created_by IS NOT NULL
    ORDER BY policy_id, version DESC
) source
WHERE rp.id = source.policy_id AND rp.maker_id IS NULL;

ALTER TABLE delegations DROP CONSTRAINT IF EXISTS delegations_status_check;
ALTER TABLE delegations
    ADD COLUMN IF NOT EXISTS approved_by uuid REFERENCES principals(id),
    ADD COLUMN IF NOT EXISTS submitted_at timestamptz,
    ADD COLUMN IF NOT EXISTS approved_at timestamptz,
    ADD COLUMN IF NOT EXISTS revoked_by uuid REFERENCES principals(id),
    ADD COLUMN IF NOT EXISTS revoked_at timestamptz,
    ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    ADD COLUMN IF NOT EXISTS version bigint NOT NULL DEFAULT 1;
ALTER TABLE delegations
    ADD CONSTRAINT delegations_status_check
    CHECK (status IN ('DRAFT','PENDING_APPROVAL','APPROVED','ACTIVE','REVOKED','EXPIRED')),
    ADD CONSTRAINT delegation_approver_independence_check
    CHECK (approved_by IS NULL OR (approved_by IS DISTINCT FROM created_by AND approved_by <> from_principal_id AND approved_by <> to_principal_id));

CREATE TABLE governance_decisions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    object_type text NOT NULL CHECK (object_type IN ('ROUTING_POLICY','DELEGATION','SEGREGATION_RULE')),
    object_id uuid NOT NULL,
    from_state text NOT NULL,
    to_state text NOT NULL,
    actor_type text NOT NULL DEFAULT 'PRINCIPAL' CHECK (actor_type IN ('PRINCIPAL','SYSTEM')),
    actor_id uuid REFERENCES principals(id),
    rationale text NOT NULL DEFAULT '',
    decided_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK ((actor_type='PRINCIPAL' AND actor_id IS NOT NULL) OR (actor_type='SYSTEM' AND actor_id IS NULL))
);
CREATE INDEX governance_decisions_object_idx
    ON governance_decisions(tenant_id, object_type, object_id, decided_at DESC);

CREATE TABLE segregation_rules (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    code text NOT NULL,
    responsibility text NOT NULL,
    prohibited_role_code text NOT NULL,
    status text NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE','RETIRED')),
    valid_from timestamptz NOT NULL DEFAULT clock_timestamp(),
    valid_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (valid_until IS NULL OR valid_from < valid_until)
);
CREATE UNIQUE INDEX segregation_rules_active_code_idx
    ON segregation_rules(tenant_id, code) WHERE status='ACTIVE' AND valid_until IS NULL;
CREATE INDEX segregation_rules_resolution_idx
    ON segregation_rules(tenant_id, responsibility, prohibited_role_code)
    WHERE status='ACTIVE';

CREATE TABLE workflow_timers (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    workflow_id uuid NOT NULL REFERENCES workflow_instances(id),
    task_id uuid REFERENCES workflow_tasks(id),
    timer_type text NOT NULL,
    due_at timestamptz NOT NULL,
    state text NOT NULL DEFAULT 'READY' CHECK (state IN ('READY','CLAIMED','FIRED','CANCELLED')),
    dedupe_key text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    attempts integer NOT NULL DEFAULT 0,
    locked_by text,
    lease_until timestamptz,
    last_error text,
    fired_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(tenant_id, dedupe_key),
    CHECK ((state='CLAIMED' AND locked_by IS NOT NULL AND lease_until IS NOT NULL) OR state<>'CLAIMED')
);
CREATE INDEX workflow_timers_claim_idx
    ON workflow_timers(state, due_at, id) WHERE state='READY';
CREATE INDEX workflow_timers_workflow_idx
    ON workflow_timers(tenant_id, workflow_id, due_at, id);

ALTER TABLE outbox_events
    ADD COLUMN IF NOT EXISTS next_attempt_at timestamptz,
    ADD COLUMN IF NOT EXISTS locked_by text,
    ADD COLUMN IF NOT EXISTS lease_until timestamptz,
    ADD COLUMN IF NOT EXISTS last_error text;
CREATE INDEX IF NOT EXISTS outbox_retry_claim_idx
    ON outbox_events(COALESCE(next_attempt_at, available_at), id)
    WHERE published_at IS NULL;

CREATE TABLE inbox_receipts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id),
    consumer text NOT NULL,
    event_id text NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(tenant_id, consumer, event_id)
);
CREATE INDEX inbox_receipts_time_idx ON inbox_receipts(tenant_id, processed_at DESC);

COMMIT;
