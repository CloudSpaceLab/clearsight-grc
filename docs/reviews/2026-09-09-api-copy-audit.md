# API copy and reviewer identity audit

Reviewed 9 September 2026 against the local working tree on `codex/vendor-evidence-reconciliation`, including the earlier uncommitted reconciliation work. Audit only; no production changes in this audit.

## Scope and evidence

The [reproducible inventory](../evidence/2026-09-09-ui-audit/inventory-api-copy.py) scans 86 non-test Go files in `internal/httpapi`: 719 `httpx.WriteError` call sites, 582 with a literal message and 134 forwarding `err.Error()`. These are source-site counts, **not defect counts**. Dynamic errors, notifications, data-provided text and backend packages outside HTTP handlers need producer tracing rather than simple literal scanning. The inventory does not claim every dynamic message was exercised.

Customer visibility is verified by `web/src/http.ts:48`: the API message becomes an `ApiError.message`. For example, `components/forms/ResponseAssessment.tsx:42,65,92` retains and renders it. `components/DocumentImportWorkspace.tsx:257` also renders caught error text. A frontend wording correction alone therefore cannot resolve this class of defect.

## Findings

### API-01 · P1 · Shared errors assume a banking customer

Confirmed shared review errors in `internal/httpapi/response_assessment.go:53,74,82` say “bank assessment”. Authorization errors say “signed-in bank scope” in `command_guard.go:236,260`, `handlers.go:106`, `route_registry.go:448,506`; `continuity_handlers.go:928` says a record was not found in “this bank scope”. These are general runtime routes, not bank reference content.

Corrections:

| Current | Proposed |
| --- | --- |
| Check the bank assessment entries and try again. | Check the assessment entries and try again. |
| This response or bank assessment has changed. Reload it before recording a decision. | This response or assessment has changed. Reload before saving. |
| The bank assessment could not be loaded or saved. Try again. | Assessment unavailable. Try again. |
| This request is outside your signed-in bank scope. | This request is outside your organization. |
| This upload is outside your signed-in bank scope. | This upload is outside your organization. |
| The requested program or issue was not found in this bank scope. | Program or issue not found in your organization. |

Keep the organization boundary, verified identity, legal-entity restrictions and fail-closed behavior unchanged. If the UI provides a recovery action, it must reflect the actual supported sign-in or organization-switch flow; do not promise a switch that does not exist.

### API-02 · P2 · Recovery errors expose implementation concepts

| Source | Current problem | Proposed correction |
| --- | --- | --- |
| `document_handoff_handlers.go:248` | “deterministic conversion identity ... different canonical content” | “This proposal conflicts with an existing import. Reload the import and review the proposal.” Preserve conflict diagnostics in protected logs. |
| `document_handoff_handlers.go:254` | “canonical Program object” | “The Program could not be created. No approval was recorded.” |
| `form_score_preview.go:31` | “exact form revision ... 500 bounded answers” combines different failures | Return the actual condition: “Choose a form revision”, “Use 500 answers or fewer”, or a specific answer-length limit. Do not replace all branches with an inaccurate answer-count error. |
| `oversight_handlers.go:19,24,28` | Requires an executive to understand a projection worker/cycle | “Oversight unavailable. Try again.” For the not-yet-calculated condition: “Oversight has not been calculated for this legal entity.” Offer a real retry action; retain operational diagnostics for support. |
| `evidence_handlers.go:427` | `tenant_id and request_id are required.` | If exposed by the upload workflow: “Upload unavailable. Reopen the evidence request and try again.” This branch was source-audited, not reproduced through a normal browser upload. |

Keep consequence text when it prevents unsafe repetition. For example, “No approval was recorded” adds useful information and should remain. Do not use “Try again” for a non-retryable conflict without the necessary reload/review instruction.

### API-03 · P2 · Validation producers bypass the frontend copy gate

`internal/formcontract/assessment.go:52,65` emits “bank review” in field validation. `internal/httpapi/forms_handlers.go:429` forwards invalid-form errors. The frontend copy regression does not inspect these sources. Likewise, 134 HTTP call sites forward raw errors; this is a coverage risk, not evidence that all 134 messages are defective.

Use reviewed domain validation messages and code-specific recovery text. Keep technical JSON/identifier diagnostics in API documentation or specialist diagnostics when appropriate. Add focused regression cases for domain validation, authorization, conflict and unavailable messages actually displayed by each workflow. Avoid a blanket prohibition on internal enum names or banking reference fixtures.

### API-04 · P1 · A pending reviewer name cannot be inferred from the current assessment DTO

`web/src/formAssessmentApi.ts:6–14` includes historical decision `reviewer_id`, review permissions, and assessment state. It does not provide a current assigned reviewer display name. A field's `reviewer_role` is a routing input, not proof that a particular person is assigned. Other contracts have display-name fields (`web/src/types.ts:69–70`), but their existence does not make them authoritative for a pending field review.

The compact status should be **Awaiting review**. In the detailed context, use **Awaiting review from {full name}** only when the current, scope-filtered authority route provides that person. Use a role/group label when routing genuinely resolves to a group. Where responsibility is unresolved, show that condition separately; do not invent a name or fall back to the vendor owner. Historical decisions must retain their original reviewer, even if the assignment later changes.

This is a contract requirement for the requested named label, not a recommendation to add verbose names to every badge. Do not query broad user populations in the browser to resolve IDs.

### API-05 · P2 · Server-provided guides repeat the banking assumption

The default role guides in `internal/onboarding/service.go:184,205,227` say “held by the bank” and “known bank records”. These are shared guide definitions for reviewers, respondents and vendor owners. They bypass a frontend source scan.

Use “Check existing evidence before requesting a response”, “Review the purpose, deadline and existing evidence”, and “Use existing records, then request missing information”. Keep the guide optional and resumable. The current action buttons and scope-specific guidance remain useful; do not remove an actionable guide merely to reduce string count. This producer was source-reviewed; role-by-role browser guide execution was not repeated in this audit.

### API-06 · P3 · Notification copy and template previews need the same concise vocabulary

`internal/evidence/operational_notification_render.go:49–51` repeats “What needs to happen next” in plain-text and HTML notifications. Use **Next action** plus the actual work title, followed by the useful action/authority limitation where required. Preserve the subject, recipient, deadline and protected link handling. The internal `BankName` field is rendered as an actual supplied brand name; the field name itself is not customer-visible sector leakage.

`internal/evidence/communication_render.go:168` supplies `[Sample bank]` in a universal template preview; use `[Sample organization]`. The `bank_name` template placeholder is a persisted compatibility contract. A neutral display label and a versioned/compatible alias are safer than breaking saved templates. User-authored communication text still requires review in its approved template context. No messages were sent or real invitation tokens rendered during this audit.

## Wording boundaries

- Shared UI: assessment, review, organization, reviewer, vendor, approval.
- Genuine banking content: bank organization names, bank-specific reference journeys, policy titles and applicable legal/source text may remain specific. For example, `bank_vertical_handlers.go:14,24` belongs to an explicitly banking reference route.
- Stable API values such as `BANK_VALIDATED` are compatibility concerns, not visible label choices. Map them to clear UI copy without casually renaming the protocol.
- User-authored policy content is not a target for silent rewriting.

## Verification and limitations

Ran `vitest run src/copyQuality.test.ts` with the bundled Node 24 runtime: **1 test passed**. This confirms the current guardrail gap; it does not certify the copy. Reproduced API error rendering by source tracing, not a full authenticated failure-injection run. No server behavior, authority rule or customer data was changed.
