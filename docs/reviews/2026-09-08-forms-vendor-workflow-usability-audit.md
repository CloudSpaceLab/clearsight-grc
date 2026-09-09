# Forms and vendor workflow usability audit

Reviewed 8 September 2026 against working-tree commit `11687656`.

## Assessment

There is substantial avoidable complexity. The largest problem is that users must choose between implementation-shaped workflows before they can do a straightforward business task. Collection, field assessment, due-diligence conclusion, evidence acceptance and activation have legitimate differences, but the interface does not assemble them into a clear route to the user's intended outcome.

The recommendation is to consolidate entry points and handoffs, remove technical inputs from routine work, and make advanced configuration conditional. Preserve independent review, evidence lineage, current authority, scoped access and explicit outcome checks.

This review does not establish that a feature is never used. No production usage or representative bank-user timing was available. The removal and demotion recommendations below are product judgments grounded in the implemented workflow, not measured adoption claims.

## Evidence and limits

- Read the root README and design contract, documentation map, governed-forms specification, ease-of-use standard, application architecture, current implementation ledger, vendor assessment plan and acceptance evidence, and relevant UI/API code and tests.
- Built the current evidence application and inspected the rendered vendor detail, ordinary vendor request, response assessment, field-assessment builder, Forms navigation and general Send form flow using local sample fixtures at the browser's 1280 × 720 viewport.
- Reproduced unsaved review-rationale loss by typing a rationale, closing the review sheet and reopening the same response. The text was empty and no discard warning appeared.
- Ran eight focused Vitest files covering vendor workspace, due diligence, vendor work, vendor form requests, vendor forms, response assessment, sent forms and copy quality: **146 tests passed**. The evidence Vite build also passed. These checks do not establish overall usability.
- No application source was changed, no vendor email was sent, and no material bank decision was recorded. Findings described as code-confirmed were traced through source; they were not exercised against a live bank or a PostgreSQL deployment. This is not a new mobile, accessibility, load or end-to-end production certification.

Priority: **P1** means material interruption, loss of work, incomplete operational results or an unreliable handoff. **P2** means recurring unnecessary effort or complexity. “Observed” refers to the local rendered sample, not customer production data.

## Findings

### 1. P1 — Three vendor request paths make the user choose the underlying workflow

**Flow:** Open a vendor to obtain evidence for onboarding or a review.

**Evidence:** The detail mounts Forms and responses, Due diligence, Activation and Vendor requests together. It offers Request form, Start due diligence and Request vendor work. The ordinary form path produces a response assessment; due diligence has a separate conclusion; vendor work requires a Program/issue link and its own acceptance. The governed-forms specification explicitly says a generic distribution cannot advance due diligence.

**Consequence:** A user can send the right-looking questionnaire through a route that cannot complete the intended onboarding review. The sample page already shows assessed form work alongside “No due diligence review has been started.” That combination is semantically possible, but the entry actions do not make the choice clear enough.

**Correction:** Start from a named task: onboard this service, collect additional evidence, or refresh an existing review. Route to the correct existing workflow and state the expected completion result before sending. Present one dominant next action for the current actor. Reuse submitted evidence where permitted, with an explicit review/linking action; do not silently convert a generic submission into approval.

**Acceptance:** A first-time owner can identify the onboarding path without understanding distributions or request origins; completing that path reaches the required decision without sending the same questionnaire again.

Sources: [detail composition](../../web/src/components/VendorsWorkspace.tsx#L757), [vendor work prerequisites](../../web/src/components/VendorWorkPanel.tsx#L222), [governed forms boundaries](../product/governed-forms.md).

### 2. P1 — General Send form requires internal identifiers

**Flow:** Forms → Sent forms → Send form.

**Evidence:** Observed editable Subject type and Subject identifier, with `CONTROL` as the default subject type. Sent-form filters also expose Subject type, Subject ID and Owner as text inputs. No business-record picker supplies this context. Detail renders the form ID and raw subject type.

**Consequence:** The sender must know internal identifiers or leave the workflow to find them. This fails the repository's requirement that routine work not require implementation terminology.

**Correction:** Use scoped, searchable business-record choices and carry the originating vendor, Program, issue and selected form into the composer. Keep raw identifiers in specialist/audit detail. Require explicit record selection when there is no originating context.

**Acceptance:** Send and find a request entirely through names and business references, without copying any ID.

Sources: [composer](../../web/src/components/forms/DistributionComposer.tsx#L123), [filters](../../web/src/components/forms/sent/SentFormsFilters.tsx), [detail](../../web/src/components/forms/sent/SentFormDetail.tsx#L14).

### 3. P1 — Vendor concern and overdue filters cannot search the full portfolio

**Flow:** Find all high-concern vendors or all overdue vendor responses.

**Evidence:** The register loads relationships in pages of 50, then `shownRecords` filters the already-loaded array using fetched summaries. Missing summaries do not match. The UI truthfully labels this “Form work in loaded relationships” and tells users to load more vendors.

**Consequence:** Users must repeatedly load unrelated relationships to discover later matches. It is an explicitly limited view, but it is not a practical portfolio triage flow. Unknown summaries also disappear from a filtered result rather than forming an actionable unknown group.

**Correction:** Apply the selected operational filter in the scoped repository query before keyset pagination. Return counts and availability for that same query. Keep summary failures visible as unknown work.

**Acceptance:** A matching relationship beyond the first 50 appears on the first filtered page; missing summaries cannot imply that the portfolio has no outstanding work.

Sources: [filter](../../web/src/components/VendorsWorkspace.tsx#L611), [load-more query](../../web/src/components/VendorsWorkspace.tsx#L471), [unknown handling](../../web/src/components/VendorsWorkspace.tsx#L812).

### 4. P1 — Closing a bank review silently discards unsaved work

**Flow:** Vendor → Review response → enter a rationale → close → reopen.

**Evidence:** Reproduced in the rendered fixture. Judgements live in component state; the enclosing sheet unmounts the assessment immediately on close. The form builder also has direct Back/section-change paths that clear its editor without a dirty-state check in the inspected handlers.

**Consequence:** Reviewers lose partially prepared work when interrupted or navigating to related context. Saving a final judgement should not be the only way to protect an unfinished rationale.

**Correction:** Provide scoped draft recovery or a dirty-state exit prompt, while keeping draft text distinct from a recorded bank judgement. Preserve entries through permitted navigation and transient failures. Protect in-flight saves against dismissal.

**Acceptance:** Close, Escape, navigation and reload either recover the draft or explicitly explain and confirm its loss. No draft is treated as an approved decision.

Sources: [local judgement state](../../web/src/components/forms/ResponseAssessment.tsx#L25), [sheet dismissal](../../web/src/components/VendorFormsPanel.tsx#L72), [builder exit](../../web/src/components/FormsWorkspace.tsx#L469).

### 5. P1 — Assessment completion does not refresh activation, and blockers have no direct action

**Flow:** Complete due diligence and continue toward vendor activation.

**Evidence:** Completing the assessment updates assessment/review state. Activation reloads only when relationship ID, version or status changes. Its successful read can therefore continue displaying the earlier failed assessment gate. A loaded but ineligible panel has no refresh control; a command conflict says to reload the checks without supplying that control in the ready state. Gate rows contain labels and explanations, but no relevant record link or responsible person.

**Consequence:** The user can finish a required step and still see it as incomplete, or be told to complete an approval/address check without a direct route to it. The server rechecks activation; this finding concerns the stale display and recovery path, not a demonstrated authorization bypass.

**Correction:** Invalidate activation eligibility when contributing assessment, decision, evidence or outcome versions change; expose an explicit refresh action and freshness. Each blocker should identify its record, current responsible role/person and next permitted action.

**Acceptance:** Complete an assessment without editing the relationship; the activation checklist refreshes. Every actionable blocker opens the exact decision or verification task, and conflict recovery is possible on the same screen.

Sources: [assessment completion](../../web/src/components/VendorsWorkspace.tsx#L421), [refresh dependencies and gates](../../web/src/components/VendorActivationPanel.tsx#L34), [server recheck](../../internal/thirdparty/activation_policy.go#L405).

### 6. P1 — Closing a batch request can lose the retry identity

**Flow:** Prepare requests for several vendors, dispatch, then close or navigate away during an uncertain/partial result.

**Evidence:** `VendorFormRequest` keeps input, batch ID and per-target receipts only in component state. Its sheet remains dismissible while `busy`. Reopening and previewing generates a new UUID. The server deduplicates using batch ID plus relationship ID. Existing-request counts warn but do not resume the previous batch.

**Consequence:** The carefully implemented retry protection applies while the current sheet state survives. Restarting after dismissal can create a new logical request for a target that already succeeded. This risk is code-confirmed; no real dispatch was performed during the audit.

**Correction:** Temporarily prevent dismissal during dispatch, and make the saved batch discoverable/resumable after navigation or reload. Keep per-target receipts and the original identity until reconciliation finishes. Reuse the existing Vendor Work sheet's in-flight protection pattern.

**Acceptance:** Under delayed and partly successful responses, close/reload/resume returns the original batch and retries only unresolved targets.

Sources: [batch state and dismissal](../../web/src/components/VendorFormRequest.tsx#L21), [server idempotency key](../../internal/httpapi/vendor_form_requests.go#L97), [existing protected send sheet](../../web/src/components/VendorWorkPanel.tsx#L232).

### 7. P2 — Routine requests ask the sender to reconstruct known context and technical access settings

**Flow:** Request an approved vendor form or send a clarification.

**Evidence:** The vendor composer starts with empty title, purpose, recipient and dates despite selected service/form context. It exposes estimated minutes, two date-time controls and three access mechanisms. Due diligence separates review date, response date and a link lifetime that defaults to 24 hours. Clarification clears the email before validating the other fields, so even a missing due date requires re-entry.

**Consequence:** A routine request becomes a configuration exercise. A link can expire well before the business deadline, producing further resend work. Re-entering addresses introduces effort and error opportunities.

**Correction:** Derive title/purpose from the approved form and task; offer authorized saved-contact references or an explicit new contact. Apply bank-approved access defaults and show the actual expiry in the preview. Keep review and response deadlines distinct where needed, with useful defaults. Preserve permitted entries on validation failures. Do not weaken audience binding or persist invitation tokens to accomplish this.

**Acceptance:** The normal send path primarily asks for recipient confirmation and response deadline. Security exceptions remain explicit, and correcting one invalid field does not erase unrelated valid entries.

Sources: [composer initial state](../../web/src/components/forms/DistributionComposer.tsx#L29), [vendor recipients](../../web/src/components/VendorFormRequest.tsx#L9), [clarification clearing](../../web/src/components/VendorDueDiligence.tsx#L314), [send controls](../../web/src/components/VendorDueDiligence.tsx#L551).

### 8. P2 — Evidence is separated from the judgement that depends on it

**Flow:** Review a document answer and record its evidence outcome.

**Evidence:** Observed a document-count sentence beside the judgement input; the shared “Review submitted evidence” button follows the entire question list and opens a second sheet for all response documents. Review defaults to all submitted fields even when only one field needs bank review.

**Consequence:** Reviewers scroll past unrelated answers, open another surface, locate the corresponding file, then return to record the judgement. This scales poorly across many document questions.

**Correction:** Put the submitted file name, availability and exact document action on its question. Keep preview and judgement together where practical. Start with fields needing the current actor's review, with all answers, context and history readily available.

**Acceptance:** Open the exact evidence for a pending field without searching a second document list; preserve the field and rationale when returning.

Sources: [assessment rendering](../../web/src/components/forms/ResponseAssessment.tsx#L99), [shared evidence action](../../web/src/components/forms/ResponseAssessment.tsx#L120).

### 9. P2 — Scoring configuration is split across overlapping locations

**Flow:** Create a simple questionnaire, then add bank review or scoring where needed.

**Evidence:** The overview contains an expanded “Assessment and scoring” section and a separate collapsed “Scoring” section. Question settings expose assessment mode, weight, review requirement, responsibility and rubric; answer scoring is elsewhere under response options. Advanced scoring adds contributions, weights and cross-field effects. Question type and requiredness appear in both canvas and inspector.

**Consequence:** Authors must understand how multiple settings interact and which editor governs the resulting score. A simple collection-only question still encounters assessment configuration prominently. The advanced functionality can be useful, but its current placement makes it feel mandatory.

**Correction:** Offer one coherent assessment setup with clear collection-only, automatic, bank-review and combined paths. Reveal rubric/weights/rules only for the selected mode. Summarize configured behavior once and make its source editable directly. Preserve advanced rules and their approval requirements for specialists.

**Acceptance:** Create an unscored collection form without visiting risk configuration; configure a mixed form and explain each field's effective scoring from one place.

Sources: [overview inspectors](../../web/src/components/forms/builder/FormInspector.tsx#L55), [field editor](../../web/src/components/forms/FieldAssessmentEditor.tsx), [advanced scoring](../../web/src/components/forms/builder/AdvancedScoringEditor.tsx).

### 10. P2 — Saving a form interrupts the edit-and-test loop

**Flow:** Change a scoring rule, save the required revision, then test it.

**Evidence:** Score preview intentionally evaluates the stored revision. The ordinary Forms workspace's save callback closes the editor, selects the saved form and refreshes the library. The author must reopen the editor and return to scoring settings to continue testing. A response-policy link in the inspector also leaves the builder; section changes clear the editor.

**Consequence:** Frequent saving creates navigation work and loses editing position. The requirement to test an identified stored revision is sound; leaving the editing context is unnecessary.

**Correction:** Save and continue in the same editor with the returned revision and selection. Preview should clearly identify whether it covers the latest saved changes. Guard navigation to policies when edits are unsaved.

**Acceptance:** Edit → save → test → adjust takes place in one persistent workspace, with exact revision evidence and no repeated search/reopening.

Sources: [save callback](../../web/src/components/FormsWorkspace.tsx#L388), [stored preview](../../web/src/components/forms/builder/AdvancedScoringEditor.tsx#L45), [section exit](../../web/src/components/FormsWorkspace.tsx#L247).

### 11. P2 — Administrative and navigation-only sections occupy routine Forms navigation

**Flow:** A sender or reviewer opens Forms to complete operational work.

**Evidence:** Seven peer tabs are always rendered: Templates, Sent forms, Responses, Documents, Policies, Imports and Communications. Imports is a static instruction to open the separate Imports workspace. Policies and Communications expose specialist governance/configuration rather than the routine collection task.

**Consequence:** Users must scan several sections unrelated to their current role. Imports adds a workspace transition without doing import work itself. Policy and communication governance need not be visible as equal-priority daily tasks for every actor.

**Correction:** Remove the navigation-only Imports tab and provide Import from the creation flow. Place policy and communication administration under role-appropriate settings, with contextual links from a form or request. Keep the shared document inventory available; its multiple scoped entry points are useful when they open the right evidence directly.

**Acceptance:** Routine senders/reviewers see their operational tasks first; administrators can still reach the full governed configuration lifecycle without copied IDs.

Sources: [tabs](../../web/src/components/forms/FormsNavigation.tsx#L4), [Imports stub](../../web/src/components/forms/FormsTabContent.tsx#L19), [policy configuration](../../web/src/components/forms/FormPolicyEditor.tsx).

### 12. P2 — Users must invent technical references for ordinary review work

**Flow:** Configure a held-value refresh, start a periodic reassessment, or record a vendor finding.

**Evidence:** Held-value configuration requires free-text “Record target key” and “Required subject type.” Reassessment requires a bank review reference. Findings require a stable lowercase reference matching a technical character pattern; the explanatory copy makes the user responsible for duplicate prevention.

**Consequence:** Bank users must know field keys and create stable identifiers simply to record a gap or review. Business references can be useful, but should not substitute for system-managed identity and deduplication.

**Correction:** Provide named, scoped field mappings; generate internal review/finding identities; suggest and open existing findings based on the response and field. Retain a bank's external review reference as optional metadata unless its actual process requires it. Never merge distinct findings solely because their titles match.

**Acceptance:** Record a finding from a failed field without typing a slug, and configure “Confirm registered address” without knowing its record-target key.

Sources: [record mapping](../../web/src/components/forms/FormFieldPropertyEditor.tsx#L97), [reassessment setup](../../web/src/components/VendorDueDiligence.tsx#L534), [finding reference](../../web/src/components/VendorDueDiligence.tsx#L591).

## What to simplify, retain and validate

| Treatment | Scope |
| --- | --- |
| Fix first | Lost review drafts; stale activation checks and missing recovery; batch resumption; portfolio filtering before pagination. |
| Consolidate | Vendor request entry points; request detail, response review and evidence context; assessment/scoring setup. |
| Remove from routine input | Raw subject IDs, record-target keys, user-generated deduplication references, manually repeated form purpose/title where an approved default exists. |
| Demote to specialist settings | Communication templates/branding, response automation policies, cross-field score effects, layout overrides and exceptional access settings. |
| Remove as a separate stop | Forms → Imports instruction-only tab. |
| Preserve | Vendor identity versus service relationship; immutable response versus bank judgement; submission versus acceptance versus activation; independent approvals; specific clarification requests; current authority; secure access expiry/revocation; outcome verification; shared document history. |
| Validate before deleting | AI drafting, starter selection, advanced scoring and response automation. Measure actual usage, setup effort and recurring benefit; the repository alone cannot justify declaring them unused. |

## Target journey and acceptance order

Use four clear task routes without replacing the existing records beneath them:

1. **Onboard a vendor service:** find/reuse identity → record service → request required evidence → review exceptions → obtain required decisions → activate when eligible.
2. **Request additional information:** start from the vendor/Program/issue → choose approved form and recipient → confirm deadline/access → track the same request through response and appropriate review.
3. **Review a response:** open assigned pending fields → inspect exact evidence → record judgements or request specific clarification → continue to the governing review/decision where one exists.
4. **Maintain a form:** reuse/create → edit → save and test in place → independent approval → make available for requests. Advanced scoring and automation appear only when required.

Deliver the defect corrections before a broad visual redesign. Then test these journeys with a relationship owner, bank reviewer, form administrator and invited vendor respondent. Use the existing effort targets: routine request median under three minutes, routine contextual approval under two minutes, and no more than three major workspace transitions without justification. Measure wrong-path starts, duplicate information requested, interruption recovery and reviewer corrections as well as completion time. Fast execution that bypasses evidence or authority does not pass.
