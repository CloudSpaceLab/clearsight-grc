BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM monitoring_adverse_episodes LIMIT 1) THEN
        RAISE EXCEPTION 'Retain monitoring adverse episode history before downgrade';
    END IF;
END;
$$;

DROP TRIGGER IF EXISTS monitoring_adverse_episode_no_delete ON monitoring_adverse_episodes;
DROP FUNCTION IF EXISTS protect_monitoring_adverse_episode_delete();
DROP TABLE IF EXISTS monitoring_adverse_episodes;
DROP INDEX IF EXISTS monitoring_results_episode_scope_idx;

DROP INDEX monitoring_outbox_event_uq;
CREATE UNIQUE INDEX monitoring_outbox_event_uq
    ON outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,(COALESCE(payload->>'version','1')))
    WHERE aggregate_type IN ('MONITORING_FORM','MONITORING_CHECK','MONITORING_RESULT');

ALTER TABLE monitoring_events
    DROP CONSTRAINT monitoring_events_aggregate_type_check,
    ADD CONSTRAINT monitoring_events_aggregate_type_check
        CHECK (aggregate_type IN ('MONITORING_FORM','MONITORING_CHECK','MONITORING_RESULT'));

COMMIT;
