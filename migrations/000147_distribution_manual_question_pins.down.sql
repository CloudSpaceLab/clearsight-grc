BEGIN;
DO $$ BEGIN IF EXISTS (SELECT 1 FROM capture_distribution_manual_question_pins) THEN RAISE EXCEPTION 'Distribution manual question pins must be retained'; END IF; END $$;
DROP TABLE capture_distribution_manual_question_pins;
DROP FUNCTION prevent_distribution_manual_pin_mutation();
COMMIT;
