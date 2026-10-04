BEGIN;

CREATE TABLE metric_observation_daily_rollups (
    tenant_id uuid NOT NULL,
    legal_entity_id uuid NOT NULL,
    metric_id text NOT NULL,
    definition_revision text NOT NULL,
    bucket_date date NOT NULL,
    observation_id uuid NOT NULL,
    source_id uuid NOT NULL,
    source_revision text NOT NULL,
    source_high_water jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(source_high_water)='object'),
    generated_at timestamptz NOT NULL,
    period_start timestamptz NOT NULL,
    period_end timestamptz NOT NULL,
    posture_as_of timestamptz NOT NULL,
    value bigint NOT NULL CHECK (value >= 0),
    condition text NOT NULL CHECK (condition IN ('CLEAR','ATTENTION')),
    freshness text NOT NULL CHECK (freshness IN ('CURRENT','STALE')),
    completeness text NOT NULL CHECK (completeness IN ('COMPLETE','PARTIAL','UNKNOWN')),
    population bigint NOT NULL CHECK (population >= 0),
    excluded bigint CHECK (excluded IS NULL OR excluded >= 0),
    unknown bigint CHECK (unknown IS NULL OR unknown >= 0),
    rolled_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CHECK (period_start<=period_end),
    PRIMARY KEY (tenant_id,legal_entity_id,metric_id,definition_revision,bucket_date),
    CONSTRAINT metric_daily_rollup_entity_fk
        FOREIGN KEY (legal_entity_id,tenant_id)
        REFERENCES legal_entities(id,tenant_id),
    CONSTRAINT metric_daily_rollup_definition_fk
        FOREIGN KEY (metric_id,definition_revision)
        REFERENCES metric_definitions(metric_id,revision)
);

CREATE INDEX metric_daily_rollup_trend_idx
    ON metric_observation_daily_rollups(
        tenant_id,legal_entity_id,metric_id,definition_revision,bucket_date DESC
    );

DROP TRIGGER metric_observations_immutable ON metric_observations;
CREATE TRIGGER metric_observations_immutable
    BEFORE UPDATE ON metric_observations
    FOR EACH ROW EXECUTE FUNCTION prevent_metric_observation_mutation();

CREATE FUNCTION guard_metric_observation_delete() RETURNS trigger
LANGUAGE plpgsql
AS $metric_observation_delete$
BEGIN
    IF OLD.generated_at >= clock_timestamp()-interval '14 days' THEN
        RAISE EXCEPTION 'Metric observation is inside the 14-day raw retention window';
    END IF;
    IF NOT EXISTS (
        SELECT 1
        FROM metric_observation_daily_rollups rollup
        WHERE rollup.tenant_id=OLD.tenant_id
          AND rollup.legal_entity_id=OLD.legal_entity_id
          AND rollup.metric_id=OLD.metric_id
          AND rollup.definition_revision=OLD.definition_revision
          AND rollup.bucket_date=(OLD.generated_at AT TIME ZONE 'UTC')::date
          AND rollup.generated_at>=OLD.generated_at
    ) THEN
        RAISE EXCEPTION 'Metric observation has no retained daily rollup';
    END IF;
    RETURN OLD;
END;
$metric_observation_delete$;

CREATE TRIGGER metric_observations_guard_delete
    BEFORE DELETE ON metric_observations
    FOR EACH ROW EXECUTE FUNCTION guard_metric_observation_delete();

COMMIT;
