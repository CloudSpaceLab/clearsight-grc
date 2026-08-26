BEGIN;

ALTER TABLE third_party_events DROP CONSTRAINT third_party_events_event_type_check;
ALTER TABLE third_party_events ADD CONSTRAINT third_party_events_event_type_check CHECK (event_type IN (
    'VendorRelationshipCreated','VendorRelationshipUpdated','AssessmentStarted','AssessmentSetupCompleted','AssessmentSetupRetryQueued',
    'AssessmentRequestPrepared','AssessmentRequestIssued','AssessmentRequestReissuePrepared','AssessmentRequestReissued',
    'AssessmentSubmitted','AssessmentReviewStarted','AssessmentDeficiencyLinked','AssessmentDocumentValidated',
    'AssessmentDocumentRejected','AssessmentCompleted','AssessmentCancelled'
));

COMMIT;
