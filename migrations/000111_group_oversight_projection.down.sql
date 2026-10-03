BEGIN;

DROP TABLE IF EXISTS group_oversight_child_facts;
DROP TABLE IF EXISTS group_oversight_runs;
DROP FUNCTION IF EXISTS prevent_group_oversight_projection_mutation();

COMMIT;
