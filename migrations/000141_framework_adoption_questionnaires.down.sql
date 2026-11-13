BEGIN;
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM framework_adoption_questionnaires LIMIT 1) THEN
  RAISE EXCEPTION 'Cannot remove existing framework questionnaire links';
 END IF;
END $$;
DROP TABLE framework_adoption_questionnaire_events;
DROP TABLE framework_adoption_questionnaires;
COMMIT;
