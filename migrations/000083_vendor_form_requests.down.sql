BEGIN;
DROP INDEX IF EXISTS capture_vendor_workspace_progress_idx;
DROP INDEX IF EXISTS capture_vendor_form_requests_idx;
DROP TABLE IF EXISTS capture_distribution_creation_receipts;
COMMIT;
