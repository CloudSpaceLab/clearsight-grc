BEGIN;
DROP TABLE capture_artifact_scan_receipts;
DROP TABLE capture_artifact_scan_jobs;
ALTER TABLE capture_artifacts DROP CONSTRAINT capture_artifacts_id_tenant_key;
COMMIT;
