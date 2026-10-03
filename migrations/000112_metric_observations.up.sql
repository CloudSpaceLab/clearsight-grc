BEGIN;

CREATE TABLE metric_definitions (
    metric_id text NOT NULL,
    revision text NOT NULL,
    label text NOT NULL,
    unit text NOT NULL CHECK (unit IN ('COUNT')),
    basis text NOT NULL CHECK (basis IN ('CURRENT_POSTURE')),
    condition_rule text NOT NULL CHECK (condition_rule IN ('ZERO_CLEAR_POSITIVE_ATTENTION')),
    aggregation_rule text NOT NULL CHECK (aggregation_rule IN ('SUM_DISJOINT_COUNTS')),
    drill_workspace text NOT NULL,
    drill_filter text NOT NULL,
    drill_consistency text NOT NULL CHECK (drill_consistency IN ('CURRENT_STATE')),
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (metric_id, revision)
);

INSERT INTO metric_definitions(
    metric_id,revision,label,unit,basis,condition_rule,aggregation_rule,
    drill_workspace,drill_filter,drill_consistency
) VALUES
    ('critical_high_open','home-oversight-v2','Critical and high','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','critical-high','CURRENT_STATE'),
    ('overdue_open','home-oversight-v2','Overdue','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','overdue','CURRENT_STATE'),
    ('routing_gaps','home-oversight-v2','Routing gaps','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','routing-gaps','CURRENT_STATE'),
    ('outcome_failures','home-oversight-v2','Outcome failures','COUNT','CURRENT_POSTURE','ZERO_CLEAR_POSITIVE_ATTENTION','SUM_DISJOINT_COUNTS','oversight','outcome-failures','CURRENT_STATE');

CREATE TABLE metric_observations (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    legal_entity_id uuid NOT NULL,
    metric_id text NOT NULL,
    definition_revision text NOT NULL,
    source_kind text NOT NULL CHECK (source_kind='OVERSIGHT_SNAPSHOT'),
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
    created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    CONSTRAINT metric_observation_entity_fk
        FOREIGN KEY (legal_entity_id, tenant_id)
        REFERENCES legal_entities(id, tenant_id),
    CONSTRAINT metric_observation_definition_fk
        FOREIGN KEY (metric_id, definition_revision)
        REFERENCES metric_definitions(metric_id, revision),
    CONSTRAINT metric_observation_period_check CHECK (period_start <= period_end),
    CONSTRAINT metric_observation_source_unique UNIQUE (source_kind, source_id, metric_id, definition_revision)
);

CREATE INDEX metric_observations_trend_idx
    ON metric_observations(tenant_id, legal_entity_id, metric_id, generated_at DESC, id DESC);

CREATE FUNCTION prevent_metric_definition_mutation() RETURNS trigger
LANGUAGE plpgsql
AS $metric_definition$
BEGIN
    RAISE EXCEPTION 'Metric definitions are immutable; insert a new revision';
END;
$metric_definition$;

CREATE TRIGGER metric_definitions_immutable
    BEFORE UPDATE OR DELETE ON metric_definitions
    FOR EACH ROW EXECUTE FUNCTION prevent_metric_definition_mutation();

CREATE FUNCTION prevent_metric_observation_update() RETURNS trigger
LANGUAGE plpgsql
AS $metric_observation$
BEGIN
    RAISE EXCEPTION 'Metric observations are immutable';
END;
$metric_observation$;

CREATE TRIGGER metric_observations_immutable
    BEFORE UPDATE ON metric_observations
    FOR EACH ROW EXECUTE FUNCTION prevent_metric_observation_update();

COMMIT;
