BEGIN;

CREATE UNIQUE INDEX directory_group_role_bindings_current_unique_idx
    ON directory_group_role_bindings(tenant_id, group_id, role_template_id, legal_entity_id, department_path)
    WHERE valid_until IS NULL;

COMMIT;
