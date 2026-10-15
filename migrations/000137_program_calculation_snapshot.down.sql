BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM program_state_snapshots WHERE calculation IS NOT NULL) THEN
        RAISE EXCEPTION 'Cannot drop Program calculations while retained scored snapshots exist';
    END IF;
END $$;

ALTER TABLE program_state_snapshots DROP CONSTRAINT program_state_calculation_object;
ALTER TABLE program_state_snapshots DROP COLUMN calculation;

COMMIT;
