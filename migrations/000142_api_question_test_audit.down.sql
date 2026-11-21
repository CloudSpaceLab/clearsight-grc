BEGIN;
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM api_question_test_attempts LIMIT 1) THEN
  RAISE EXCEPTION 'API question test audit exists; rollback is not safe';
 END IF;
END $$;
DROP TABLE api_question_test_outcomes;
DROP TABLE api_question_test_attempts;
DROP FUNCTION prevent_api_question_test_audit_mutation();
COMMIT;
