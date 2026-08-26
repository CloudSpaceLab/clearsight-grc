BEGIN;

CREATE TABLE third_party_work_requests (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    relationship_id uuid NOT NULL,
    program_link_id uuid,
    matter_link_id uuid,
    target_type text NOT NULL CHECK (target_type IN ('PROGRAM','MATTER')),
    target_id uuid NOT NULL,
    purpose text NOT NULL CHECK (purpose=btrim(purpose) AND char_length(purpose) BETWEEN 1 AND 500),
    instructions text NOT NULL CHECK (instructions=btrim(instructions) AND char_length(instructions) BETWEEN 1 AND 2000),
    owner_principal_id uuid NOT NULL,
    reviewer_principal_id uuid,
    form_template_id uuid NOT NULL,
    form_template_version bigint NOT NULL CHECK (form_template_version > 0),
    presentation text NOT NULL CHECK (presentation IN ('AUTOMATIC','CLASSIC','WIZARD')),
    current_request_id uuid,
    current_invitation_id uuid,
    current_capture_sequence integer NOT NULL DEFAULT 0 CHECK (current_capture_sequence >= 0),
    submission_id uuid,
    state text NOT NULL CHECK (state IN ('PREPARING','AWAITING_VENDOR','RESPONSE_RECEIVED','UNDER_REVIEW','CHANGES_REQUESTED','ACCEPTED','CANCELLED')),
    delivery_state text NOT NULL CHECK (delivery_state IN ('NOT_SENT','LINK_CREATED_EMAIL_NOT_SENT','DELIVERED','RETRY_REQUIRED')),
    recovery text NOT NULL DEFAULT '' CHECK (recovery=btrim(recovery) AND char_length(recovery) <= 1000),
    review_rationale text NOT NULL DEFAULT '' CHECK (review_rationale=btrim(review_rationale) AND char_length(review_rationale) <= 2000),
    cancellation_reason text NOT NULL DEFAULT '' CHECK (cancellation_reason=btrim(cancellation_reason) AND char_length(cancellation_reason) <= 1000),
    due_at timestamptz NOT NULL,
    response_received_at timestamptz,
    review_started_at timestamptz,
    accepted_at timestamptz,
    cancelled_at timestamptz,
    version bigint NOT NULL CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    UNIQUE (id,tenant_id,legal_entity_id),
    FOREIGN KEY (relationship_id,tenant_id,legal_entity_id) REFERENCES third_party_relationships(id,tenant_id,legal_entity_id),
    FOREIGN KEY (program_link_id,tenant_id,legal_entity_id) REFERENCES third_party_relationship_program_links(id,tenant_id,legal_entity_id),
    FOREIGN KEY (matter_link_id,tenant_id,legal_entity_id) REFERENCES third_party_relationship_matter_links(id,tenant_id,legal_entity_id),
    FOREIGN KEY (tenant_id,form_template_id,form_template_version) REFERENCES monitoring_form_templates(tenant_id,id,version),
    FOREIGN KEY (current_request_id,tenant_id) REFERENCES capture_requests(id,tenant_id),
    FOREIGN KEY (current_invitation_id,tenant_id,current_request_id) REFERENCES capture_invitations(id,tenant_id,request_id),
    FOREIGN KEY (submission_id,tenant_id,current_request_id) REFERENCES capture_submissions(id,tenant_id,request_id),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (reviewer_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK ((target_type='PROGRAM' AND program_link_id IS NOT NULL AND matter_link_id IS NULL) OR
           (target_type='MATTER' AND matter_link_id IS NOT NULL AND program_link_id IS NULL)),
    CHECK (due_at > created_at),
    CHECK ((state='ACCEPTED' AND accepted_at IS NOT NULL AND reviewer_principal_id IS NOT NULL AND review_rationale<>'') OR
           (state<>'ACCEPTED' AND accepted_at IS NULL)),
    CHECK ((state='CANCELLED' AND cancelled_at IS NOT NULL AND cancellation_reason<>'') OR
           (state<>'CANCELLED' AND cancelled_at IS NULL))
);
CREATE UNIQUE INDEX third_party_work_requests_active_link_idx
    ON third_party_work_requests(tenant_id,legal_entity_id,COALESCE(program_link_id,matter_link_id))
    WHERE state NOT IN ('ACCEPTED','CANCELLED');
CREATE INDEX third_party_work_requests_relationship_idx
    ON third_party_work_requests(tenant_id,legal_entity_id,relationship_id,updated_at DESC,id DESC);
CREATE INDEX third_party_work_requests_target_idx
    ON third_party_work_requests(tenant_id,legal_entity_id,target_type,target_id,updated_at DESC,id DESC);

CREATE TABLE third_party_work_capture_links (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    work_request_id uuid NOT NULL,
    request_id uuid NOT NULL,
    sequence integer NOT NULL CHECK (sequence > 0),
    purpose text NOT NULL CHECK (purpose IN ('INITIAL','CLARIFICATION')),
    origin_type text NOT NULL CHECK (origin_type='THIRD_PARTY_WORK'),
    origin_id uuid NOT NULL,
    origin_version bigint NOT NULL CHECK (origin_version > 0),
    invitation_id uuid,
    submission_id uuid,
    created_at timestamptz NOT NULL,
    UNIQUE (tenant_id,work_request_id,sequence),
    UNIQUE (tenant_id,work_request_id,request_id),
    UNIQUE (tenant_id,origin_type,origin_id,origin_version),
    FOREIGN KEY (work_request_id,tenant_id,legal_entity_id) REFERENCES third_party_work_requests(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (request_id,tenant_id) REFERENCES capture_requests(id,tenant_id),
    FOREIGN KEY (invitation_id,tenant_id,request_id) REFERENCES capture_invitations(id,tenant_id,request_id),
    FOREIGN KEY (submission_id,tenant_id,request_id) REFERENCES capture_submissions(id,tenant_id,request_id),
    CHECK (origin_id=work_request_id AND origin_version=sequence)
);
CREATE INDEX third_party_work_capture_links_request_idx
    ON third_party_work_capture_links(tenant_id,request_id,work_request_id);

CREATE TABLE third_party_work_reactions (
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    work_request_id uuid NOT NULL,
    reaction_kind text NOT NULL CHECK (reaction_kind='SUBMITTED'),
    causation_id text NOT NULL CHECK (causation_id=btrim(causation_id) AND char_length(causation_id) BETWEEN 1 AND 128),
    request_id uuid NOT NULL,
    submission_id uuid NOT NULL,
    resulting_version bigint NOT NULL CHECK (resulting_version > 1),
    result_snapshot jsonb NOT NULL CHECK (jsonb_typeof(result_snapshot)='object' AND octet_length(result_snapshot::text) <= 32768),
    applied_at timestamptz NOT NULL,
    PRIMARY KEY (tenant_id,reaction_kind,causation_id),
    FOREIGN KEY (work_request_id,tenant_id,legal_entity_id) REFERENCES third_party_work_requests(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (request_id,tenant_id) REFERENCES capture_requests(id,tenant_id),
    FOREIGN KEY (submission_id,tenant_id,request_id) REFERENCES capture_submissions(id,tenant_id,request_id)
);

CREATE TABLE third_party_work_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    work_request_id uuid NOT NULL,
    work_version bigint NOT NULL CHECK (work_version > 0),
    actor_principal_id uuid,
    event_type text NOT NULL CHECK (event_type IN ('VendorWorkPrepared','VendorWorkCaptureAttached','VendorWorkSent','VendorWorkResponseReceived','VendorWorkReviewStarted','VendorWorkChangesRequested','VendorWorkAccepted','VendorWorkCancelled','VendorWorkDeliveryRetryRequired','VendorWorkPreparationRetryRequired')),
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload)='object' AND octet_length(payload::text) <= 32768),
    occurred_at timestamptz NOT NULL,
    UNIQUE (tenant_id,work_request_id,work_version),
    FOREIGN KEY (work_request_id,tenant_id,legal_entity_id) REFERENCES third_party_work_requests(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (actor_principal_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX third_party_work_events_history_idx
    ON third_party_work_events(tenant_id,legal_entity_id,work_request_id,work_version,id);

CREATE TABLE third_party_work_jobs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    work_request_id uuid NOT NULL,
    job_type text NOT NULL CHECK (job_type='DELIVERY_RETRY'),
    dedupe_key text NOT NULL CHECK (dedupe_key=btrim(dedupe_key) AND char_length(dedupe_key) BETWEEN 1 AND 256),
    state text NOT NULL CHECK (state IN ('READY','LEASED','COMPLETED','FAILED')),
    attempts integer NOT NULL DEFAULT 0 CHECK (attempts BETWEEN 0 AND 20),
    available_at timestamptz NOT NULL,
    lease_token uuid,
    lease_expires_at timestamptz,
    last_failure_code text NOT NULL DEFAULT '' CHECK (last_failure_code=btrim(last_failure_code) AND char_length(last_failure_code) <= 128),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    UNIQUE (tenant_id,dedupe_key),
    FOREIGN KEY (work_request_id,tenant_id,legal_entity_id) REFERENCES third_party_work_requests(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    CHECK ((state='LEASED' AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL) OR
           (state<>'LEASED' AND lease_token IS NULL AND lease_expires_at IS NULL))
);
CREATE INDEX third_party_work_jobs_claim_idx
    ON third_party_work_jobs(state,available_at,id) WHERE state IN ('READY','LEASED');

COMMIT;
