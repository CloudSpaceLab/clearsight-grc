BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM matters WHERE organization_scope_id IS NOT NULL LIMIT 1) THEN
        RAISE EXCEPTION 'Retain Matter organization-scope attribution before downgrade';
    END IF;
END;
$$;

DROP INDEX IF EXISTS matters_organization_scope_updated_idx;
ALTER TABLE matters
    DROP CONSTRAINT IF EXISTS matters_organization_scope_fk,
    DROP COLUMN IF EXISTS organization_scope_id;

COMMIT;
