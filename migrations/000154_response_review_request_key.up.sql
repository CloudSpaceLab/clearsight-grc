BEGIN;
ALTER TABLE capture_response_reviews ADD COLUMN request_key uuid;
CREATE UNIQUE INDEX capture_response_reviews_request_key_unique ON capture_response_reviews(tenant_id,legal_entity_id,reviewer_id,request_key) WHERE request_key IS NOT NULL;
COMMIT;
