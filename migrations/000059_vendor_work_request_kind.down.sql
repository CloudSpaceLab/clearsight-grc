BEGIN;

ALTER TABLE third_party_work_requests
    DROP COLUMN IF EXISTS request_kind;

COMMIT;
