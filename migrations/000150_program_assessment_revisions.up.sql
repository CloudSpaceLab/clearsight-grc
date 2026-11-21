BEGIN;

-- Earlier assessment submissions remain revision 1, including their existing receipts.
ALTER TABLE capture_submissions ADD COLUMN assessment_revision bigint NOT NULL DEFAULT 1;
ALTER TABLE capture_submissions ADD COLUMN correction_of uuid;
ALTER TABLE capture_submissions ADD CONSTRAINT capture_assessment_revision_check
 CHECK ((assessment_revision=1 AND correction_of IS NULL) OR (assessment_revision>1 AND correction_of IS NOT NULL));
ALTER TABLE capture_submissions ADD CONSTRAINT capture_assessment_correction_fk
 FOREIGN KEY(correction_of,tenant_id,request_id) REFERENCES capture_submissions(id,tenant_id,request_id);
CREATE UNIQUE INDEX capture_assessment_revision_idx
 ON capture_submissions(tenant_id,request_id,assessment_revision) WHERE assessment_revision>1;
CREATE UNIQUE INDEX capture_assessment_correction_parent_idx ON capture_submissions(correction_of) WHERE correction_of IS NOT NULL;

ALTER TABLE capture_submission_reviews ADD COLUMN correction_due_at timestamptz;
ALTER TABLE capture_submission_reviews ADD CONSTRAINT capture_assessment_correction_deadline_check
 CHECK (correction_due_at IS NULL OR (status='FOLLOW_UP_REQUIRED' AND correction_due_at>created_at));
ALTER TABLE capture_response_drafts ADD COLUMN correction_review_id uuid REFERENCES capture_submission_reviews(id);
ALTER TABLE capture_response_drafts ADD CONSTRAINT capture_assessment_draft_principal_check
 CHECK (correction_review_id IS NULL OR principal_id IS NOT NULL);

COMMIT;
