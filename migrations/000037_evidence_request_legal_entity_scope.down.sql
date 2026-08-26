BEGIN;

DROP TRIGGER IF EXISTS capture_requests_legal_entity_immutable ON capture_requests;
DROP INDEX IF EXISTS capture_requests_entity_subject_idx;
DROP INDEX IF EXISTS capture_requests_entity_queue_idx;
ALTER TABLE capture_requests DROP CONSTRAINT IF EXISTS capture_requests_legal_entity_tenant_fk;
ALTER TABLE capture_requests DROP COLUMN IF EXISTS legal_entity_id;

COMMIT;
