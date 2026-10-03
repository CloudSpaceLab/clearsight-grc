BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM rcsa_cycles
        WHERE first_line_response_revision_id IS NOT NULL
    ) THEN
        RAISE EXCEPTION 'Retain governed RCSA first-line response references before downgrade';
    END IF;
END;
$$;

ALTER TABLE rcsa_cycles
    DROP CONSTRAINT IF EXISTS rcsa_cycles_first_line_response_scope_fk,
    DROP COLUMN IF EXISTS first_line_response_revision_id;

DROP INDEX IF EXISTS capture_response_revisions_rcsa_scope_idx;

COMMIT;
