BEGIN;

DROP TABLE IF EXISTS metric_observations;
DROP FUNCTION IF EXISTS prevent_metric_observation_update();
DROP TABLE IF EXISTS metric_definitions;
DROP FUNCTION IF EXISTS prevent_metric_definition_mutation();

COMMIT;
