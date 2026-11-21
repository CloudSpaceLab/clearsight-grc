BEGIN;

ALTER TABLE capture_response_drafts DROP COLUMN correction_review_id;
ALTER TABLE capture_submission_reviews DROP COLUMN correction_due_at;
DROP INDEX capture_assessment_correction_parent_idx;
DROP INDEX capture_assessment_revision_idx;
ALTER TABLE capture_submissions DROP COLUMN correction_of, DROP COLUMN assessment_revision;

COMMIT;
