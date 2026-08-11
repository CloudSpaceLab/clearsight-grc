BEGIN;

ALTER TABLE role_templates
    DROP CONSTRAINT IF EXISTS role_templates_capability_count_check,
    DROP COLUMN IF EXISTS capabilities;

ALTER TABLE org_positions
    DROP CONSTRAINT IF EXISTS org_positions_department_path_depth_check,
    DROP COLUMN IF EXISTS department_path;

COMMIT;
