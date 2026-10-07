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
        CHECK (currency IS NULL OR currency ~ '^[A-Z]{3}$'),
    ADD CONSTRAINT metric_daily_rollups_member_count_check
        CHECK (member_count IS NULL OR member_count>=0),
    ADD CONSTRAINT metric_daily_rollups_typed_value_check
        CHECK ((currency IS NULL AND value>=0) OR currency IS NOT NULL),
    ADD CONSTRAINT metric_observation_daily_rollups_condition_check
        CHECK (condition IN ('CLEAR','ATTENTION','NEUTRAL'));

COMMIT;
