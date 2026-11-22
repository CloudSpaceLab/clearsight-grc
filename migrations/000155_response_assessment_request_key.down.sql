BEGIN;
DO $$ BEGIN IF EXISTS (SELECT 1 FROM capture_response_assessments WHERE request_key IS NOT NULL) THEN RAISE EXCEPTION 'assessment request receipts exist'; END IF; END $$;
DROP INDEX capture_response_assessment_request_key_unique;
ALTER TABLE capture_response_assessments DROP CONSTRAINT capture_response_assessment_request_receipt_check, DROP COLUMN request_payload, DROP COLUMN request_reviewer_id, DROP COLUMN request_key;
COMMIT;
