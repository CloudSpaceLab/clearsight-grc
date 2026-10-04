BEGIN;

DO $data_boundary_history$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM governance_decisions
        WHERE object_type IN ('LEGAL_ENTITY_DATA_BOUNDARY','LEGAL_ENTITY_DATA_BOUNDARY_REVISION')
    ) THEN
        RAISE EXCEPTION 'legal-entity data-boundary governance decisions exist; refusing to erase data-boundary history';
    END IF;
END;
$data_boundary_history$;

DROP TABLE IF EXISTS legal_entity_data_boundary_revisions;
DROP TABLE IF EXISTS legal_entity_data_boundaries;
DROP FUNCTION IF EXISTS valid_legal_entity_data_regions(text[]);

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
        'ORGANIZATION_POSITION_REVISION'
    ));

COMMIT;
