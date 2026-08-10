BEGIN;

DROP TABLE IF EXISTS directory_group_role_bindings;
DROP TABLE IF EXISTS directory_group_members;
DROP TABLE IF EXISTS directory_groups;
DROP TABLE IF EXISTS scim_users;
DROP TABLE IF EXISTS scim_sources;

ALTER TABLE role_templates
    DROP CONSTRAINT IF EXISTS role_templates_tenant_identity_unique;

COMMIT;
