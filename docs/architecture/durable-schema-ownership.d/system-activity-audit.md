# System activity and governed audit-export schema ownership

This fragment extends the executable durable-schema ownership register for issue #171. System activity remains a normalized read over canonical domain/runtime records; this table owns only export request/result metadata and never duplicates business-event payloads.

The Activity/Audit read model federates existing owned sources instead of creating another event warehouse: safe `outbox_events` metadata for committed domain activity, selected Identity & Access administration history from `governance_decisions`, and governed retry decisions from `operational_recovery_events`. Those sources retain their existing owners and lifecycle contracts; this fragment does not reclassify them.

Terminal worker/delivery failure history is deliberately not inferred from mutable `failed_at`, `dead_lettered_at`, or `last_error` state. A later #171 tranche must add or reuse an immutable terminal-failure receipt before such failures are represented as historical audit events. Purpose-bound `AUDIT_READ`/protected-object visibility and asynchronous large exports also remain outside this merged tranche.

<!-- schema-ownership:begin -->
| Table | Classification | Owner | Writers | Readers | Lifecycle / valid time | Retention / deletion | Executable evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `audit_export_receipts` | active infrastructure ledger | system activity / governed audit export | audit export service transaction | authorized audit-export status/download handlers and operational audit reconstruction | generating → ready/failed with immutable requester, exact filter/as-of boundary, checksums and expiry | retain through the configured audit-export evidence horizon; downgrade refuses to discard retained receipts; generated object bytes expire independently and contain only normalized safe activity fields | `internal/activity/export.go`; `internal/activity/export_postgres.go`; migration `000072_system_activity_audit_exports`; export service/PostgreSQL tests |
<!-- schema-ownership:end -->
