# T5 receipts, response controls and approval acceptance

T5 closes the governance loop without turning ordinary AI traffic into another alert/workflow system.

## Compact decision receipts

`ai_gateway_decision_receipts` stores only bounded identifiers, exact policy revision, effective/proposed action, stable reason codes/obligations, selected model/route, outcome/error code and timestamps. It never stores prompt, response, source payload or credentials.

Receipt IDs are tenant-scoped and content-bound: an exact replay is idempotent; reusing an ID for different content conflicts. Material/non-ALLOW outcomes are retained; ordinary `ALLOW` traffic is deterministically sampled by runtime policy. Expired receipts are deleted in bounded batches by the existing shared worker runtime.

Repeated `REQUIRE_APPROVAL` receipts converge on one stable Signal/Drift episode key derived from tenant, workload, policy revision and reason set. Routine successful traffic creates no Signal, Matter or Today item.

## Response controls

Whole-response deny/redaction is deterministic and bounded by byte/pattern limits. A policy that needs whole-response inspection cannot claim transparent SSE streaming: contradictory streaming configuration is rejected, and governed streaming fails closed when whole-response controls are active. The gateway does not splice, buffer without bounds or silently skip inspection.

## Matter-backed execution grants

`ai_execution_grants` is the only new execution-authority record. A grant requires:

- an existing Matter;
- the exact current `AI_EXECUTION_GRANT` Decision in `APPROVED` or `CONDITIONALLY_APPROVED` state;
- the approving authority principal;
- an exact SHA-256 hash of canonical action arguments matching the Decision selected option;
- short expiry (15 minutes default, one hour maximum);
- a random bearer token whose plaintext is returned once and never persisted.

Consumption atomically binds tenant, workload, action hash and token digest and changes `ACTIVE` to `USED`. Changed arguments, expiry and replay all fail. Existing Matter/Workflow/Today projections remain the human approval surface; T5 does not add a parallel AI task model.

## Product surface

Configure exposes a compact AI governance view of active workloads, policy rollout and degraded state. Approval work remains in the existing Matter/Today experience and launches the exact governed action context; routine allowed traffic never becomes dashboard or Today noise.

## Automated evidence

- `internal/aigovernance` tests cover maker/checker, shadow-before-enforce, global credential uniqueness, receipt idempotency/episode dedupe, retention, exact-action grants and replay/argument tampering.
- `internal/aigateway` tests cover deterministic fact precedence, mutation, response deny/redaction and streaming restrictions.
- `internal/aigovernance/postgres_integration_test.go` proves policy/workload lifecycle, compact receipt idempotency and single-use grants against the real PostgreSQL schema.
- web rendered/accessibility tests cover compact and unavailable states.
- CI applies and rollback/reapplies migrations `000035_ai_governance_enforcement` and `000036_ai_governance_receipts_grants`, then runs serialized PostgreSQL integration, race-enabled unit tests, vet, web typecheck/tests/build.
