BEGIN;
DROP INDEX IF EXISTS third_party_assessment_request_document_idx;
DROP INDEX IF EXISTS capture_submissions_document_order_idx;
DROP INDEX IF EXISTS capture_response_revisions_submission_idx;
COMMIT;
