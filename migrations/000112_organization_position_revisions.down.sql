BEGIN;

DO $position_revision_down$
BEGIN
    IF EXISTS (SELECT 1 FROM organization_position_revisions) THEN
        RAISE EXCEPTION 'organization position revisions exist; refusing to erase hierarchy governance history';
    END IF;
    IF EXISTS (
        SELECT 1 FROM governance_decisions
        WHERE object_type='ORGANIZATION_POSITION_REVISION'
    ) THEN
        RAISE EXCEPTION 'organization position governance decisions exist; refusing to erase hierarchy governance history';
    END IF;
END;
$position_revision_down$;

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

DROP TRIGGER IF EXISTS organization_position_revisions_history_guard ON organization_position_revisions;
DROP FUNCTION IF EXISTS protect_organization_position_revision();
DROP INDEX IF EXISTS organization_position_revisions_queue_idx;
DROP INDEX IF EXISTS organization_position_revisions_pending_idx;
DROP TABLE IF EXISTS organization_position_revisions;

COMMIT;
