BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM source_binding_checkpoints LIMIT 1) THEN
        RAISE EXCEPTION 'source binding checkpoints exist; migrate them before rolling back source-access runtime state';
    END IF;
    IF EXISTS (SELECT 1 FROM source_observations WHERE scope_kind<>'SOURCE' LIMIT 1) THEN
        RAISE EXCEPTION 'scoped source observations exist; migrate them before rolling back source-access runtime state';
    END IF;
END
$$;

DROP TRIGGER source_binding_checkpoint_guard_trigger ON source_binding_checkpoints;
DROP FUNCTION source_binding_checkpoint_guard();
DROP TABLE source_binding_checkpoints;

DROP INDEX source_observations_binding_history_idx;
DROP INDEX source_observations_view_history_idx;
DROP INDEX source_observations_connection_history_idx;
DROP TRIGGER source_observation_scope_guard_trigger ON source_observations;
DROP FUNCTION source_observation_scope_guard();

ALTER TABLE source_observations
    DROP CONSTRAINT source_observations_binding_fk,
    DROP CONSTRAINT source_observations_view_fk,
    DROP CONSTRAINT source_observations_connection_fk,
    DROP CONSTRAINT source_observations_scope_shape_ck,
    DROP CONSTRAINT source_observations_scope_kind_ck,
    DROP COLUMN binding_version,
    DROP COLUMN binding_id,
    DROP COLUMN view_version,
    DROP COLUMN view_id,
    DROP COLUMN connection_version,
    DROP COLUMN connection_id,
    DROP COLUMN scope_kind;

COMMIT;
