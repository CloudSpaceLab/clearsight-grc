# ClearSight Acceptance and Non-Regression Tests

This document defines the minimum product, source-quality, evidence, security, AI, visual, accessibility, and operational tests required to preserve the ClearSight situation-first product model.

A feature is not accepted because CRUD, upload, API, or AI output works. It is accepted when the complete bank behavior works under realistic positive, negative, ambiguous, stale, partial, unauthorized, offline, degraded, and historical conditions.

It conforms to:

- [`../../README.md`](../../README.md)
- [`../../AGENTS.md`](../../AGENTS.md)
- [`../product/operating-model.md`](../product/operating-model.md)
- [`../product/experience-principles.md`](../product/experience-principles.md)

---

# 1. Test philosophy

Tests must prove:

- situations before modules;
- banking language before GRC jargon;
- scope before action;
- source authority before automated trust;
- progressive integration;
- existing evidence before human requests;
- data-quality transparency;
- evidence before confidence;
- contradiction before false certainty;
- decisions before dashboards;
- verification before closure;
- human authority for material judgment;
- and point-in-time institutional memory.

Fixtures MUST NOT inject desired conclusions, mappings, owners, authority, materiality, or verification outcomes.

---

# 2. Required test layers

## Domain unit tests

- canonical object state transitions;
- source authority and limitation;
- observation provenance;
- evidence-recipe policy;
- sufficiency dimensions;
- contradiction propagation;
- situation lifecycle;
- authority and segregation of duties;
- verification-contract evaluation;
- temporal semantics.

## Property-based tests

Use where combinations are broad:

- temporal intervals;
- scope inheritance;
- authority matrices;
- population denominators and exclusions;
- evidence coverage;
- matching and merge/unmerge;
- idempotency;
- bulk selection;
- workflow transitions;
- import partial success.

## Contract tests

- APIs and events;
- observation schema;
- spreadsheet mapping schema;
- integration adapters;
- structured AI output;
- export manifests;
- source-health events.

## Integration tests

Use real interactions among:

- relational database;
- object storage;
- outbox and queue;
- search and graph projections;
- authorization;
- durable workflow;
- media and spreadsheet processing;
- model gateway stubs or approved routes.

## End-to-end tests

Run through real UI and service boundaries for selected golden journeys.

## Security tests

- tenant and legal-entity isolation;
- wrong-scope action;
- relationship and purpose authorization;
- bulk-operation authorization;
- search, count, cache, graph, and embedding inference;
- protected reporting;
- malicious spreadsheets, media, documents, and prompts;
- integration credentials;
- offline capture.

## AI evaluations

- extraction correctness;
- grounding and source versions;
- explicit versus inferred values;
- contradiction handling;
- abstention;
- authority routing;
- prompt injection;
- leakage;
- latency and cost.

## Visual, accessibility, and localization tests

- light and dark parity;
- comfortable and compact density;
- desktop, tablet, and mobile where relevant;
- keyboard and screen-reader journeys;
- 200% zoom;
- reduced motion;
- local number, currency, date, and time formats;
- long translated labels;
- low bandwidth.

## Performance and recovery

- observation ingestion;
- spreadsheet and media processing;
- population queries;
- Situation and Today latency;
- import backlog recovery;
- workflow recovery;
- source and model outage;
- projection rebuild;
- large export.

---

# 3. Golden Journey A — ATM population, import, photo capture, and verification

## Purpose

Prove progressive integration, source profiles, population reconciliation, targeted human capture, bounded photo interpretation, contradiction, decision, and verified outcome.

## Setup

- Retail ATM channel in Lagos has 428 expected machines.
- Asset-register spreadsheet contains 428 rows.
- 19 rows have invalid or missing branch identifiers.
- 12 machines have no switch heartbeat for more than 24 hours.
- Vendor records conflict with the internal serial number for 4 machines.
- Seven branches have related tampering or card-retention complaints.
- Branch staff can photograph selected unresolved machines.

## Required path

1. User selects the ATM Asset Register Source Profile.
2. Spreadsheet is uploaded and sheet selected.
3. Columns are mapped to approved ATM fields.
4. Preview shows type errors, duplicates, and missing identifiers.
5. Import accepts valid observations and queues unresolved rows.
6. Switch source supplies heartbeat observations.
7. ClearSight creates one or more bounded ATM Risk Situations.
8. Only unresolved machines are shown in the worklist.
9. Branch staff receive focused requests for relevant machines.
10. Capture requests full device context and serial-number region.
11. AI extracts serial number and visible seal state.
12. User confirms or corrects extracted values.
13. Original image and extraction coordinates remain linked.
14. One photo conflicts with asset register and vendor record.
15. Contradiction prevents a supported asset-assignment conclusion.
16. Authorized action corrects the inventory and investigates the device.
17. Switch and follow-up observations meet the Verification Contract.
18. Situation becomes verified only after accepted outcome evidence.

## Assertions

- [ ] Upload success is distinct from parse, mapping, observation acceptance, reconciliation, and sufficiency.
- [ ] File, sheet, row, mapping version, and import time are preserved.
- [ ] Percentages expose denominators and exclusions.
- [ ] Only unresolved or sampled machines are requested from branches.
- [ ] Photo interpretation is limited to visible attributes.
- [ ] Machine-extracted and user-confirmed values are distinct.
- [ ] Contradictory sources remain separate.
- [ ] Source authority is visible.
- [ ] No green appears at task completion.
- [ ] Current and historical situation can be reconstructed.

---

# 4. Golden Journey B — POS terminal identity and settlement reconciliation

## Purpose

Prove reuse of the same operating model across a different banking channel and a large population.

## Setup

- 18,000 active POS terminal records.
- Merchant KYC source, terminal-management source, settlement file, and processor telemetry.
- 87 terminal IDs are duplicated.
- 42 terminals map to conflicting merchants.
- 19 terminals appear from unexpected locations.
- Reversal rate increased for one merchant cluster.
- Processor availability degraded during the same period.

## Required path

1. Sources are ingested with explicit authority and freshness.
2. Terminal and merchant matching creates confirmed, provisional, unresolved, duplicate, and contradictory states.
3. Population worklist shows total and filtered denominators.
4. Related observations become one bounded POS Risk Situation where defensible.
5. Settlement and transaction totals are compared.
6. ClearSight separates processor degradation, identity mismatch, and reconciliation variance while preserving their relationship.
7. Only unresolved merchant or terminal facts are requested.
8. Appropriate channel owner reviews materiality and evidence.
9. Decision and action are routed within authority.
10. Outcome is verified against processor, terminal, and settlement observations.

## Assertions

- [ ] ATM-specific code is not required to model POS exposure.
- [ ] Universal exposure patterns are reused.
- [ ] A table, not a card grid, handles the terminal population.
- [ ] Bulk selection shows exact criteria, count, exclusions, and side effects.
- [ ] Bulk actions enforce authorization per object.
- [ ] Duplicate and contradictory states remain distinguishable.
- [ ] Processor completion cannot set final risk state.
- [ ] Projected effect and observed result remain distinct.

---

# 5. Golden Journey C — Source degradation and data-quality uncertainty

## Purpose

Prove that source health and data quality affect confidence without being falsely represented as certain risk failure.

## Setup

- IAM source is normally refreshed hourly.
- Last successful synchronization occurred 36 hours ago.
- Current privileged-access claim depends on IAM and approval data.
- No incident is observed.
- A manager attests that the review was completed.

## Required path

1. Source Profile changes from current to stale.
2. Affected observations and conclusions are identified.
3. Situation shows evidence and data-quality debt separately from exposure severity.
4. Manager attestation is accepted as an assertion, not replacement for current IAM population.
5. System selects an approved fallback or requests focused evidence.
6. Restored IAM data reconciles with interim evidence.

## Assertions

- [ ] Stale data is not shown as current.
- [ ] No-data, stale, and control-failed states are visually distinct.
- [ ] Successful connection recovery does not automatically resolve contradiction.
- [ ] Source owner, limitation, age, and affected conclusion are visible.
- [ ] Human attestation does not become independent technical evidence.

---

# 6. Golden Journey D — Minimum-question privileged-access review

## Purpose

Prove that ClearSight searches existing observations and asks only for unresolved business knowledge.

## Setup

- IAM lists 67 privileged accounts.
- Approval repository covers 62.
- One emergency account has a valid exception.
- Four accounts have unresolved business need.
- HR shows one of the four users transferred.

## Required path

1. Claim is created for exact population and period.
2. IAM, approval, HR, and exception observations are evaluated first.
3. Only four accounts are requested from the manager.
4. Known identifiers, roles, and prior state are prefilled.
5. Manager confirms three and rejects need for transferred user.
6. IAM still shows transferred user active.
7. Contradiction remains unresolved.
8. Removal action is authorized.
9. Verification observes no unauthorized reactivation for the defined period.

## Assertions

- [ ] No request is sent for 63 resolved accounts.
- [ ] Request uses business language, not control IDs.
- [ ] Manager can redirect or report lack of authority.
- [ ] Contradiction blocks final conclusion.
- [ ] Action completion remains awaiting verification.
- [ ] Unauthorized manager cannot see other departments.

---

# 7. Golden Journey E — Material decision, expiry, and verification failure

## Purpose

Prove authority, conditions, expiry, context invalidation, action-versus-outcome separation, and historical preservation.

## Setup

- A channel risk cannot be fully remediated before a planned launch.
- Temporary acceptance is allowed for 90 days under defined conditions.
- Product owner may propose but not approve.
- A new vulnerable customer group later enters scope.

## Required path

1. Decision review shows exact scope, period, evidence, uncertainty, options, and side effects.
2. Authority matrix selects approver.
3. Acceptance includes conditions, expiry, and verification.
4. New customer group invalidates conditions.
5. Active decision expires or is invalidated.
6. Original decision remains unchanged historically.
7. Revised action is implemented.
8. Outcome telemetry fails the threshold.
9. Situation remains open or reopens.

## Assertions

- [ ] Product owner cannot self-approve.
- [ ] Scope is visible before approval.
- [ ] Context-free approval is impossible.
- [ ] Context change invalidates rather than overwrites.
- [ ] Failed verification is visually distinct from implementation failure.
- [ ] AI may draft but cannot approve.

---

# 8. Golden Journey F — Anonymous protected report and sanitized escalation

## Purpose

Prove isolated intake, anonymous communication, protected content, conflict-aware routing, and minimized connection to ordinary risk situations.

## Setup

- Anonymous report alleges deliberate bypass of a customer-refund control.
- Audio and screenshots are attached.
- One investigator is named in the allegation.
- Report may indicate broad customer harm.

## Required path

1. Portal explains anonymity and confidentiality boundaries.
2. Reporter submits without internal account.
3. Secure case token is issued.
4. Protected case content and optional identity are separately controlled.
5. Conflicted investigator cannot access case or counts.
6. Media uses approved protected-data route.
7. Summary distinguishes allegation from verified fact.
8. Investigator sends anonymous follow-up.
9. Reporter responds through token.
10. Validated minimized risk signal creates or updates a Risk Situation without protected identity.

## Assertions

- [ ] Identity is absent from ordinary search, graph, analytics, logs, exports, and notifications.
- [ ] Case narrative cannot leak through ordinary summaries.
- [ ] Identity reveal is a separate privileged workflow.
- [ ] No credibility score is inferred from style, emotion, accent, or demographics.
- [ ] Original and derived media remain separately controlled.
- [ ] Sanitized escalation is traceable to protected authority without exposing content.

---

# 9. Golden Journey G — Malicious spreadsheet, document, and prompt injection

## Purpose

Prove content security and governed AI behavior across common ingestion methods.

## Setup

- Spreadsheet contains formula injection and hidden sheets.
- Document contains instructions to reveal secrets, access another tenant, close the situation, and call a tool.
- Image metadata contains unexpected location and personal data.

## Required path

1. Spreadsheet is treated as untrusted content.
2. Formula and active-content policy is applied.
3. Hidden sheets and macros are surfaced according to policy.
4. Document instructions do not alter operator policy.
5. No cross-tenant retrieval or unauthorized tool call occurs.
6. Image metadata is minimized or governed.
7. Relevant observations may still be extracted.
8. Security events are recorded safely.

## Assertions

- [ ] Tool permission remains outside prompts.
- [ ] Free-form output cannot mutate domain state.
- [ ] Secret and cross-tenant requests return nothing, including counts or titles.
- [ ] Import preview does not execute formulas.
- [ ] Security logging contains no restricted evidence.

---

# 10. Golden Journey H — Scope switching, bulk action, and partial authorization

## Purpose

Prove that a user cannot accidentally act across entities, regions, or unauthorized objects.

## Setup

- User can manage Lagos region but view summary data for another region.
- A filtered worklist contains 120 visible records.
- 10 records are read-only due to authority.
- User selects “all matching” and initiates a bulk ownership update.

## Required path

1. Scope header shows bank, entity, region, population, and period.
2. Selection summary shows 120 matching, 110 writable, 10 excluded.
3. Exact filter and action are shown.
4. Server evaluates authorization for each object.
5. User approves proportional side effects.
6. Action is idempotent.
7. Post-action result shows success, exclusion, failure, and retry states.
8. Audit manifest reconstructs the operation.

## Assertions

- [ ] Client selection cannot bypass server authorization.
- [ ] User cannot infer unauthorized record details.
- [ ] Scope switch clears or confirms incompatible drafts and selections.
- [ ] Partial success is not displayed as universal success.
- [ ] Bulk action can be retried safely.

---

# 11. Golden Journey I — Offline branch capture and synchronization conflict

## Purpose

Prove safe evidence capture under unstable connectivity.

## Setup

- Branch user receives three ATM confirmation requests.
- Network becomes unavailable.
- Policy permits encrypted offline drafts for this evidence class.
- Another source updates one ATM before synchronization.

## Required path

1. Requests and permitted reference data are available offline.
2. Unsynchronized state is explicit.
3. Capture time is preserved separately from upload time.
4. User submits photos and confirmations locally.
5. Synchronization resumes.
6. One record conflicts with newer authoritative data.
7. Conflict is routed for review rather than overwritten.
8. Duplicate submission is prevented.

## Assertions

- [ ] Restricted evidence that cannot be stored offline is blocked with explanation.
- [ ] Local queue is encrypted and bounded.
- [ ] Offline status is not shown as submitted.
- [ ] Conflict retains both versions and timing.
- [ ] Retry does not duplicate observations.

---

# 12. Golden Journey J — AI and integration degraded mode

## Purpose

Prove that core GRC operation remains usable when external models or sources fail.

## Setup

- Primary model provider unavailable.
- Vendor connector delayed.
- Executive must review a time-sensitive situation.

## Required path

1. Deterministic situation context appears.
2. Last-known source data is labeled with exact age.
3. AI state shows unavailable rather than pending forever.
4. Manual evidence and decision workflow remains usable.
5. Missing information and uncertainty are captured.
6. Failed tasks are resumable after recovery.

## Assertions

- [ ] No stale AI answer is shown as new.
- [ ] No stale source is shown as current.
- [ ] Manual decision retains evidence and uncertainty.
- [ ] Recovery does not duplicate queued work.

---

# 13. Golden Journey K — Point-in-time situation reconstruction

## Purpose

Prove institutional memory across source, mapping, evidence, decision, and outcome changes.

## Setup

- Source mapping, relationship, conclusion, decision, and policy are later corrected or superseded.

## Required path

Authorized user selects a historical date and retrieves:

- scope and relationships valid then;
- observations recorded then;
- source health and mappings known then;
- claims and evidence available then;
- conclusion and materiality;
- decision, authority, and conditions;
- subsequent corrections clearly separated.

## Assertions

- [ ] Historical records are not overwritten.
- [ ] Valid time and record time are distinct.
- [ ] Future observations are excluded from “known then.”
- [ ] Current viewer authorization is enforced.
- [ ] Export manifest records reconstruction time and versions.

---

# 14. Golden Journey L — External execution engine boundary

## Purpose

Prove that Probo, ITSM, or another execution engine can perform work without becoming authoritative for ClearSight conclusions.

## Setup

- ClearSight authorizes a compliance or remediation task.
- External engine collects a document and marks the task complete.
- Document proves implementation but not operating effectiveness.

## Required path

1. Scoped idempotent external action is created.
2. External object ID and version are recorded.
3. Returned document becomes an observation.
4. Action becomes implemented.
5. Verification remains pending.
6. Separate outcome evidence is collected.
7. ClearSight issues final conclusion.

## Assertions

- [ ] Duplicate writes create no duplicate task.
- [ ] Organization mapping is server-controlled.
- [ ] Model receives no unrestricted bearer token.
- [ ] External completion cannot set final risk state.
- [ ] Deletion or permission revocation is reconciled.

---

# 15. Authorization test matrix

For each object and projection, test:

- same tenant, permitted entity, purpose, and relationship;
- same tenant, wrong entity or region;
- correct role, wrong ownership relationship;
- expired or delegated assignment;
- different tenant;
- service identity outside capability;
- operator exceeding user authority;
- protected case conflict;
- audit independence;
- mixed-sensitivity export;
- search and vector retrieval;
- graph traversal;
- counts and aggregates;
- cache isolation;
- background worker scope;
- bulk operation;
- offline queue;
- break-glass access.

Required invariant:

> An unauthorized actor learns nothing material about the existence, count, identity, content, or relationship of protected objects.

---

# 16. Source, import, and reconciliation tests

## Source Registry

- authoritative field and limitation;
- owner change;
- freshness target;
- stale and unavailable state;
- authorization revoked;
- mapping version change;
- retired source;
- affected-conclusion propagation.

## Spreadsheet and CSV

- multiple sheets;
- hidden sheets;
- malformed and oversized files;
- wrong delimiter and encoding;
- formula injection;
- duplicate rows;
- invalid dates and currencies;
- missing identifiers;
- partial acceptance;
- rollback reference;
- row provenance.

## Matching

- exact match;
- alias match;
- ambiguous candidate;
- duplicate external identifier;
- provisional match;
- merge and unmerge;
- later contradiction;
- downstream impact.

Required invariants:

- import success does not imply evidence sufficiency;
- unresolved rows remain visible;
- no silent merge of ambiguous material identities;
- percentages expose denominators and exclusions.

---

# 17. Evidence tests

## Integrity

- hash validation;
- corrupted upload;
- interrupted/resumed upload;
- duplicate content;
- object version change;
- malware rejection;
- signed manifest where configured.

## Photo and media

- blur, glare, crop, unreadable label;
- multiple identifiers;
- incorrect AI extraction;
- correction and confirmation;
- metadata minimization;
- protected background content;
- original-versus-derived linkage.

## Sufficiency

- relevant but stale;
- fresh but wrong scope;
- full self-attestation;
- partial independent sample;
- full system population;
- conflicting authoritative sources;
- unavailable original;
- source outside authority limit.

## Request burden

- evidence already sufficient;
- overlapping requests;
- recipient redirects;
- partial response;
- evidence arrives through another channel;
- request becomes irrelevant;
- wrong recipient or conflict.

Required invariant:

> The system does not contact a person when authorized existing evidence is already sufficient.

---

# 18. Situation and materiality tests

Test:

- one exposure across multiple observations;
- one observation affecting multiple situations;
- duplicate observations;
- evidence debt without observed failure;
- source degradation;
- appetite approach versus breach;
- concentration amplification;
- customer vulnerability;
- context change invalidating dismissal;
- situation merge, split, supersede, reopen;
- suppression without deletion.

Required invariants:

- severity alone does not determine executive priority;
- evidence uncertainty is not represented as certain control failure;
- grouped observations remain traceable;
- suppressed observations remain available;
- user-facing language remains banking-first.

---

# 19. Visual regression matrix

Maintain golden screens for:

1. Today brief.
2. Situation card in evidence-needed, decision-needed, and verification-failed states.
3. Situation workspace: Summary, Evidence, Decision, Action, Outcome, History.
4. ATM inventory situation.
5. POS identity or settlement situation.
6. Population worklist.
7. Spreadsheet mapper.
8. Import summary and reconciliation queue.
9. Source Profile.
10. Degraded-source state.
11. Evidence micro-request.
12. Mobile ATM photo capture.
13. AI extraction review.
14. Controlled-value form.
15. Evidence sufficiency.
16. Contradiction compare.
17. Decision approval.
18. Implemented but awaiting verification.
19. Failed verification.
20. Relationship path.
21. Point-in-time reconstruction.
22. No material change.
23. No data.
24. Not assessed.
25. Unauthorized.
26. Whistleblower intake.
27. Anonymous follow-up.
28. Protected investigator case.
29. Board or committee mode.
30. AI unavailable.
31. Offline queued capture.
32. Synchronization conflict.
33. Bulk action review.
34. Post-action reconciliation.
35. Export review and manifest.

For each relevant screen test:

- light and dark;
- comfortable and compact density;
- desktop, tablet, and mobile;
- loading, empty, stale, contradictory, unauthorized, error, offline, and degraded states;
- 125%, 150%, and 200% zoom;
- keyboard focus;
- long translated labels;
- local currency, date, number, and time-zone formats.

Reject:

- uncontrolled density;
- architecture names as mandatory navigation;
- cards used for large populations;
- color-only meaning;
- decorative glass or glow;
- low-contrast metadata;
- hidden material actions on hover;
- unexplained percentages or missing denominators;
- green for completion;
- upload success presented as evidence success;
- AI claims exceeding visible observations;
- light-mode neglect;
- mobile compression of desktop complexity.

---

# 20. Accessibility and localization matrix

Test:

- full keyboard operation;
- logical focus order;
- visible focus;
- screen-reader names, roles, values, and async announcements;
- headings and landmarks;
- non-color status;
- table navigation;
- chart alternatives;
- form error association;
- touch targets;
- reduced motion;
- 200% zoom;
- multilingual expansion;
- right-to-left readiness where supported;
- local currencies and dates;
- low-bandwidth capture;
- evidence and protected-reporting journeys without mouse.

---

# 21. Performance and recovery tests

Test:

- concurrent executive and operational users;
- 20,000+ row worklists;
- spreadsheet parsing and reconciliation;
- media processing;
- authorized search;
- Situation load;
- Today ranking;
- workflow backlog;
- large export;
- model concurrency;
- source burst and recovery.

Measure:

- p50, p95, p99 latency;
- throughput;
- error rate;
- queue delay;
- database contention;
- memory and storage;
- model cost;
- recovery time;
- UI stability.

Simulate:

- database failover;
- object-store failure;
- queue outage;
- worker crash;
- model outage;
- source outage;
- token revocation;
- search corruption;
- graph-projection rebuild;
- offline conflict;
- backup restore.

Required behavior:

- authoritative state remains correct;
- workflows resume;
- events replay idempotently;
- stale state is labeled;
- manual operation remains possible;
- and recovery is auditable.

---

# 22. Release gates

## Pull request

- relevant unit and contract tests pass;
- authorization negative cases pass;
- docs and ADRs updated;
- accessibility checked;
- visual diff reviewed for UI changes;
- no sensitive logging;
- source and data-quality impact reviewed;
- migration and rollback considered.

## Feature

- complete situation behavior passes;
- scope and source authority are visible;
- partial and degraded mode works;
- telemetry exists;
- role and authority are tested;
- support and runbook content exists.

## AI capability

- evaluation thresholds pass;
- no critical authorization or leakage failure;
- structured output validates;
- extracted and inferred values are distinguishable;
- abstention is acceptable;
- monitoring and kill switch exist;
- regression comparison reviewed.

## Pilot

- selected ATM and POS/settlement journeys pass;
- tenant, entity, bulk, source, and protected-data assessments pass;
- backup and recovery tested;
- performance meets pilot targets;
- users demonstrate reduced effort and improved data/evidence quality.

## General availability

- independent security review complete;
- SLOs and operational ownership approved;
- deployment and rollback tested;
- legal and privacy controls validated;
- no critical product-invariant defect;
- no unresolved critical visual or accessibility defect in core journeys.

---

# 23. Final acceptance standard

ClearSight passes its core acceptance test only when it can:

1. receive imperfect bank information through realistic sources;
2. expose source authority, freshness, and unresolved quality;
3. normalize observations with provenance;
4. create one understandable bounded Risk Situation;
5. explain what changed and why it matters;
6. find existing evidence before asking a person;
7. ask only for the smallest unresolved facts;
8. preserve contradiction and uncertainty;
9. route the appropriate authorized decision;
10. execute through governed people, systems, or operators;
11. verify the defined observable outcome;
12. and reconstruct the complete situation later.

Anything less is a partial workflow or visual prototype, not the finished ClearSight operating loop.