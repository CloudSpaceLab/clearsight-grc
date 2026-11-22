BEGIN;
ALTER TABLE capture_response_assessments ADD COLUMN request_key uuid, ADD COLUMN request_reviewer_id uuid, ADD COLUMN request_payload jsonb;
ALTER TABLE capture_response_assessments ADD CONSTRAINT capture_response_assessment_request_receipt_check CHECK ((request_key IS NULL AND request_reviewer_id IS NULL AND request_payload IS NULL) OR (request_key IS NOT NULL AND request_reviewer_id IS NOT NULL AND request_payload IS NOT NULL AND jsonb_typeof(request_payload)='object'));
CREATE UNIQUE INDEX capture_response_assessment_request_key_unique ON capture_response_assessments(tenant_id,legal_entity_id,request_reviewer_id,request_key) WHERE request_key IS NOT NULL;
COMMIT;
