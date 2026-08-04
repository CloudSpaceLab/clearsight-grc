# Regulatory and Enforcement Intelligence Acceptance Tests

These tests are mandatory for the capability defined in [`../product/regulatory-and-enforcement-intelligence.md`](../product/regulatory-and-enforcement-intelligence.md).

They extend the general ClearSight acceptance tests. A feature is not accepted because a document was summarized, obligations were generated, a case was opened, or files were attached. It is accepted only when source authority, exact provisions, interpretation, applicability, legal authority, actions, evidence, response, protection and temporal history work together.

---

# 1. Core invariants under test

- primary source before register row;
- exact provision before obligation or directive;
- document class before workflow;
- final versus draft status is explicit;
- applicability requires authorized review;
- legal authority precedes case action and disclosure;
- authority request does not imply guilt, suspicion or reportability;
- case directives remain distinct from internal decisions;
- external response requires complete directive reconciliation;
- protected case content remains outside general search and AI context;
- amendments supersede rather than overwrite;
- task completion is not implementation or remediation effectiveness;
- KRI values derive from governed cases and sources where possible.

---

# 2. Golden Journey A — Final regulatory circular to verified rule package

## Setup

- An official regulator publishes a final circular in PDF with definitions, numbered requirements, an effective date, one implementation deadline, a table of thresholds and an annex.
- The circular applies to one legal entity and selected payment channels.
- The bank has an existing policy and partially overlapping controls.
- One threshold is already implemented; another is not.

## Required path

1. Source is captured from the official channel with original bytes, hash and retrieval metadata.
2. Document is classified as final circular.
3. Definitions, provisions, table rows and annex references receive stable source coordinates.
4. Candidate Directive Atoms are extracted.
5. Compliance reviewer sees source and candidate interpretation side by side.
6. One ambiguous provision is routed to legal review rather than guessed.
7. Applicability is assessed against licence, entity, channel and system context.
8. Existing policies and controls are reconciled.
9. Covered, partially covered and uncovered obligations remain distinct.
10. ClearSight drafts internal requirement, policy change, control implementation, technical rule where appropriate, Evidence Recipe and test.
11. Required owners and deadlines are approved.
12. Implementation evidence is collected.
13. Operating effectiveness is observed for the required population and period.
14. Readiness changes only after accepted evidence.

## Assertions

- [ ] Every obligation links to an exact source provision and version.
- [ ] Table thresholds are extracted with units and effective context.
- [ ] General model knowledge does not create a requirement.
- [ ] Existing control similarity does not automatically mean full coverage.
- [ ] Applicability and legal interpretation are human approved.
- [ ] Task completion does not produce compliant or verified state.
- [ ] Prior and current obligation versions are reconstructable.

---

# 3. Golden Journey B — Exposure draft must not create final controls

## Setup

- An authority publishes an exposure draft with a consultation deadline and proposed effective date.
- The document contains mandatory-sounding language inside proposed text.

## Required path

1. Source is classified as exposure draft.
2. Candidate requirements are extracted as proposed.
3. System creates impact-assessment and consultation work, not active obligations.
4. A user attempts to publish the candidates as final institutional controls.
5. Policy blocks final publication without an approved override and source-status change.
6. Later final publication is linked and compared with the draft.

## Assertions

- [ ] Proposed language is never presented as current law.
- [ ] Consultation deadline and possible future effective date remain distinct.
- [ ] Final source comparison identifies retained, changed and removed provisions.
- [ ] Draft interpretations remain historical and do not overwrite the final source.

---

# 4. Golden Journey C — Amendment, FAQ and effective-date extension

## Setup

- A final circular is followed by an FAQ, then an amendment extending one deadline and narrowing an exception.

## Required path

1. New sources are related to the original.
2. Provision comparison identifies affected obligations.
3. Existing implementation actions and readiness are reassessed.
4. Extended deadline changes planning but does not erase prior overdue history.
5. Narrowed exception reopens affected applicability decisions.
6. Committee and owner views show what changed and why.

## Assertions

- [ ] No source silently overwrites another.
- [ ] An FAQ cannot become a new primary obligation without interpretation.
- [ ] Deadline changes preserve original and revised dates.
- [ ] Affected controls, actions and reports are identified deterministically.

---

# 5. Golden Journey D — Unverified compliance-register row

## Setup

- A legacy spreadsheet contains a requirement summary, regulator name and deadline but no official source, reference number or provision.

## Required path

1. Row imports as a secondary working observation.
2. Candidate obligation state is `SOURCE_UNVERIFIED`.
3. ClearSight searches approved sources and proposes possible matches.
4. Ambiguous matches are presented with reasons.
5. Compliance reviewer selects the correct source or records inability to verify.
6. Only verified rows can become approved obligations.

## Assertions

- [ ] Spreadsheet upload cannot establish law.
- [ ] Original file, sheet, row and mapping version are retained.
- [ ] Similar wording alone does not auto-link a source.
- [ ] Unverified requirements remain visible without being represented as approved obligations.

---

# 6. Golden Journey E — Supervisory finding and remediation response

## Setup

- A supervisory report contains three findings, a management-response deadline and a later evidence-submission date.
- One finding overlaps an existing internal issue.

## Required path

1. Source and exact finding provisions are captured.
2. Supervisory Matters are created without duplicating the existing issue.
3. Authority observation and institution interpretation remain distinct.
4. Management responses, actions and milestones are approved.
5. Evidence is collected against each commitment.
6. Response package reconciles every finding and requested deliverable.
7. Authority feedback requests additional evidence for one matter.
8. The matter reopens without deleting prior response history.
9. Closure requires accepted outcome evidence and approved authority state where available.

## Assertions

- [ ] One internal issue may support multiple supervisory matters without duplicate truth.
- [ ] Submission is distinct from authority acceptance.
- [ ] Management response is versioned and approved.
- [ ] Evidence and later effectiveness remain separate.

---

# 7. Golden Journey F — Protected authority request with multiple accounts

## Setup

- A verified authority request names several customers and accounts, defines a transaction period, requests records, and includes a legal instrument.
- One subject identifier is exact, one has two candidate customer matches, and one does not exist.
- The request also asks for KYC and address information.

## Required path

1. Request enters a protected Authority Request Case.
2. Source authenticity, receipt time, deadline, confidentiality and legal-instrument state are captured.
3. Legal/compliance reviewer approves disclosure scope and response owner.
4. Requested records, subjects, periods and deliverables become separate Case Directives.
5. Exact subject resolves automatically under policy.
6. Ambiguous subject requires human selection.
7. Missing subject remains unresolved and is not guessed.
8. Existing KYC, address and transaction evidence is searched first.
9. Missing KYC or address facts create focused remediation tasks.
10. Complete records are assembled for the approved period.
11. Response package maps every directive to a deliverable or approved explanation.
12. Signatory approves package and submission occurs through the approved channel.
13. Delivery confirmation or acknowledgement is retained.
14. Case closes operationally while legal hold remains active where required.

## Assertions

- [ ] Case content is absent from general search, counts, suggestions and AI context.
- [ ] Ambiguous identities do not merge automatically.
- [ ] The request itself does not produce guilt or suspicious-report status.
- [ ] KYC update is verified in the authoritative system.
- [ ] Response package excludes unauthorized adjacent customer data.
- [ ] Every access, export and submission is audited.

---

# 8. Golden Journey G — Request with missing or disputed legal basis

## Setup

- A communication requests customer information or action but the expected legal instrument is absent, inconsistent or unclear under institutional policy.

## Required path

1. Case is created and deadline captured.
2. Source is not discarded merely because legal basis is unresolved.
3. Legal review state becomes `INFORMATION_REQUIRED` or equivalent.
4. Operational actions that do not require disputed authority may proceed only if approved.
5. Disclosure, restriction or other high-impact actions remain blocked.
6. Clarification or additional instrument is requested and linked.
7. New source version updates the case without rewriting prior state.

## Assertions

- [ ] AI does not determine final legal sufficiency.
- [ ] “Without court order” or equivalent status is explicit and reviewable.
- [ ] Blocked actions show the missing authority and reviewer.
- [ ] Deadline escalation does not bypass legal policy.

---

# 9. Golden Journey H — Suspicious-activity decision remains independent

## Setup

- An authority request concerns unusual transactions and asks for records.
- Internal monitoring has mixed evidence.

## Required path

1. Authority request and internal AML/fraud review are linked but distinct.
2. Case Directive requests records only as stated by source.
3. Internal policy determines whether enhanced review is required.
4. Authorized AML reviewer assesses reporting threshold using approved evidence.
5. AI may summarize and identify missing facts but cannot file or decide reportability.
6. Filing decision, rationale, source evidence, approver and submission proof are retained where a report is filed.
7. Authority response package does not disclose protected internal reporting information unless explicitly authorized.

## Assertions

- [ ] Authority request does not automatically create a suspicious report.
- [ ] No model confidence becomes a reportability decision.
- [ ] Internal report and external response have separate access and disclosure rules.
- [ ] Negative or no-file decision remains auditable without implying exoneration.

---

# 10. Golden Journey I — Address capture and branch evidence

## Setup

- A case requires current address verification for selected customers.
- Existing address data is stale or incomplete.
- Branch or field staff may capture evidence.

## Required path

1. Existing authoritative and prior address evidence is assembled.
2. Only unresolved facts are requested.
3. Capture flow explains acceptable evidence and sensitivity.
4. Image or document extraction proposes address fields with coordinates.
5. User confirms or corrects extraction.
6. Original artifact remains protected and versioned.
7. Address update requires approved review and authoritative-system write.
8. Verification confirms the update and records effective date.

## Assertions

- [ ] A photo or document proves only visible attributes and approved interpretation.
- [ ] Geolocation metadata is collected only where policy permits and is disclosed.
- [ ] Customer information is scoped to the case.
- [ ] Upload completion is not address verification.

---

# 11. Golden Journey J — Bulk request and partial success

## Setup

- An authority request contains hundreds of accounts in an attached spreadsheet.
- Some rows are duplicated, malformed, outside scope or unauthorized for the assigned user.

## Required path

1. Attachment is scanned and parsed as untrusted input.
2. Column mapping and preview show errors and duplicates.
3. Valid subjects become case items; invalid rows enter reconciliation.
4. Bulk actions show exact criteria, counts, exclusions and authority.
5. Authorization is enforced per subject and directive.
6. Partial failure is visible and recoverable.
7. Response package identifies complete, unresolved and excluded items.

## Assertions

- [ ] One bad row does not silently reject or accept the entire request.
- [ ] Duplicate subjects do not duplicate case tasks or disclosures.
- [ ] Unauthorized subjects reveal no material metadata.
- [ ] Denominators and exclusions are explicit.

---

# 12. Golden Journey K — Duplicate, amendment and follow-up request

## Setup

- The same authority sends an amended request with a changed period and additional subject.

## Required path

1. ClearSight detects similarity and proposes relation to the original.
2. Reviewer confirms duplicate, amendment or follow-up classification.
3. Changed directives are compared.
4. Existing evidence is reused where authorized and in scope.
5. New or changed tasks are created without duplicating completed work.
6. Response packages preserve their own versions and source basis.

## Assertions

- [ ] Original request and response remain immutable.
- [ ] Changed period invalidates evidence outside the new scope where appropriate.
- [ ] Reuse respects purpose and authorization.

---

# 13. Golden Journey L — Malicious or misleading source content

## Setup

- A document contains embedded instructions aimed at the AI, hidden text, malformed tables and a false deadline in an attachment.

## Required path

1. All content is treated as untrusted source material.
2. Prompt-like text cannot modify policy or tool access.
3. Visible and hidden text differences are surfaced.
4. Deadline conflict is identified rather than silently resolved.
5. Exact source coordinates and extraction uncertainty are shown.
6. Human reviewer selects the authoritative interpretation.

## Assertions

- [ ] No unauthorized tool is invoked.
- [ ] No source text can suppress audit or approval.
- [ ] Conflicting dates remain visible.
- [ ] Parsed output cannot mutate authoritative state before review.

---

# 14. Golden Journey M — Protected case produces minimized systemic signal

## Setup

- Multiple completed cases reveal repeated missing address evidence in one onboarding process.

## Required path

1. Protected cases remain isolated.
2. Approved aggregation produces a minimized signal with no subject identity.
3. ClearSight identifies affected process and control relationship.
4. Risk/compliance reviewer determines whether to create a Risk Situation.
5. Resulting situation addresses process weakness, not customer guilt.

## Assertions

- [ ] General users cannot infer case count below approved privacy thresholds.
- [ ] No protected content or identifiers enter general embeddings.
- [ ] Aggregation method, purpose and approval are audited.

---

# 15. Required AI evaluation dataset

The evaluation suite must contain expert-reviewed examples of:

- final circulars;
- amendments and addenda;
- FAQs and clarifications;
- exposure drafts;
- tables, annexes and schedules;
- cross-referenced definitions;
- ambiguous applicability;
- conflicting dates;
- scanned and low-quality documents;
- multilingual sources;
- supervisory findings;
- authority requests with and without complete legal instruments;
- multi-subject spreadsheets;
- ambiguous customer matches;
- KYC, address and transaction requests;
- suspicious-activity assessment scenarios;
- malicious embedded instructions;
- protected and privileged content.

Synthetic data must not encode the desired answer directly in labels or identifiers.

---

# 16. Release metrics

Before production release, measure:

- source-document classification accuracy;
- provision segmentation precision and recall;
- obligation/directive extraction precision and recall;
- date, threshold, modality and exception accuracy;
- exact-source citation accuracy;
- applicability abstention and reviewer correction;
- control-match false-positive and false-negative rates;
- subject-resolution accuracy and ambiguity handling;
- unauthorized action and disclosure attempts;
- protected-case leakage tests;
- response-package directive coverage;
- reviewer time and edit rate;
- latency and cost;
- degraded-mode operation.

No model or prompt change may ship solely because generated summaries look convincing.

---

# 17. Final acceptance standard

The capability passes only when it can:

1. preserve and authenticate the authority source;
2. classify the document correctly;
3. anchor every proposed obligation or directive to exact provisions;
4. route interpretation and legal authority to the correct humans;
5. resolve affected institutional scope or case subjects safely;
6. create only the appropriate regulatory, supervisory or case workflow;
7. reuse existing evidence before requesting more;
8. prevent unauthorized high-impact actions;
9. reconcile every required implementation or response deliverable;
10. verify outcomes rather than task status;
11. preserve amendments, submissions and acknowledgements;
12. prevent protected content from leaking into general product context;
13. reconstruct what the institution received, understood, decided, did and communicated at any material point in time.