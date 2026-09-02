BEGIN;

ALTER TABLE automation_policies
    DROP CONSTRAINT IF EXISTS automation_policies_updated_after_created_ck,
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_at;

COMMIT;
