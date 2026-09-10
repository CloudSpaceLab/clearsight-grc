BEGIN;

-- Migration 000005 created separate type and length checks. Migration 000036
-- replaced the type check with the shared 200-field contract, but PostgreSQL's
-- automatically suffixed length check remained and still rejected >50 fields.
-- Keep the current array/type/200-field constraint; remove only the obsolete cap.
ALTER TABLE capture_requests DROP CONSTRAINT IF EXISTS capture_requests_fields_check1;

COMMIT;
