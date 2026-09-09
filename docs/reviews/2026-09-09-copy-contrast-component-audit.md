# Copy, contrast and component audit

**9 September 2026 · ClearSight · Current local working tree**

The interface needs a coordinated correction across status language, visual contrast and repeated workflow controls. Replacing “bank” in a few headings will leave the same wording in errors, accessibility labels, starter forms and saved requirement defaults. Some labels also misstate the data, which takes priority over cosmetic simplification.

This audit includes the earlier uncommitted vendor evidence reconciliation work on `codex/vendor-evidence-reconciliation` (base commit `11687656`). It preserves the current implementation as the review baseline. **No production code, customer data or deployed configuration was changed in this audit.**

## Read the findings

- [Frontend copy audit](2026-09-09-copy-audit.md): 11 findings, source anchors, before/after examples and affected workflow coverage.
- [API copy and reviewer identity](2026-09-09-api-copy-audit.md): shared errors, validation, recovery and the contract needed for reviewer names.
- [Rendered contrast audit](2026-09-09-contrast-audit.md): measured ratios, CSS causes, light/dark and desktop/mobile evidence, exclusions and incomplete checks.
- [Component and workflow duplication](2026-09-09-component-duplication-audit.md): 11 findings, Keep/Merge/Remove/Move decisions, implementation risk and acceptance checks.

The reports group related defects by cause. Their counts should not be added as if each were a separate screen defect; several findings overlap deliberately across copy, rendering and composition.

## Correction order

| Priority | Finding | Correction | Evidence |
| --- | --- | --- | --- |
| 1 | Unknown data can look clean or ready | Keep **Unknown**, **Unavailable** and **Out of date** distinct from zero, complete and ready. Never claim recalculation is running from a version mismatch alone. | COPY-01: Program open-issue count falls back to 0; empty reasons claim no latest exceptions; missing/invalid check time says Ready now. |
| 1 | Historical review can be attributed to today's assignee | Resolve the recorded reviewer for that result. Use **Reviewer name unavailable** when enrichment is absent. | COPY-02: `MatterOutcomePanel.tsx:283,296`. |
| 1 | New requirements persist invented banking semantics | Stop assigning “The bank”, “maintain the stated safeguard” and “the monitored channel” regardless of the entered requirement. Capture or derive source-supported values with explicit review. | COPY-03: `continuityCommands.ts:156–158`, called by Program setup. This is persisted business meaning, not a placeholder. |
| 2 | Shared copy assumes a banking customer | Review the whole runtime path: setup, status, filter, review, receipt, error, accessibility label and starter template. Use neutral labels, retaining genuine organization names and banking policy content. | COPY-04/05/09, API-01/03. |
| 2 | Received responses are called completed work | Label the actual event and retain a separate review state. Verify timestamp semantics before renaming a date. | COPY-06: Forms Responses mounts a pending review below “Completed work”. |
| 2 | Status badges, placeholders and essential field boundaries have insufficient contrast | Correct the shared text cascade and tokens, then remove local overrides that recreate the defect. Verify normal, hover, selected and focus states in each host. | Rendered contrast report and JSON measurements. |
| 2 | Vendors presents competing request actions | Keep the current assessment action dominant. Retain one secondary generic request entry and the separate register batch request. | C01/C04: two selected-vendor Request form buttons use the same callback; additional linked work serves a distinct purpose. |
| 2 | Forms repeats the same response results and evidence action | Show one automatic/reviewed result comparison, one automatic explanation and one document entry. Preserve revision history and review decisions. | C02: ResponsesView and its ResponseAssessment child render the same automatic result and open the same response documents. |
| 2 | One uncovered document restores all legacy document rows | Filter residual requirements/documents individually. Preserve separate decisions when one file supports two requirements. | C03: mixed checklist coverage sets a boolean that renders the entire legacy document list. |
| 2 | Mobile Forms puts a long filter form before responses | Show search and a compact **Filters** control; expand advanced filters on demand. Keep active filters and reset visible. | C11 and the mobile Responses render. |
| 3 | Configure and recovery text narrate implementation | Replace architectural explanations with the task, condition or recovery. Do not claim a refreshed version was loaded until refresh succeeds. | COPY-07/08, API-02. |
| 3 | Old components and CSS preserve inconsistent behavior | Migrate complete control groups; retire confirmed unused editor branches; share empty-state and overlay mechanics. Keep legitimate domain variants. | C05–C10 and the component inventory. |

Priorities describe the correction sequence, not a claim that every source condition was reproduced against a live authenticated backend. The individual reports identify source-confirmed, render-confirmed and proposed hierarchy changes separately.

## Measured contrast failures

The final matrix covers **104 fixture/theme/viewport combinations**: 26 states in light/dark at 1440px and 390px. Axe reports **17 text-node failures across 11 captures**, plus **2,715 incomplete node occurrences**. Repeated components contribute to those counts; incomplete checks are not passes. Supplemental checks establish placeholder and essential input-boundary failures that axe did not report.

| Element | Measured contrast | Required contrast |
| --- | --- | --- |
| Builder guidance placeholder | 2.03:1 light / 2.58:1 dark | 4.5:1 |
| Shared search placeholder | 3.14:1 light / 4.21:1 dark | 4.5:1 |
| Same-fill shared input boundary | 1.52:1 light / 1.51:1 dark | 3:1 |
| Overdue status | 4.12:1 light | 4.5:1 |
| Low concern status | 4.12:1 light | 4.5:1 |
| Vendor activation explanation | 4.35:1 light | 4.5:1 |

Text and essential control-boundary requirements follow [W3C text contrast](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html) and [W3C non-text contrast](https://www.w3.org/WAI/WCAG22/Understanding/non-text-contrast.html). Decorative borders and disabled controls are assessed separately. These are rendered failures, not a recommendation to make every separator visually heavy.

Two causes recur: transparent placeholder/badge colors lose contrast on their actual surfaces, and broad feature selectors override shared component text colors. Correcting the token alone will not fix an overriding selector. The [contrast report](2026-09-09-contrast-audit.md) records exact selectors, sources, screenshots and measurement limits.

## Copy contract

Use the shortest label that accurately names the state. Supporting text is conditional: add the owner, reason, deadline, source or recovery only when it helps with the current action. Avoid an eyebrow, heading and paragraph all repeating the same task.

| Context | Compact label | Detail when useful |
| --- | --- | --- |
| Required document absent | **Missing** | Requirement name; **Use existing document** or **Request document**, when available |
| Partial response or required information absent | **Incomplete** | Specific missing fields; do not imply an upload is absent when it already exists |
| Submission/document received | **Received** | Received date or linked source; receipt alone does not establish acceptance |
| Required review pending | **Awaiting review** | **Awaiting review from {full name}**, only from the current scoped review route |
| Review started | **In review** | Reviewer and due date where recorded |
| Evidence accepted for a requirement | **Accepted** | Decision, reviewer and date; this is not blanket vendor approval |
| Evidence rejected | **Rejected** | Reason and replacement action |
| Evidence no longer valid | **Expired** | Expiry date and replacement action |
| Applicability undecided | **Applicability pending** | The unanswered question or required decision |
| Applicability explicitly excluded | **Not applicable** | Recorded decision and reason |
| Data cannot be read | **Unavailable** | Retry or the actual recovery action |
| Population/value not known | **Unknown** | Do not substitute 0 or a reassuring completion claim |
| Assessment predates a material change | **Out of date** | Previous calculation date; update state only when known |
| Authorized approval with conditions | **Approved with conditions** | Conditions, owners and deadlines; mandatory blockers remain governed by policy |

**Awaiting review** is the preferred shared pending-review label. The previous **Pending review** is understandable but should not vary arbitrarily across Forms, Vendors and Documents for the same state. Review status must remain distinct from relationship approval and evidence acceptance. The display vocabulary should map typed domain states; a generic underscore-to-title-case formatter cannot establish their meaning.

Reviewer names are not currently available everywhere they are needed. The assessment DTO supplies a historical reviewer ID, not a pending full name. Extend the scoped runtime contract where necessary. Do not substitute the vendor owner, a different operation's assignee or the first eligible candidate. If routing names a group, show the actual group rather than inventing a person. Keep generic **Awaiting review** when the pending name is unavailable.

The new industry-neutral rule applies to shared operating UI. Do not silently rewrite an approved template, a user's requirement, an imported quotation, legal content or an actual organization name. Stable API enum values can stay unchanged behind neutral presentation. The universally offered starter template needs explicit correction/versioning; it is production authoring content, even though its source resembles a fixture.

## The two vendor journeys after correction

### Existing vendor, new requirements

1. Select the vendor/service and review the new requirements, including applicability and source.
2. Reconcile existing evidence before requesting more. The checklist shows **Accepted**, **Awaiting review**, **Missing**, **Expired** and **Not applicable** from the actual current records.
3. Request only the unresolved vendor contributions. A document already held for review is not an outstanding vendor upload.
4. Review pending evidence and record the assessment conclusion. Show the remaining gaps and their owners without reopening duplicate collection paths.

### New vendor onboarding

1. Enter basic vendor and service information; select applicable requirement sets.
2. Review the requirements and reusable evidence, then send one scoped request for the vendor's missing information/documents.
3. On receipt, put the checklist and the reviewer's next action first. Move response history and unrelated linked requests into compact secondary sections.
4. Record the assessment and route the relationship decision. Where policy permits, **Approved with conditions** retains visible conditions, owners and deadlines. It must not recolor unresolved conditions as accepted or bypass mandatory blockers.

These are target task sequences, not a claim that all onboarding, import and conditional-approval gaps are implemented. Retain the earlier [journey audit](2026-09-08-vendor-journeys-requirements-and-onboarding.md) for requirement-import and governed-approval work outside this presentation audit.

## Evidence and guardrails

Source inventory covers 260 non-test frontend TS/TSX files and 86 HTTP-handler Go files. Component reachability identifies 169 TSX files and 76 CSS files from the production entry. These measure different populations. Candidate copy lines and native-control source sites are discovery aids, not defect or visible-control counts.

The rendered review uses local sample fixtures with current components. Representative baseline renders:

- [Mobile response review](../evidence/2026-09-09-ui-audit/contrast/forms-response-light-390.png): expanded filters and review content.
- [Vendor checklist and surrounding workflow](../evidence/2026-09-09-ui-audit/contrast/vendor-checklist-light-1440.png): current checklist plus repeated request/history sections.
- [Mobile form builder](../evidence/2026-09-09-ui-audit/contrast/forms-builder-light-390.png): faint field guidance.

Full-page screenshots retain fixed/sticky UI at the capture scroll position; do not interpret their position within the tall image as a standalone overlap defect. Contrast conclusions use browser-computed colors and the qualifications in the contrast report, not antialiased screenshot pixels.

The existing copy-quality regression **passes (1 test)** under Node 24 despite these findings. Its shallow glob excludes 80 nested frontend component files and all API copy. Extend coverage recursively with explicit content boundaries, test runtime status helpers and API messages, and review whole rendered workflows. A phrase scanner cannot establish semantic accuracy.

Update shared-language examples in `DESIGN.md`, `enterprise-copy-and-content-design.md` and `plain-language-content-standard.md` during remediation. Their bank-centric examples and competing label guidance can recreate the same drift. Preserve bank-vertical documentation as sector-specific content.

## Completion criteria for corrections

- Unknown, stale, missing, received, reviewed and approved states remain distinguishable across summary, detail and history.
- Reviewer reassignment cannot alter historical attribution; pending named labels use verified current responsibility.
- Each workflow has one dominant action for the actor/state. No supported document or required review disappears during deduplication.
- A nonbank sample organization exercises shared setup, assessment, external capture, errors and starter-form creation without banking assumptions.
- Re-run copy and affected workflow tests; render each changed host in both themes, at desktop/mobile widths and 200% zoom. Include missing, rejected, expired, historical, unavailable, conflict, conditional approval and routing-gap fixtures.
- Contrast checks cover text, placeholders, necessary field boundaries and focus. Automated incomplete results remain unresolved until manually assessed; they are never counted as passes.
- Preserve before/after renders, document changed component contracts and verify accessibility/recovery before claiming visual completion.

This is a broad source and rendered-fixture audit, not an exhaustive certification of every permission combination, every generated guide, customer-authored text, translation, live connector response or WCAG criterion. The detailed reports record the remaining proof required for each correction.
