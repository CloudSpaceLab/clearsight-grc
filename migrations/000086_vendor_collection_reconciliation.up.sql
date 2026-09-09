BEGIN;
ALTER TABLE third_party_assessments ADD COLUMN collection_completed_at timestamptz;
DO $$ DECLARE constraint_name text; BEGIN
 FOR constraint_name IN SELECT conname FROM pg_constraint WHERE conrelid='third_party_assessments'::regclass AND contype='c' AND pg_get_constraintdef(oid) LIKE '%review_started_at IS NULL%' AND pg_get_constraintdef(oid) LIKE '%submitted_at IS NOT NULL%'
 LOOP EXECUTE format('ALTER TABLE third_party_assessments DROP CONSTRAINT %I',constraint_name); END LOOP;
END $$;
ALTER TABLE third_party_assessments ADD CONSTRAINT third_party_assessment_review_receipt_check CHECK (review_started_at IS NULL OR (submitted_at IS NOT NULL AND review_started_at>=submitted_at) OR (collection_completed_at IS NOT NULL AND review_started_at>=collection_completed_at));
CREATE TABLE capture_field_collection_resolutions (
 id uuid PRIMARY KEY,
 tenant_id uuid NOT NULL REFERENCES tenants(id),
 legal_entity_id uuid NOT NULL,
 assessment_id uuid NOT NULL,
 assessment_version bigint NOT NULL CHECK(assessment_version>0),
 request_id uuid NOT NULL,
 request_version bigint NOT NULL CHECK(request_version>0),
 field_id text NOT NULL CHECK(char_length(field_id) BETWEEN 1 AND 200),
 version bigint NOT NULL CHECK(version>0),
 source_submission_id uuid NOT NULL,
 source_request_id uuid NOT NULL,
 source_artifact_request_id uuid NOT NULL,
 source_field_id text NOT NULL,
 source_artifact_id uuid NOT NULL,
 actor_principal_id uuid NOT NULL,
 receipt jsonb NOT NULL CHECK(jsonb_typeof(receipt)='object'),
 created_at timestamptz NOT NULL,
 UNIQUE(tenant_id,request_id,field_id,version),
 FOREIGN KEY(assessment_id,tenant_id,legal_entity_id) REFERENCES third_party_assessments(id,tenant_id,legal_entity_id),
 FOREIGN KEY(request_id,tenant_id) REFERENCES capture_requests(id,tenant_id),
 FOREIGN KEY(source_submission_id,tenant_id,source_request_id) REFERENCES capture_submissions(id,tenant_id,request_id),
 FOREIGN KEY(source_artifact_id,tenant_id,source_artifact_request_id) REFERENCES capture_artifacts(id,tenant_id,request_id),
 FOREIGN KEY(actor_principal_id,tenant_id) REFERENCES principals(id,tenant_id)
);
CREATE INDEX capture_collection_request_history_idx ON capture_field_collection_resolutions(tenant_id,legal_entity_id,request_id,field_id,version DESC);
ALTER TABLE third_party_events DROP CONSTRAINT third_party_events_event_type_check;
ALTER TABLE third_party_events ADD CONSTRAINT third_party_events_event_type_check CHECK (event_type IN (
    'VendorIdentityCreated','VendorIdentityUpdated','VendorBrandDiscovered','VendorBrandApproved','VendorBrandRemoved',
    'VendorRelationshipCreated','VendorRelationshipUpdated','VendorRelationshipActivated','AssessmentStarted','AssessmentSetupCompleted',
    'AssessmentSetupRetryQueued','AssessmentRequestPrepared','AssessmentRequestIssued','AssessmentRequestReissuePrepared',
    'AssessmentRequestReissued','AssessmentSubmitted','AssessmentReviewStarted','AssessmentDeficiencyLinked',
    'AssessmentDocumentValidated','AssessmentDocumentRejected','AssessmentDocumentExpired','AssessmentResponseApplied','AssessmentCompleted','AssessmentCancelled','AssessmentDocumentReconciled','AssessmentReusedDocumentReviewed'
));

CREATE FUNCTION prevent_collection_resolution_mutation() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'Collection receipts are immutable'; END $$;
CREATE TRIGGER capture_collection_resolutions_immutable BEFORE UPDATE OR DELETE ON capture_field_collection_resolutions FOR EACH ROW EXECUTE FUNCTION prevent_collection_resolution_mutation();
COMMIT;
