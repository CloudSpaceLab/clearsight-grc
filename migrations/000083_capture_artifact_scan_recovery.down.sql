BEGIN;

ALTER TABLE capture_artifact_scan_receipts
    DROP CONSTRAINT IF EXISTS capture_artifact_scan_receipts_cycle_attempt_key,
    ADD CONSTRAINT capture_artifact_scan_receipts_artifact_id_attempt_key UNIQUE (artifact_id, attempt);

ALTER TABLE capture_artifact_scan_receipts DROP COLUMN recovery_cycle;
ALTER TABLE capture_artifact_scan_jobs DROP COLUMN recovery_cycle;

COMMIT;
