BEGIN;
DROP TABLE capture_field_collection_resolutions;
DROP FUNCTION prevent_collection_resolution_mutation();
ALTER TABLE third_party_assessments DROP CONSTRAINT third_party_assessment_review_receipt_check;
ALTER TABLE third_party_assessments DROP COLUMN collection_completed_at;
ALTER TABLE third_party_assessments ADD CHECK(review_started_at IS NULL OR (submitted_at IS NOT NULL AND review_started_at>=submitted_at));
ALTER TABLE third_party_events DROP CONSTRAINT third_party_events_event_type_check;
ALTER TABLE third_party_events ADD CONSTRAINT third_party_events_event_type_check CHECK (event_type IN (
    'VendorIdentityCreated','VendorIdentityUpdated','VendorBrandDiscovered','VendorBrandApproved','VendorBrandRemoved',
    'VendorRelationshipCreated','VendorRelationshipUpdated','VendorRelationshipActivated','AssessmentStarted','AssessmentSetupCompleted',
    'AssessmentSetupRetryQueued','AssessmentRequestPrepared','AssessmentRequestIssued','AssessmentRequestReissuePrepared',
    'AssessmentRequestReissued','AssessmentSubmitted','AssessmentReviewStarted','AssessmentDeficiencyLinked',
    'AssessmentDocumentValidated','AssessmentDocumentRejected','AssessmentDocumentExpired','AssessmentResponseApplied','AssessmentCompleted','AssessmentCancelled'
));

COMMIT;
