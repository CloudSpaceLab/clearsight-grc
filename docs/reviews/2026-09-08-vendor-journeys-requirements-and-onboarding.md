# Vendor requirements refresh and onboarding: practical journey review

Reviewed 8 September 2026 against checkout `11687656`. This is a proposed product direction, not an approved implementation specification. It extends the [forms and vendor usability audit](2026-09-08-forms-vendor-workflow-usability-audit.md).

The user has selected **governed conditional approval** for new vendors. Mandatory blockers remain controlled by bank policy; a conditional approval must not represent outstanding requirements as satisfied.

## Main conclusion

The central usability problem is that the bank must assemble collection, evidence review, internal remediation, assessment and authorization across separate entry points. The individual controls have legitimate purposes. Routine users need a continuous journey to a business outcome, with one next action for their role and current state.

Recommend two entry actions in Vendors: **Review existing vendors** and **Onboard vendor**. Both should use the existing governed collection, evidence, issue, decision and authority capabilities. Forms remains the reusable collection tool and specialist configuration area, rather than a prerequisite destination for doing vendor work.

The unit of assessment is the vendor's service for a particular bank legal entity. A vendor's corporate identity can be reused, but approval for one service, entity or data scope must not silently approve another.

## What the supplied workbooks actually contain

Sources were opened read-only with openpyxl. Neither workbook was changed.

| Source | Observed contents | Correct starting interpretation |
|---|---|---|
| [Sample Third-Party Risk Register (1).xlsx](<C:/Users/Son/Downloads/fidelitybankgrcusecasesandrequirements/Sample Third-Party Risk Register (1).xlsx>), `2026 Register!A1:P6` | Five populated finding rows across two service groups. Assessor, service, overall rating and other values use merged cells. Rows after 6 are empty or formatting-only. | Historical findings, recommendations and commitments to reconcile against existing vendor records. These are not five automatically approved requirements for every vendor. |
| [NDPA_Compliance_Checklist (1).xlsx](<C:/Users/Son/Downloads/fidelitybankgrcusecasesandrequirements/NDPA_Compliance_Checklist (1).xlsx>), `Checklist!A1:F48` | 47 checklist items with references, applicability, timing and evidence columns. | Candidate requirements requiring scope, source and responsibility review. These are not 47 questions to send unchanged to every vendor. |

The register's five source findings are all labelled Medium and Open, with recorded deadlines of 31 March 2026. Those are historical source facts, not a current bank assessment. The deadlines are past at this review date, but current fulfillment remains unverified. Import must retain the original commitments and reconcile any later evidence instead of silently resetting deadlines or announcing a current compliance failure.

Both service-provider cells contain `xxxxx`. This is not a reliable identity match. A provider name embedded in a recommendation is a candidate clue, not permission to bind that row automatically. Service labels help a bank user choose an existing relationship. Re-importing the same source must recover or update the linked work, not create duplicate findings.

### Translating the risk register into practical work

| Source finding | Vendor contribution | Bank contribution and completion evidence |
|---|---|---|
| `F2:O2`: information-security standard absent; comment says surveillance audit underway | Supply current certification/status and the referenced attestation, if not already held. | Security reviewer reconciles the apparent inconsistency and checks entity, service scope, issuer and validity. An attestation alone does not automatically resolve the finding. |
| `F3:O3`: vulnerability assessment and penetration test absent | Supply relevant test report and remediation/retest evidence, or explain absence and intended delivery. | IT risk reviews test coverage and material outstanding vulnerabilities. Receipt of a report is not proof that vulnerabilities were corrected. |
| `F4:O4`: agreement lacks right-to-audit clause | Provide contractual input/signature when requested. | Business owner and legal/procurement prepare the amendment. Closure requires checking the executed, effective terms. The bank-owned drafting task must not become a vendor questionnaire checkbox. |
| `F5:O5`: ISO 27001 and ISO 22301 certificates not provided | Supply each relevant assurance document or record each gap. | Security and resilience reviewers evaluate their respective evidence. Preserve one source finding while tracking its two evidence checks. |
| `F6:O6`: expired PCI DSS evidence | Supply current applicable validation evidence or explain the gap. | Reviewer checks applicability, current validation and service coverage against bank policy. Do not infer acceptance from a filename or upload date. |

### Interpreting the NDPA checklist

Examples below describe the supplied sheet, not independently verified legal rules:

- `D2:D13` makes registration, classification, reporting and DPO items conditional on organizational circumstances. Bank and vendor obligations must be distinguished.
- `B18:F26` covers processing-lawfulness and privacy assessments. A vendor can supply facts and relevant evidence; the bank retains decisions about its own processing and applicable assessments.
- `B38:F40` explicitly covers processor agreements, due diligence and agreement updates. These generate internal contract/review work as well as vendor requests.
- `B41:F43` depends on international transfers. Ask about storage, processing, remote access and subprocessors before deciding what applies.
- `E24:E25` and `E46` contain deadlines relative to an instrument's issuance; `E31:E32` concern breach events. These cannot become a fresh onboarding due date or a permanent request for a non-existent breach report.

Before activating a reusable requirement set, the bank must confirm source/version, jurisdiction, legal entity, applicability, timing triggers, accepted evidence and authority. The checklist's references do not by themselves establish that its interpretations or deadlines are current. The official GAID PDF could not be retrieved in this review, so its individual articles were not verified.

“ISO” also needs a specific meaning in the requirement set: [ISO/IEC 27001](https://www.iso.org/standard/27001) concerns information security management, while [ISO 22301](https://www.iso.org/standard/75106.html) concerns business continuity management. A certificate can support several checks, but does not decide unrelated privacy or contractual requirements. [PCI SSC](https://www.pcisecuritystandards.org/standards/pci-dss/) likewise provides a distinct payment-data standard and validation resources. These sources were checked on 8 September 2026; the full licensed standards were not reproduced or audited.

## Journey 1: apply new requirements to existing vendors

### 1. Start with the intended review

From Vendors, choose **Review existing vendors**, select an approved requirement set or upload a source, and name the purpose. A genuine new requirement and a historical risk register lead to different preparation screens inside the same journey.

For this register, show “5 findings across 2 service groups need matching.” Resolve the masked vendor identities before issuing any requests. For a new checklist, show the proposed requirements, unresolved applicability and source references before bank approval. Deterministic/manual mapping must remain available without AI.

Reviewing and approving a reusable policy is a specialist task performed once per material revision. Routine owners select that approved version; they do not redesign its questions, scoring and automation for each vendor.

### 2. Confirm affected services and what changed

Choose services using business attributes: bank entity, owner, service category, criticality and processing profile. Show the complete authorized matching population using server-side filtering and bounded queries. Unknown classification remains visible as needing review rather than disappearing from the population.

For each service, compare the new requirement version with the prior review. Separate newly applicable checks, changed checks, still-valid prior conclusions and unknowns. Preserve prior approval history. A changed requirement raises review work; it does not silently revoke an existing relationship or change material risk.

### 3. Search existing evidence before requesting more

Inspect current contracts, accepted documents, previous answers and linked open findings. Reuse eligible evidence by exact revision, service scope, permission and freshness. A file may support several requirements; each requirement retains its own acceptance conclusion.

Show a simple preparation list: already supported, needs vendor evidence, needs bank action, or applicability unresolved. Resolve conflicts, such as the ISO surveillance comment, before treating either statement as authoritative.

### 4. Send only missing work and route internal work

Prepare a recipient preview grouped by vendor/contact and permitted service scope. Ask the vendor once for overlapping information where the collection rules permit it. Preserve service-specific decisions and do not expose unrelated bank records through shared links.

The owner reviews known contacts, missing items and deadlines, then uses **Send requests**. The system should retain a durable batch receipt, delivery failures and retry controls across navigation or reload. In parallel, route contract amendments and bank assessments to their own responsible people.

Vendor-facing copy names the actual missing evidence and why it is requested. It does not expose raw subject identifiers, internal assessment rubrics or bank-only findings. A lack of a required document can be declared explicitly with an explanation; it remains a gap for review.

### 5. Review changes and resolve findings

Each reviewer opens their assigned requirements with answer, evidence, prior result and requested decision together. Focus initially on new, changed, rejected or missing items. Ask a targeted follow-up against the exact item rather than resending the whole questionnaire.

Bank-held record updates, document acceptance, requirement satisfaction and final authorization remain distinct decisions, but share context and navigation. Avoid repeating the same explanation across screens when one saved rationale can be referenced. Do not weaken separate authority where different judgments are required.

When an issue requires remediation, record an owner, due date and closure evidence. A vendor promise is a commitment; an uploaded amendment is evidence; a signed effective clause checked by the bank supports the outcome.

### 6. Finish with a current, scoped result and continued oversight

The result identifies the service, approved requirement version, evaluated population and checked time. Separate satisfied, gaps, awaiting review and justified non-applicability. Missing knowledge never counts as a passed check.

For an existing active service, the decision concerns continued use and treatment of gaps, not another first-time activation. Renewal restrictions or suspensions follow explicit bank policy and current authority. Programs maintain ongoing obligations; linked issues handle changes and exceptions.

Evidence expiry, relevant source changes and missed commitments create or update bounded follow-up work. Escalation and any automated action need the applicable approved policy. Previously accepted evidence and the prior decision remain reconstructable.

## Journey 2: onboard a vendor with minimal bank entry

### 1. Enter the essentials and save a draft

Start with vendor name, service/purpose, business owner and vendor contact. Derive bank entity from verified context when unambiguous; otherwise let the owner select within permitted scope. Suggest existing vendor identity matches before creating another record.

Let the bank enter what it knows and leave the remainder for the vendor. Unknown criticality or privacy role must be an explicit pending classification, not an implicit assertion of low exposure or no personal-data role. Draft saving must not require an invitation-ready contact or every eventual approval fact.

### 2. Establish scope and prepare one invitation

Ask a short, progressive set of service questions: access to bank systems, data handled, service criticality, processing locations and subcontractors. Let the vendor complete missing facts and let the bank confirm consequential classifications.

Apply the bank's approved third-party and relevant privacy/security/continuity requirements. Show which checks apply and why. Avoid asking the owner to choose one “primary framework” as a substitute for several simultaneous obligations.

Before sending, summarize what the vendor must complete, existing information to confirm, requested evidence, deadline and internal reviewers. Existing contact and purpose values should be prefilled. Routine owners should not configure scoring policies, invitation audiences or technical result bindings.

### 3. Vendor completes one resumable request

Organize the request around company details, service/data use, relevant documents and final confirmation. Display known values for confirmation or correction, with disclosure limited to the invitation's purpose.

For each document, allow **Provide document**, **Not available yet**, or **Does not apply**, with an explanation for gaps and applicability claims. Require a response to the requirement; require a file when the response claims the document is available. A declaration of absence permits review, not approval or a passed check.

Support several relevant documents and evidence reuse without forcing duplicate uploads. File errors occur beside the affected item. Save/resume, expired-link recovery and authorized contact changes preserve submitted history and do not grant access simply because a link was forwarded.

If new answers make additional requirements applicable, add a clearly explained follow-up and preserve the earlier submission. Do not ask the vendor to repeat unchanged company details.

### 4. Bank reviewers work in parallel in one review workspace

Present requirement, vendor answer, evidence and judgment together. Route privacy, security, resilience, contracting and any policy-required identity verification separately; a single business owner does not become every reviewer or signatory.

Validation has several levels: safe/readable file, claimed issuer/entity/dates/scope, evidence relevance and bank acceptance. Extraction or automated checks may suggest discrepancies. They must not claim independent authenticity when that was not verified. The bank needs a manual path when an issuer or integration cannot be reached.

Review known record corrections as proposed changes with conflict handling. Request clarification for a named item. Save review drafts and warn before discarding unsaved judgments. Once the required reviews finish, show the next decision and responsible role in the same journey.

### 5. Reach approval, conditional approval or refusal/deferment

Prepare an approval summary from completed work: service scope, requirement version, material gaps, accepted evidence, reviewer conclusions, uncertainties and proposed terms. Authorized people make their distinct decisions through the current route.

Conditional approval, as selected by the user, requires each condition to identify the unresolved requirement, permitted activity/restriction, accountable owner, due date, required proof, verifier and review/expiry consequence. Missing mandatory approval gates cannot be converted into conditions by the sender. An assessment conclusion alone is not the final authorization.

After valid decisions, the business owner sees the approved result and next permitted action. If activation is a separate bank operation, present it as the next named step with its responsible actor and current blockers. A completed review must refresh eligibility without requiring the owner to reconstruct the workflow.

### 6. Carry conditions into operational follow-up

An approved service with conditions continues to display those conditions until separately verified. Do not show “all requirements met” because the relationship became active. Reminders, missed deadlines, evidence aging and changes enter the existing ongoing oversight process. Consequences follow the recorded policy; the system does not invent automatic shutdown rules.

## Specific corrections in the current product

### Agreed refinement: reconcile existing documents and make pending work unmistakable

The user emphasized that a matching document already submitted by the vendor must be reconcilable against a new request. The interface must distinguish satisfied requirements from pending work prominently, and must not continue treating bank review as a vendor submission obligation.

**Separate evidence receipt from requirement satisfaction.** Store and present whether the requested evidence has been supplied separately from whether the bank has accepted it for the applicable requirement. Reconciliation links a specific existing document revision to the requirement and collection item; it does not manufacture a new vendor submission or silently create a favorable assessment.

Before sending a request, present candidate documents already held in the same requirement row. Each candidate shows document name, submission date, vendor/service scope, validity information where known, and previous review status. Show the reason for the suggested match. A filename or an “ISO” tag alone is insufficient. Search only evidence the actor may use for this purpose; support manual selection if suggested matching is unavailable or incomplete.

Use the row action **Use existing document** where bank reconciliation is required. If the document fulfills the collection request but still needs substantive review, the result is **Submitted · Bank review pending**, with **No vendor action**. Remove that collection item from the outbound missing-item preview and vendor-pending count. If an existing acceptance is still valid for this exact requirement version and scope under the approved reuse policy, retain **Satisfied** without requiring a redundant new acceptance click. A new scope or changed requirement can require bank review even when the same file remains usable.

When several candidate documents exist, show the most relevant candidate with an option to inspect alternatives. A reviewer can decline the match without deleting the source document. An expired certificate, wrong entity, incomplete service coverage or inaccessible/unavailable artifact must not be presented as satisfying the request. State the concrete reason and, when needed, prepare a targeted replacement request. If applicability or suitability is not yet resolved, show that bank decision explicitly rather than assuming the vendor owes a new upload.

For requests already issued, authorized reconciliation updates the outstanding collection work through its workflow and retains who linked what, why and when. The vendor's task list must then acknowledge the evidence already received; reminders must exclude fulfilled collection items. Preserve original answers, submissions and dispatch history. If later review finds the evidence inadequate, create an explicit follow-up for the missing proof, explaining why the previous document was insufficient.

**Recommended visual hierarchy.** A vendor review opens with a large service name, a concise overall state, and a bold summary strip: **Satisfied**, **Vendor action needed**, **Bank action needed**, **Not applicable**. These are mutually exclusive primary work buckets for the current scoped requirements, with pending applicability included under bank action. Show an “Also waiting on…” detail where an item has additional actors, so the primary counts do not double-count requirements. Show approval conditions separately as decision follow-up; they are never counted as satisfied merely because approval was granted. Label scope and checked time beside the summary. Unknown data remains unknown.

Below the summary, show pending requirements first in a compact list with requirement name, plain-language status, current evidence, responsible person and one next action for the current actor. Satisfied requirements remain one click away with their accepted evidence and checked date. Bank-owned actions such as preparing an amendment sit in the same list, with a clear owner and deadline. On smaller screens, each row becomes a card retaining status, evidence and next action. Use weight, spacing and text labels for hierarchy; color supports rather than carries meaning. Open document details beside the requirement on desktop and in a focused detail view on mobile, preserving the user's place and unsaved work.

Illustrative rows, not live bank data:

| Requirement | Visible state | Evidence/context | Current next action |
|---|---|---|---|
| ISO 27001 assurance | Satisfied | Previously submitted certificate; acceptance current for this scope | View evidence |
| ISO 22301 assurance | Bank review pending | Existing certificate linked; no vendor action | Review document, for the assigned reviewer |
| Vulnerability testing | Vendor action needed | Requested report not yet supplied | Vendor: provide report or explain gap; bank: track request |
| Right-to-audit clause | Bank action needed | Legal amendment outstanding | Assigned bank owner: complete amendment |

The final decision area summarizes the same underlying requirements, blockers and conditions. It cannot show “Ready for approval” solely because every document was received. The workspace's dominant action follows the current actor: reconcile evidence, send missing requests, review evidence, complete an internal action, or make the routed approval decision.

Additional acceptance criteria:

- Existing valid accepted evidence satisfies a compatible requirement without duplicate upload or unnecessary re-review, subject to approved reuse rules.
- Linking an already supplied but unreviewed certificate clears its vendor collection obligation and leaves bank review visibly pending.
- A request preview and subsequent reminders contain only outstanding vendor work; the displayed summary refreshes after reconciliation.
- Wrong-scope, expired and contradictory evidence receives a specific explanation and cannot produce a satisfied status through suggested matching.
- One document can support several compatible requirements without duplicate uploads; each requirement retains its own acceptance and source/version lineage.
- Two reviewers reconciling concurrently cannot overwrite newer evidence or decisions. A later invalidation creates visible review/follow-up work without rewriting history.
- With mixed satisfied, vendor-pending and bank-pending fixtures, a bank user can identify what is already done, who owes the next action and why within the proposed 30-second usability target. No status requires opening several panels to interpret.

| Priority | Current evidence | Correction needed for these journeys |
|---|---|---|
| P1 | [Form proposal importer](../../internal/documentimport/form_template_proposal.go) proposes fields. The [existing acceptance record](../acceptance/vendor-form-assessment.md) reports 16 proposed fields and 32 unresolved items from this five-finding register; that recorded import was not rerun here. | Add source interpretation and work mapping before form generation. Preserve finding, requirement, internal action and evidence semantics. Exact worksheet/row lineage and repeat-import handling are acceptance requirements. |
| P1 | [Vendor detail](../../web/src/components/VendorsWorkspace.tsx) composes generic form requests, due diligence, activation and vendor work as separate choices. | Lead with the two business journeys and connect handoffs. Reuse existing services; preserve the rule that a generic submission cannot silently advance due diligence. |
| P1 | [Starter form](../../web/src/vendorDueDiligenceForm.ts:20) permits framework “None” while requiring a single assurance PDF unconditionally. | Allow explicit document absence and multiple applicable evidence checks. An honest gap must reach review without a fabricated attachment. |
| P1 | [Creation defaults](../../web/src/components/VendorsWorkspace.tsx:58) set Standard criticality and None privacy role. The starter collects only a contact email, service/data facts, one framework/document and attestation. | Separate a minimal draft from confirmed classification. Provide a coherent bank-to-vendor completion path for remaining corporate and service information. Do not let unanswered risk facts look confirmed. |
| P1 | [Conclusion panel](../../web/src/components/VendorDueDiligence.tsx:618) collects conclusion, rationale, uncertainty and next review date. [Activation gates](../../internal/thirdparty/activation_policy.go:410) separately check conclusions, terms, decisions, authority, issues and contradictions. | Connect conditional recommendation to governed terms and authorized decisions. Show every condition's follow-up lifecycle and the actual next approver. Do not replace the existing activation controls with a form score. |
| P1 | [Activation facts](../../internal/thirdparty/activation_policy_postgres.go:328) establish `ConditionsRecorded` from nonempty conditions on qualifying decisions. | Presence of terms is not evidence of the full owner/deadline/verification lifecycle requested here. Prove end-to-end term enforcement and follow-up in acceptance; this inspection does not establish that upstream decision validation is absent. |
| P1 | [Activation panel](../../web/src/components/VendorActivationPanel.tsx) does not directly consume assessment completion as a refresh dependency; blockers are explanatory text rather than work links. | Refresh on material progress and link each outstanding gate to the responsible person's task. Preserve server-side revalidation at execution. |
| P1 | First audit reproduced unsaved judgment loss; batch state and retries depend on a transient composer session. | Preserve review drafts and durable batch recovery. Show delivery, response, review and decision as separate progress. |
| P2 | The vendor register's form-work filtering concerns loaded relationships; this is explicitly qualified in the UI. | Provide server-filtered affected-service and requirement-outcome views with accurate authorized counts, pagination and freshness. Form completion is only one aspect of the review. |
| P2 | Forms exposes overlapping assessment/scoring controls, technical send inputs and configuration navigation during routine work. | Keep advanced form/policy administration available to specialists. Routine owners select an approved requirement version, confirm scope and send missing requests. |

The backend correctly permits required documents to be either validated or rejected before an assessment review can finish ([completion check](../../internal/thirdparty/assessment_document.go:189)). Rejection can support an unsatisfactory conclusion; this is not itself an approval bypass. The usability correction concerns submitting an honest missing-document answer and carrying its consequence through the entire journey.

## Delivery direction and acceptance

Three options were considered: isolated fixes to existing forms; a guided journey over the existing governed capabilities; and a separate vendor workflow engine. Recommend the guided journey. Isolated fixes leave ownership and final-decision handoffs unresolved; another engine would duplicate lifecycle and authority work. Ship small reliability corrections while implementing one complete service journey at a time.

First prove the onboarding journey, including missing documents and governed conditional approval. Then prove the existing-vendor journey against the actual risk register and the NDPA checklist. Their shared review/evidence/decision workspace should reduce duplicated functionality.

Required scenario outcomes:

1. **Actual register:** detect five findings and two service groups; ignore empty rows; require resolution of masked identities; preserve merged context, source comments, dates and ratings; do not duplicate on re-import.
2. **Mixed responsibility:** the right-to-audit finding produces a bank-owned contract action and appropriate vendor input, reaching checked executed terms before closure.
3. **Prior evidence:** scoped valid evidence is offered for reuse; expired/wrong-scope evidence is not silently accepted; contradictory source statements remain reviewable.
4. **Minimal onboarding:** bank enters basic facts, vendor completes permitted remaining details, reviewer applies accepted changes, and the journey reaches an actual authorized outcome without starting a second questionnaire manually.
5. **Missing assurance:** a vendor selecting “not available yet” can submit; the item remains a gap; a mandatory blocker prevents approval; an eligible nonblocking gap can become a governed condition.
6. **Privacy applicability:** unknown processing cannot select the no-personal-data route; bank-owned assessments remain internal; irrelevant reporting/event documents are not demanded from every vendor.
7. **Multiple standards:** one service can require distinct ISO 27001, ISO 22301 and applicable payment-data evidence. One file may be referenced where relevant, while acceptance is tracked per requirement.
8. **Parallel review:** different current reviewers see only their authorized work. Conflict, absence, delegation and revoked authority lead to a valid routed next step, not blanket owner approval.
9. **Conditional lifecycle:** approval records explicit enforceable terms; deadline/expiry handling reaches the designated owner and reviewer; verified closure and continued restrictions remain distinct.
10. **Existing active service:** requirement changes preserve prior decisions, generate targeted review and lead to an authorized continued-use/treatment decision; they do not rerun first activation or silently revoke approval.
11. **Recovery:** browser reload retains drafts and send receipts; retries do not duplicate requests; stale evidence/decision versions require reconciliation; final eligibility refreshes after progress.
12. **Truthful portfolio:** an affected service outside the first loaded page still appears in the review population; unauthorized services remain excluded; unknowns, pending reviews and conditions cannot inflate compliant counts.

Proposed effort targets for usability trials, not measured performance: routine onboarding setup with known facts and approved requirements should take about three minutes; a returning owner should find the next actionable blocker within 30 seconds; unchanged facts should require no retyping; a vendor should not re-upload the same usable evidence to answer overlapping requests. Measure document preparation and legal/security judgment separately from interface overhead.

Before claiming implementation completion, create before/after fixtures for both journeys, including missing evidence, conditions, partial dispatch, restricted access and degraded integration. Render desktop, mobile and reflow states; verify keyboard and draft recovery; run copy-quality and affected workflow tests plus material authority/transaction/recovery checks. No new implementation or rendered completion is claimed by this review.

## Review limits

This follow-up inspected both workbooks and traced relevant source and existing acceptance evidence. It did not send invitations, modify bank records, implement the proposal, run a new live onboarding, or measure production adoption. The earlier audit's 146 passing tests and local browser findings remain supporting evidence for that earlier scope, not proof that these proposed journeys already work.
