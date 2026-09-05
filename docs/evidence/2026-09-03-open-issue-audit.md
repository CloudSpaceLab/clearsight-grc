# Open issue audit — 3 September 2026

Baseline: `bbe7397423d69894c8fa6a6f063477bf0ffd7795` (`origin/main`).

All 14 open issues were reviewed against their acceptance criteria, current code, merged PRs and recorded hosted evidence. None was wholly ready for closure at this baseline. An unchecked historical implementation list does not mean the implementation is absent; a merged implementation does not prove the complete hosted outcome.

## Disposition

| Issue | What is already present | What still prevents closure |
| --- | --- | --- |
| [#137](https://github.com/CloudSpaceLab/clearsight-grc/issues/137) | Repaired external workspace and distribution command contracts (#133–136). | PascalCase client alternatives, scalar answer decoding, retired draft facade and query-token discovery remain. Complete the canonical-contract inventory and rejection gates. |
| [#138](https://github.com/CloudSpaceLab/clearsight-grc/issues/138) | Customer entry cannot import the evidence transport; `DemoTasks()` is removed and tasks derive from Matter aggregates. | Full persisted identity/authority/vendor/distribution/response/policy installation and restart/role-switch acceptance; reconcile #129. Historical claims that the customer bundle still uses the static transport are stale. |
| [#139](https://github.com/CloudSpaceLab/clearsight-grc/issues/139) | Governed activation/address verification and capture repairs (#150, #162–170). | Real recipient registration, staff evidence, independent sign-off, outcome verification, closure, activation and certification-refresh journey remains unproven. Delivery alone is insufficient. |
| [#140](https://github.com/CloudSpaceLab/clearsight-grc/issues/140) | Immutable linked-form remediation and exact response handling (#150, #157, #165, #170); hosted request and binding exist. | Recorded acceptance still awaits real response submission, independent application, deterministic verification and authorized closure. |
| [#141](https://github.com/CloudSpaceLab/clearsight-grc/issues/141) | Position/reporting inventory, narrow manager-handoff resolver and assignment notification infrastructure. | Governed editable hierarchy, versioned preview/approval/activation/rollback, complete handoff adversarial tests and rendered/hosted acceptance. Existing manager traversal can accept a candidate before a cycle or invalid-chain truncation; a prerequisite correction is in progress. |
| [#142](https://github.com/CloudSpaceLab/clearsight-grc/issues/142) | Multi-level OVERDUE runtime, current-authority candidate restrictions and independently approved role/group guard revisions. | Full sequence authoring, representative recipient/active-work simulation, effective-dated rollback and three-level recovery/hosted proof. Department preview is not full recipient simulation. |
| [#143](https://github.com/CloudSpaceLab/clearsight-grc/issues/143) | Advanced scoring, bounded response queries, form-policy runtime and historical integration evidence; #158 preserves profiles and makes replacement approval concurrency-safe. | Current-main extraction of unique #129 changes, approved-form selector and exact-head persisted good/borderline/poor response-to-policy-to-Matter-to-verified-closure acceptance. Historical branch evidence is not current-head proof. |
| [#147](https://github.com/CloudSpaceLab/clearsight-grc/issues/147) | Main CI, managed UI review and deployment have passed for the baseline. | Remaining V1 children and complete exact-head hosted business journeys. |
| [#128](https://github.com/CloudSpaceLab/clearsight-grc/issues/128) | #144, #145 and #146 have recorded closure evidence; #124 is closed. | #137–143 and #147 remain blockers. The parent checklist was corrected for the three closed children without closing the parent. |
| [#172](https://github.com/CloudSpaceLab/clearsight-grc/issues/172) | #173 adds baseline/overlay governance; #176 adds provider/model transport revisions, opaque secret references and atomic runtime application. | Stable proxy/capability UI, safe tests/simulation, bounded baseline exceptions, full tool governance, genuine emergency outbound freeze and complete enterprise acceptance. Suspending a revision is not proof that existing known-good routing stops. |
| [#74](https://github.com/CloudSpaceLab/clearsight-grc/issues/74) | Governed backend lifecycle and read-only workload/policy panel. | Workload registration/revision/rotation, policy authoring/simulation/promotion and bounded decision investigation UX. Gateway configuration in #172 does not complete these. |
| [#80](https://github.com/CloudSpaceLab/clearsight-grc/issues/80) | Vendor collection/review plus activation/address-verification implementation. | Contract/obligation metadata, broader reuse/import reconciliation, continuation/restriction/suspension/verified exit, assurance monitoring and production/bank-user proof. |
| [#57](https://github.com/CloudSpaceLab/clearsight-grc/issues/57) | Deterministic rule kernel, isolated source executor and shared source-access foundations (#58–70). | Full assurance population/rule/run/episode lifecycle, scheduled execution, intervention UX, CDC/KRI semantics and bank-scale proof. |
| [#13](https://github.com/CloudSpaceLab/clearsight-grc/issues/13) | Substantial identity, Forms, governance, gateway and activity/audit foundations. | Explicit pilot/GA scope and completed or expressly deferred security, storage, resilience, vertical and representative-user requirements. Keep as the readiness umbrella; do not restart completed foundations. |

## Traceable evidence

- Closed child evidence: [#144](https://github.com/CloudSpaceLab/clearsight-grc/issues/144#issuecomment-5516580716), [#145](https://github.com/CloudSpaceLab/clearsight-grc/issues/145#issuecomment-5517004072), [#146](https://github.com/CloudSpaceLab/clearsight-grc/issues/146#issuecomment-5516581617).
- Runtime boundary: `web/scripts/runtime-fixture-boundary.nodecheck.mjs`, `internal/runtimecontext`, `cmd/api/services_memory.go`, `cmd/seed-bank-reference/main.go`.
- Contract residuals: `web/src/formsDistributionApi.ts`, `web/src/captureInvitationBrowser.ts`, `internal/formcontract/model.go`, `internal/evidence/draft_compatibility.go`.
- Hierarchy/escalation: `internal/access/postgres.go`, `web/src/components/access/OrganizationInventory.tsx`, `internal/governance/escalation_revision.go`, `internal/httpapi/identity_access_handlers.go`, `internal/workflow/matter_escalation_postgres.go`.
- Remediation/activation: `internal/continuity/matter_form_remediation*_test.go`, `internal/thirdparty/activation_policy*`, and the explicit remaining hosted boundaries in #139/#140 comments.
- Gateway: `docs/acceptance/ai-gateway-control-plane.md`, `internal/aigateway/transport_runtime.go`, `internal/aigovernance/gateway_transport_service_test.go`, `web/src/components/configure/AIGatewayTransportControl.tsx`.
- Scoring: `docs/evidence/form-scoring-policy-release.md`, `internal/formpolicy`, `web/src/components/forms/FormPolicyEditor.tsx`; #129 remains draft and is not safe to merge wholesale.

The audit ran focused unit/architecture checks for runtime boundaries, reference installation, remediation, activation and gateway baseline/transport behavior. These passed. They were not represented as new hosted, real-mailbox or production-scale acceptance. No open issue was closed by this audit, and no business record was modified to improve displayed counts.

## Operator-requested pause

Implementation stopped on 3 September. The #141 safety prerequisite was committed locally as `c88c26004e1df57563a0d062f7978af69453431e` before the stop instruction arrived. Its targeted PostgreSQL/HTTP regressions pass, but independent review, full clean-database release gates, push, merge and deployment remain outstanding. The baseline findings above still describe main, not an already-released correction.

The [remaining-issue closure plan](../superpowers/plans/2026-09-03-remaining-issue-closure.md) records dependency order, exact files, test contracts, hosted acceptance and broader-roadmap boundaries. It is paused, not an automation or instruction to continue working unattended.
