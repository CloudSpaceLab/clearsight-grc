BEGIN;

ALTER TABLE third_party_relationships
    ADD CONSTRAINT third_party_relationships_scoped_id_key UNIQUE (id,tenant_id,legal_entity_id);
ALTER TABLE capture_artifacts
    ADD CONSTRAINT capture_artifacts_id_tenant_request_key UNIQUE (id,tenant_id,request_id);

ALTER TABLE matters DROP CONSTRAINT matters_matter_type_check;
ALTER TABLE matters ADD CONSTRAINT matters_matter_type_check CHECK (matter_type IN (
    'REGULATORY_CHANGE','SUPERVISORY_FINDING','AUTHORITY_REQUEST','RISK_SITUATION','CONTROL_GAP','AUDIT_FINDING','EXCEPTION','INCIDENT',
    'OPERATIONAL_LOSS','DATA_BREACH','VENDOR_REVIEW','VENDOR_DEFICIENCY','CUSTOMER_CONCERN','OVERDUE_OBLIGATION','FAILED_VERIFICATION',
    'EVIDENCE_CONTRADICTION','KRI_BREACH'
));

CREATE TABLE third_party_assessments (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    relationship_id uuid NOT NULL,
    review_kind text NOT NULL CHECK (review_kind IN ('ONBOARDING')),
    stable_episode_key text NOT NULL CHECK (stable_episode_key=btrim(stable_episode_key) AND char_length(stable_episode_key)=64),
    status text NOT NULL CHECK (status IN ('SETUP_PENDING','READY_TO_SEND','COLLECTING','SUBMITTED','UNDER_REVIEW','COMPLETED','CANCELLED')),
    form_template_id uuid NOT NULL,
    form_template_version bigint NOT NULL CHECK (form_template_version > 0),
    current_request_id uuid,
    submission_id uuid,
    review_matter_id uuid,
    review_due_at timestamptz NOT NULL,
    started_by_principal_id uuid NOT NULL,
    started_at timestamptz NOT NULL,
    submitted_at timestamptz,
    review_started_at timestamptz,
    completed_at timestamptz,
    reviewer_principal_id uuid,
    conclusion text CHECK (conclusion IS NULL OR conclusion IN ('SATISFACTORY','SATISFACTORY_WITH_CONDITIONS','UNSATISFACTORY','INCONCLUSIVE')),
    conclusion_uncertainty text NOT NULL DEFAULT '' CHECK (conclusion_uncertainty=btrim(conclusion_uncertainty) AND char_length(conclusion_uncertainty) <= 2000),
    conclusion_rationale text NOT NULL DEFAULT '' CHECK (conclusion_rationale=btrim(conclusion_rationale) AND char_length(conclusion_rationale) <= 4000),
    next_review_recommended_at timestamptz,
    cancellation_reason text NOT NULL DEFAULT '' CHECK (cancellation_reason=btrim(cancellation_reason) AND char_length(cancellation_reason) <= 2000),
    version bigint NOT NULL CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    UNIQUE (id,tenant_id,legal_entity_id),
    UNIQUE (tenant_id,legal_entity_id,stable_episode_key),
    FOREIGN KEY (relationship_id,tenant_id,legal_entity_id) REFERENCES third_party_relationships(id,tenant_id,legal_entity_id),
    FOREIGN KEY (tenant_id,form_template_id,form_template_version) REFERENCES monitoring_form_templates(tenant_id,id,version),
    FOREIGN KEY (current_request_id,tenant_id) REFERENCES capture_requests(id,tenant_id),
    FOREIGN KEY (submission_id,tenant_id,current_request_id) REFERENCES capture_submissions(id,tenant_id,request_id),
    FOREIGN KEY (review_matter_id,tenant_id) REFERENCES matters(id,tenant_id),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY (started_by_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (reviewer_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK (review_due_at > started_at),
    CHECK ((status='COMPLETED' AND completed_at IS NOT NULL AND reviewer_principal_id IS NOT NULL AND conclusion IS NOT NULL AND conclusion_rationale<>'')
        OR (status<>'COMPLETED' AND completed_at IS NULL AND conclusion IS NULL AND conclusion_rationale='')),
    CHECK ((status='CANCELLED' AND cancellation_reason<>'') OR (status<>'CANCELLED' AND cancellation_reason='')),
    CHECK (submitted_at IS NULL OR submitted_at >= started_at),
    CHECK (review_started_at IS NULL OR (submitted_at IS NOT NULL AND review_started_at >= submitted_at)),
    CHECK (completed_at IS NULL OR (review_started_at IS NOT NULL AND completed_at >= review_started_at))
);
CREATE INDEX third_party_assessments_scoped_list_idx
    ON third_party_assessments(tenant_id,legal_entity_id,status,updated_at DESC,id DESC);
CREATE INDEX third_party_assessments_relationship_idx
    ON third_party_assessments(tenant_id,legal_entity_id,relationship_id,updated_at DESC,id DESC);

CREATE TABLE third_party_assessment_matter_links (
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    assessment_id uuid NOT NULL,
    matter_id uuid NOT NULL,
    link_kind text NOT NULL CHECK (link_kind IN ('REVIEW','DEFICIENCY')),
    created_at timestamptz NOT NULL,
    PRIMARY KEY (tenant_id,assessment_id,matter_id),
    FOREIGN KEY (assessment_id,tenant_id,legal_entity_id) REFERENCES third_party_assessments(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (matter_id,tenant_id) REFERENCES matters(id,tenant_id)
);
CREATE UNIQUE INDEX third_party_assessment_review_matter_idx
    ON third_party_assessment_matter_links(tenant_id,assessment_id)
    WHERE link_kind='REVIEW';

CREATE TABLE third_party_assessment_request_links (
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    assessment_id uuid NOT NULL,
    request_id uuid NOT NULL,
    purpose text NOT NULL CHECK (purpose IN ('INITIAL','CLARIFICATION')),
    sequence integer NOT NULL CHECK (sequence > 0),
    origin_type text NOT NULL CHECK (origin_type='THIRD_PARTY_ASSESSMENT'),
    origin_id uuid NOT NULL,
    origin_sequence bigint NOT NULL CHECK (origin_sequence > 0),
    invitation_id uuid,
    is_current boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (tenant_id,assessment_id,sequence),
    UNIQUE (tenant_id,assessment_id,request_id),
    UNIQUE (tenant_id,origin_type,origin_id,origin_sequence),
    FOREIGN KEY (assessment_id,tenant_id,legal_entity_id) REFERENCES third_party_assessments(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (request_id,tenant_id) REFERENCES capture_requests(id,tenant_id),
    FOREIGN KEY (invitation_id,tenant_id,request_id) REFERENCES capture_invitations(id,tenant_id,request_id),
    CHECK (origin_id=assessment_id)
);
CREATE UNIQUE INDEX third_party_assessment_current_request_idx
    ON third_party_assessment_request_links(tenant_id,assessment_id)
    WHERE is_current;

CREATE TABLE third_party_assessment_reactions (
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    assessment_id uuid NOT NULL,
    reaction_kind text NOT NULL CHECK (reaction_kind IN ('SETUP_COMPLETED','SUBMITTED')),
    causation_id text NOT NULL CHECK (causation_id=btrim(causation_id) AND char_length(causation_id) BETWEEN 1 AND 128),
    job_id uuid,
    event_id uuid,
    matter_id uuid,
    request_id uuid,
    submission_id uuid,
    resulting_version bigint NOT NULL CHECK (resulting_version > 1),
    result_snapshot jsonb NOT NULL CHECK (jsonb_typeof(result_snapshot)='object' AND octet_length(result_snapshot::text) <= 32768),
    applied_at timestamptz NOT NULL,
    PRIMARY KEY (tenant_id,reaction_kind,causation_id),
    FOREIGN KEY (assessment_id,tenant_id,legal_entity_id) REFERENCES third_party_assessments(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (matter_id,tenant_id) REFERENCES matters(id,tenant_id),
    FOREIGN KEY (request_id,tenant_id) REFERENCES capture_requests(id,tenant_id),
    FOREIGN KEY (submission_id,tenant_id,request_id) REFERENCES capture_submissions(id,tenant_id,request_id),
    CHECK ((reaction_kind='SETUP_COMPLETED' AND job_id IS NOT NULL AND event_id IS NULL AND matter_id IS NOT NULL AND request_id IS NULL AND submission_id IS NULL)
        OR (reaction_kind='SUBMITTED' AND job_id IS NULL AND event_id IS NOT NULL AND matter_id IS NULL AND request_id IS NOT NULL AND submission_id IS NOT NULL))
);

CREATE TABLE third_party_documents (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    relationship_id uuid NOT NULL,
    assessment_id uuid NOT NULL,
    request_id uuid NOT NULL,
    artifact_id uuid NOT NULL,
    document_type text NOT NULL CHECK (document_type=btrim(document_type) AND char_length(document_type) BETWEEN 1 AND 128),
    reference text NOT NULL DEFAULT '' CHECK (reference=btrim(reference) AND char_length(reference) <= 256),
    issued_by text NOT NULL DEFAULT '' CHECK (issued_by=btrim(issued_by) AND char_length(issued_by) <= 512),
    issued_on date,
    expires_on date,
    evidence_class text NOT NULL CHECK (evidence_class IN ('VENDOR_SUPPLIED','BANK_VALIDATED','OFFICIAL_SOURCE')),
    status text NOT NULL CHECK (status IN ('SUBMITTED','VALIDATED','REJECTED','EXPIRED')),
    validated_by_principal_id uuid,
    validated_at timestamptz,
    validation_note text NOT NULL DEFAULT '' CHECK (validation_note=btrim(validation_note) AND char_length(validation_note) <= 2000),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    version bigint NOT NULL CHECK (version > 0),
    UNIQUE (id,tenant_id),
    UNIQUE (tenant_id,assessment_id,artifact_id),
    FOREIGN KEY (assessment_id,tenant_id,legal_entity_id) REFERENCES third_party_assessments(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    FOREIGN KEY (relationship_id,tenant_id,legal_entity_id) REFERENCES third_party_relationships(id,tenant_id,legal_entity_id),
    FOREIGN KEY (artifact_id,tenant_id,request_id) REFERENCES capture_artifacts(id,tenant_id,request_id),
    FOREIGN KEY (validated_by_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK (expires_on IS NULL OR issued_on IS NULL OR expires_on >= issued_on),
    CHECK ((status IN ('VALIDATED','REJECTED') AND validated_by_principal_id IS NOT NULL AND validated_at IS NOT NULL)
        OR (status IN ('SUBMITTED','EXPIRED') AND validated_by_principal_id IS NULL AND validated_at IS NULL))
);
CREATE INDEX third_party_documents_assessment_idx
    ON third_party_documents(tenant_id,legal_entity_id,assessment_id,status,updated_at DESC,id DESC);

CREATE TABLE third_party_assessment_jobs (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    assessment_id uuid NOT NULL,
    job_type text NOT NULL CHECK (job_type IN ('SETUP_REVIEW')),
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
    FOREIGN KEY (assessment_id,tenant_id,legal_entity_id) REFERENCES third_party_assessments(id,tenant_id,legal_entity_id) ON DELETE CASCADE,
    CHECK ((state='LEASED' AND lease_token IS NOT NULL AND lease_expires_at IS NOT NULL)
        OR (state<>'LEASED' AND lease_token IS NULL AND lease_expires_at IS NULL))
);
CREATE INDEX third_party_assessment_jobs_claim_idx
    ON third_party_assessment_jobs(state,available_at,id)
    WHERE state IN ('READY','LEASED');

ALTER TABLE third_party_events
    DROP CONSTRAINT third_party_events_aggregate_type_check,
    DROP CONSTRAINT third_party_events_event_type_check,
    DROP CONSTRAINT third_party_events_aggregate_id_tenant_id_fkey,
    DROP CONSTRAINT third_party_events_tenant_id_aggregate_id_aggregate_version_key;
ALTER TABLE third_party_events
    ALTER COLUMN actor_principal_id DROP NOT NULL,
    ADD CONSTRAINT third_party_events_aggregate_type_check CHECK (aggregate_type IN ('VENDOR_RELATIONSHIP','THIRD_PARTY_ASSESSMENT')),
    ADD CONSTRAINT third_party_events_event_type_check CHECK (event_type IN (
        'VendorRelationshipCreated','VendorRelationshipUpdated','AssessmentStarted','AssessmentSetupCompleted','AssessmentRequestPrepared','AssessmentRequestIssued','AssessmentRequestReissuePrepared','AssessmentRequestReissued',
        'AssessmentSubmitted','AssessmentReviewStarted','AssessmentCompleted','AssessmentCancelled'
    )),
    ADD CONSTRAINT third_party_events_typed_version_key UNIQUE (tenant_id,aggregate_type,aggregate_id,aggregate_version);
DROP INDEX third_party_events_history_idx;
CREATE INDEX third_party_events_history_idx
    ON third_party_events(tenant_id,aggregate_type,aggregate_id,aggregate_version,id);

CREATE FUNCTION third_party_event_aggregate_guard() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.aggregate_type='VENDOR_RELATIONSHIP' AND NOT EXISTS (
        SELECT 1 FROM third_party_relationships WHERE id=NEW.aggregate_id AND tenant_id=NEW.tenant_id
    ) THEN
        RAISE EXCEPTION 'third-party relationship event aggregate is outside tenant scope';
    END IF;
    IF NEW.aggregate_type='THIRD_PARTY_ASSESSMENT' AND NOT EXISTS (
        SELECT 1 FROM third_party_assessments WHERE id=NEW.aggregate_id AND tenant_id=NEW.tenant_id
    ) THEN
        RAISE EXCEPTION 'third-party assessment event aggregate is outside tenant scope';
    END IF;
    RETURN NEW;
END $$;
CREATE CONSTRAINT TRIGGER third_party_event_aggregate_scope
    AFTER INSERT OR UPDATE OF tenant_id,aggregate_type,aggregate_id ON third_party_events
    DEFERRABLE INITIALLY IMMEDIATE FOR EACH ROW EXECUTE FUNCTION third_party_event_aggregate_guard();

COMMIT;
