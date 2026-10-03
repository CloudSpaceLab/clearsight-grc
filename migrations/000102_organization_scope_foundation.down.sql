BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM organization_scopes WHERE origin='MANAGED' LIMIT 1) THEN
        RAISE EXCEPTION 'Retain managed organization scope history before downgrade';
    END IF;
END;
$$;

DROP TRIGGER IF EXISTS directory_group_role_bindings_bind_organization_scope ON directory_group_role_bindings;
DROP TRIGGER IF EXISTS org_positions_bind_organization_scope ON org_positions;
DROP FUNCTION IF EXISTS bind_legacy_organization_scope();
DROP FUNCTION IF EXISTS reconcile_organization_scopes();
DROP FUNCTION IF EXISTS organization_scope_for_department_path(uuid,uuid,text[],timestamptz);

DROP INDEX IF EXISTS directory_group_role_bindings_organization_scope_idx;
ALTER TABLE directory_group_role_bindings
    DROP CONSTRAINT IF EXISTS directory_group_role_bindings_organization_scope_fk,
    DROP COLUMN IF EXISTS organization_scope_id;

DROP INDEX IF EXISTS org_positions_organization_scope_idx;
ALTER TABLE org_positions
    DROP CONSTRAINT IF EXISTS org_positions_organization_scope_fk,
    DROP COLUMN IF EXISTS organization_scope_id;

DROP TABLE IF EXISTS organization_scopes;

COMMIT;
