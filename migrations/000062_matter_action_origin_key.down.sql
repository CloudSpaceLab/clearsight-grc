BEGIN;

DROP INDEX IF EXISTS matter_actions_origin_key_idx;
ALTER TABLE matter_actions DROP COLUMN IF EXISTS origin_key;

COMMIT;
