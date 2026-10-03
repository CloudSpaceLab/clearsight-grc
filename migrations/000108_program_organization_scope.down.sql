BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM programs WHERE organization_scope_id IS NOT NULL LIMIT 1) THEN
        RAISE EXCEPTION 'Retain Program organization-scope attribution before downgrade';
    END IF;
END;
$$;

DROP INDEX IF EXISTS programs_organization_scope_updated_idx;
ALTER TABLE programs
    DROP CONSTRAINT IF EXISTS programs_organization_scope_fk,
    DROP COLUMN IF EXISTS organization_scope_id;

COMMIT;
