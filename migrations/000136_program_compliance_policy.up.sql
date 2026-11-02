BEGIN;

ALTER TABLE programs
    ADD COLUMN calculation_policy jsonb;

ALTER TABLE programs
    ADD CONSTRAINT programs_calculation_policy_object
    CHECK (
        calculation_policy IS NULL OR (
            jsonb_typeof(calculation_policy)='object'
            AND octet_length(calculation_policy::text) <= 32768
        )
    );

COMMENT ON COLUMN programs.calculation_policy IS
    'Current approved and pending Program compliance calculation revisions; authoritative change history remains in continuity_events.';

COMMIT;
