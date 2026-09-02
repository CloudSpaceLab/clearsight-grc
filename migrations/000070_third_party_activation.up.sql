BEGIN;

CREATE TABLE third_party_activation_policies (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    policy_number integer NOT NULL CHECK (policy_number > 0),
    allowed_conclusions text[] NOT NULL CHECK (cardinality(allowed_conclusions) > 0 AND allowed_conclusions <@ ARRAY['SATISFACTORY','SATISFACTORY_WITH_CONDITIONS']::text[]),
    maximum_assessment_age_days integer NOT NULL CHECK (maximum_assessment_age_days BETWEEN 1 AND 730),
    required_decision_types text[] NOT NULL DEFAULT '{}'::text[],
    address_verification_required boolean NOT NULL,
    blocking_matter_types text[] NOT NULL DEFAULT '{}'::text[],
    conditional_conclusion_needs_terms boolean NOT NULL,
    effective_from timestamptz NOT NULL,
    effective_until timestamptz,
    status text NOT NULL CHECK (status IN ('DRAFT','PENDING_APPROVAL','ACTIVE','RETIRED')),
    proposed_by uuid NOT NULL,
    approved_by uuid,
    proposal_rationale text NOT NULL CHECK (char_length(proposal_rationale) BETWEEN 20 AND 2000),
    approval_rationale text NOT NULL DEFAULT '' CHECK (char_length(approval_rationale) <= 2000),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    version bigint NOT NULL CHECK (version > 0),
    UNIQUE (id,tenant_id,legal_entity_id),
    UNIQUE (tenant_id,legal_entity_id,policy_number),
    CONSTRAINT third_party_activation_policy_entity_fk FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id),
    CONSTRAINT third_party_activation_policy_maker_fk FOREIGN KEY (proposed_by,tenant_id) REFERENCES principals(id,tenant_id),
    CONSTRAINT third_party_activation_policy_checker_fk FOREIGN KEY (approved_by,tenant_id) REFERENCES principals(id,tenant_id),
    CHECK (effective_until IS NULL OR effective_until > effective_from),
    CHECK ((status IN ('ACTIVE','RETIRED') AND approved_by IS NOT NULL AND approval_rationale <> '') OR (status IN ('DRAFT','PENDING_APPROVAL') AND approved_by IS NULL AND approval_rationale = '')),
    CHECK (approved_by IS NULL OR approved_by <> proposed_by)
);
CREATE INDEX third_party_activation_policy_current_idx
    ON third_party_activation_policies(tenant_id,legal_entity_id,status,effective_from DESC,id DESC);

CREATE TABLE third_party_activation_policy_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    policy_id uuid NOT NULL,
    policy_version bigint NOT NULL CHECK (policy_version > 0),
    event_type text NOT NULL CHECK (event_type IN ('PROPOSED','SUBMITTED','APPROVED','RETIRED')),
    actor_principal_id uuid NOT NULL,
    rationale text NOT NULL CHECK (char_length(rationale) BETWEEN 1 AND 2000),
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload)='object'),
    occurred_at timestamptz NOT NULL,
    UNIQUE (tenant_id,policy_id,policy_version),
    CONSTRAINT third_party_activation_policy_event_scope_fk FOREIGN KEY (policy_id,tenant_id,legal_entity_id) REFERENCES third_party_activation_policies(id,tenant_id,legal_entity_id),
    CONSTRAINT third_party_activation_policy_event_actor_fk FOREIGN KEY (actor_principal_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX third_party_activation_policy_events_history_idx
    ON third_party_activation_policy_events(tenant_id,legal_entity_id,policy_id,policy_version,id);

CREATE TABLE third_party_activation_receipts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    relationship_id uuid NOT NULL,
    relationship_version bigint NOT NULL CHECK (relationship_version > 1),
    policy_id uuid NOT NULL,
    policy_version bigint NOT NULL CHECK (policy_version > 0),
    assessment_id uuid NOT NULL,
    assessment_version bigint NOT NULL CHECK (assessment_version > 0),
    decision_ids uuid[] NOT NULL DEFAULT '{}'::uuid[],
    address_matter_id uuid,
    verification_result_id uuid,
    activated_by uuid NOT NULL,
    activated_at timestamptz NOT NULL,
    rationale text NOT NULL CHECK (char_length(rationale) BETWEEN 1 AND 2000),
    UNIQUE (tenant_id,relationship_id),
    CONSTRAINT third_party_activation_receipt_relationship_fk FOREIGN KEY (relationship_id,tenant_id,legal_entity_id) REFERENCES third_party_relationships(id,tenant_id,legal_entity_id),
    CONSTRAINT third_party_activation_receipt_policy_fk FOREIGN KEY (policy_id,tenant_id,legal_entity_id) REFERENCES third_party_activation_policies(id,tenant_id,legal_entity_id),
    CONSTRAINT third_party_activation_receipt_assessment_fk FOREIGN KEY (assessment_id,tenant_id,legal_entity_id) REFERENCES third_party_assessments(id,tenant_id,legal_entity_id),
    CONSTRAINT third_party_activation_receipt_address_matter_fk FOREIGN KEY (address_matter_id,tenant_id) REFERENCES matters(id,tenant_id),
    CONSTRAINT third_party_activation_receipt_verification_fk FOREIGN KEY (verification_result_id) REFERENCES verification_results(id),
    CONSTRAINT third_party_activation_receipt_actor_fk FOREIGN KEY (activated_by,tenant_id) REFERENCES principals(id,tenant_id)
);

ALTER TABLE third_party_events DROP CONSTRAINT third_party_events_event_type_check;
ALTER TABLE third_party_events ADD CONSTRAINT third_party_events_event_type_check CHECK (event_type IN (
    'VendorRelationshipCreated','VendorRelationshipUpdated','VendorRelationshipActivated',
    'AssessmentStarted','AssessmentSetupCompleted','AssessmentRequestPrepared','AssessmentRequestIssued','AssessmentRequestReissuePrepared','AssessmentRequestReissued',
    'AssessmentSubmitted','AssessmentReviewStarted','AssessmentCompleted','AssessmentCancelled'
));

COMMIT;
