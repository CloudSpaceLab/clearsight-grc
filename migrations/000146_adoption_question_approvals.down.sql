BEGIN;
DO $$ BEGIN IF EXISTS (SELECT 1 FROM program_adoption_question_approvals) THEN RAISE EXCEPTION 'Adoption question approvals must be retained'; END IF; END $$;
DROP TABLE program_adoption_question_approvals;
DROP FUNCTION prevent_adoption_question_approval_mutation();
COMMIT;
