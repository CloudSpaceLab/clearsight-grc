BEGIN;
DO $$ BEGIN IF EXISTS (SELECT 1 FROM form_question_binding_approvals) THEN RAISE EXCEPTION 'Form question binding approvals must be retained'; END IF; END $$;
DROP TABLE form_question_binding_approvals;
DROP FUNCTION prevent_form_question_binding_approval_mutation();
COMMIT;
