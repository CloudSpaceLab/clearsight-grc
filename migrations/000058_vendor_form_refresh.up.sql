ALTER TABLE third_party_assessments
    ADD COLUMN scope_kind text NOT NULL DEFAULT 'FULL',
    ADD COLUMN scope_version bigint NOT NULL DEFAULT 1,
    ADD COLUMN selected_field_ids jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD CONSTRAINT third_party_assessments_scope_kind_check CHECK (scope_kind IN ('FULL','FOCUSED')),
    ADD CONSTRAINT third_party_assessments_scope_version_check CHECK (scope_version > 0),
    ADD CONSTRAINT third_party_assessments_selected_fields_check CHECK (
        jsonb_typeof(selected_field_ids)='array'
        AND jsonb_array_length(selected_field_ids) <= 200
        AND ((scope_kind='FULL' AND jsonb_array_length(selected_field_ids)=0)
          OR (scope_kind='FOCUSED' AND jsonb_array_length(selected_field_ids)>0))
    );

ALTER TABLE third_party_documents
    DROP CONSTRAINT third_party_documents_status_check,
    ADD COLUMN supersedes_document_id uuid,
    ADD CONSTRAINT third_party_documents_status_check CHECK (status IN ('SUBMITTED','VALIDATED','REJECTED','EXPIRED','SUPERSEDED')),
    ADD CONSTRAINT third_party_documents_supersedes_fk
        FOREIGN KEY (supersedes_document_id,tenant_id) REFERENCES third_party_documents(id,tenant_id);

DO $$
DECLARE validation_constraint text;
BEGIN
    SELECT c.conname INTO validation_constraint
    FROM pg_constraint c
    WHERE c.conrelid='third_party_documents'::regclass
      AND c.contype='c'
      AND pg_get_constraintdef(c.oid) LIKE '%validated_by_principal_id%'
      AND pg_get_constraintdef(c.oid) LIKE '%validated_at%'
    LIMIT 1;
    IF validation_constraint IS NOT NULL THEN
        EXECUTE format('ALTER TABLE third_party_documents DROP CONSTRAINT %I', validation_constraint);
    END IF;
END $$;
ALTER TABLE third_party_documents
    ADD CONSTRAINT third_party_documents_validation_state_check CHECK (
        (status IN ('VALIDATED','REJECTED') AND validated_by_principal_id IS NOT NULL AND validated_at IS NOT NULL)
        OR (status='SUBMITTED' AND validated_by_principal_id IS NULL AND validated_at IS NULL)
        OR status IN ('EXPIRED','SUPERSEDED')
    );

CREATE INDEX third_party_documents_current_type_idx
    ON third_party_documents(tenant_id,legal_entity_id,relationship_id,document_type)
    WHERE status IN ('VALIDATED','EXPIRED');
CREATE INDEX third_party_documents_due_idx
    ON third_party_documents(tenant_id,legal_entity_id,expires_on,id)
    WHERE status='VALIDATED' AND expires_on IS NOT NULL;

CREATE TABLE third_party_response_application_receipts (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    assessment_id uuid NOT NULL,
    distribution_id uuid,
    response_revision_id uuid NOT NULL,
    vendor_id uuid NOT NULL,
    actor_principal_id uuid NOT NULL,
    accepted_field_ids jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(accepted_field_ids)='array'),
    rejected_field_ids jsonb NOT NULL DEFAULT '[]'::jsonb CHECK (jsonb_typeof(rejected_field_ids)='array'),
    decisions jsonb NOT NULL CHECK (jsonb_typeof(decisions)='array' AND jsonb_array_length(decisions)>0),
    prior_vendor_version bigint NOT NULL CHECK (prior_vendor_version > 0),
    result_vendor_version bigint NOT NULL CHECK (result_vendor_version >= prior_vendor_version),
    result_assessment_version bigint NOT NULL CHECK (result_assessment_version > 0),
    applied_at timestamptz NOT NULL,
    UNIQUE (tenant_id,assessment_id,response_revision_id),
    UNIQUE (id,tenant_id,legal_entity_id),
    FOREIGN KEY (assessment_id,tenant_id,legal_entity_id) REFERENCES third_party_assessments(id,tenant_id,legal_entity_id),
    FOREIGN KEY (distribution_id,tenant_id,legal_entity_id) REFERENCES capture_form_distributions(id,tenant_id,legal_entity_id),
    FOREIGN KEY (vendor_id,tenant_id) REFERENCES third_parties(id,tenant_id),
    FOREIGN KEY (actor_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id)
);

CREATE INDEX third_party_response_application_receipts_assessment_idx
    ON third_party_response_application_receipts(tenant_id,legal_entity_id,assessment_id,applied_at DESC,id DESC);

CREATE TABLE third_party_refresh_attentions (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    relationship_id uuid NOT NULL,
    owner_principal_id uuid NOT NULL,
    target_keys jsonb NOT NULL CHECK (jsonb_typeof(target_keys)='array' AND jsonb_array_length(target_keys) BETWEEN 1 AND 206),
    reason text NOT NULL CHECK (reason=btrim(reason) AND char_length(reason) BETWEEN 1 AND 512),
    observed_versions jsonb NOT NULL CHECK (jsonb_typeof(observed_versions)='object'),
    dedupe_key text NOT NULL CHECK (dedupe_key ~ '^[0-9a-f]{64}$'),
    state text NOT NULL CHECK (state IN ('OPEN','RESOLVED')),
    version bigint NOT NULL CHECK (version > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL CHECK (updated_at >= created_at),
    UNIQUE (id,tenant_id,legal_entity_id),
    UNIQUE (tenant_id,dedupe_key),
    FOREIGN KEY (relationship_id,tenant_id,legal_entity_id) REFERENCES third_party_relationships(id,tenant_id,legal_entity_id),
    FOREIGN KEY (owner_principal_id,tenant_id) REFERENCES principals(id,tenant_id),
    FOREIGN KEY (legal_entity_id,tenant_id) REFERENCES legal_entities(id,tenant_id)
);
CREATE INDEX third_party_refresh_attentions_open_idx
    ON third_party_refresh_attentions(tenant_id,legal_entity_id,owner_principal_id,updated_at DESC,id DESC)
    WHERE state='OPEN';

CREATE TABLE third_party_refresh_attention_events (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    attention_id uuid NOT NULL,
    relationship_id uuid NOT NULL,
    attention_version bigint NOT NULL CHECK (attention_version > 0),
    event_type text NOT NULL CHECK (event_type IN ('VendorRefreshAttentionCreated','VendorRefreshAttentionResolved')),
    payload jsonb NOT NULL CHECK (jsonb_typeof(payload)='object'),
    occurred_at timestamptz NOT NULL,
    UNIQUE (tenant_id,attention_id,attention_version),
    FOREIGN KEY (attention_id,tenant_id,legal_entity_id) REFERENCES third_party_refresh_attentions(id,tenant_id,legal_entity_id),
    FOREIGN KEY (relationship_id,tenant_id,legal_entity_id) REFERENCES third_party_relationships(id,tenant_id,legal_entity_id)
);
CREATE INDEX third_party_refresh_attention_events_history_idx
    ON third_party_refresh_attention_events(tenant_id,legal_entity_id,attention_id,attention_version,id);

ALTER TABLE third_party_events DROP CONSTRAINT third_party_events_event_type_check;
ALTER TABLE third_party_events ADD CONSTRAINT third_party_events_event_type_check CHECK (event_type IN (
    'VendorIdentityCreated','VendorIdentityUpdated','VendorBrandDiscovered','VendorBrandApproved','VendorBrandRemoved',
    'VendorRelationshipCreated','VendorRelationshipUpdated','AssessmentStarted','AssessmentSetupCompleted',
    'AssessmentSetupRetryQueued','AssessmentRequestPrepared','AssessmentRequestIssued','AssessmentRequestReissuePrepared',
    'AssessmentRequestReissued','AssessmentSubmitted','AssessmentReviewStarted','AssessmentDeficiencyLinked',
    'AssessmentDocumentValidated','AssessmentDocumentRejected','AssessmentDocumentExpired','AssessmentCompleted','AssessmentCancelled'
));
