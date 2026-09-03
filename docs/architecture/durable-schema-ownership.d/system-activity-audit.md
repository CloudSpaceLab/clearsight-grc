# System activity and governed audit-export schema ownership

This fragment extends the executable durable-schema ownership register for issue #171. System activity remains a normalized read over canonical domain/runtime records; this table owns only export request/result metadata and never duplicates business-event payloads.

<!-- schema-ownership:begin -->
| Table | Classification | Owner | Writers | Readers | Lifecycle / valid time | Retention / deletion | Executable evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `audit_export_receipts` | active infrastructure ledger | system activity / governed audit export | audit export service transaction | authorized audit-export status/download handlers and operational audit reconstruction | generating → ready/failed with immutable requester, exact filter/as-of boundary, checksums and expiry | retain through the configured audit-export evidence horizon; downgrade refuses to discard retained receipts; generated object bytes expire independently and contain only normalized safe activity fields | `internal/activity/export.go`; `internal/activity/export_postgres.go`; migration `000072_system_activity_audit_exports`; export service/PostgreSQL tests |
<!-- schema-ownership:end -->
