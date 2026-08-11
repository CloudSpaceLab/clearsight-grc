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
| `tenants` | active authoritative state | identity / tenancy | provisioning and reference installers | identity, authority, tenant-scoped repositories | tenant lifetime | retain while tenant data is referenced | identity middleware; PostgreSQL integration |
| `legal_entities` | active authoritative state | identity / tenancy | provisioning and reference installers | identity context, authority | `valid_from` / `valid_until` | retain governed history while referenced | identity and authority PostgreSQL paths |
| `principals` | active authoritative state | identity / authority | provisioning and reference installers | identity, authority, governance, capture, workflow | status plus valid interval | retain identity history while referenced | identity authenticator; authority integration |
| `principal_identities` | active authoritative state | identity / federation | trusted enterprise provisioning and identity administration | OIDC principal correlation and local access resolver | active/revoked mapping from immutable issuer + subject to local principal; legal-entity access is resolved separately from current organization state | retain correlation history while identity attribution is required | `internal/access/postgres.go`; migration `000025_oidc_session_access`; access integration |
| `scim_sources` | active authoritative state | identity / provisioning | governed identity configuration and bootstrap tooling | SCIM bearer authentication and provisioning repository | active/revoked tenant-scoped source configuration | retain source attribution while provisioned resources are referenced | `internal/scimapi/postgres.go`; migration `000026_scim_provisioning` |
| `scim_users` | active authoritative state | identity / provisioning | authenticated SCIM User lifecycle | SCIM reads, OIDC correlation and local access resolver | active/inactive source-owned principal projection with soft deletion | retain provisioning attribution and stable SCIM resource identity | `internal/scimapi/postgres.go`; SCIM PostgreSQL integration |
| `directory_groups` | active authoritative state | identity / provisioning | authenticated SCIM Group lifecycle | SCIM reads and local access resolver | current source group fact with soft deletion | retain group identity while mappings or reconstruction require it | `internal/scimapi/postgres.go`; local access integration |
| `directory_group_members` | active authoritative state | identity / provisioning | authenticated SCIM Group membership lifecycle | SCIM reads and local access resolver | direct current User membership only | remove or rebuild from source truth; no nested closure is stored | `internal/scimapi/postgres.go`; SCIM and access integration |
| `directory_group_role_bindings` | active authoritative state | authority configuration / access | governed ClearSight identity configuration only | local access resolver | effective-dated legal-entity and exact department role eligibility | retain governed mapping history; SCIM cannot write this table | `internal/access/postgres.go`; migration `000026_scim_provisioning`; access integration |
| `responsibility_assignments` | active authoritative state | authority | governed authority configuration and installers | effective authority resolver | valid interval, priority and scope | retain superseded assignments | `internal/authority/postgres.go`; authority integration |
| `authority_grants` | active authoritative state | authority | governed authority configuration and installers | effective authority resolver | valid interval and decision limits | retain superseded grants | `internal/authority/postgres.go`; authority integration |
| `workflow_instances` | active projection | workflow | Matter Action, Matter lifecycle and Evidence Request projectors | workflow projection and reads | one derived workflow per accountable subject where defined | rebuild only with source-domain reconciliation | `internal/workflow/matter_action_projector_postgres.go`; `internal/workflow/matter_lifecycle_projector_postgres.go`; `internal/workflow/evidence_request_projector_postgres.go` |
| `outbox_events` | active infrastructure ledger | runtime delivery | transactional domain/configuration writers | publisher and bounded workers | pending → published or retry/failure | retain per delivery/audit policy | `internal/runtime/postgres.go`; worker integration |
| `web_sessions` | active infrastructure ledger | identity / session | SCS session middleware through pgx store | SCS session middleware | opaque server-side session data until idle/absolute expiry or logout | expired rows are removed by bounded store cleanup; no authorization truth retained in browser | `internal/federation/oidc.go`; SCS pgx store; migration `000025_oidc_session_access` |
| `org_positions` | active authoritative state | authority | governed organisation configuration and installers | authority and local capability resolvers | valid interval plus current hierarchical `department_path` | retain governed organisation history | `internal/authority/postgres.go`; `internal/access/postgres.go` |
| `role_templates` | active authoritative state | authority | governed organisation configuration and installers | authority resolver and local capability resolver | valid interval plus bounded coarse capabilities | retain governed role history | authority resolver; `internal/access/postgres.go`; actor-context tests |
| `position_role_bindings` | active authoritative state | authority | governed organisation configuration and installers | authority and local capability resolvers | valid interval | retain governed binding history | `internal/authority/postgres.go`; `internal/access/postgres.go` |
| `delegations` | active authoritative state | governance / authority | governance service | authority resolver and governance reads | draft/active/revoked/expired plus interval | retain decision history | governance and authority integration |
| `routing_policies` | active authoritative state | governance | governance service | governance reads and authority route projection | draft/active/retired with current version | retain policy lineage | `internal/governance/postgres.go` |
| `routing_policy_versions` | active authoritative state | governance | governance service | effective authority route projection and escalation policy parser | immutable version lineage with effective interval | retain approved/rejected versions | governance tests; escalation validation; migration 000014 triggers |
| `workflow_tasks` | active projection | workflow | Matter Action, Matter lifecycle and Evidence Request projectors | Today and workflow reads | actor-facing status derived from accountable work | rebuild/reconcile from canonical source-domain truth | workflow projector integration; Today tests |
| `workflow_events` | active projection | workflow | Matter Action, Matter lifecycle and Evidence Request projectors | workflow projection history and diagnostics | append-only projection events | retain with projected workflow history | workflow projector integration |
| `user_onboarding_state` | active authoritative state | onboarding | onboarding service | onboarding and guide reads | per principal/guide version | retain current guidance state by guide version | onboarding PostgreSQL tests |
| `compliance_signals` | active authoritative state | autonomy | autonomy service / signal ingestion | drift assessment and diagnostics | immutable observation/effective timestamps; deduplicated | retain governed observation history | `internal/autonomy/postgres.go`; recovery tests |
| `drift_assessments` | active authoritative state | autonomy | autonomy assessment service | Readiness and autonomy reads | active/resolved/superseded | retain assessment history | `internal/autonomy/postgres.go`; service tests |
| `automation_policies` | active authoritative state | autonomy configuration | governed configuration/installers | autonomy policy reads and Configure UI | versioned status/effective interval | retain policy lineage; never execution evidence | autonomy repository; rendered tests |
| `governance_decisions` | active authoritative state | governance | governance transition service | governance history/decision reads | append-only decision record | retain governed decision history | `internal/governance/postgres.go`; integration |
| `segregation_rules` | active authoritative state | authority / governance | governed authority configuration/installers | effective authority resolver | active status and scope | retain superseded rules | `internal/authority/postgres.go`; authority integration |
| `workflow_timers` | active infrastructure ledger | runtime scheduler | runtime timer scheduler | timer claimer/firer and queue health | scheduled → claimed/fired with lease | retain per worker operational policy | `internal/runtime/background_jobs_postgres.go` |
| `inbox_receipts` | active infrastructure ledger | runtime delivery | idempotent consumers/projectors | duplicate-delivery guards | one receipt per consumer/event | retain through deduplication horizon | runtime inbox; projector integration |
| `evidence_sources` | active authoritative state | evidence | evidence service/installers | evidence health and continuity source-quality | active/inactive source configuration | retain while assessments depend on source | evidence repository; continuity source tests |
| `source_observations` | active authoritative state | evidence | evidence observation service | source health and reconciliation | append-only observed/valid timestamps | retain governed source-health history | evidence/reconciliation integration |
| `capture_requests` | active authoritative state | evidence capture | evidence service and recipient lifecycle service | internal/external capture reads and Evidence Request work projector | draft/ready/in-progress/submitted/expired/cancelled plus canonical recipient state/revision | retain request/submission/recipient traceability | evidence PostgreSQL integration; recipient lifecycle tests |
| `capture_recipient_history` | active authoritative state | evidence capture | recipient lifecycle service | recipient audit/reconstruction and diagnostics | append-only wrong-recipient/reassignment chronology | retain with request traceability | `internal/evidence/recipient_lifecycle_postgres.go`; recipient lifecycle integration |
| `capture_submissions` | active authoritative state | evidence capture | evidence service | evidence/capture reads and reconstruction | append-only submission receipt | retain governed submission history | evidence capture integration |
| `capture_invitations` | active authoritative state | evidence capture | evidence invitation service and recipient lifecycle revocation | redemption/revocation | issued/redeemed/revoked/expired | retain bounded invitation traceability | invitation/session and recipient lifecycle tests |
| `capture_sessions` | active authoritative state | evidence capture | evidence session service and recipient lifecycle revocation | bounded external capture authorization | active/revoked/expired with last use | retain security/session traceability | session-guard/revocation and recipient lifecycle tests |
| `capture_artifacts` | active authoritative state | evidence capture | artifact upload/inspection service | capture and evidence reads | pending inspection → available/rejected | retain per evidence policy/legal hold | artifact integration tests |
| `programs` | active authoritative state | continuity | continuity command service | current Program reads, projection, summaries | draft/active/paused/ended with effective interval/version | retain governed Program history | continuity/current-read integration |
| `program_requirements` | active authoritative state | continuity | continuity commands | Program state/current reads | `effective_from` / `effective_until` | retain superseded requirement history | P1 state-truth tests |
| `program_applicability` | active authoritative state | continuity | continuity commands | Program state/current reads | `valid_from` / `valid_until` | retain applicability history | P1 state-truth tests |
| `control_objectives` | active authoritative state | continuity | continuity commands | Program/current reads | Program-scoped lifecycle | retain while referenced by Program/history | continuity repository tests |
| `control_implementations` | active authoritative state | continuity | continuity commands | Program state/current reads | valid interval | retain implementation history | P1 state-truth tests |
| `requirement_control_links` | active authoritative state | continuity | continuity commands | Program state/current reads | link lifetime inside Program | retain governed mapping history | continuity PostgreSQL tests |
| `evidence_contracts` | active authoritative state | continuity | continuity commands | Program state and evidence assessment | active flag, target and freshness contract | retain contract lineage | P1 evidence/state tests |
| `evidence_contract_sources` | active authoritative state | continuity | continuity commands | Program source-quality derivation | contract/source dependency lifetime | retain dependency history with contracts | source-dependency integration |
| `evidence_assessments` | active authoritative state | continuity | continuity verification/evidence commands | Program state/current reads | assessed/valid-until interval | retain assessment history | P1 validity/source-state tests |
| `program_state_snapshots` | active projection | continuity projection | projection worker/reconciler | Program detail/summaries/readiness UI | assessed Program version plus projection version | rebuildable from owned inputs/history | projection and stale-state tests |
| `program_review_checkpoints` | active authoritative state | continuity review acknowledgement | continuity review service | actor-scoped Program review digest | append-only accepted Program/projection version baseline per principal | retain governed acknowledgement history | `internal/continuity/program_review_postgres.go`; Program review integration |
| `program_trigger_events` | active authoritative state | continuity | continuity trigger commands | Program history/reconstruction | append-only trigger events | retain governed trigger history | continuity event/replay tests |
| `matters` | active authoritative state | continuity | continuity command service | Matter current reads, closure, summaries | typed lifecycle/state/version | retain governed Matter history | P1 closure/current-read integration |
| `matter_links` | active authoritative state | continuity | continuity commands | Matter/Program relationship reads | link lifetime | retain governed relationship history | continuity PostgreSQL tests |
| `matter_decisions` | active authoritative state | continuity | decision lifecycle service | Matter closure/current reads | current lineage by event order and lifecycle state | retain complete decision lineage | P1.2/P1.3 tests |
| `matter_actions` | active authoritative state | continuity | Matter action commands | Matter closure and workflow projector | planned/in-progress/blocked/implemented/cancelled with owner/version | retain action history; source of Task truth | P1.4 projector/current-read tests |
| `verification_contracts` | active authoritative state | continuity | continuity commands | verification recording and Matter closure | active contract, observation period, authority | retain contract history | P1.2 verification tests |
| `verification_results` | active authoritative state | continuity | verification service | Matter closure/current reads | append-only result chronology | retain independent outcome evidence | P1.2 verification tests |
| `response_packages` | active authoritative state | continuity | response lifecycle service | Matter closure/current reads | proposal/review/approval/transmission/acknowledgement lineage | retain complete response history | P1.2/P1.3 response tests |
| `continuity_events` | active authoritative state | continuity | continuity transactional repository | replay, history, reconciliation, projectors | append-only aggregate event order | retain governed reconstruction history | replay/reconciliation integration |
| `continuity_projection_jobs` | active infrastructure ledger | continuity projection operations | projection reconciler/rebuild commands | projection workers and health endpoints | pending/claimed/completed/failed retry lifecycle | retain per projection operations policy | projection integration tests |
| `document_imports` | active authoritative state | document import | document-import service/worker/reviewer | Imports API/UI and processing worker | pending → extraction/analysis terminal → versioned human review | retain artifact reference and review history | P1.5 restart/hostile-file/PostgreSQL tests |
| `effective_authority_routes` | active projection | authority / governance | database projection function triggered by routing policy versions | authority resolver/integrity checks | current effective route set with valid interval | rebuild from active routing policy versions | migration 000014; authority integration |
<!-- schema-ownership:end -->

## Historical dispositions

These tables are intentionally absent from the live register:

| Historical table | Final classification | Disposition |
| --- | --- | --- |
| `evidence_requests` | deprecated/migration-only | Foundation-era duplicate removed by migration `000013_capture_consolidation`; `capture_requests` is the executable owner. |
| `invitation_grants` | deprecated/migration-only | Foundation-era duplicate removed by migration `000013_capture_consolidation`; `capture_invitations` is the executable owner. |
| `audit_events` | removable | Generic unused ledger removed by migration `000019_schema_ownership_cleanup`; narrower owned histories and delivery ledgers remain. Migration refuses removal if unexpected data exists. |
| `readiness_snapshots` | removable | Unused snapshot table removed by migration `000019_schema_ownership_cleanup`; current Readiness is derived from active drift assessments and does not claim a known population baseline. Migration refuses removal if unexpected data exists. |

## API/schema ownership boundary

`internal/httpapi/route_registry.go` is the executable route inventory. `api/runtime.openapi.json` is the mechanically verified route/access/permission projection generated from that inventory. It is the only repository API artifact that may be used as executable authorization or route-existence truth.

Bounded domain specifications such as `api/bank-journeys.openapi.yaml` and `api/document-imports.openapi.yaml` describe domain payloads and examples only. They do not create routes and do not override runtime access policy.

The former broad manually maintained `api/openapi.yaml` duplicated the executable route catalogue and had drifted far enough to advertise removed Workflow Task mutations and retired capture aliases. P2 removed that duplicate instead of maintaining two route truths.

## Change rule

Any migration that changes the live table set must update the machine-checked register in the same change. CI reconstructs the live table set from ordered `*.up.sql` migrations, applies `CREATE TABLE` / `DROP TABLE` changes in migration order, and requires an exact one-to-one match with the rows between the ownership markers above.

Do not add a generic runtime metadata catalogue for this. The migration chain plus this checked architecture register is the smaller ownership mechanism.