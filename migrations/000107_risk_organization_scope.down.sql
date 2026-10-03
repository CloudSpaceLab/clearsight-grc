BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM risks WHERE organization_scope_id IS NOT NULL LIMIT 1) THEN
        RAISE EXCEPTION 'Retain Risk organization-scope attribution before downgrade';
    END IF;
END;
$$;

DROP INDEX IF EXISTS risks_organization_scope_updated_idx;
ALTER TABLE risks
    DROP CONSTRAINT IF EXISTS risks_organization_scope_fk,
    DROP COLUMN IF EXISTS organization_scope_id;

COMMIT;
