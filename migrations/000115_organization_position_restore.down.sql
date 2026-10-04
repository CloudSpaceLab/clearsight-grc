BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM organization_position_revisions
        WHERE restored_from_revision_id IS NOT NULL
    ) THEN
        RAISE EXCEPTION 'organization position restore history exists; refusing to erase restore provenance';
    END IF;
END;
$$;

DROP INDEX IF EXISTS organization_position_revisions_restore_source_idx;
ALTER TABLE organization_position_revisions
    DROP CONSTRAINT IF EXISTS organization_position_revisions_restore_source_fk,
    DROP COLUMN IF EXISTS restored_from_revision_id;

COMMIT;
