BEGIN;

ALTER TABLE capture_artifacts ADD CONSTRAINT capture_artifacts_id_tenant_key UNIQUE(id,tenant_id);

CREATE TABLE capture_artifact_scan_jobs (
 artifact_id uuid PRIMARY KEY,
 tenant_id uuid NOT NULL,
 sha256 text NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
 size_bytes bigint NOT NULL CHECK (size_bytes >= 0),
 state text NOT NULL DEFAULT 'PENDING' CHECK (state IN ('PENDING','RUNNING','COMPLETED','FAILED')),
 attempt integer NOT NULL DEFAULT 0 CHECK (attempt BETWEEN 0 AND 5),
 worker_id text NOT NULL DEFAULT '',
 lease_until timestamptz,
 next_attempt_at timestamptz NOT NULL,
 failure_code text NOT NULL DEFAULT '',
 FOREIGN KEY (artifact_id,tenant_id) REFERENCES capture_artifacts(id,tenant_id),
 CHECK ((state='RUNNING') = (lease_until IS NOT NULL AND worker_id<>''))
);
CREATE INDEX capture_artifact_scan_due_idx ON capture_artifact_scan_jobs(next_attempt_at,artifact_id) WHERE state='PENDING';
CREATE INDEX capture_artifact_scan_lease_idx ON capture_artifact_scan_jobs(lease_until,artifact_id) WHERE state='RUNNING';
CREATE INDEX capture_artifact_scan_health_idx ON capture_artifact_scan_jobs(state,attempt,next_attempt_at) WHERE state<>'COMPLETED';

CREATE TABLE capture_artifact_scan_receipts (
 id uuid PRIMARY KEY DEFAULT uuidv7(),
 artifact_id uuid NOT NULL,
 tenant_id uuid NOT NULL,
 sha256 text NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
 size_bytes bigint NOT NULL CHECK (size_bytes >= 0),
 attempt integer NOT NULL CHECK (attempt BETWEEN 1 AND 5),
 verdict text NOT NULL CHECK (verdict IN ('CLEAN','INFECTED','UNAVAILABLE')),
 scanner text NOT NULL CHECK (length(scanner)<=100),
 scanner_version text NOT NULL CHECK (length(scanner_version)<=512),
 inspected_at timestamptz NOT NULL,
 failure_code text NOT NULL,
 UNIQUE(artifact_id,attempt),
 FOREIGN KEY (artifact_id,tenant_id) REFERENCES capture_artifacts(id,tenant_id),
 CHECK ((verdict IN ('CLEAN','INFECTED') AND scanner<>'' AND scanner_version<>'' AND failure_code='') OR
        (verdict='UNAVAILABLE' AND failure_code IN ('SCANNER_UNAVAILABLE','OBJECT_UNAVAILABLE','INTEGRITY_MISMATCH')))
);
CREATE INDEX capture_artifact_scan_receipt_history_idx ON capture_artifact_scan_receipts(tenant_id,artifact_id,inspected_at DESC);

-- Legacy pending uploads receive work, never an invented clean receipt.
INSERT INTO capture_artifact_scan_jobs(artifact_id,tenant_id,sha256,size_bytes,next_attempt_at)
SELECT id,tenant_id,sha256,size_bytes,created_at FROM capture_artifacts WHERE status='STORED_UNSCANNED';
COMMIT;
