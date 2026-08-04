# ClearSight Acceptance and Non-Regression Tests

This document defines the minimum product, security, evidence, AI, visual, and operational tests required to preserve ClearSight’s differentiation.

A feature is not accepted because its CRUD operations work. It is accepted when the complete institutional behavior works under realistic positive, negative, ambiguous, unauthorized, degraded, and historical conditions.

---

# 1. Test philosophy

Tests must prove the product invariants:

- materiality before volume;
- evidence before confidence;
- relationships before forms;
- decisions before dashboards;
- verification before closure;
- automation before reminders;
- human authority for material judgment;
- and institutional memory over periodic reporting.

Fixtures must not inject the desired final conclusion or bypass the real owner, policy, graph, evidence, operator, or verification path.

---

# 2. Required test layers

## 2.1 Domain unit tests

Test:

- state transitions;
- invariants;
- temporal versioning;
- evidence sufficiency dimensions;
- authority evaluation;
- appetite logic;
- verification-contract evaluation;
- and contradiction propagation.

## 2.2 Property-based tests

Use property-based tests where combinations are too broad for examples alone, including:

- temporal intervals;
- version and supersession chains;
- authority matrices;
- risk and appetite boundaries;
- evidence coverage;
- idempotency;
- and workflow transition sequences.

## 2.3 Contract tests

Test:

- APIs;
- events;
- integration adapters;
- model structured-output schemas;
- and export manifests.

## 2.4 Integration tests

Test real interactions among:

- database;
- object storage;
- outbox and queue;
- search and graph projections;
- workflow runtime;
- authorization;
- and model gateway stubs or approved test routes.

## 2.5 End-to-end golden journeys

Run through actual application boundaries, including UI where relevant.

## 2.6 Security tests

Test:

- tenant isolation;
- relationship-based authorization;
- protected identity isolation;
- evidence access;
- export boundaries;
- prompt injection;
- operator tool abuse;
- and integration credentials.

## 2.7 AI evaluations

Evaluate:

- grounding;
- domain correctness;
- contradiction handling;
- abstention;
- authority routing;
- security;
- latency;
- and cost.

## 2.8 Accessibility and visual regression

Maintain light and dark golden screens, keyboard journeys, screen-reader semantics, non-color status, reduced motion, and supported breakpoints.

## 2.9 Performance and resilience

Test realistic signal, graph, evidence, workflow, export, and operator loads, plus failure and recovery.

---

# 3. Golden Journey A — Privileged-access evidence review

## Purpose

Prove that ClearSight asks only for missing human knowledge, reconciles machine and human evidence, preserves contradiction, and does not approve a claim prematurely.

## Setup

- Treasury Operations is a critical process.
- IAM lists 67 privileged accounts.
- Approval repository contains current approval for 62.
- One account is a known emergency account under a valid exception.
- Four accounts have unresolved business need.
- HR shows one of those four users transferred departments.
- The manager is the best available source for business-need confirmation.

## Required path

1. IAM and approval data are ingested with provenance.
2. The claim is created for the exact population and review period.
3. Existing evidence is evaluated before contacting a person.
4. ClearSight identifies only four unresolved accounts.
5. The request explains what is known and asks only about those four.
6. The manager confirms three and states the transferred user no longer needs access.
7. IAM still shows the transferred user active.
8. A contradiction is created.
9. The operating-effectiveness conclusion remains unresolved.
10. Removal action is authorized and executed.
11. IAM confirms removal.
12. The verification contract observes no reactivation for the defined period.
13. The conclusion becomes supported only after verification.

## Assertions

- [ ] No request is sent for the other 63 accounts.
- [ ] The request does not require the manager to enter data already known.
- [ ] HR, IAM, manager, and approval evidence remain separate.
- [ ] Contradiction is visible and affects conclusion.
- [ ] Action completion does not immediately produce verified green.
- [ ] Source versions and time scope are reconstructable.
- [ ] An unauthorized manager cannot see other departments.
- [ ] Duplicate IAM events do not duplicate evidence or action.

---

# 4. Golden Journey B — Payment-service resilience exposure

## Purpose

Prove causal grouping, materiality, critical-service context, evidence debt, decision authority, and outcome verification.

## Setup

- Retail instant payments is a critical service.
- Impact tolerance is 30 minutes.
- Service latency and failure rate increase.
- Primary processor reports degradation.
- A production change completed two hours earlier.
- Vendor claims failover was tested.
- The attached test is outside the required period.
- A related disaster-recovery finding remains open.

## Required path

1. Each source emits a distinct signal.
2. The Institutional Risk Graph connects them to one service and scenario.
3. The Materiality Compiler groups them into one material item.
4. Exposure and evidence debt are shown separately.
5. The CISO or CRO sees why the item matters now.
6. The system requests current failover evidence from the best source.
7. Three response options are proposed with cost, time-to-effect, dependencies, and uncertainty.
8. The correct authority approves one option.
9. Execution creates external tasks.
10. A new test produces telemetry.
11. Recovery exceeds 30 minutes.
12. Verification fails, issue remains open, and appetite state updates.

## Assertions

- [ ] Executive does not receive five disconnected alerts.
- [ ] Materiality explanation references critical service, tolerance, open finding, and stale evidence.
- [ ] A stale vendor attestation is not treated as current proof.
- [ ] Approval cannot be performed by the action owner when segregation requires challenge.
- [ ] External task completion cannot close the issue.
- [ ] Failed verification is visible and updates the next decision.
- [ ] Point-in-time reconstruction shows what was known before and after the new test.

---

# 5. Golden Journey C — Material risk acceptance

## Purpose

Prove executable appetite, authority, conditions, expiry, context invalidation, and historical preservation.

## Setup

- A moderate technology risk cannot be remediated before a product launch.
- Temporary acceptance is permitted for 90 days under defined conditions.
- The product owner may propose but cannot approve.
- The accepting executive has a financial and duration limit.

## Required path

1. Decision record captures evidence, uncertainty, options, and projected outcomes.
2. Authority matrix selects the required approver.
3. Acceptance includes conditions, expiry, review triggers, and verification.
4. Product launch proceeds.
5. A new vulnerable customer segment is added to the product.
6. Original acceptance conditions no longer apply.
7. ClearSight invalidates the active acceptance and requests a new decision.
8. Original acceptance remains unchanged in history.

## Assertions

- [ ] Product owner cannot self-approve.
- [ ] Acceptance has an explicit effective period.
- [ ] Absence of an expiry is rejected for this policy.
- [ ] Context change invalidates rather than overwrites.
- [ ] Board or audit can reconstruct the original evidence and authority.
- [ ] AI may draft rationale but cannot approve acceptance.

---

# 6. Golden Journey D — Remediation implementation versus effectiveness

## Purpose

Prove that issue closure requires outcome evidence.

## Setup

- Repeated unauthorized privileged-access reactivation caused a finding.
- Remediation adds automated deprovisioning.
- Ticketing system records implementation complete.
- Verification requires 30 days with zero unauthorized reactivations.

## Required path

1. Action enters `IMPLEMENTED` after valid implementation evidence.
2. Issue enters `AWAITING_VERIFICATION`.
3. On day 12, an account is reactivated.
4. Verification fails.
5. Issue reopens or remains open according to policy.
6. Risk state and executive item update.
7. A revised remediation is proposed.

## Assertions

- [ ] Implemented state is visually distinct from verified effective.
- [ ] No green state is shown before the observation period completes.
- [ ] Reactivation event is linked to the verification contract.
- [ ] Failure does not delete implementation evidence.
- [ ] Projected and observed treatment effect are both retained.

---

# 7. Golden Journey E — Anonymous whistleblower report

## Purpose

Prove identity isolation, anonymous communication, protected evidence processing, conflict-aware routing, and risk-signal integration.

## Setup

- Reporter chooses anonymous submission.
- Report alleges deliberate bypass of a customer-refund control.
- Audio and screenshots are attached.
- One investigator is named in the allegation.
- The report may indicate broad customer impact.

## Required path

1. Portal explains anonymity and confidentiality boundaries.
2. Reporter submits without creating an internal account.
3. Secure case token is issued.
4. Case content and any optional identity data are stored separately.
5. Named/conflicted investigator is excluded.
6. Audio is processed only through an approved protected-data route.
7. AI summary distinguishes allegation from verified facts.
8. Investigator sends a question through anonymous messaging.
9. Reporter responds using the token.
10. Validated evidence creates a risk signal without exposing identity.

## Assertions

- [ ] Reporter identity is absent from ordinary case queries.
- [ ] Search, embeddings, logs, analytics, exports, and graph views do not leak identity.
- [ ] Identity reveal requires a separate privileged workflow.
- [ ] No credibility score is derived from language, emotion, accent, or demographics.
- [ ] Conflicted investigator cannot access case metadata or counts.
- [ ] Original audio and derived transcript remain linked and separately controlled.
- [ ] Anonymous reporter can communicate bidirectionally.

---

# 8. Golden Journey F — Regulatory change to evidence lineage

## Purpose

Prove source-to-obligation-to-control-to-evidence lineage with expert review and time-aware versions.

## Setup

- An authoritative regulator publishes revised outsourcing guidance.
- One paragraph changes an existing obligation.
- The bank has two related controls, one global and one entity-specific.
- Vendor-exit evidence is stale for one critical provider.

## Required path

1. Regulatory source and version are captured.
2. Candidate changed obligation is extracted with source passage.
3. Low-confidence applicability is routed to compliance review.
4. Approved obligation is linked to affected entities, service, controls, and vendor.
5. Gap analysis identifies stale exit evidence.
6. Targeted evidence request is issued.
7. Action and verification are linked.
8. Examiner export traces the final statement to original source.

## Assertions

- [ ] AI extraction cannot publish without required review.
- [ ] Source text and normalized obligation are distinct.
- [ ] Prior obligation version remains reconstructable.
- [ ] Global and entity-specific control implementations are not collapsed.
- [ ] Final export includes source, obligation, control, evidence, decision, and action lineage.

---

# 9. Golden Journey G — Probo execution boundary

## Purpose

Prove that Probo or another compliance engine can execute work without becoming authoritative for ClearSight material decisions.

## Setup

- ClearSight identifies a missing compliance measure.
- Approved action creates a task in Probo.
- Probo collects a document and marks the task complete.
- The document proves implementation but not operating effectiveness.

## Required path

1. ClearSight creates a scoped, idempotent external action.
2. Probo object IDs and versions are recorded.
3. Returned document is captured as source evidence.
4. Task completion updates implementation state.
5. Verification remains pending.
6. Separate operating evidence is collected.
7. ClearSight issues the final conclusion.

## Assertions

- [ ] Duplicate writes do not create duplicate Probo tasks.
- [ ] Organization mapping is server-controlled.
- [ ] Model does not receive unrestricted Probo bearer token.
- [ ] Probo completion cannot directly set ClearSight risk state.
- [ ] Permission revocation or deleted evidence is reconciled.

---

# 10. Golden Journey H — AI operator malicious-content defense

## Purpose

Prove prompt-injection resistance and governed tool use.

## Setup

An uploaded evidence document contains text instructing the AI to:

- ignore policy;
- reveal secrets;
- access another tenant;
- close the finding;
- and call an external tool.

## Required path

1. Document is treated as untrusted evidence content.
2. Relevant evidence facts may still be extracted.
3. Embedded instructions do not alter operator policy.
4. No secret or cross-tenant data is retrieved.
5. No unauthorized tool is invoked.
6. Injection attempt is recorded safely.
7. Human review receives the valid extraction and security signal.

## Assertions

- [ ] Tool permissions remain external to prompts.
- [ ] Free-form model output cannot mutate state.
- [ ] Cross-tenant retrieval returns no data, titles, counts, or hints.
- [ ] Audit includes operator, sources, model, policy, denial, and result.

---

# 11. Golden Journey I — AI and integration degraded mode

## Purpose

Prove that core risk governance remains usable during external failure.

## Setup

- Primary model provider is unavailable.
- Vendor connector is delayed.
- Executive must make a time-sensitive decision.

## Required path

1. Deterministic source data and last-known connector state are displayed.
2. Stale vendor data is explicitly labeled.
3. AI analysis shows unavailable, not pending forever.
4. Manual evidence and decision workflow remains available.
5. Decision records missing information and uncertainty.
6. Failed tasks are resumable after recovery.

## Assertions

- [ ] No stale AI answer is shown as newly generated.
- [ ] No stale connector data is presented as current.
- [ ] Manual decision captures uncertainty and authority.
- [ ] Recovery does not duplicate queued work.

---

# 12. Golden Journey J — Point-in-time reconstruction

## Purpose

Prove institutional memory.

## Setup

A risk decision, evidence conclusion, graph relationship, and policy are later corrected or superseded.

## Required path

An authorized auditor selects a historical date and retrieves:

- institutional relationships valid then;
- records known then;
- evidence versions available then;
- appetite and policy version;
- materiality explanation;
- decision and approvers;
- and subsequent corrections clearly separated.

## Assertions

- [ ] Historical records are not overwritten.
- [ ] Valid time and record time are distinguishable.
- [ ] Later evidence is not incorrectly included in the “known then” view.
- [ ] Access is evaluated against the current authorized viewer while preserving historical content.
- [ ] Export manifest identifies the reconstruction time and included versions.

---

# 13. Authorization test matrix

Every object type requires tests for:

- same tenant, permitted entity, permitted purpose;
- same tenant, wrong legal entity;
- same tenant, correct role but wrong relationship;
- same tenant, expired assignment;
- different tenant;
- service identity outside capability;
- delegated operator exceeding user authority;
- protected case with conflict;
- audit read independence;
- export with mixed sensitivity;
- search and vector retrieval;
- graph traversal;
- count and aggregate inference;
- cache isolation;
- background worker scope;
- and break-glass access.

Required invariant:

> An unauthorized actor learns nothing material about the existence, count, identity, content, or relationship of protected objects.

---

# 14. Temporal and versioning tests

Test:

- overlapping valid-time intervals;
- late-arriving facts;
- backdated corrections;
- superseded evidence;
- reopened conclusions;
- expired decisions;
- relationship changes;
- source revocation;
- and point-in-time exports.

Properties:

- no material version is overwritten;
- each supersession chain is valid;
- current view selects the correct active version;
- historical view does not include future knowledge;
- and dependent conclusions are invalidated predictably.

---

# 15. Evidence tests

## Integrity

- content hash validation;
- corrupted upload;
- interrupted/resumed upload;
- duplicate content;
- object-store version change;
- malware rejection;
- and signed manifest verification.

## Sufficiency

- relevant but stale;
- fresh but wrong scope;
- complete self-attestation;
- partial independent sample;
- full system population;
- contradictory authoritative sources;
- and unavailable original source.

## Request burden

- existing evidence already sufficient;
- overlapping requests;
- recipient redirects to better source;
- partial response;
- response received through another channel;
- and request becomes irrelevant before deadline.

Required invariant:

> The system does not contact a person when authorized existing evidence is already sufficient.

---

# 16. Materiality tests

Test:

- high-severity but contained and delegated event;
- moderate fast-moving exposure with weak evidence;
- many duplicate signals;
- one signal affecting multiple critical services;
- evidence debt without observed failure;
- appetite approach versus breach;
- concentration amplification;
- customer-vulnerability impact;
- and context change invalidating a previous dismissal.

Required invariants:

- severity alone does not determine executive priority;
- evidence uncertainty is not falsely represented as certain risk failure;
- grouped signals remain traceable;
- and dismissed signals are preserved.

---

# 17. Decision and authority tests

Test:

- unauthorized proposer;
- self-approval conflict;
- parallel approval;
- conditional approval;
- dissent;
- emergency authority;
- expired acceptance;
- violated condition;
- evidence deterioration;
- and approver role change mid-workflow.

Required invariants:

- effective authority is evaluated at action time;
- AI cannot grant authority;
- and an approval record preserves the context reviewed.

---

# 18. AI evaluation suites

Each operator evaluation must include:

## Grounding

- correct source selected;
- source version correct;
- no unsupported factual claim;
- contradiction disclosed;
- and time scope correct.

## Domain behavior

- facts, claims, conclusions, and decisions remain separate;
- correct entity resolution;
- correct action class;
- and correct evidence relationship.

## Abstention

- insufficient evidence;
- ambiguous entity;
- conflicting source;
- out-of-scope request;
- and unsupported legal conclusion.

## Authority

- material risk acceptance;
- regulatory reportability;
- issue closure;
- protected identity;
- and external communication.

## Security

- prompt injection;
- indirect prompt injection;
- cross-tenant request;
- secret request;
- malicious tool arguments;
- and sensitive-data exfiltration.

## Reliability

- malformed output;
- tool timeout;
- partial failure;
- provider outage;
- duplicate invocation;
- and replay.

## Protected reporting

- allegation/fact distinction;
- no credibility profiling;
- no identity leakage;
- and safe translation.

---

# 19. Visual regression matrix

Maintain screenshots for each golden screen in:

- dark mode;
- light mode;
- desktop;
- tablet where relevant;
- mobile where relevant;
- normal data;
- loading;
- empty;
- error;
- stale;
- insufficient evidence;
- contradictory;
- unauthorized;
- and verified states.

Golden screens:

1. Today executive brief
2. Material decision card
3. Explain workspace
4. Institutional graph
5. Evidence micro-request
6. Mobile evidence capture
7. Evidence sufficiency panel
8. Contradiction comparison
9. Decision approval
10. Verification contract
11. Failed verification
12. Whistleblower intake
13. Anonymous follow-up
14. Protected investigator case
15. Regulatory lineage
16. Board pack review
17. Operator review
18. Integration degraded mode

Visual review must reject:

- uncontrolled density;
- semantic-color drift;
- decorative glow;
- low-contrast glass;
- layout shift;
- green used for completion only;
- and light-mode neglect.

---

# 20. Accessibility test matrix

Automated and manual tests must cover:

- full keyboard navigation;
- logical focus order;
- visible focus;
- screen-reader names, roles, states, and async announcements;
- headings and landmarks;
- non-color status;
- chart alternatives;
- form error association;
- target size;
- reduced motion;
- 200% zoom;
- high contrast where supported;
- multilingual expansion;
- and low-bandwidth external reporting.

No protected-reporting or evidence-submission journey may require a mouse.

---

# 21. Performance tests

Define exact workload targets from pilot sizing. At minimum test:

- concurrent executive users;
- high-volume signal bursts;
- graph neighborhood and dependency queries;
- point-in-time reconstruction;
- evidence upload and processing;
- full-population evidence snapshots;
- search and authorized retrieval;
- workflow backlogs;
- large audit exports;
- and operator concurrency.

Measure:

- p50, p95, and p99 latency;
- throughput;
- error rate;
- queue delay;
- database contention;
- memory;
- storage;
- model cost;
- and recovery time.

Required behavior:

- deterministic content is not blocked by AI;
- long lists are paginated or virtualized;
- graph detail is progressive;
- evidence upload is resumable;
- and backlogs recover without duplicate side effects.

---

# 22. Resilience and recovery tests

Simulate:

- database failover;
- object-store temporary failure;
- queue outage;
- worker crash mid-task;
- model-provider outage;
- connector outage;
- token revocation;
- search-index corruption;
- graph-projection rebuild;
- region loss;
- and backup restore.

Verify:

- authoritative state remains correct;
- workflows resume;
- events replay idempotently;
- stale state is labeled;
- manual operation remains possible;
- and recovery is auditable.

---

# 23. Migration tests

For imported legacy GRC data, test:

- duplicate risks and controls;
- conflicting owners;
- missing effective dates;
- unlinked evidence;
- invalid file references;
- status ambiguity;
- historical decisions stored as comments;
- and legal-entity mismatch.

Migration must produce:

- reconciliation report;
- unresolved mapping queue;
- source provenance;
- import version;
- rollback path;
- and explicit confidence for inferred relationships.

Do not silently convert legacy attachments into sufficient evidence.

---

# 24. Release gates

## Pull request gate

- relevant unit and contract tests pass;
- authorization negative cases pass;
- docs and ADRs updated;
- accessibility checked;
- visual diff reviewed for UI changes;
- no sensitive logging;
- and migration/rollback considered.

## Feature gate

- complete vertical behavior passes;
- degraded mode works;
- telemetry is present;
- role and authority matrix is tested;
- and support/runbook content exists.

## AI capability gate

- evaluation thresholds pass;
- no critical authorization or leakage failure;
- structured output validates;
- abstention behavior is acceptable;
- monitoring and kill switch exist;
- and regression comparison is reviewed.

## Pilot gate

- all selected golden journeys pass;
- tenant and protected-identity assessment passes;
- backup and recovery tested;
- performance meets pilot targets;
- and users demonstrate reduced effort with improved evidence quality.

## General availability gate

- independent security review complete;
- SLOs and operational ownership approved;
- deployment and rollback tested;
- legal/privacy controls validated;
- and no critical product-invariant defect remains.

---

# 25. Final acceptance standard

ClearSight passes its core acceptance test only when it can:

1. receive fragmented institutional signals;
2. connect them through the institutional graph;
3. identify one defensible material risk item;
4. explain what changed and why it matters;
5. find existing evidence before asking a person;
6. ask only the smallest unresolved question;
7. preserve contradictions and uncertainty;
8. route the appropriate human decision;
9. execute through governed people, systems, or operators;
10. verify the intended outcome;
11. update risk based on accepted evidence;
12. and reconstruct the entire path later.

Anything less is a partial workflow, not the finished ClearSight operating loop.