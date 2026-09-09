# Frontend copy audit — 9 September 2026

## Decision

The shared interface does not yet meet the requested industry-neutral, concise working language. The most important problems are stronger than verbosity: missing data can read as a known clean state, submitted responses are described as completed work, and one history view can attribute an old result to today's reviewer. Reusable assessment and capture flows repeatedly hard-code “bank”. Configure contains substantial implementation and product-design narration.

This is a **read-only audit of the current working tree**, including the existing uncommitted vendor-evidence reconciliation changes. No production source, tests, fixtures, policy documents or existing work were changed. Suggested wording below is review material, not an approved implementation. Backend/API-originated copy is covered by the coordinating audit; this report identifies frontend consumers of that copy.

The user's latest instruction takes precedence over the bank-centric examples in AGENTS.md and existing specifications: shared controls and statuses use industry-neutral language; a real organization name or reviewer full name is shown only when supplied by the relevant runtime record or authority route. Banking laws, genuine banking policy text and named sample banking organizations retain their meaning. “Missing”, “Incomplete”, “Received” and “Accepted” are distinct states, not interchangeable short labels.

## Scope and method

Read AGENTS.md, README.md, DESIGN.md, the docs map, `docs/product/enterprise-copy-and-content-design.md` and `docs/product/plain-language-content-standard.md`. Reviewed shared screen composition and strings in Today, Programs, Work, Forms, Vendors, Configure, Imports, external capture, document review, guides and sample-entry boundaries. Searches covered visible JSX, text properties, label/helper maps, receipts, validation, accessible names, actor fallbacks and API error presentation. High-impact matches were followed into their surrounding condition and consumer rather than treated as prohibited words in isolation.

Reproducible inventory: `node docs/evidence/2026-09-09-ui-audit/copy-inventory.cjs` from the repository root. Supporting outputs are [frontend-copy-candidates.json](../evidence/2026-09-09-ui-audit/frontend-copy-candidates.json) and [frontend-copy-candidates.txt](../evidence/2026-09-09-ui-audit/frontend-copy-candidates.txt).

| Measure | Count | Meaning |
| --- | ---: | --- |
| Non-test TypeScript/TSX files scanned | 260 | Recursive `web/src`; excludes `.test.*`, declarations and `web/src/test` |
| Files with copy candidate lines | 229 | Heuristic, not proof every line is customer-facing |
| Candidate source lines | 4,231 | JSX text, common copy props, or literals with natural-language spacing |
| Distinct candidate line values | 4,034 | Source-line deduplication, **not unique rendered strings** |
| Candidate lines containing bank/banking | 116 in 38 files | Includes code references, runtime, starter content and sample fixtures; not 116 defects |
| Files reached by current copy-quality source glob | 180 | Root TS/TSX, immediate components, immediate `components/forms` |
| Files outside that glob | 80 | Nested component directories, including Configure/capture/access/documents/builder/sent/dashboard/filters/UI/Oversight |

The inventory is deliberately a discovery aid: one compressed JSX line may contain many sentences; dynamic text may have no static literal; identifiers and code can match. It is not an AST, translation catalogue, complete human sign-off of 4,231 lines, or a defect count. An initial attempt to use the locally installed TypeScript compiler API was unavailable under the installed TypeScript 7 package layout; the delivered script uses the documented heuristic and needs no dependencies. The scan and source review completed successfully.

Priority: **P1** can mislead a material decision, scope or attribution; **P2** materially increases ambiguity or violates the user's requested copy standard; **P3** is repetitive or inconsistent presentation with a clear meaning retained. Locations below are one-based in the audited working tree.

## Findings

### COPY-01 — P1: Unknown or stale information can read as current, clean or ready

`web/src/components/ProgramCurrentPosition.tsx:49` correctly computes stale state, but `:74` displays `current?.open_matter_count ?? 0`. `:77` displays “No status exceptions are recorded for the latest calculation” whenever reasons are empty, including when the current calculation is missing. A stale record with old reasons also lists them under “Why this status” without an adjacent prior-calculation label. The same panel says “Status is being recalculated” at `:71` although the condition proves version mismatch, not an active worker.

`web/src/components/TodayInterventions.tsx:84` uses “Ready now” when an outcome check has no `next_check_at`; `:153` also returns it for an invalid date. A missing schedule is not evidence of eligibility to check the outcome.

Suggested behavior and copy:

| Condition | Before | After |
| --- | --- | --- |
| Program calculation missing | Open issues: 0 | Open issues: Unknown |
| No calculation | No status exceptions are recorded for the latest calculation. | Status has not been calculated. |
| Old calculation | Updating status / Why this status | Status out of date / Previous assessment |
| Outcome schedule absent/invalid | Ready now | Check time not recorded |

Only say “Updating” when processing state confirms it. Preserve stored counts as previous counts with their date/version. Required fixture: missing current state, stale empty reasons, stale nonempty reasons, failed update processing, missing/invalid outcome schedule.

### COPY-02 — P1: Historical outcome attribution can fall back to the current reviewer

`web/src/components/MatterOutcomePanel.tsx:283` resolves an outcome history reviewer from its result-specific responsible party or exact reviewer ID, then falls back to `recordOperation?.assigned_to?.display_name`. At `:296`, that value renders as “Recorded [date] by [name]”. After reassignment or unavailable historical identity enrichment, the current reviewer may be named as the author of an older result. `:293` also falls back from the expected independent reviewer to the definition operation assignee, whose task is different.

Use the historical result's recorded reviewer identity and its scoped display name. If it cannot be resolved, show **“Reviewer name unavailable”** and retain the recorded identifier in history detail. For a pending outcome check, show **“Awaiting review”** or **“Awaiting review by {full name}”** only when the current reviewer route supplies that person. Do not infer this from the owner, maker, current assignee for a different operation or first candidate.

Related visible gaps: `web/src/components/forms/ResponseAssessment.tsx:109` prints `decision.reviewer_id` after “Reviewed by”; `web/src/components/forms/VendorResponseReview.tsx:53` prints `receipt.actor_principal_id` as “Reviewer”. These are P2 usability gaps, not fabricated identity. The form assessment API currently supplies a reviewer ID, not a pending reviewer's display name; resolving names is a data-contract task, not a string substitution. Test reassignment, absent enrichment, historical IDs and multiple eligible reviewers.

### COPY-03 — P1: Program creation persists assumed banking requirement semantics

`web/src/components/ProgramSetupWorkspace.tsx:71` calls `addProgramRequirement` in `web/src/continuityCommands.ts:148`. The helper writes `actor: "The bank"`, `action: "maintain the stated safeguard"`, and `object: "the monitored channel"` at `:156–158` regardless of the user's requirement text. These are stored requirement semantics, not just placeholder wording. In the full editor, `web/src/components/ProgramRequirementsPanel.tsx:24`, `:29` and `:30` default a new or absent requirement actor to “The bank”; `:35` sends that value.

Use a source-supported obligated party and action/object or request those values when required. For an unfilled editable field, use a concise prompt such as **“Who must meet this requirement?”**, with no assumed actor value. Do not mechanically replace “The bank” with “The organization” and preserve an unsupported action/object. The requirement actor here is the obligated party, distinct from the verified command actor; both semantics must remain intact. Test a nonbank organization and a requirement whose obligated party is a vendor or regulator.

### COPY-04 — P2: Shared review states and actions hard-code the banking sector

This is a repeated workflow issue, including accessible names, not an isolated heading.

| Source | Exact examples | Suggested copy |
| --- | --- | --- |
| `web/src/components/VendorFormsPanel.tsx:12`, `:124` | Awaiting bank review; Bank review in progress; Bank review not required | Awaiting review; Review in progress; Review not required |
| `web/src/components/VendorsWorkspace.tsx:637`, `:816` | Awaiting bank review, including aria-label and count link | Awaiting review; {n} awaiting review |
| `web/src/components/forms/ResponseAssessment.tsx:14`, `:83`, `:122` | Bank assessment; Bank assessment complete; Save bank assessment | Assessment; Assessment complete; Save assessment |
| `web/src/components/forms/ResponseAssessment.tsx:109`, `:111` | Saved bank judgement; Bank judgement for {field} | Saved decision; Decision for {field} |
| `web/src/components/forms/FieldAssessmentEditor.tsx:38`, `:43` | Require bank review; Bank rubric | Require review; Assessment rubric |
| `web/src/components/forms/fieldAssessment.ts:5`, `:7` | Bank review; Automatic rules, then bank review | Reviewer assessment; Automatic rules, then review |
| `web/src/components/forms/FormPolicyEditor.tsx:111`, `:127` | bank records; Choose bank records | records; Choose record type |
| `web/src/components/VendorDueDiligence.tsx:212`, `:564`, `:631` | bank review reference; Bank validated | review reference; Accepted by reviewer |
| `web/src/components/forms/VendorResponseReview.tsx:64` | Current bank record | Current record |
| `web/src/components/EvidenceWorkspace.tsx:24–25` | Bank system; Official bank record | Internal system; Official record |
| `web/src/components/VendorsWorkspace.tsx:902` | These details are shared across the bank… | These vendor details are shared across your organization… |

Review the full field-assessment path: configuration → policy → response list → response assessment → saved decision → vendor summary → error/retry text → accessibility labels. Keep automatic results distinct from reviewer assessments; “Assessment complete” does not become “Accepted” unless the stored outcome actually means acceptance. Existing `BANK_*` API enum names are compatibility details and need not be renamed to correct shared display copy.

Additional shared setup leakage: `ProgramSetupWorkspace.tsx:90`, `:93`, `:97`, `:106`; `MatterSetupWorkspace.tsx:90`; `access/IdentityAccessComposers.tsx:80`; `ProgramSafeguardsPanel.tsx:174`; `AIGovernancePanel.tsx:52`; `GovernanceAdminWorkspace.tsx:238`. Replace generic creation placeholders with neutral operational examples, without rewriting a saved banking requirement.

### COPY-05 — P2: External capture repeats institutional review prose where a concise state is needed

`web/src/components/CapturePanel.tsx:358` shows **Received** plus “All required documents received. No response needed. Bank review is separate.” The same qualification appears in each already-received field at `web/src/components/capture/CaptureFieldControl.tsx:24` and its review summary at `CaptureReview.tsx:42`. `web/src/components/VendorEvidenceChecklist.tsx:138` uses “Bank reviewer · No upload needed.” `VendorFormsPanel.tsx:58` uses “Bank review is separate” as the assessment value for received evidence, which explains the lifecycle rather than showing its actual state.

Use **Received** for the collection state; **Awaiting review** only when pending review is recorded; **Accepted** only for a recorded acceptance. At a received field, retain **“No upload needed.”** if it prevents duplicate work. At the overall receipt, **“All required documents received. No response needed.”** is sufficient; show review status in its own compact field if relevant to this recipient. Do not repeat a review disclaimer on every field, receipt and summary.

The checklist now correctly uses “Checklist” (`:90`) and “Missing”/“Accepted” (`:134`, `:136`). Preserve that improvement. The bare detail “Vendor” at `:134` names a party without an action; omit it when context is obvious or use **“Request document”** for a real available action. “Linked” at `:139` describes provenance, not receipt/review state; keep the link source in supporting detail while the primary status describes the collection state.

### COPY-06 — P2: Forms calls submitted responses “Completed work” before assessment

`web/src/components/forms/ResponsesView.tsx:158` labels the screen “Completed work”; `:149`, `:162–199`, `:227` and `:234` repeatedly use “Completed”, “completed responses” and completion dates. The same detail mounts `ResponseAssessment` at `:247`, which explicitly supports awaiting/in-progress review. The work is not complete merely because a response was submitted.

Use **“Responses”** as the heading, **“Submitted”** as the date column, **“Submitted from/until”** as filters, and **“{n} responses on this page”** for the scoped count. Remove “Completed work”. A submitted response with incomplete review should show **“Received · Awaiting review”** in the appropriate response and assessment fields. Keep the submitted version's immutability message; it is a useful limitation. Verify the backend meaning of `completed_at` before changing its label: the UI should name the actual recorded event, not invent a submission timestamp.

### COPY-07 — P2: Configure still narrates architecture and defends screen organization

These examples occur on shared operational screens, not developer documentation:

| Location | Before | After |
| --- | --- | --- |
| `web/src/components/configure/ConfigureWorkspace.tsx:40` | Keep ClearSight connected, governed and operational without mixing administration into daily bank work. | Manage access, approval routes, integrations and system settings. |
| `configure/ConfigureOverview.tsx:12` | Control plane / Open one administrative area at a time… | Configuration areas; remove the navigation instruction |
| `configure/AuthorityRoutingSection.tsx:50` | Supporting context only; assigned work remains canonical in Today and Work. | Review assignments affected by these approval routes. |
| `configure/AutomationSection.tsx:27` | …without mixing them with AI model and workload governance. | Review automation policies and action limits. |
| `configure/SystemOperationsSection.tsx:81` | …without mixing operational recovery with governed business work. | Review processing failures and retry eligible jobs. |
| `configure/DataIntegrationsSection.tsx:4` | Exact import review can still open directly from Today… | Review imports and connected sources. |
| `configure/SystemActivityPanel.tsx:229` | A bounded view of recently committed system and business activity. | Recent system and business activity. Open a record to review it. |
| `configure/SystemActivityPanel.tsx:285` | The server fixes an exact as-of boundary and exports only normalized audit fields… | Export audit events as of the selected time. Each export supports up to 10,000 events. |
| `access/OrganizationInventory.tsx:37` | Parent positions outside this bounded view | Parent positions not shown |

The export example must preserve the actual allowed fields and explicit size-limit recovery in secondary detail. “Bounded”, “exact” and “server” do not themselves communicate an operator action. Further concentrated review is needed at `configure/AIGatewaySimulationPanel.tsx:111`, `:131`, `:149–150`; `AIGatewayControlPlane.tsx:99`; `AIGatewayTransportControl.tsx:76`, `:160`; and `AIGovernancePanel.tsx:40`. For example: “Test the selected policy and routing versions. This simulation makes no model calls.” Technical provider, region, TLS and secret-reference terms are appropriate in specialist configuration when they determine a real choice; do not flatten these into vague business copy.

### COPY-08 — P2: Error recovery text is bypassed or claims success before confirmation

`web/src/http.ts:54` accepts backend message strings verbatim and falls back to “Request failed with {status}”. Many consumers prefer any `Error.message` over their operational fallback: `forms/ResponseAssessment.tsx:154`, `forms/ResponsesView.tsx:289`, `ProgramRequirementsPanel.tsx:41`, and `DocumentImportWorkspace.tsx:52`, `:120`, `:135`. A normal `fetch` rejection is an Error, so a useful fallback such as “The assessment could not be loaded. Retry…” is replaced with browser prose such as “Failed to fetch”. Domain error codes need task-specific approved presentation while raw diagnostic detail remains available separately.

Imports also says “The latest version has been loaded” at `DocumentImportWorkspace.tsx:148` and “The latest comparison has been loaded” at `:170`, `:188` **before** awaiting `refresh`. `refresh` (`:35–54`) can fail or still be loading. Instead show **“This import changed. Reload it before saving your review.”** until refresh succeeds, then **“Latest version loaded. Review the changes before saving again.”** Keep draft-retention promises only when that path actually retains the draft.

`WorkspaceErrorBoundary.tsx:26` says to reload the view but `:27` only resets the boundary state. Align the label and help with the actual retry mechanism. Required tests: network failure, unmapped error code, forbidden/conflict, refresh failure after conflict, retry with retained draft, and failure after a successfully committed command.

### COPY-09 — P2: One reviewer-neutral starter form is actually a bank-specific runtime template

`web/src/vendorDueDiligenceForm.ts:10`, `:17`, `:18` contain “bank information”, “Bank information used”, “No bank information” and “Do subcontractors process bank information?”. This file is not merely a fixture: `web/src/components/VendorFormReadiness.tsx:70` passes it to `createFormTemplate` in the production workflow.

For the shared starter, use **“Information used”**, **“No organization information”**, and **“Do subcontractors process your organization's information?”**, or a verified requesting organization's name where the template contract supports safe substitution. The source question, option values and existing response/scoring compatibility require review together. Do not silently rewrite already-approved/sent template revisions. If keeping a specific banking starter, name and select it explicitly as a banking template rather than making it the universal default.

### COPY-10 — P3: Repeated headings and explanatory wrappers increase reading without adding state

`web/src/AppViews.tsx:22` supplies Today heading and assigned-work description; `TodayInterventions.tsx:26` repeats Today, another action heading and another assigned-work description; its unavailable state adds a third Today title at `:31`. Consolidate the main title, retain the count as state, and show the error once. “Nothing needs your action right now” at `TodayInterventions.tsx:34` can become **“No assigned work”**, with the existing scoped explanation retained.

`web/src/components/MatterCurrentHandoff.tsx:39–41` stacks “What needs to happen next”, “Current handoff” and the actual next action. Use **“Next action”** once and the concrete action as the heading. `:42` and `:53` repeat the operation reason in the read-only state. The no-operation fallback also claims “No person or role is currently assigned” even though `:23` may have a stored owner; use **“No action is available for your current role”** without claiming the whole issue is unassigned.

`forms/ResponsesView.tsx:242` stacks “Submitted versions” and “Version history”; `:259` stacks “Calculation detail” and “Why this score was assigned”. Prefer **“Version history”** and **“Score calculation”**. `forms/FormQualityPanel.tsx:12` stacks “Quality gate”, “Approval readiness” and “Ready”; use **“Approval checks”** with the blocker count. Keep task-specific help where it changes the decision, such as why an assessment cannot lower a critical automatic score.

### COPY-11 — P2: The current copy test cannot enforce app-wide review

`web/src/copyQuality.test.ts:3–5` scans only three shallow directory levels. All 80 nested component files are excluded, including many COPY-05/07 examples. Its `what is still needed` rule (`:10`) only matches direct plain text inside h1–h6; an eyebrow, expression, helper, span or equivalent wording bypasses it. `canonical` only has narrower patterns; “assigned work remains canonical” is not caught. It scans raw source rather than rendered strings, so widening indiscriminately could flag valid internal identifiers and developer-only gallery documentation.

The coordinating agent ran the existing test under Node 24: **1 test passed, 5.61 seconds**, despite the visible findings above. Passing that test is not copy acceptance.

Expand source coverage recursively; explicitly classify test/demo/developer-gallery content; test semantic string helpers and representative rendered workflows; add focused patterns for newly recognized narration, avoiding broad bans on technical identifiers, actual legal text or organization names. Tests must distinguish received/submitted/accepted/verified, missing/unknown/incomplete, and unavailable reviewer names. Update the standards' shared-interface examples to the latest industry-neutral requirement so future features do not recreate the same drift.

## Surface coverage and retained good behavior

| Surface | Source review coverage | Result / remaining proof |
| --- | --- | --- |
| Today | AppViews, TodayInterventions, guide hosts, dynamic attention fields | COPY-01/10; API supplies titles/state/owner/reasons, requiring backend audit |
| Programs | List, record/current position, setup, requirements, evidence, safeguards, review digest, lifecycle | COPY-01/03/04; scoped named-owner fallbacks in ProgramDetailsPanel are preferable to fabricated names |
| Work / issues | MattersWorkspace, current handoff, actions, details, outcome, information, evidence requests | COPY-02/10; existing completed-action vs outcome wording is materially useful |
| Forms | Workspace/navigation, creation, builder/assessment, sent, responses, policies, communications, history | COPY-04/06/08/10/11; assessment formulas and legal/signature copy need state-specific acceptance |
| Vendors | Register/detail, due diligence, checklist, form progress/request/review, activation, work, identity, relationships | COPY-04/05/09; recent Checklist/Missing/Received/Accepted simplification retained |
| Imports | Import list, comparison, proposal review/handoff and result links | COPY-08; source quotations and proposal content must retain provenance and meaning |
| Configure | Overview, access/directory, authority, data, automation, AI and system activity | COPY-04/07/11; specialist technical fields kept where operationally necessary |
| External capture | ExternalCaptureApp, CapturePanel, field/form/review, access, conflict/recovery | COPY-05/08; connection-retry copy and separate submission/review are generally useful |
| Documents | Browser, occurrence/file state, preview and review | “Awaiting review” already appears at `documents/DocumentFile.tsx:18`; reconcile “Validated by reviewer” with accepted-evidence semantics rather than blanket renaming |
| Guides | IntroGuide, RoleAwareOnboarding, CinematicGuidePanel and their dynamic props | Dismissal failure explicitly leaves the guide closed for the session (`RoleAwareOnboarding.tsx:123`); guide titles/descriptions are backend data, audited separately |
| Demo/reference | AppViews reference header, BankJourneysWorkspace, demo menu, static fixtures, capture evidence entry | Keep Nigerian banking reference notice at `BankJourneysWorkspace.tsx:86`; fixture organizations and real regulations are not shared-sector defects |

## Implementation acceptance and limits

Resolve P1 truth/attribution issues first, then review each full workflow's copy in one pass. A rename must reach headings, filter options, accessible names, receipts, notices, API messages, empty states, errors, starter templates and documentation. Do not invent full names to improve a mockup or treat “Accepted” as a generic green success label. Render with a verified named reviewer, unavailable name, changed reviewer, no route, pending review, received evidence, rejected evidence, incomplete response, stale status and failed refresh.

This source audit does **not** claim all role/permission states were executed in a browser. It provides exact source evidence and state conditions; contrast and component-composition renders are owned by the parallel audit. No legal-content rewriting, screen-reader session, translation audit or jurisdiction review was performed. API-served guide text, authority explanations, arbitrary user-authored form/requirement content, imported quotations and deployed connector errors cannot be exhaustively assessed from frontend literals. Source findings about absent/stale data and historical attribution require targeted tests before a fix is declared complete. Preserve sample/reference notices and existing approved material versions throughout remediation.
