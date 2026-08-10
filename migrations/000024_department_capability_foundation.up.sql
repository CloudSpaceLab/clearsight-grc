BEGIN;

ALTER TABLE org_positions
    ADD COLUMN department_path text[] NOT NULL DEFAULT '{}'::text[];

ALTER TABLE org_positions
    ADD CONSTRAINT org_positions_department_path_depth_check
    CHECK (cardinality(department_path) <= 12);

CREATE INDEX org_positions_department_path_idx
    ON org_positions USING gin (department_path)
    WHERE valid_until IS NULL;

ALTER TABLE role_templates
    ADD COLUMN capabilities text[] NOT NULL DEFAULT '{}'::text[];

ALTER TABLE role_templates
    ADD CONSTRAINT role_templates_capability_count_check
    CHECK (cardinality(capabilities) <= 64);

CREATE TABLE principal_role_bindings (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    principal_id uuid NOT NULL REFERENCES principals(id),
    role_template_id uuid NOT NULL REFERENCES role_templates(id),
    department_path text[] NOT NULL DEFAULT '{}'::text[],
    valid_from timestamptz NOT NULL DEFAULT clock_timestamp(),
    valid_until timestamptz,
    recorded_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (valid_until IS NULL OR valid_from < valid_until),
    CHECK (cardinality(department_path) <= 12)
);

CREATE UNIQUE INDEX principal_role_bindings_active_idx
    ON principal_role_bindings(tenant_id, principal_id, role_template_id, department_path)
    WHERE valid_until IS NULL;

CREATE INDEX principal_role_bindings_principal_idx
    ON principal_role_bindings(tenant_id, principal_id, valid_from, valid_until);

COMMIT;
