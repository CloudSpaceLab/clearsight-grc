BEGIN;

ALTER TABLE monitoring_events
    DROP CONSTRAINT monitoring_events_aggregate_type_check,
    ADD CONSTRAINT monitoring_events_aggregate_type_check
        CHECK (aggregate_type IN ('MONITORING_FORM','MONITORING_CHECK','MONITORING_RESULT','MONITORING_ADVERSE_EPISODE'));

DROP INDEX monitoring_outbox_event_uq;
CREATE UNIQUE INDEX monitoring_outbox_event_uq
    ON outbox_events(tenant_id,aggregate_type,aggregate_id,event_type,(COALESCE(payload->>'version','1')))
    WHERE aggregate_type IN ('MONITORING_FORM','MONITORING_CHECK','MONITORING_RESULT','MONITORING_ADVERSE_EPISODE');

CREATE UNIQUE INDEX monitoring_results_episode_scope_idx
    ON monitoring_results(id,tenant_id,program_id,monitoring_check_id);

CREATE TABLE monitoring_adverse_episodes (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    program_id uuid NOT NULL,
    monitoring_check_id uuid NOT NULL,
    state text NOT NULL CHECK (state IN ('OPEN','CLOSED')),
    matter_id uuid,
    first_result_id uuid NOT NULL,
    last_result_id uuid NOT NULL,
    last_check_version bigint NOT NULL CHECK (last_check_version>0),
    last_band text NOT NULL CHECK (last_band IN ('LOW','MODERATE','HIGH','CRITICAL','NOT_ASSESSED')),
    last_score numeric(7,4) CHECK (last_score IS NULL OR last_score BETWEEN 0 AND 100),
    last_coverage numeric(8,6) NOT NULL CHECK (last_coverage BETWEEN 0 AND 1),
    opened_at timestamptz NOT NULL,
    closed_at timestamptz,
    updated_at timestamptz NOT NULL,
    record_version bigint NOT NULL DEFAULT 1 CHECK (record_version>0),
    UNIQUE(id,tenant_id,legal_entity_id),
    FOREIGN KEY(legal_entity_id,tenant_id)
        REFERENCES legal_entities(id,tenant_id),
    FOREIGN KEY(program_id,tenant_id,legal_entity_id)
        REFERENCES programs(id,tenant_id,legal_entity_id),
    FOREIGN KEY(tenant_id,monitoring_check_id,last_check_version,program_id)
        REFERENCES monitoring_checks(tenant_id,id,version,program_id),
    FOREIGN KEY(first_result_id,tenant_id,program_id,monitoring_check_id)
        REFERENCES monitoring_results(id,tenant_id,program_id,monitoring_check_id),
    FOREIGN KEY(last_result_id,tenant_id,program_id,monitoring_check_id)
        REFERENCES monitoring_results(id,tenant_id,program_id,monitoring_check_id),
    FOREIGN KEY(matter_id,tenant_id,legal_entity_id)
        REFERENCES matters(id,tenant_id,legal_entity_id),
    CHECK ((state='OPEN' AND closed_at IS NULL) OR (state='CLOSED' AND closed_at IS NOT NULL)),
    CHECK (updated_at>=opened_at)
);

CREATE UNIQUE INDEX monitoring_adverse_episode_open_uq
    ON monitoring_adverse_episodes(tenant_id,legal_entity_id,monitoring_check_id)
    WHERE state='OPEN';

CREATE INDEX monitoring_adverse_episode_history_idx
    ON monitoring_adverse_episodes(tenant_id,legal_entity_id,monitoring_check_id,opened_at DESC,id DESC);

CREATE FUNCTION protect_monitoring_adverse_episode_delete() RETURNS trigger LANGUAGE plpgsql AS $episode$
BEGIN
    RAISE EXCEPTION 'monitoring adverse episode history cannot be deleted';
END;
$episode$;

CREATE TRIGGER monitoring_adverse_episode_no_delete
    BEFORE DELETE ON monitoring_adverse_episodes
    FOR EACH ROW EXECUTE FUNCTION protect_monitoring_adverse_episode_delete();

COMMIT;
