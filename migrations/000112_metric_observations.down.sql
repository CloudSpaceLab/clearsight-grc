BEGIN;

DO $metric_observation_downgrade$
BEGIN
    IF EXISTS (SELECT 1 FROM metric_observations) THEN
        RAISE EXCEPTION 'Refusing to discard retained metric observations';
    END IF;
END;
$metric_observation_downgrade$;

DROP TABLE IF EXISTS metric_observations;
DROP INDEX IF EXISTS oversight_snapshots_metric_source_uq;
DROP FUNCTION IF EXISTS prevent_metric_observation_mutation();
DROP TABLE IF EXISTS metric_definitions;
DROP FUNCTION IF EXISTS prevent_metric_definition_mutation();

COMMIT;
