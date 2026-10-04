BEGIN;

DO $position_role_history$
BEGIN
    IF EXISTS (SELECT 1 FROM organization_position_role_revisions) THEN
        RAISE EXCEPTION 'organization position role revisions exist; refusing to erase role governance history';
    END IF;
    IF EXISTS (
        SELECT 1
        FROM governance_decisions
        WHERE object_type IN ('ORGANIZATION_POSITION_ROLE_BINDING','ORGANIZATION_POSITION_ROLE_REVISION')
    ) THEN
        RAISE EXCEPTION 'organization position role governance decisions exist; refusing to erase role governance history';
    END IF;
END;
$position_role_history$;

DROP TABLE IF EXISTS organization_position_role_revisions;
DROP INDEX IF EXISTS position_role_workspace_active_uidx;
ALTER TABLE position_role_bindings DROP COLUMN IF EXISTS binding_purpose;

ALTER TABLE governance_decisions
    DROP CONSTRAINT IF EXISTS governance_decisions_object_type_check;
ALTER TABLE governance_decisions
    ADD CONSTRAINT governance_decisions_object_type_check
    CHECK (object_type IN (
        'ROUTING_POLICY',
        'DELEGATION',
        'SEGREGATION_RULE',
        'SCIM_SOURCE',
        'DIRECTORY_GROUP_ROLE_BINDING',
        'ORGANIZATION_SCOPE',
        'ORGANIZATION_SCOPE_REVISION',
        'ORGANIZATION_POSITION',
        'ORGANIZATION_POSITION_REVISION',
        'LEGAL_ENTITY_DATA_BOUNDARY',
        'LEGAL_ENTITY_DATA_BOUNDARY_REVISION'
    ));

COMMIT;
