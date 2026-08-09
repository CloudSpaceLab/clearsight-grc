BEGIN;

DROP INDEX IF EXISTS capture_requests_internal_recipient_queue_idx;
ALTER TABLE capture_requests DROP CONSTRAINT IF EXISTS capture_requests_recipient_shape_check;
ALTER TABLE capture_requests
    DROP COLUMN IF EXISTS recipient_hint,
    DROP COLUMN IF EXISTS recipient_audience_hash,
    DROP COLUMN IF EXISTS recipient_principal_id,
    DROP COLUMN IF EXISTS recipient_type;

COMMIT;
