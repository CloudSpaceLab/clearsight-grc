BEGIN;
DO $$ BEGIN IF EXISTS (SELECT 1 FROM api_question_draft_revisions) THEN RAISE EXCEPTION 'API question draft revisions must be retained'; END IF; END $$;
DROP TABLE api_question_draft_revisions;
DROP FUNCTION prevent_api_question_draft_mutation();
COMMIT;
