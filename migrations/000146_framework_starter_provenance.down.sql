BEGIN;
-- Verified citation corrections are intentionally retained; rollback must not
-- restore an incorrect source or mutate immutable historical records.
COMMIT;
