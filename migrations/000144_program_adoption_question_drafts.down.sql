BEGIN;
DO $$ BEGIN IF EXISTS (SELECT 1 FROM program_adoption_question_draft_revisions) THEN RAISE EXCEPTION 'Adoption question draft revisions must be retained'; END IF; END $$;
DROP TABLE program_adoption_question_draft_revisions;
DROP FUNCTION prevent_adoption_question_draft_mutation();
COMMIT;
