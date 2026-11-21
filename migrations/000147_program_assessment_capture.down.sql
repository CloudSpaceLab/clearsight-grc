BEGIN;
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM capture_submission_reviews) OR EXISTS(SELECT 1 FROM capture_response_drafts WHERE principal_id IS NOT NULL)
  OR EXISTS(SELECT 1 FROM capture_requests WHERE origin_type='FRAMEWORK_ASSESSMENT') THEN
  RAISE EXCEPTION 'Program assessment records must be retained; downgrade refused';
 END IF;
END $$;
DROP TRIGGER capture_adopted_assessment_created ON capture_requests;
DROP FUNCTION record_adopted_assessment_creation();
DROP TRIGGER capture_adopted_assessment_scope ON capture_requests;
DROP FUNCTION validate_adopted_assessment_request();
DROP FUNCTION framework_assessment_recipient_eligible(uuid,uuid,uuid,jsonb);
DROP TABLE capture_submission_reviews;
DROP INDEX capture_submission_review_scope_idx;
DROP INDEX capture_internal_draft_identity_idx;
ALTER TABLE capture_response_drafts DROP CONSTRAINT capture_draft_identity_check;
ALTER TABLE capture_response_drafts DROP CONSTRAINT capture_draft_principal_fk;
ALTER TABLE capture_response_drafts DROP COLUMN principal_id;
ALTER TABLE capture_response_drafts ALTER COLUMN session_id SET NOT NULL;
COMMIT;
