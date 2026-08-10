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

COMMIT;
