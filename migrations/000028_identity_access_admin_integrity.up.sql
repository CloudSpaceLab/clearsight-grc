BEGIN;

CREATE UNIQUE INDEX directory_group_role_bindings_current_unique_idx
    ON directory_group_role_bindings(tenant_id, group_id, role_template_id, legal_entity_id, department_path)
    WHERE valid_until IS NULL;

ALTER TABLE governance_decisions
    DROP CONSTRAINT IF EXISTS governance_decisions_object_type_check;
ALTER TABLE governance_decisions
    ADD CONSTRAINT governance_decisions_object_type_check
    CHECK (object_type IN (
        'ROUTING_POLICY',
        'DELEGATION',
        'SEGREGATION_RULE',
        'SCIM_SOURCE',
        'DIRECTORY_GROUP_ROLE_BINDING'
    ));

COMMIT;
