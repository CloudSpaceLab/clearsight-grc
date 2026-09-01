BEGIN;

CREATE INDEX form_response_policy_effective_form_idx
    ON form_response_policy_definitions(tenant_id,legal_entity_id,form_template_id,form_template_version,status,effective_from,effective_until)
    WHERE status='ACTIVE';

ALTER TABLE form_response_policy_executions
    ADD CONSTRAINT form_response_policy_executions_id_scope_key
    UNIQUE(id,tenant_id,legal_entity_id);

-- Claimed by the policy maintenance worker with FOR UPDATE SKIP LOCKED.
CREATE TABLE form_response_policy_maintenance_jobs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    job_type text NOT NULL CHECK (job_type IN ('RECONCILE','OUTCOME_CHECK','COMPENSATION')),
    response_revision_id uuid NOT NULL,
    policy_execution_id uuid,
    adverse_episode_id uuid,
    matter_id uuid,
    rollback_policy_id uuid,
    rollback_policy_version bigint CHECK (rollback_policy_version IS NULL OR rollback_policy_version > 0),
    due_at timestamptz NOT NULL,
    state text NOT NULL DEFAULT 'READY' CHECK (state IN ('READY','CLAIMED','COMPLETED','FAILED')),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    locked_by text,
    lease_until timestamptz,
    last_error text NOT NULL DEFAULT '' CHECK (char_length(last_error) <= 1000),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (response_revision_id,tenant_id,legal_entity_id)
        REFERENCES capture_response_revisions(id,tenant_id,legal_entity_id),
    FOREIGN KEY (policy_execution_id,tenant_id,legal_entity_id)
        REFERENCES form_response_policy_executions(id,tenant_id,legal_entity_id),
    FOREIGN KEY (adverse_episode_id,tenant_id,legal_entity_id)
        REFERENCES form_response_policy_adverse_episodes(id,tenant_id,legal_entity_id),
    FOREIGN KEY (matter_id,tenant_id,legal_entity_id)
        REFERENCES matters(id,tenant_id,legal_entity_id),
    FOREIGN KEY (rollback_policy_id,tenant_id,legal_entity_id)
        REFERENCES form_response_policy_definitions(id,tenant_id,legal_entity_id),
    CHECK (
        (job_type='RECONCILE' AND policy_execution_id IS NULL AND adverse_episode_id IS NULL AND matter_id IS NULL AND rollback_policy_id IS NULL AND rollback_policy_version IS NULL)
        OR
        (job_type='OUTCOME_CHECK' AND policy_execution_id IS NOT NULL AND adverse_episode_id IS NOT NULL AND matter_id IS NOT NULL AND rollback_policy_id IS NULL AND rollback_policy_version IS NULL)
        OR
        (job_type='COMPENSATION' AND policy_execution_id IS NOT NULL AND adverse_episode_id IS NULL AND matter_id IS NOT NULL AND rollback_policy_id IS NOT NULL AND rollback_policy_version IS NOT NULL)
    ),
    CHECK ((state='CLAIMED')=(locked_by IS NOT NULL AND lease_until IS NOT NULL)),
    CHECK (updated_at >= created_at)
);

CREATE UNIQUE INDEX form_response_policy_reconcile_job_uq
    ON form_response_policy_maintenance_jobs(tenant_id,legal_entity_id,response_revision_id)
    WHERE job_type='RECONCILE';
CREATE UNIQUE INDEX form_response_policy_outcome_job_uq
    ON form_response_policy_maintenance_jobs(tenant_id,legal_entity_id,policy_execution_id)
    WHERE job_type='OUTCOME_CHECK';
CREATE UNIQUE INDEX form_response_policy_compensation_job_uq
    ON form_response_policy_maintenance_jobs(tenant_id,legal_entity_id,rollback_policy_id,rollback_policy_version,policy_execution_id)
    WHERE job_type='COMPENSATION';
CREATE INDEX form_response_policy_maintenance_claim_idx
    ON form_response_policy_maintenance_jobs(due_at,id)
    WHERE state IN ('READY','CLAIMED');

CREATE TABLE form_response_policy_execution_failures (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    policy_id uuid NOT NULL,
    policy_version bigint NOT NULL CHECK (policy_version > 0),
    automation_policy_id uuid NOT NULL,
    automation_policy_version bigint NOT NULL CHECK (automation_policy_version > 0),
    response_revision_id uuid NOT NULL,
    event_id text NOT NULL CHECK (event_id=btrim(event_id) AND char_length(event_id) BETWEEN 1 AND 256),
    reason_code text NOT NULL CHECK (reason_code=btrim(reason_code) AND char_length(reason_code) BETWEEN 1 AND 128),
    created_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,policy_id,policy_version,response_revision_id,event_id),
    UNIQUE(id,tenant_id,legal_entity_id),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (policy_id,tenant_id,legal_entity_id)
        REFERENCES form_response_policy_definitions(id,tenant_id,legal_entity_id),
    FOREIGN KEY (automation_policy_id,tenant_id) REFERENCES automation_policies(id,tenant_id),
    FOREIGN KEY (response_revision_id,tenant_id,legal_entity_id)
        REFERENCES capture_response_revisions(id,tenant_id,legal_entity_id)
);
CREATE INDEX form_response_policy_execution_failure_response_idx
    ON form_response_policy_execution_failures(tenant_id,legal_entity_id,response_revision_id,created_at,id);

CREATE TABLE form_response_policy_compensations (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    rollback_policy_id uuid NOT NULL,
    rollback_policy_version bigint NOT NULL CHECK (rollback_policy_version > 0),
    original_execution_id uuid NOT NULL,
    matter_id uuid NOT NULL,
    review_matter_id uuid NOT NULL,
    actor_id uuid NOT NULL,
    reviewer_principal_id uuid NOT NULL,
    state text NOT NULL CHECK (state='REVIEW_REQUIRED'),
    reason_code text NOT NULL CHECK (reason_code=btrim(reason_code) AND char_length(reason_code) BETWEEN 1 AND 128),
    created_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,rollback_policy_id,rollback_policy_version,original_execution_id),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (rollback_policy_id,tenant_id,legal_entity_id)
        REFERENCES form_response_policy_definitions(id,tenant_id,legal_entity_id),
    FOREIGN KEY (original_execution_id,tenant_id,legal_entity_id)
        REFERENCES form_response_policy_executions(id,tenant_id,legal_entity_id),
    FOREIGN KEY (matter_id,tenant_id,legal_entity_id)
        REFERENCES matters(id,tenant_id,legal_entity_id),
    FOREIGN KEY (review_matter_id,tenant_id,legal_entity_id)
        REFERENCES matters(id,tenant_id,legal_entity_id),
    FOREIGN KEY (actor_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (reviewer_principal_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX form_response_policy_compensation_matter_idx
    ON form_response_policy_compensations(tenant_id,legal_entity_id,matter_id,created_at,id);

COMMIT;
