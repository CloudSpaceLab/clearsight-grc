# AI gateway transport configuration ownership

<!-- schema-ownership:begin -->
| Table | Classification | Owner | Writers | Readers | Lifecycle / valid time | Retention / deletion | Executable evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `ai_gateway_config_revisions` | active authoritative state | AI governance transport configuration | governed AI gateway configuration lifecycle | Configure AI governance and tenant-scoped AI gateway runtime snapshot loader | immutable tenant/environment revisions with DRAFT → PENDING_APPROVAL → APPROVED → ACTIVE → SUSPENDED/RETIRED lifecycle and one ACTIVE revision per tenant/environment | retain complete configuration lineage while required for audit and operational reconstruction; provider secret values are never stored, only opaque references | `internal/aigovernance/gateway_transport_postgres.go`; `internal/aigateway/transport_runtime.go`; migration `000078_ai_governance_gateway_transport` |
<!-- schema-ownership:end -->
