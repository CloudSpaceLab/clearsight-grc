# Living Evidence Fabric

The Living Evidence Fabric is the central differentiating subsystem of ClearSight.

It is a claim-centric, temporal, provenance-preserving system that continuously gathers, validates, reconciles, refreshes, and governs evidence from systems, staff, vendors, customers, and confidential reporters.

The goal is not to collect more documents. The goal is to establish the strongest defensible understanding of material institutional claims with the least reasonable human effort.

---

# 1. Core principle

> **Evidence exists to support or contradict a precise claim for a defined purpose, scope, and period.**

An evidence object without a linked claim, provenance, effective period, and intended use is incomplete.

Examples of claims:

- “All active Treasury Operations privileged users were reviewed and approved during July.”
- “The payment service can fail over within the approved impact tolerance.”
- “The vendor encrypts customer data at rest in the relevant production environment.”
- “The remediation removed unapproved standing access.”
- “The institution notified affected customers within the applicable deadline.”

A policy, screenshot, ticket, attestation, log extract, interview, video, or system record may support part of a claim, contradict it, or be irrelevant. The fabric must preserve that distinction.

---

# 2. Conceptual model

## 2.1 Claim

A statement that can be supported, contradicted, qualified, or remain unresolved.

Required attributes:

- claim ID;
- claim type;
- normalized statement;
- subject entities;
- scope;
- effective period;
- purpose;
- materiality;
- required evidence policy;
- current conclusion;
- confidence state;
- lifecycle state;
- and version.

Claim types may include:

- design claim;
- implementation claim;
- operating-effectiveness claim;
- compliance claim;
- risk-state claim;
- incident fact claim;
- remediation-completion claim;
- remediation-effectiveness claim;
- customer-impact claim;
- resilience claim;
- and identity or ownership claim.

## 2.2 Evidence item

An immutable version of an artifact, observation, statement, or system-produced fact.

Required attributes:

- evidence ID and version;
- source type and source identity;
- capture method;
- original content or protected object reference;
- content hash;
- capture time;
- effective period;
- scope;
- classification and sensitivity;
- tenant and legal-entity boundaries;
- chain-of-custody metadata;
- retention and legal-hold state;
- extraction status;
- and access policy.

Evidence types include:

- system event;
- configuration snapshot;
- transaction or population extract;
- document;
- image or screenshot;
- audio or video;
- email or message;
- structured attestation;
- observation;
- interview record;
- customer report;
- confidential report;
- external intelligence;
- test result;
- approval record;
- and physical-world verification.

## 2.3 Evidence assertion

A structured assertion extracted or confirmed from an evidence item.

Examples:

- account `A123` remained active on 2026-07-31;
- approval `APR-992` applies to role `TreasuryAdmin` until 2026-09-30;
- failover test completed in 43 minutes;
- customer notification sent at a specific time;
- control owner stated that a review occurred.

An assertion records:

- source evidence version;
- extraction or entry method;
- actor or model version;
- confidence;
- user confirmation state;
- and normalized entities and time.

Machine-extracted assertions are never silently treated as source facts without validation policy.

## 2.4 Claim-evidence evaluation

A versioned evaluation of how an evidence item or assertion relates to a claim.

Possible relationships:

- supports;
- partially supports;
- contradicts;
- limits;
- duplicates;
- supersedes;
- irrelevant;
- unverifiable;
- or pending review.

The evaluation includes dimension scores, rationale, evaluator, model or rule version, and review state.

## 2.5 Conclusion

A reasoned determination about the claim based on the current evidence set.

Possible conclusion states:

- supported;
- partially supported;
- unsupported;
- contradicted;
- indeterminate;
- expired;
- or not applicable.

A conclusion is versioned and must identify:

- included evidence;
- excluded evidence and reason;
- contradiction state;
- assumptions;
- evidence sufficiency;
- evaluator;
- approval authority where needed;
- and valid period.

## 2.6 Evidence request

A targeted request for unresolved evidence.

A request includes:

- target claim;
- unresolved facts;
- proposed source;
- source-ranking rationale;
- selected channel;
- recipient;
- prefilled context;
- acceptable response forms;
- deadline;
- sensitivity;
- estimated effort;
- reminders and escalation policy;
- and stop condition.

## 2.7 Contradiction

A first-class record linking conflicting assertions or evidence evaluations.

It includes:

- disputed claim;
- conflict type;
- involved evidence versions;
- affected conclusions and decisions;
- materiality;
- resolver;
- deadline;
- resolution outcome;
- and residual uncertainty.

## 2.8 Evidence debt

A measure of uncertainty created by missing, stale, weak, incomplete, or contradictory evidence.

Evidence debt is tracked separately from risk severity. It may alter confidence, materiality, assurance, and escalation.

---

# 3. Evidence lifecycle

The lifecycle is:

```text
Need identified
  → existing evidence searched
  → best source selected
  → evidence observed or requested
  → original captured immutably
  → assertions extracted
  → claim relationship evaluated
  → sufficiency and contradiction assessed
  → human review where required
  → conclusion issued
  → evidence monitored for expiry or invalidation
  → conclusion refreshed, superseded, or reopened
```

## 3.1 Need identification

Evidence needs may be triggered by:

- a new claim;
- a control or risk review;
- a material signal;
- a regulatory change;
- a threshold breach;
- an incident or loss;
- an audit sample;
- a remediation milestone;
- approaching evidence expiry;
- a contradictory source;
- a changed business relationship;
- a customer or protected report;
- or a verification contract.

The trigger must identify the exact claim and required evidence policy.

## 3.2 Existing evidence search

Before contacting a person, the fabric searches authorized sources for:

- current evidence already linked to the claim;
- evidence linked to equivalent or overlapping claims;
- source-system data;
- previous review evidence still within scope;
- evidence from another control implementation;
- and evidence scheduled for collection by another workflow.

The system must respect purpose and authorization. Evidence gathered for one purpose is not automatically reusable for another.

## 3.3 Best-placed-source resolution

Candidate sources are ranked across:

- authority;
- directness;
- independence;
- freshness;
- population coverage;
- historical reliability;
- accessibility;
- human burden;
- cost;
- response time;
- sensitivity;
- conflict of interest;
- and legal or contractual constraints.

Source selection must remain explainable.

Example order for a privileged-access claim:

1. authoritative IAM snapshot;
2. approval repository;
3. HR status and role data;
4. access-review system output;
5. manager confirmation;
6. account-holder attestation.

Human attestation may still be necessary to explain business need, but it should not replace authoritative technical evidence when that evidence is available.

## 3.4 Minimum-question generation

The request generator determines:

- what facts are already known;
- what exact facts remain unresolved;
- what response structure minimizes ambiguity;
- which attachments are necessary;
- and when no human request is needed.

A generated question must be reviewed against:

- clarity;
- burden;
- sensitivity;
- leading language;
- jargon;
- recipient authority;
- and whether the answer can actually support the claim.

## 3.5 Capture

Evidence capture supports:

- API and event ingestion;
- secure upload;
- email and enterprise messaging;
- mobile camera and document scan;
- audio and video;
- structured forms;
- external portals;
- anonymous reporting;
- and controlled batch import.

At capture, the system records:

- original bytes or source reference;
- hash;
- source identity;
- channel;
- timestamps;
- source-system version or cursor;
- device or integration metadata where policy allows;
- consent or notice state where required;
- classification;
- and chain of custody.

## 3.6 Validation

Validation may include:

- file integrity;
- malware scanning;
- source authentication;
- digital-signature verification;
- schema validation;
- completeness;
- page and attachment consistency;
- date and scope coverage;
- duplicate detection;
- image or document readability;
- and policy checks.

Validation does not determine whether the evidence proves the claim. That occurs during evaluation.

## 3.7 Extraction and normalization

Approved AI or deterministic processors may:

- transcribe audio or video;
- extract text;
- identify dates, entities, systems, products, accounts, controls, and obligations;
- normalize identifiers;
- redact unnecessary personal data;
- classify evidence type;
- detect signatures and approvals;
- compare versions;
- and propose structured assertions.

Requirements:

- preserve the original;
- record processor and version;
- retain extraction coordinates or time offsets where feasible;
- separate inferred values from explicit values;
- allow correction;
- and mark user-confirmed assertions.

## 3.8 Claim evaluation

Each evidence item is evaluated against the exact claim and purpose.

The same item may strongly support one claim and weakly support another.

Example:

A screenshot of an IAM console may support that an account appeared disabled at capture time. It may not establish that all privileged accounts were reviewed across the required monthly period.

## 3.9 Conclusion and approval

Conclusion authority depends on materiality and use.

Examples:

- low-risk control evidence may be auto-evaluated under approved policy;
- material control effectiveness may require second-line review;
- audit conclusions remain with internal audit;
- regulatory representations require authorized compliance or legal review;
- protected-case facts may require investigator validation.

## 3.10 Refresh and invalidation

Evidence and conclusions may be invalidated by:

- expiry;
- source revocation;
- superseding evidence;
- business-scope change;
- control or policy change;
- incident realization;
- contradiction;
- integration failure;
- model extraction error;
- or legal hold and access changes.

The system must propagate invalidation to affected conclusions, decisions, reports, and assurance packs.

---

# 4. Evidence sufficiency model

Evidence sufficiency must be multidimensional.

## 4.1 Relevance

Does the evidence address the exact claim?

Consider:

- subject;
- time period;
- environment;
- population;
- control implementation;
- legal entity;
- and intended conclusion.

## 4.2 Authenticity

Can the source and integrity be trusted?

Signals include:

- authenticated system source;
- digital signature;
- cryptographic hash;
- chain of custody;
- source identity;
- metadata consistency;
- and tamper indicators.

## 4.3 Coverage

How much of the required scope is represented?

Coverage may be:

- full population;
- sample;
- time-window coverage;
- entity coverage;
- process coverage;
- or scenario coverage.

A sample must record selection method and population.

## 4.4 Freshness

Is the evidence recent enough for the claim and risk velocity?

Freshness policy depends on:

- control frequency;
- rate of environmental change;
- risk severity;
- and regulatory or audit requirements.

## 4.5 Independence

How independent is the evidence from the person or process making the claim?

Possible levels:

- self-attested;
- manager-confirmed;
- system-generated;
- independent internal test;
- internal audit;
- external assurance;
- or regulator-observed.

Independence does not automatically determine accuracy, but it affects sufficiency.

## 4.6 Completeness

Are required elements present?

Examples:

- approvals;
- signatures;
- dates;
- population definition;
- exception records;
- result details;
- and supporting attachments.

## 4.7 Consistency

Does the evidence agree with other trusted sources?

The system must identify:

- direct contradiction;
- scope mismatch;
- temporal mismatch;
- identity mismatch;
- and unexplained variance.

## 4.8 Reliability

How dependable has the source or collection method been?

Reliability may consider:

- historical corrections;
- integration failures;
- source control maturity;
- known data-quality issues;
- and reviewer outcomes.

## 4.9 Traceability

Can an independent reviewer reconstruct:

- where the evidence came from;
- how it was transformed;
- why it was linked to the claim;
- and how the conclusion was reached?

## 4.10 Sufficiency decision

A policy may define minimum requirements by claim type and materiality.

Example:

```yaml
claim_type: operating_effectiveness
materiality: high
requirements:
  relevance: required
  authenticity: strong
  coverage:
    minimum: full_population
  freshness:
    maximum_age: 31d
  independence:
    minimum: system_generated
  contradiction:
    unresolved_allowed: false
  approval:
    role: second_line_control_assurance
```

A weighted score may aid prioritization, but policy gates and dimension explanations remain authoritative.

---

# 5. Dynamic request orchestration

## 5.1 Request state machine

Suggested states:

```text
DRAFTED
→ POLICY_CHECKED
→ READY
→ DELIVERED
→ VIEWED
→ PARTIALLY_ANSWERED
→ VALIDATING
→ FOLLOW_UP_REQUIRED
→ SUFFICIENT
→ DECLINED
→ REDIRECTED
→ EXPIRED
→ CANCELLED
```

State transitions must be explicit and audited.

## 5.2 Recipient resolution

Recipient ranking can use graph relationships such as:

- system owner;
- process owner;
- control performer;
- manager;
- evidence custodian;
- vendor contact;
- investigator;
- and service owner.

The system must support redirecting to a better source without treating redirection as non-compliance.

## 5.3 Channel selection

Channel policy considers:

- sensitivity;
- response format;
- user preference;
- authentication strength;
- expected effort;
- urgency;
- record-retention requirements;
- and accessibility.

Protected or highly sensitive evidence must not be collected through channels that cannot preserve required confidentiality and chain of custody.

## 5.4 Reminder policy

Reminder frequency must consider:

- materiality;
- deadline;
- recipient workload;
- duplicate requests;
- delegated ownership;
- and whether the missing evidence remains decision-relevant.

The system should cancel reminders automatically when sufficient evidence is obtained elsewhere.

## 5.5 Burden measurement

ClearSight should measure:

- time to answer;
- number of fields;
- number of follow-ups;
- evidence rejection rate;
- duplicate requests avoided;
- redirection rate;
- and recipient-reported difficulty.

The request engine should improve from these outcomes without silently changing policy.

---

# 6. Continuous evidence collection

## 6.1 Connectors

Connectors may collect:

- IAM users, roles, and approvals;
- cloud configurations;
- security-tool findings;
- ticket and change records;
- HR status;
- vendor documents and performance;
- service telemetry;
- backup and recovery results;
- training completion;
- policy acknowledgements;
- and transaction or operational samples.

Each connector must record:

- source system;
- authorization scope;
- source object IDs;
- version, ETag, event ID, or cursor;
- collection time;
- effective time;
- mapping version;
- and sync health.

## 6.2 Polling versus event collection

Prefer event-driven capture when the source supports reliable events. Use polling when required.

Polling jobs must be:

- incremental;
- idempotent;
- rate-limited;
- observable;
- and replay-safe.

## 6.3 Evidence snapshots

Some claims require point-in-time or period snapshots.

Snapshots should include:

- population definition;
- query or collection logic;
- source versions;
- row or object counts;
- hash manifest;
- exceptions;
- and reproducibility metadata.

## 6.4 Source degradation

When a connector fails or becomes stale:

- affected evidence is marked;
- dependent conclusions are reassessed;
- material confidence changes are surfaced;
- fallback sources may be selected;
- and the system must not present stale evidence as current.

---

# 7. Human, customer, vendor, and protected evidence

## 7.1 Staff evidence

Staff requests should:

- use plain language;
- explain purpose;
- prefill known facts;
- show estimated effort;
- support delegation or redirection;
- and avoid exposing unnecessary confidential context.

## 7.2 Vendor evidence

Vendor collection should support:

- scoped portal access;
- reusable evidence packages;
- contract and service context;
- expiration and refresh;
- evidence request negotiation;
- and mapping to multiple bank claims without unnecessary duplicate uploads.

Vendor-supplied evidence must be distinguished from independent validation.

## 7.3 Customer evidence

Customer reports may include:

- complaint narratives;
- transaction details;
- screenshots;
- call or interaction records;
- service-impact reports;
- consent and privacy concerns;
- and supporting documents.

The system must distinguish allegation, observation, and verified fact.

## 7.4 Whistleblower and protected evidence

Protected evidence requires:

- identity/content separation;
- separate encryption keys or equivalent isolation;
- need-to-know access;
- conflict-aware routing;
- anonymous two-way communication;
- legal privilege markers;
- restricted search and analytics;
- controlled export;
- and identity-reveal workflow.

AI may assist with translation and triage but must not:

- infer credibility from style or emotion;
- expose identity through summaries;
- retrieve protected content outside approved purpose;
- or automatically close a report.

---

# 8. Chain of custody and integrity

Each evidence version should support an integrity manifest containing:

- evidence ID and version;
- content hash;
- storage object version;
- capture actor and method;
- source identity;
- capture timestamp;
- effective period;
- transformations;
- derivative artifact hashes;
- access events;
- review events;
- export events;
- retention policy;
- and legal-hold status.

For high-assurance cases, consider:

- trusted timestamping;
- signed manifests;
- write-once storage;
- or externally verifiable audit anchors.

These mechanisms must be justified by threat model and regulatory need rather than used as marketing decoration.

---

# 9. Authorization model

Evidence access may depend on:

- tenant;
- legal entity;
- jurisdiction;
- business unit;
- evidence classification;
- data subject;
- case assignment;
- control or risk ownership;
- audit independence;
- investigation conflict;
- legal privilege;
- purpose;
- and time-limited authorization.

Requirements:

- authorization is enforced on original evidence, derivatives, assertions, search indexes, embeddings, graph edges, exports, and AI retrieval;
- counts and metadata must not leak inaccessible objects;
- access is re-evaluated at export time;
- and protected identity access uses a separate privileged action.

---

# 10. Storage architecture

A reference architecture may use:

- relational authoritative metadata;
- immutable versioned object storage for evidence content;
- transactional outbox for events;
- search projection for text and metadata;
- vector projection for authorized semantic retrieval;
- graph projection for relationships;
- and an append-only audit store.

Principles:

- search, vectors, and graph projections are rebuildable;
- object references are content- and version-aware;
- source data and derivatives are separated;
- encryption and retention follow classification;
- and deletion or legal hold propagates consistently.

Do not place raw evidence directly into logs, events, or analytics streams.

---

# 11. AI use within the fabric

Approved AI tasks may include:

- document classification;
- transcription;
- field extraction;
- entity resolution;
- claim suggestion;
- evidence-to-claim mapping;
- contradiction detection;
- summarization;
- redaction suggestion;
- request drafting;
- and follow-up question generation.

Every AI result must include:

- operator identity;
- model and version;
- prompt-policy version;
- input evidence versions;
- output schema version;
- confidence;
- validation result;
- and human review state where required.

Material conclusions cannot rely solely on unvalidated free-form model output.

---

# 12. APIs and events

## 12.1 Representative APIs

- create claim;
- version claim;
- search authorized evidence;
- initiate evidence need;
- rank sources;
- create request;
- capture evidence;
- complete upload;
- extract assertions;
- evaluate evidence against claim;
- record contradiction;
- issue conclusion;
- approve conclusion;
- invalidate evidence;
- apply retention or legal hold;
- generate evidence manifest;
- and export assurance package.

## 12.2 Representative events

- `ClaimCreated`
- `EvidenceNeedIdentified`
- `EvidenceRequestDelivered`
- `EvidenceVersionCaptured`
- `EvidenceValidationFailed`
- `EvidenceAssertionsExtracted`
- `EvidenceLinkedToClaim`
- `ContradictionDetected`
- `EvidenceSufficiencyChanged`
- `ClaimConclusionIssued`
- `ClaimConclusionSuperseded`
- `EvidenceExpired`
- `EvidenceSourceDegraded`
- `ProtectedIdentityAccessed`
- `EvidenceExported`

Events carry IDs and safe metadata, not raw evidence content.

---

# 13. Metrics

## Evidence quality

- material claims with sufficient evidence;
- claims with unresolved contradiction;
- average evidence age by claim type;
- evidence debt by materiality;
- source reliability;
- and conclusion reversal rate.

## Human effort

- median response time;
- median active effort;
- questions per accepted evidence item;
- follow-up rate;
- duplicate requests avoided;
- redirection rate;
- and rejection reasons.

## Automation

- evidence collected without human input;
- automated evaluations accepted;
- automated evaluations overridden;
- connector freshness;
- and failed collection jobs.

## Trust and security

- unauthorized access attempts;
- protected identity access;
- evidence integrity failures;
- export volume and purpose;
- and chain-of-custody exceptions.

---

# 14. Acceptance scenarios

The subsystem is not complete until it can pass at least these scenarios.

## Scenario A: privileged access review

- IAM provides current population.
- Approval system provides prior approvals.
- HR indicates one employee transferred.
- Four accounts remain unresolved.
- The system asks the appropriate manager only about those four.
- The response is reconciled with IAM.
- One contradiction is detected.
- The conclusion remains pending until resolved.

## Scenario B: resilience test

- Vendor attests that failover is tested.
- Latest attached report is outside required period.
- Internal telemetry shows no recent test event.
- Evidence sufficiency is weak despite the attestation.
- A targeted vendor request is generated.
- The affected critical service and decision deadline are visible.

## Scenario C: remediation verification

- A ticket records that access was removed.
- IAM initially confirms removal.
- The verification contract requires 30 days with no reactivation.
- The action is complete but the issue remains awaiting verification.
- A reactivation event during the observation period fails verification and reopens action.

## Scenario D: protected report

- An anonymous report includes an audio attachment.
- Audio is transcribed using an approved protected-data model route.
- The summary excludes identifying metadata.
- Investigator search does not expose reporter identity.
- Identity reveal is impossible without a separate authorized workflow.

## Scenario E: point-in-time reconstruction

- An auditor selects a past date.
- The system reconstructs the claim, evidence versions, conclusion, policy, and decision known at that time.
- Later corrections are visible but do not rewrite the historical view.

---

# 15. Prohibited shortcuts

Do not:

- store evidence only as a file URL;
- overwrite evidence versions;
- treat all attachments as supporting evidence;
- use AI confidence as evidence sufficiency;
- treat self-attestation as independent evidence;
- silently discard contradictions;
- request evidence before searching existing sources;
- ask broad questions when the unresolved facts are known;
- reuse protected evidence without purpose validation;
- index restricted content without equivalent authorization;
- close a claim because a request received any response;
- or mark evidence current when the source integration is stale.

---

# 16. Definition of success

The Living Evidence Fabric succeeds when:

- staff are asked fewer and better questions;
- machine and human evidence are reconciled;
- material claims have clear proof or clearly disclosed uncertainty;
- contradictions become actionable rather than hidden;
- auditors can reconstruct the evidence chain;
- protected reporters remain protected across every subsystem;
- and remediation is verified through observed outcomes rather than task completion.