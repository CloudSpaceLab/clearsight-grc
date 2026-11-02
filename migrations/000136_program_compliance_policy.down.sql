BEGIN;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM programs WHERE calculation_policy IS NOT NULL) THEN
        RAISE EXCEPTION 'Cannot drop Program calculation policy: approved or pending configurations exist';
    END IF;
END $$;

ALTER TABLE programs DROP CONSTRAINT programs_calculation_policy_object;
ALTER TABLE programs DROP COLUMN calculation_policy;

COMMIT;
