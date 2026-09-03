BEGIN;

DROP TABLE IF EXISTS monitoring_collection_cycles;

DROP INDEX IF EXISTS capture_requests_origin_idx;
ALTER TABLE capture_requests
    DROP CONSTRAINT IF EXISTS capture_requests_predecessor_tenant_fk,
    DROP CONSTRAINT IF EXISTS capture_requests_previous_responses_check,
    DROP CONSTRAINT IF EXISTS capture_requests_origin_shape_check,
    DROP COLUMN IF EXISTS previous_responses,
    DROP COLUMN IF EXISTS predecessor_request_id,
    DROP COLUMN IF EXISTS origin_sequence,
    DROP COLUMN IF EXISTS origin_id,
    DROP COLUMN IF EXISTS origin_type;

ALTER TABLE monitoring_checks
    DROP CONSTRAINT IF EXISTS monitoring_checks_collection_policy_check,
    DROP COLUMN IF EXISTS reminder_count,
    DROP COLUMN IF EXISTS renewal_window_days,
    DROP COLUMN IF EXISTS validity_months;

COMMIT;
