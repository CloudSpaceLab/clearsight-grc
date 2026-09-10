BEGIN;

-- Preserve every prior inspection receipt while allowing an explicitly
-- requeued artifact to start a new bounded scan cycle.
ALTER TABLE capture_artifact_scan_jobs
    ADD COLUMN recovery_cycle integer NOT NULL DEFAULT 0 CHECK (recovery_cycle >= 0);

ALTER TABLE capture_artifact_scan_receipts
    ADD COLUMN recovery_cycle integer NOT NULL DEFAULT 0 CHECK (recovery_cycle >= 0);

ALTER TABLE capture_artifact_scan_receipts
    DROP CONSTRAINT IF EXISTS capture_artifact_scan_receipts_artifact_id_attempt_key,
    ADD CONSTRAINT capture_artifact_scan_receipts_cycle_attempt_key UNIQUE (artifact_id, recovery_cycle, attempt);

COMMIT;
