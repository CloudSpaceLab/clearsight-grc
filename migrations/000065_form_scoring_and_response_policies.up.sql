BEGIN;

ALTER TABLE monitoring_form_templates
    ADD COLUMN score_profile jsonb,
    ADD CONSTRAINT monitoring_form_templates_score_profile_ck CHECK (
        score_profile IS NULL OR (
            jsonb_typeof(score_profile)='object'
            AND octet_length(score_profile::text) <= 262144
        )
    );

ALTER TABLE capture_requests
    ADD COLUMN scoring_mode text NOT NULL DEFAULT 'NONE',
    ADD COLUMN score_profile jsonb,
    ADD CONSTRAINT capture_requests_scoring_mode_ck CHECK (scoring_mode IN ('NONE','RISK','COMPLIANCE')),
    ADD CONSTRAINT capture_requests_score_profile_ck CHECK (
        score_profile IS NULL OR (
            jsonb_typeof(score_profile)='object'
            AND octet_length(score_profile::text) <= 262144
        )
    );

ALTER TABLE capture_response_revisions
    ADD COLUMN score_mode text,
    ADD COLUMN score_direction text,
    ADD COLUMN raw_score numeric(7,4),
    ADD COLUMN adverse_score numeric(7,4),
    ADD COLUMN concern_band text,
    ADD COLUMN score_state text NOT NULL DEFAULT 'NOT_CONFIGURED',
    ADD COLUMN score_result jsonb,
    ADD COLUMN score_profile_checksum text NOT NULL DEFAULT '',
    ADD COLUMN score_calculated_at timestamptz,
    ADD CONSTRAINT capture_response_revisions_score_mode_ck CHECK (score_mode IS NULL OR score_mode IN ('NONE','RISK','COMPLIANCE')),
    ADD CONSTRAINT capture_response_revisions_score_direction_ck CHECK (score_direction IS NULL OR score_direction IN ('HIGH_IS_POOR','LOW_IS_POOR')),
    ADD CONSTRAINT capture_response_revisions_raw_score_ck CHECK (raw_score IS NULL OR raw_score BETWEEN 0 AND 100),
    ADD CONSTRAINT capture_response_revisions_adverse_score_ck CHECK (adverse_score IS NULL OR adverse_score BETWEEN 0 AND 100),
    ADD CONSTRAINT capture_response_revisions_concern_band_ck CHECK (concern_band IS NULL OR concern_band IN ('LOW','MODERATE','HIGH','CRITICAL')),
    ADD CONSTRAINT capture_response_revisions_score_state_ck CHECK (score_state IN ('NOT_CONFIGURED','FINAL','PROVISIONAL','FAILED')),
    ADD CONSTRAINT capture_response_revisions_score_result_ck CHECK (
        score_result IS NULL OR (
            jsonb_typeof(score_result)='object'
            AND octet_length(score_result::text) <= 1048576
        )
    ),
    ADD CONSTRAINT capture_response_revisions_score_checksum_ck CHECK (
        char_length(score_profile_checksum) <= 128
    );

CREATE INDEX capture_response_revisions_current_adverse_idx
    ON capture_response_revisions(tenant_id,legal_entity_id,adverse_score DESC,created_at DESC,id DESC)
    WHERE is_current AND score_state IN ('FINAL','PROVISIONAL') AND adverse_score IS NOT NULL;

CREATE INDEX capture_response_revisions_current_raw_idx
    ON capture_response_revisions(tenant_id,legal_entity_id,raw_score DESC,created_at DESC,id DESC)
    WHERE is_current AND score_state IN ('FINAL','PROVISIONAL') AND raw_score IS NOT NULL;

ALTER TABLE capture_response_revisions
    ADD CONSTRAINT capture_response_revisions_id_tenant_entity_key UNIQUE(id,tenant_id,legal_entity_id);

ALTER TABLE matters
    ADD CONSTRAINT matters_id_tenant_entity_key UNIQUE(id,tenant_id,legal_entity_id);

CREATE TABLE form_response_policy_definitions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    code text NOT NULL CHECK (code=btrim(code) AND code ~ '^[a-z0-9][a-z0-9-]{0,63}$'),
    name text NOT NULL CHECK (name=btrim(name) AND char_length(name) BETWEEN 1 AND 160),
    purpose text NOT NULL CHECK (purpose=btrim(purpose) AND char_length(purpose) BETWEEN 1 AND 1000),
    action_class text NOT NULL DEFAULT 'FORM_RESPONSE_CREATE_MATTER' CHECK (action_class='FORM_RESPONSE_CREATE_MATTER'),
    automation_policy_id uuid NOT NULL,
    automation_policy_version bigint NOT NULL CHECK (automation_policy_version > 0),
    form_template_id uuid NOT NULL,
    form_template_version bigint NOT NULL CHECK (form_template_version > 0),
    eligibility jsonb NOT NULL CHECK (jsonb_typeof(eligibility)='object' AND octet_length(eligibility::text) <= 65536),
    matter_action jsonb NOT NULL CHECK (jsonb_typeof(matter_action)='object' AND octet_length(matter_action::text) <= 32768),
    blast_radius jsonb NOT NULL CHECK (jsonb_typeof(blast_radius)='object' AND octet_length(blast_radius::text) <= 4096),
    outcome_contract jsonb NOT NULL CHECK (jsonb_typeof(outcome_contract)='object' AND octet_length(outcome_contract::text) <= 32768),
    rollout_mode text NOT NULL CHECK (rollout_mode IN ('SHADOW','ENFORCE')),
    status text NOT NULL CHECK (status IN ('DRAFT','PENDING_APPROVAL','APPROVED','ACTIVE','SUSPENDED','RETIRED')),
    maker_id uuid NOT NULL,
    checker_id uuid,
    checksum text NOT NULL CHECK (char_length(checksum)=64),
    approved_simulation_id uuid,
    supersedes_policy_id uuid,
    rollback_of_policy_id uuid,
    effective_from timestamptz,
    effective_until timestamptz,
    submitted_at timestamptz,
    approved_at timestamptz,
    activated_at timestamptz,
    suspended_at timestamptz,
    retired_at timestamptz,
    version bigint NOT NULL CHECK (version > 0),
    record_version bigint NOT NULL DEFAULT 1 CHECK (record_version > 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(id,tenant_id,legal_entity_id),
    UNIQUE(tenant_id,legal_entity_id,code,version),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (automation_policy_id,tenant_id) REFERENCES automation_policies(id,tenant_id),
    FOREIGN KEY (tenant_id,legal_entity_id,form_template_id,form_template_version)
        REFERENCES monitoring_form_templates(tenant_id,legal_entity_id,id,version),
    FOREIGN KEY (maker_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (checker_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (supersedes_policy_id,tenant_id,legal_entity_id)
        REFERENCES form_response_policy_definitions(id,tenant_id,legal_entity_id),
    FOREIGN KEY (rollback_of_policy_id,tenant_id,legal_entity_id)
        REFERENCES form_response_policy_definitions(id,tenant_id,legal_entity_id),
    CHECK (effective_until IS NULL OR effective_from IS NULL OR effective_until > effective_from),
    CHECK (updated_at >= created_at)
);
CREATE UNIQUE INDEX form_response_policy_active_code_uq
    ON form_response_policy_definitions(tenant_id,legal_entity_id,code)
    WHERE status='ACTIVE';
CREATE INDEX form_response_policy_list_idx
    ON form_response_policy_definitions(tenant_id,legal_entity_id,code,version DESC,id DESC);

CREATE TABLE form_response_policy_simulations (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    policy_id uuid NOT NULL,
    policy_version bigint NOT NULL CHECK (policy_version > 0),
    policy_checksum text NOT NULL CHECK (char_length(policy_checksum)=64),
    actor_id uuid NOT NULL,
    population_count integer NOT NULL CHECK (population_count >= 0),
    eligible_count integer NOT NULL CHECK (eligible_count BETWEEN 0 AND population_count),
    would_create_count integer NOT NULL CHECK (would_create_count BETWEEN 0 AND eligible_count),
    would_reuse_count integer NOT NULL CHECK (would_reuse_count BETWEEN 0 AND eligible_count),
    blast_suppressed_count integer NOT NULL CHECK (blast_suppressed_count BETWEEN 0 AND eligible_count),
    restricted_excluded_count integer NOT NULL CHECK (restricted_excluded_count >= 0),
    population_high_water text NOT NULL DEFAULT '' CHECK (char_length(population_high_water) <= 256),
    population_checksum text NOT NULL CHECK (char_length(population_checksum)=64),
    impact_checksum text NOT NULL CHECK (char_length(impact_checksum)=64),
    observed_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL CHECK (expires_at > observed_at),
    UNIQUE(id,tenant_id,legal_entity_id,policy_id),
    FOREIGN KEY (policy_id,tenant_id,legal_entity_id)
        REFERENCES form_response_policy_definitions(id,tenant_id,legal_entity_id),
    FOREIGN KEY (actor_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX form_response_policy_simulation_policy_idx
    ON form_response_policy_simulations(tenant_id,legal_entity_id,policy_id,observed_at DESC,id DESC);

ALTER TABLE form_response_policy_definitions
    ADD CONSTRAINT form_response_policy_approved_simulation_fk
    FOREIGN KEY (approved_simulation_id,tenant_id,legal_entity_id,id)
    REFERENCES form_response_policy_simulations(id,tenant_id,legal_entity_id,policy_id);

CREATE TABLE form_response_policy_executions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    policy_id uuid NOT NULL,
    policy_version bigint NOT NULL CHECK (policy_version > 0),
    automation_policy_id uuid NOT NULL,
    automation_policy_version bigint NOT NULL CHECK (automation_policy_version > 0),
    response_revision_id uuid NOT NULL,
    state text NOT NULL CHECK (state IN ('NOT_MATCHED','SHADOW','APPLIED','REUSED','BLAST_SUPPRESSED','FAILED')),
    matter_id uuid,
    reason_code text NOT NULL DEFAULT '' CHECK (char_length(reason_code) <= 128),
    created_matter boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE(tenant_id,legal_entity_id,policy_id,policy_version,response_revision_id),
    FOREIGN KEY (policy_id,tenant_id,legal_entity_id)
        REFERENCES form_response_policy_definitions(id,tenant_id,legal_entity_id),
    FOREIGN KEY (automation_policy_id,tenant_id) REFERENCES automation_policies(id,tenant_id),
    FOREIGN KEY (response_revision_id,tenant_id,legal_entity_id)
        REFERENCES capture_response_revisions(id,tenant_id,legal_entity_id),
    FOREIGN KEY (matter_id,tenant_id,legal_entity_id) REFERENCES matters(id,tenant_id,legal_entity_id)
);
CREATE INDEX form_response_policy_execution_response_idx
    ON form_response_policy_executions(tenant_id,legal_entity_id,response_revision_id,created_at DESC,id DESC);

CREATE TABLE form_response_policy_adverse_episodes (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    policy_code text NOT NULL CHECK (policy_code=btrim(policy_code) AND char_length(policy_code) BETWEEN 1 AND 64),
    policy_id uuid NOT NULL,
    policy_version bigint NOT NULL CHECK (policy_version > 0),
    subject_type text NOT NULL CHECK (subject_type=btrim(subject_type) AND char_length(subject_type) BETWEEN 1 AND 80),
    subject_id uuid NOT NULL,
    state text NOT NULL CHECK (state IN ('OPEN','CLOSED')),
    matter_id uuid,
    last_response_revision_id uuid NOT NULL,
    opened_at timestamptz NOT NULL,
    closed_at timestamptz,
    updated_at timestamptz NOT NULL,
    record_version bigint NOT NULL DEFAULT 1 CHECK (record_version > 0),
    UNIQUE(id,tenant_id,legal_entity_id),
    FOREIGN KEY (policy_id,tenant_id,legal_entity_id)
        REFERENCES form_response_policy_definitions(id,tenant_id,legal_entity_id),
    FOREIGN KEY (matter_id,tenant_id,legal_entity_id) REFERENCES matters(id,tenant_id,legal_entity_id),
    FOREIGN KEY (last_response_revision_id,tenant_id,legal_entity_id)
        REFERENCES capture_response_revisions(id,tenant_id,legal_entity_id),
    CHECK ((state='OPEN' AND closed_at IS NULL) OR (state='CLOSED' AND closed_at IS NOT NULL)),
    CHECK (updated_at >= opened_at)
);
CREATE UNIQUE INDEX form_response_policy_open_episode_uq
    ON form_response_policy_adverse_episodes(tenant_id,legal_entity_id,policy_code,subject_type,subject_id)
    WHERE state='OPEN';

CREATE TABLE form_response_policy_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    policy_id uuid NOT NULL,
    policy_version bigint NOT NULL CHECK (policy_version > 0),
    record_version bigint NOT NULL CHECK (record_version > 0),
    event_type text NOT NULL CHECK (event_type=btrim(event_type) AND char_length(event_type) BETWEEN 1 AND 128),
    actor_id uuid NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(payload)='object' AND octet_length(payload::text) <= 16384),
    occurred_at timestamptz NOT NULL,
    UNIQUE(tenant_id,legal_entity_id,policy_id,record_version,event_type),
    FOREIGN KEY (policy_id,tenant_id,legal_entity_id)
        REFERENCES form_response_policy_definitions(id,tenant_id,legal_entity_id),
    FOREIGN KEY (actor_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX form_response_policy_events_history_idx
    ON form_response_policy_events(tenant_id,legal_entity_id,policy_id,occurred_at,id);

CREATE UNIQUE INDEX form_response_policy_outbox_uq
    ON outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,(COALESCE(payload->>'version','1')))
    WHERE aggregate_type='FORM_RESPONSE_POLICY';

COMMIT;
