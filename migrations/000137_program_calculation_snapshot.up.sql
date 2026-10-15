BEGIN;

ALTER TABLE program_state_snapshots
    ADD COLUMN calculation jsonb;

ALTER TABLE program_state_snapshots
    ADD CONSTRAINT program_state_calculation_object
    CHECK (calculation IS NULL OR (
        jsonb_typeof(calculation)='object'
        AND octet_length(calculation::text) <= 16384
    ));

COMMENT ON COLUMN program_state_snapshots.calculation IS
    'Derived approved Program calculation summary, bound to the retained state snapshot and its exact policy/source revisions. Null until authoritative scoring is available.';

COMMIT;
