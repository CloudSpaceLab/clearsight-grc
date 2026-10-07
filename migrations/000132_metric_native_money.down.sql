BEGIN;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM metric_definitions
        WHERE unit<>'COUNT'
           OR basis<>'CURRENT_POSTURE'
           OR condition_rule<>'ZERO_CLEAR_POSITIVE_ATTENTION'
           OR aggregation_rule<>'SUM_DISJOINT_COUNTS'
    ) OR EXISTS (
        SELECT 1 FROM metric_observations
        WHERE currency IS NOT NULL OR member_count IS NOT NULL OR condition='NEUTRAL'
    ) OR EXISTS (
        SELECT 1 FROM metric_observation_daily_rollups
        WHERE currency IS NOT NULL OR member_count IS NOT NULL OR condition='NEUTRAL'
    ) THEN
        RAISE EXCEPTION 'Native metric measure history exists; refusing to erase it';
    END IF;
END;
$$;

ALTER TABLE metric_observation_daily_rollups
    DROP CONSTRAINT metric_daily_rollups_currency_check,
    DROP CONSTRAINT metric_daily_rollups_member_count_check,
    DROP CONSTRAINT metric_daily_rollups_typed_value_check,
    DROP COLUMN currency,
    DROP COLUMN member_count,
    DROP CONSTRAINT metric_observation_daily_rollups_condition_check;
ALTER TABLE metric_observation_daily_rollups
    ADD CONSTRAINT metric_observation_daily_rollups_condition_check
        CHECK (condition IN ('CLEAR','ATTENTION')),
    ADD CONSTRAINT metric_observation_daily_rollups_value_check
        CHECK (value>=0);

ALTER TABLE metric_observations
    DROP CONSTRAINT metric_observations_currency_check,
    DROP CONSTRAINT metric_observations_member_count_check,
    DROP CONSTRAINT metric_observations_typed_value_check,
    DROP COLUMN currency,
    DROP COLUMN member_count,
    DROP CONSTRAINT metric_observations_condition_check;
ALTER TABLE metric_observations
    ADD CONSTRAINT metric_observations_condition_check
        CHECK (condition IN ('CLEAR','ATTENTION')),
    ADD CONSTRAINT metric_observations_value_check
        CHECK (value>=0);

ALTER TABLE metric_definitions
    DROP CONSTRAINT metric_definitions_unit_check,
    DROP CONSTRAINT metric_definitions_basis_check,
    DROP CONSTRAINT metric_definitions_condition_rule_check,
    DROP CONSTRAINT metric_definitions_aggregation_rule_check;
ALTER TABLE metric_definitions
    ADD CONSTRAINT metric_definitions_unit_check CHECK (unit IN ('COUNT')),
    ADD CONSTRAINT metric_definitions_basis_check CHECK (basis IN ('CURRENT_POSTURE')),
    ADD CONSTRAINT metric_definitions_condition_rule_check CHECK (condition_rule IN ('ZERO_CLEAR_POSITIVE_ATTENTION')),
    ADD CONSTRAINT metric_definitions_aggregation_rule_check CHECK (aggregation_rule IN ('SUM_DISJOINT_COUNTS'));

COMMIT;
