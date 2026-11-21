BEGIN;
DROP INDEX IF EXISTS capture_response_review_outbox_uq;
DROP TABLE IF EXISTS capture_response_reviews;
COMMIT;
