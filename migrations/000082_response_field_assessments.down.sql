BEGIN;
DROP INDEX IF EXISTS capture_response_assessment_outbox_uq;
DROP TABLE IF EXISTS capture_field_assessments;
DROP TABLE IF EXISTS capture_response_assessments;
DROP FUNCTION IF EXISTS prevent_response_assessment_mutation();
COMMIT;
