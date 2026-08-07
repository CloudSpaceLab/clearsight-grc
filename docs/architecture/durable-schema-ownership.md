# Durable schema ownership

**Status:** P2 executable ownership register  
**Scope:** live PostgreSQL tables after applying the ordered migration chain  
**Executable authority:** code and migrations remain authoritative; this register is mechanically checked against the live migration result.

A durable table must have one owner and one maturity classification. A table appearing in a migration is not, by itself, evidence that a product capability exists.

## Classification

The live register uses the issue #33 categories verbatim:

- **active authoritative state** — governed current state or append-only domain/history state used to make or reconstruct product decisions;
- **active projection** — derived state that can be reconstructed from authoritative inputs;
- **active infrastructure ledger** — delivery, scheduling, retry, idempotency or operational bookkeeping that supports execution but is not business truth.

No live table is currently classified as reserved, deprecated/migration-only, or removable after migration `000019_schema_ownership_cleanup`.

<!-- schema-ownership:begin -->
| Table | Classification | Owner | Writers | Readers | Lifecycle / valid time | Retention / deletion | Executable evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `tenants` | active authoritative state | identity / tenancy | provisioning and reference installers | identity, authority, all tenant-scoped repositories | tenant lifetime | retain while any tenant data is referenced | identity middleware; PostgreSQL integration suites |
| `legal_entities` | active authoritative state | identity / tenancy | provisioning and reference installers | identity context, authority resolution | `valid_from` / `valid_until` | retain governed history while referenced | identity and authority PostgreSQL paths |
| `principals` | active authoritative state | identity / authority | provisioning and reference installers | identity, authority, governance, capture, workflow | status plus `valid_from` / `valid_until` | retain identity history while referenced | identity authenticator; authority integration tests |
| `responsibility_assignments` | active authoritative state | authority | governed authority configuration and installers | effective authority resolver | `valid_from` / `valid_until`, priority and scope | retain superseded assignments for governed traceability | `internal/authority/postgres.go`; authority integration tests |
| `authority_grants` | active authoritative state | authority | governed authority configuration and installers | effective authority resolver | `valid_from` / `valid_until`, decision limits | retain superseded grants for governed traceability | `internal/authority/postgres.go`; authority integration tests |
| `workflow_instances` | active projection | workflow | Matter Action projector | workflow task projection and runtime reads | one projection workflow per accountable subject where defined | delete/rebuild only with source-domain reconciliation | `internal/workflow/matter_action_projector_postgres.go` |
| `outbox_events` | active infrastructure ledger | runtime delivery | transactional domain/configuration writers | publisher and bounded worker classes | pending → published or failed/retried | retain per delivery/audit policy; never business truth | `internal/runtime/postgres.go`; worker integration tests |
| `org_positions` | active authoritative state | authority | governed organisation configuration and installers | authority resolver | `valid_from` / `valid_until` | retain governed organisation history | `internal/authority/postgres.go` |
| `role_templates` | active authoritative state | authority | governed organisation configuration and installers | authority resolver and actor-role projection | `valid_from` / `valid_until` | retain governed role history | authority resolver and actor-context tests |
| `position_role_bindings` | active authoritative state | authority | governed organisation configuration and installers | authority resolver | `valid_from` / `valid_until` | retain governed binding history | `internal/authority/postgres.go` |
| `delegations` | active authoritative state | governance / authority | governance service | authority resolver and governance reads | draft/active/revoked/expired plus start/end | retain decision history; do not hard-delete active traceability | governance and authority integration tests |
| `routing_policies` | active authoritative state | governance | governance service | governance reads and authority route projection | draft/active/retired with current version | retain policy lineage | `internal/governance/postgres.go` |
| `routing_policy_versions` | active authoritative state | governance | governance service | effective authority route projection | immutable version lineage with effective interval | retain all approved/rejected policy versions | governance service tests; migration 000014 triggers |
| `workflow_tasks` | active projection | workflow | Matter Action projector | Today and workflow reads | actor-facing status derived from accountable work | rebuild/reconcile from owning domain work; terminal history retained as required | workflow projector integration tests; Today tests |
| `workflow_events` | active projection | workflow | workflow projector/service | workflow history and diagnostics | append-only projection lifecycle events | retain with projected workflow history | `internal/workflow/postgres.go` |
| `user_onboarding_state` | active authoritative state | onboarding | onboarding service | onboarding and actor guide reads | per principal/guide version; completed/dismissed state | retain current user guidance state; replace by guide version | onboarding PostgreSQL repository/tests |
| `compliance_signals` | active authoritative state | autonomy | autonomy service / signal ingestion | drift assessment and diagnostics | immutable observation/effective timestamps; deduplicated | retain governed observation history | `internal/autonomy/postgres.go`; recovery tests |
| `drift_assessments` | active authoritative state | autonomy | autonomy assessment service | Readiness and Today/autonomy reads | active/resolved/superseded | retain assessment history | `internal/autonomy/postgres.go`; service tests |
| `automation_policies` | active authoritative state | autonomy configuration | governed configuration/installers | autonomy policy reads and UI configuration | versioned status/effective interval | retain policy lineage; policy is eligibility configuration, not execution evidence | autonomy repository; Automation Policies rendered tests |
| `governance_decisions` | active authoritative state | governance | governance transition service | governance history/decision reads | append-only decision record | retain governed approval/rejection history | `internal/governance/postgres.go`; integration tests |
| `segregation_rules` | active authoritative state | authority / governance | governed authority configuration/installers | effective authority resolver | active status and scope | retain superseded rules for traceability | `internal/authority/postgres.go`; authority integration tests |
| `workflow_timers` | active infrastructure ledger | runtime scheduler | runtime timer scheduler | timer claimer/firer and queue health | scheduled → claimed/fired with lease semantics | retain according to worker operational policy | `internal/runtime/background_jobs_postgres.go` |
| `inbox_receipts` | active infrastructure ledger | runtime delivery | idempotent consumers/projectors | duplicate-delivery guards | one receipt per consumer/event | retain through replay/deduplication horizon | runtime inbox and workflow projector integration tests |
| `evidence_sources` | active authoritative state | evidence | evidence service/installers | evidence health, continuity source-quality derivation | active/inactive source configuration | retain source history while assessments depend on it | evidence PostgreSQL repository; continuity source tests |
| `source_observations` | active authoritative state | evidence | evidence observation service | source health and reconciliation | append-only observed/valid timestamps | retain governed source-health history | evidence and reconciliation PostgreSQL integration tests |
| `capture_requests` | active authoritative state | evidence capture | evidence service | internal/external capture reads | draft/ready/in-progress/submitted/expired/cancelled with version | retain request/submission traceability | evidence PostgreSQL integration tests |
| `capture_submissions` | active authoritative state | evidence capture | evidence service | evidence/capture reads and audit reconstruction | append-only submission receipt | retain governed submission history | evidence capture integration tests |
| `capture_invitations` | active authoritative state | evidence capture | evidence invitation service | redemption/revocation | issued/redeemed/revoked/expired | retain bounded invitation traceability | evidence invitation/session tests |
| `capture_sessions` | active authoritative state | evidence capture | evidence session service | bounded external capture authorization | active/revoked/expired with `last_used_at` | retain security/session traceability per policy | evidence session-guard/revocation tests |
| `capture_artifacts` | active authoritative state | evidence capture | artifact upload/inspection service | capture and evidence reads | pending inspection → available/rejected | retain according to evidence retention policy and legal hold | evidence artifact integration tests |
| `programs` | active authoritative state | continuity | continuity command service | current Program reads, state projection, summaries | draft/active/paused/ended with effective interval/version | retain governed Program history | continuity PostgreSQL and current-read tests |
| `program_requirements` | active authoritative state | continuity | continuity commands | Program state derivation/current reads | `effective_from` / `effective_until` | retain superseded requirement history | P1 state-truth tests |
| `program_applicability` | active authoritative state | continuity | continuity commands | Program state derivation/current reads | `valid_from` / `valid_until` | retain applicability history | P1 state-truth tests |
| `control_objectives` | active authoritative state | continuity | continuity commands | Program/current reads | Program-scoped lifecycle | retain while Program/history references it | continuity repository tests |
| `control_implementations` | active authoritative state | continuity | continuity commands | Program state derivation/current reads | `valid_from` / `valid_until` | retain implementation history | P1 state-truth tests |
| `requirement_control_links` | active authoritative state | continuity | continuity commands | Program state/current reads | link lifetime inside Program | retain governed mapping history | continuity PostgreSQL tests |
| `evidence_contracts` | active authoritative state | continuity | continuity commands | Program state and evidence assessment | active flag, target and freshness contract | retain contract lineage | P1 evidence/state tests |
| `evidence_contract_sources` | active authoritative state | continuity | continuity commands | Program source-quality derivation | contract/source dependency lifetime | retain dependency history with contracts | source-dependency integration tests |
| `evidence_assessments` | active authoritative state | continuity | continuity verification/evidence commands | Program state derivation/current reads | assessed/valid-until interval | retain assessment history | P1 validity and source-state tests |
| `program_state_snapshots` | active projection | continuity projection | projection worker/reconciler | Program detail/summaries/readiness UI | assessed Program version plus monotonic projection version | rebuildable from normalized/current and history inputs | projection integration and stale-state rendered tests |
| `program_trigger_events` | active authoritative state | continuity | continuity trigger commands | Program history/reconstruction | append-only trigger events | retain governed trigger history | continuity event/replay tests |
| `matters` | active authoritative state | continuity | continuity command service | Matter current reads, closure and summaries | typed lifecycle/state/version | retain governed Matter history | P1 closure/current-read integration tests |
| `matter_links` | active authoritative state | continuity | continuity commands | Matter/Program relationship reads | link lifetime | retain governed relationship history | continuity PostgreSQL tests |
| `matter_decisions` | active authoritative state | continuity | decision lifecycle service | Matter closure/current reads | append/current lineage by event order and lifecycle state | retain complete decision lineage | P1.2/P1.3 tests |
| `matter_actions` | active authoritative state | continuity | Matter action commands | Matter closure and workflow projector | planned/implemented/cancelled plus accountable owner/version | retain action history; source of truth for projected Tasks | P1.4 projector/current-read tests |
| `verification_contracts` | active authoritative state | continuity | continuity commands | verification recording and Matter closure | active contract, observation period, authority | retain contract history | P1.2 verification tests |
| `verification_results` | active authoritative state | continuity | verification service | Matter closure/current reads | append-only result chronology | retain independent outcome evidence | P1.2 verification tests |
| `response_packages` | active authoritative state | continuity | response lifecycle service | Matter closure/current reads | proposal/review/approval/transmission/acknowledgement lineage | retain complete response history | P1.2/P1.3 response tests |
| `continuity_events` | active authoritative state | continuity | continuity transactional repository | replay, history, reconciliation and projectors | append-only aggregate event order | retain governed reconstruction history | replay/reconciliation/PostgreSQL integration tests |
| `continuity_projection_jobs` | active infrastructure ledger | continuity projection operations | projection reconciler/rebuild commands | projection workers and health endpoints | pending/claimed/completed/failed retry lifecycle | retain operational history per projection policy | command-integrity/projection integration tests |
| `document_imports` | active authoritative state | document import | document-import service/worker/reviewer | Imports API/UI and processing worker | pending → terminal extraction/analysis → versioned human review | retain original artifact reference and governed review history | P1.5 restart/hostile-file/PostgreSQL tests |
| `effective_authority_routes` | active projection | authority / governance | database projection function triggered by routing policy/version changes | authority resolver/integrity checks | current effective route set with valid interval | rebuild from active routing policy versions | migration 000014; authority integration tests |
<!-- schema-ownership:end -->

## Historical dispositions

These tables are intentionally absent from the live register:

| Historical table | Final classification | Disposition |
| --- | --- | --- |
| `evidence_requests` | deprecated/migration-only | Foundation-era duplicate removed by migration `000013_capture_consolidation`; `capture_requests` is the executable owner. |
| `invitation_grants` | deprecated/migration-only | Foundation-era duplicate removed by migration `000013_capture_consolidation`; `capture_invitations` is the executable owner. |
| `audit_events` | removable | Generic unused ledger removed by migration `000019_schema_ownership_cleanup`; domain events, governance decisions, workflow events and delivery ledgers keep their narrower explicit owners. Migration refuses removal if unexpected data exists. |
| `readiness_snapshots` | removable | Unused snapshot table removed by migration `000019_schema_ownership_cleanup`; current Readiness is derived from active drift assessments and explicitly has no known population baseline yet. Migration refuses removal if unexpected data exists. |

## API/schema ownership boundary

`internal/httpapi/route_registry.go` is the executable route inventory. `api/runtime.openapi.json` is the mechanically verified route/access/permission projection generated from that inventory. It is the only repository API artifact that may be used as executable authorization or route-existence truth.

Bounded domain specifications such as `api/bank-journeys.openapi.yaml` and `api/document-imports.openapi.yaml` describe domain payloads and examples only. They do not create routes and do not override runtime access policy.

The former broad manually maintained `api/openapi.yaml` duplicated the executable route catalogue and had drifted far enough to advertise removed Workflow Task mutations and retired capture aliases. P2 removes that duplicate instead of maintaining two route truths.

## Change rule

Any migration that changes the live table set must update the machine-checked register in the same change. CI reconstructs the live table set from ordered `*.up.sql` migrations, applies `CREATE TABLE` / `DROP TABLE` changes in migration order, and requires an exact one-to-one match with the rows between the ownership markers above.

Do not add a generic runtime metadata catalogue for this. The migration chain plus this checked architecture register is the smaller ownership mechanism.
