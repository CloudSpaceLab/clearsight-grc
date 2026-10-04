BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM organization_position_revisions) THEN
        RAISE EXCEPTION 'organization position revisions exist; refusing to erase hierarchy governance history';
    END IF;
    IF EXISTS (
        SELECT 1
        FROM governance_decisions
        WHERE object_type IN ('ORGANIZATION_POSITION','ORGANIZATION_POSITION_REVISION')
    ) THEN
        RAISE EXCEPTION 'organization position governance decisions exist; refusing to erase hierarchy governance history';
    END IF;
END;
$$;

DROP TRIGGER IF EXISTS organization_position_write_guard ON org_positions;
DROP FUNCTION IF EXISTS validate_organization_position_write();
DROP TABLE IF EXISTS organization_position_revisions;

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
        'ORGANIZATION_SCOPE_REVISION'
    ));

COMMIT;
