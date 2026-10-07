BEGIN;

ALTER TABLE metric_definitions
    DROP CONSTRAINT metric_definitions_unit_check,
    DROP CONSTRAINT metric_definitions_basis_check,
    DROP CONSTRAINT metric_definitions_condition_rule_check,
    DROP CONSTRAINT metric_definitions_aggregation_rule_check;

ALTER TABLE metric_definitions
    ADD CONSTRAINT metric_definitions_unit_check
        CHECK (unit IN ('COUNT','MONEY')),
    ADD CONSTRAINT metric_definitions_basis_check
        CHECK (basis IN ('CURRENT_POSTURE','PERIOD_FLOW')),
    ADD CONSTRAINT metric_definitions_condition_rule_check
        CHECK (condition_rule IN ('ZERO_CLEAR_POSITIVE_ATTENTION','NO_CONDITION')),
    ADD CONSTRAINT metric_definitions_aggregation_rule_check
        CHECK (aggregation_rule IN ('SUM_DISJOINT_COUNTS','SUM_SAME_CURRENCY'));

ALTER TABLE metric_observations
    ADD COLUMN currency text,
    ADD COLUMN member_count bigint;

ALTER TABLE metric_observations
    DROP CONSTRAINT metric_observations_value_check,
    DROP CONSTRAINT metric_observations_condition_check;

ALTER TABLE metric_observations
    ADD CONSTRAINT metric_observations_currency_check
        CHECK (currency IS NULL OR currency ~ '^[A-Z]{3}$'),
    ADD CONSTRAINT metric_observations_member_count_check
        CHECK (member_count IS NULL OR member_count>=0),
    ADD CONSTRAINT metric_observations_typed_value_check
        CHECK ((currency IS NULL AND value>=0) OR currency IS NOT NULL),
    ADD CONSTRAINT metric_observations_condition_check
        CHECK (condition IN ('CLEAR','ATTENTION','NEUTRAL'));

ALTER TABLE metric_observation_daily_rollups
    ADD COLUMN currency text,
    ADD COLUMN member_count bigint;

ALTER TABLE metric_observation_daily_rollups
    DROP CONSTRAINT metric_observation_daily_rollups_value_check,
    DROP CONSTRAINT metric_observation_daily_rollups_condition_check;

ALTER TABLE metric_observation_daily_rollups
    ADD CONSTRAINT metric_daily_rollups_currency_check
        CHECK (currency IS NULL OR currency ~ '^[A-Z]{3}
),
    ADD CONSTRAINT metric_daily_rollups_member_count_check
        CHECK (member_count IS NULL OR member_count>=0),
    ADD CONSTRAINT metric_daily_rollups_typed_value_check
        CHECK ((currency IS NULL AND value>=0) OR currency IS NOT NULL),
    ADD CONSTRAINT metric_observation_daily_rollups_condition_check
        CHECK (condition IN ('CLEAR','ATTENTION','NEUTRAL'));

CREATE FUNCTION validate_metric_native_measure() RETURNS trigger
LANGUAGE plpgsql
AS $metric_native_measure$
DECLARE
    metric_unit text;
BEGIN
    SELECT definition.unit
      INTO metric_unit
      FROM metric_definitions definition
     WHERE definition.metric_id=NEW.metric_id
       AND definition.revision=NEW.definition_revision;

    IF NOT FOUND THEN
        RAISE EXCEPTION 'Metric definition is unavailable';
    END IF;

    IF metric_unit='COUNT' THEN
        IF NEW.value<0 OR NEW.currency IS NOT NULL OR NEW.member_count IS NOT NULL THEN
            RAISE EXCEPTION 'COUNT metric observation has invalid native measure fields';
        END IF;
    ELSIF metric_unit='MONEY' THEN
        IF NEW.currency IS NULL OR NEW.member_count IS NULL OR NEW.member_count<0 THEN
            RAISE EXCEPTION 'MONEY metric observation requires currency and member count';
        END IF;
    ELSE
        RAISE EXCEPTION 'Metric observation unit is unsupported';
    END IF;
    RETURN NEW;
END;
$metric_native_measure$;

CREATE TRIGGER metric_observations_validate_native_measure
    BEFORE INSERT OR UPDATE ON metric_observations
    FOR EACH ROW EXECUTE FUNCTION validate_metric_native_measure();

CREATE TRIGGER metric_daily_rollups_validate_native_measure
    BEFORE INSERT OR UPDATE ON metric_observation_daily_rollups
    FOR EACH ROW EXECUTE FUNCTION validate_metric_native_measure();

COMMIT;
