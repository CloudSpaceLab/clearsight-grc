BEGIN;

DROP INDEX group_oversight_child_domain_source_idx;

ALTER TABLE group_oversight_child_facts
    DROP CONSTRAINT group_child_domain_posture_json_check,
    DROP CONSTRAINT group_child_domain_high_water_json_check,
    DROP CONSTRAINT group_child_domain_source_check,
    DROP COLUMN domain_source_high_water,
    DROP COLUMN domain_posture,
    DROP COLUMN domain_definition_revision,
    DROP COLUMN domain_generated_at,
    DROP COLUMN domain_source_id,
    DROP COLUMN domain_state;

COMMIT;
