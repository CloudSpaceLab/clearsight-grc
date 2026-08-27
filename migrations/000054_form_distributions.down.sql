BEGIN;

DROP INDEX IF EXISTS capture_distribution_outbox_uq;
DROP TABLE IF EXISTS capture_distribution_events;
DROP TABLE IF EXISTS capture_response_revisions;
DROP TABLE IF EXISTS capture_response_workspace_edits;
DROP TABLE IF EXISTS capture_response_workspaces;
DROP TABLE IF EXISTS capture_otp_challenges;
DROP TABLE IF EXISTS capture_access_routes;
DROP TABLE IF EXISTS capture_distribution_recipients;

DROP INDEX IF EXISTS capture_submissions_distribution_idx;
ALTER TABLE capture_submissions
    DROP CONSTRAINT IF EXISTS capture_submissions_distribution_tenant_fk,
    DROP CONSTRAINT IF EXISTS capture_submissions_id_tenant_key,
    DROP COLUMN IF EXISTS distribution_id;

DROP INDEX IF EXISTS capture_requests_distribution_idx;
ALTER TABLE capture_requests
    DROP CONSTRAINT IF EXISTS capture_requests_distribution_scope_fk,
    DROP COLUMN IF EXISTS distribution_id;

DROP TABLE IF EXISTS capture_form_distributions;

COMMIT;
