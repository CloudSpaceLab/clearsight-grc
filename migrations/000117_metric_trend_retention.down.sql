BEGIN;

DO $metric_trend_downgrade$
BEGIN
    IF EXISTS (SELECT 1 FROM metric_observation_daily_rollups) THEN
        RAISE EXCEPTION 'Refusing to discard retained metric trend rollups';
    END IF;
END;
$metric_trend_downgrade$;

DROP TRIGGER IF EXISTS metric_observations_guard_delete ON metric_observations;
DROP FUNCTION IF EXISTS guard_metric_observation_delete();
DROP TABLE IF EXISTS metric_observation_daily_rollups;

DROP TRIGGER IF EXISTS metric_observations_immutable ON metric_observations;
CREATE TRIGGER metric_observations_immutable
    BEFORE UPDATE OR DELETE ON metric_observations
    FOR EACH ROW EXECUTE FUNCTION prevent_metric_observation_mutation();

COMMIT;
