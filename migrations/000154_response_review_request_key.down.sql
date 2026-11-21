BEGIN;
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM capture_response_reviews WHERE request_key IS NOT NULL) THEN
  RAISE EXCEPTION 'cannot remove recorded response review request keys';
 END IF;
END $$;
DROP INDEX capture_response_reviews_request_key_unique;
ALTER TABLE capture_response_reviews DROP COLUMN request_key;
COMMIT;
