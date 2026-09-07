# Capture artifact inspection schema ownership

<!-- schema-ownership:begin -->
| Table | Classification | Owner | Writers | Readers | Lifecycle / valid time | Retention / deletion | Executable evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `capture_artifact_scan_jobs` | active infrastructure ledger | Evidence / Capture | manifest transaction and leased inspection worker | bounded worker claims and runtime queue health | pending → running → completed/failed; five attempts; expired leases reclaim with a new attempt fence | retain with artifact; failed jobs remain visible; no automatic deletion or disposal authority | `internal/evidence/artifact_scan_postgres.go`; migration `000080_capture_artifact_inspection`; memory and PostgreSQL scan tests |
| `capture_artifact_scan_receipts` | active authoritative state | Evidence / Capture inspection | inspection completion transaction | exact artifact reconstruction and audit | append-only completed attempt tied to artifact digest, size, scanner version and time; unavailable receipts confer no availability | retain while artifact, evidence use or audit reconstruction requires it; legal hold and disposal orchestration remain separate work | `internal/evidence/artifact_scan_postgres.go`; migration `000080_capture_artifact_inspection`; transaction rollback/replay tests |
<!-- schema-ownership:end -->
