BEGIN;

CREATE TABLE capture_form_distributions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    form_template_id uuid NOT NULL,
    form_template_version bigint NOT NULL CHECK (form_template_version > 0),
    subject_type text NOT NULL CHECK (subject_type=btrim(subject_type) AND char_length(subject_type) BETWEEN 1 AND 128),
    subject_id uuid NOT NULL,
    title text NOT NULL CHECK (title=btrim(title) AND char_length(title) BETWEEN 1 AND 512),
    purpose text NOT NULL CHECK (purpose=btrim(purpose) AND char_length(purpose) BETWEEN 1 AND 2000),
    access_policy text NOT NULL CHECK (access_policy IN ('DIRECT_MAGIC_LINK','SHARED_LINK_EMAIL_OTP','DIRECT_LINK_EMAIL_OTP')),
    status text NOT NULL CHECK (status IN ('DRAFT','READY','OPEN','LOCKED','COMPLETED','EXPIRED','REVOKED','SUPERSEDED')),
    deadline timestamptz NOT NULL,
    route_expires_at timestamptz NOT NULL,
    reminder_policy jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(reminder_policy)='object' AND octet_length(reminder_policy::text) <= 16384),
    created_by uuid NOT NULL,
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (id,tenant_id),
    UNIQUE (id,tenant_id,legal_entity_id),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (tenant_id,form_template_id,form_template_version) REFERENCES monitoring_form_templates(tenant_id,id,version),
    FOREIGN KEY (created_by,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK (route_expires_at <= deadline),
    CHECK (updated_at >= created_at)
);
CREATE INDEX capture_form_distributions_deadline_idx
    ON capture_form_distributions(tenant_id,legal_entity_id,deadline,id);
CREATE INDEX capture_form_distributions_updated_idx
    ON capture_form_distributions(tenant_id,legal_entity_id,updated_at DESC,id DESC);
CREATE INDEX capture_form_distributions_template_idx
    ON capture_form_distributions(tenant_id,legal_entity_id,form_template_id,form_template_version,created_at DESC,id DESC);

ALTER TABLE capture_requests ADD COLUMN distribution_id uuid;
ALTER TABLE capture_requests
    ADD CONSTRAINT capture_requests_distribution_scope_fk
    FOREIGN KEY (distribution_id,tenant_id,legal_entity_id)
    REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id);
CREATE INDEX capture_requests_distribution_idx
    ON capture_requests(tenant_id,legal_entity_id,distribution_id,deadline,id)
    WHERE distribution_id IS NOT NULL;

ALTER TABLE capture_submissions
    ADD COLUMN distribution_id uuid,
    ADD CONSTRAINT capture_submissions_id_tenant_key UNIQUE(id,tenant_id);
ALTER TABLE capture_submissions
    ADD CONSTRAINT capture_submissions_distribution_tenant_fk
    FOREIGN KEY (distribution_id,tenant_id)
    REFERENCES capture_form_distributions(id,tenant_id);
CREATE INDEX capture_submissions_distribution_idx
    ON capture_submissions(tenant_id,distribution_id,submitted_at DESC,id DESC)
    WHERE distribution_id IS NOT NULL;

CREATE TABLE capture_distribution_recipients (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    distribution_id uuid NOT NULL,
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    role text NOT NULL CHECK (role IN ('TO','CC')),
    recipient_type text NOT NULL CHECK (recipient_type IN ('INTERNAL_PRINCIPAL','EXTERNAL_AUDIENCE')),
    principal_id uuid,
    request_id uuid,
    address_hash bytea,
    address_ciphertext bytea,
    address_key_id text,
    audience_hint text NOT NULL DEFAULT '' CHECK (char_length(audience_hint) <= 320),
    contact_label text NOT NULL DEFAULT '' CHECK (char_length(contact_label) <= 320),
    state text NOT NULL CHECK (state IN ('PENDING','DELIVERED','VERIFIED','REVOKED','COMPLETED')),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (id,tenant_id),
    UNIQUE (id,tenant_id,legal_entity_id),
    FOREIGN KEY (distribution_id,tenant_id,legal_entity_id) REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (request_id,tenant_id) REFERENCES capture_requests(id,tenant_id),
    CHECK ((recipient_type='INTERNAL_PRINCIPAL' AND principal_id IS NOT NULL AND address_hash IS NULL AND address_ciphertext IS NULL AND address_key_id IS NULL)
        OR (recipient_type='EXTERNAL_AUDIENCE' AND principal_id IS NULL AND address_hash IS NOT NULL AND address_ciphertext IS NOT NULL AND address_key_id IS NOT NULL AND btrim(address_key_id)<>'')),
    CHECK ((role='TO' AND request_id IS NOT NULL) OR (role='CC' AND request_id IS NULL)),
    CHECK (updated_at >= created_at)
);
CREATE UNIQUE INDEX capture_distribution_recipients_request_uq
    ON capture_distribution_recipients(tenant_id,request_id)
    WHERE request_id IS NOT NULL;
CREATE INDEX capture_distribution_recipients_distribution_idx
    ON capture_distribution_recipients(tenant_id,legal_entity_id,distribution_id,role,state,id);

CREATE TABLE capture_access_routes (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    distribution_id uuid NOT NULL,
    recipient_id uuid,
    access_policy text NOT NULL CHECK (access_policy IN ('DIRECT_MAGIC_LINK','SHARED_LINK_EMAIL_OTP','DIRECT_LINK_EMAIL_OTP')),
    selector_hash bytea NOT NULL UNIQUE,
    audience_hint text NOT NULL DEFAULT '' CHECK (char_length(audience_hint) <= 320),
    expires_at timestamptz NOT NULL,
    max_redemptions integer NOT NULL DEFAULT 1 CHECK (max_redemptions BETWEEN 1 AND 20),
    redemptions integer NOT NULL DEFAULT 0 CHECK (redemptions >= 0 AND redemptions <= max_redemptions),
    revoked_at timestamptz,
    created_by uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (id,tenant_id),
    FOREIGN KEY (distribution_id,tenant_id,legal_entity_id) REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (recipient_id,tenant_id,legal_entity_id) REFERENCES capture_distribution_recipients(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (created_by,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK ((access_policy='SHARED_LINK_EMAIL_OTP' AND recipient_id IS NULL) OR (access_policy<>'SHARED_LINK_EMAIL_OTP' AND recipient_id IS NOT NULL)),
    CHECK (revoked_at IS NULL OR revoked_at >= created_at)
);
CREATE INDEX capture_access_routes_distribution_idx
    ON capture_access_routes(tenant_id,legal_entity_id,distribution_id,expires_at,id)
    WHERE revoked_at IS NULL;

CREATE TABLE capture_otp_challenges (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    distribution_id uuid NOT NULL,
    route_id uuid NOT NULL,
    recipient_id uuid,
    code_hash bytea NOT NULL,
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 5),
    max_attempts integer NOT NULL DEFAULT 5 CHECK (max_attempts BETWEEN 1 AND 5 AND attempts <= max_attempts),
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (id,tenant_id),
    FOREIGN KEY (distribution_id,tenant_id,legal_entity_id) REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (route_id,tenant_id) REFERENCES capture_access_routes(id,tenant_id) ON DELETE CASCADE,
    FOREIGN KEY (recipient_id,tenant_id,legal_entity_id) REFERENCES capture_distribution_recipients(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    CHECK (expires_at > created_at),
    CHECK (consumed_at IS NULL OR consumed_at >= created_at)
);
CREATE INDEX capture_otp_challenges_active_idx
    ON capture_otp_challenges(tenant_id,distribution_id,route_id,expires_at DESC,id DESC)
    WHERE consumed_at IS NULL AND attempts < max_attempts;

CREATE TABLE capture_response_workspaces (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    distribution_id uuid NOT NULL,
    status text NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN','LOCKED','COMPLETED','REVOKED')),
    version bigint NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (distribution_id),
    UNIQUE (id,tenant_id),
    UNIQUE (id,tenant_id,legal_entity_id),
    FOREIGN KEY (distribution_id,tenant_id,legal_entity_id) REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    CHECK (updated_at >= created_at)
);
CREATE INDEX capture_response_workspaces_updated_idx
    ON capture_response_workspaces(tenant_id,legal_entity_id,updated_at DESC,id DESC);

CREATE TABLE capture_response_workspace_edits (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    distribution_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    recipient_id uuid NOT NULL,
    request_id uuid NOT NULL,
    base_version bigint NOT NULL CHECK (base_version > 0),
    result_version bigint NOT NULL CHECK (result_version = base_version + 1),
    patch jsonb NOT NULL CHECK (jsonb_typeof(patch)='object' AND octet_length(patch::text) <= 1048576),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    FOREIGN KEY (distribution_id,tenant_id,legal_entity_id) REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (workspace_id,tenant_id,legal_entity_id) REFERENCES capture_response_workspaces(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (recipient_id,tenant_id,legal_entity_id) REFERENCES capture_distribution_recipients(id,tenant_id,legal_entity_id),
    FOREIGN KEY (request_id,tenant_id) REFERENCES capture_requests(id,tenant_id)
);
CREATE UNIQUE INDEX capture_response_workspace_edits_version_uq
    ON capture_response_workspace_edits(tenant_id,workspace_id,result_version);
CREATE INDEX capture_response_workspace_edits_history_idx
    ON capture_response_workspace_edits(tenant_id,workspace_id,created_at,id);

CREATE TABLE capture_response_revisions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    distribution_id uuid NOT NULL,
    workspace_id uuid NOT NULL,
    submission_id uuid NOT NULL,
    revision bigint NOT NULL CHECK (revision > 0),
    supersedes_revision_id uuid,
    achieved_assurance text NOT NULL CHECK (achieved_assurance IN ('LINK_POSSESSION','EMAIL_VERIFIED')),
    signoff_summary jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(signoff_summary)='object' AND octet_length(signoff_summary::text) <= 65536),
    compliance_score numeric(7,4) CHECK (compliance_score IS NULL OR compliance_score BETWEEN 0 AND 100),
    scored_weight_coverage numeric(7,4) NOT NULL DEFAULT 0 CHECK (scored_weight_coverage BETWEEN 0 AND 100),
    state text NOT NULL CHECK (state IN ('PROVISIONAL','FINAL')),
    critical_field_results jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(critical_field_results)='array' AND octet_length(critical_field_results::text) <= 65536),
    scoring_policy_version text NOT NULL CHECK (scoring_policy_version=btrim(scoring_policy_version) AND char_length(scoring_policy_version) BETWEEN 1 AND 128),
    is_current boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (id,tenant_id),
    UNIQUE (tenant_id,workspace_id,revision),
    FOREIGN KEY (distribution_id,tenant_id,legal_entity_id) REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (workspace_id,tenant_id,legal_entity_id) REFERENCES capture_response_workspaces(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (submission_id,tenant_id) REFERENCES capture_submissions(id,tenant_id),
    FOREIGN KEY (supersedes_revision_id,tenant_id) REFERENCES capture_response_revisions(id,tenant_id),
    CHECK ((revision=1 AND supersedes_revision_id IS NULL) OR (revision>1 AND supersedes_revision_id IS NOT NULL))
);
CREATE UNIQUE INDEX capture_response_revisions_current_uq
    ON capture_response_revisions(tenant_id,workspace_id)
    WHERE is_current;
CREATE INDEX capture_response_revisions_history_idx
    ON capture_response_revisions(tenant_id,workspace_id,revision DESC,id DESC);

CREATE TABLE capture_distribution_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    distribution_id uuid NOT NULL,
    distribution_version bigint NOT NULL CHECK (distribution_version > 0),
    event_type text NOT NULL CHECK (event_type=btrim(event_type) AND char_length(event_type) BETWEEN 1 AND 128),
    payload jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(payload)='object'),
    actor_id uuid,
    occurred_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    UNIQUE (tenant_id,distribution_id,distribution_version,event_type),
    FOREIGN KEY (distribution_id,tenant_id,legal_entity_id) REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (actor_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX capture_distribution_events_history_idx
    ON capture_distribution_events(tenant_id,distribution_id,occurred_at,id);

CREATE UNIQUE INDEX capture_distribution_outbox_uq
    ON outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,(COALESCE(payload->>'version','1')))
    WHERE aggregate_type='FORM_DISTRIBUTION';

COMMIT;
