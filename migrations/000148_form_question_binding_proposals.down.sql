BEGIN;
DO $$ BEGIN IF EXISTS (SELECT 1 FROM form_question_binding_proposals) THEN RAISE EXCEPTION 'Form question binding proposals must be retained'; END IF; END $$;
DROP TABLE form_question_binding_proposals;
DROP FUNCTION prevent_form_question_binding_proposal_mutation();
COMMIT;
