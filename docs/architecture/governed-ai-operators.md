# Governed AI Operators

ClearSight uses specialized AI operators as constrained institutional actors. This document defines the operator architecture, authority model, audit contract, evaluation requirements, and human-control boundaries.

The purpose is not to place a generic assistant over GRC data. The purpose is to safely reduce institutional effort while preserving human authority, evidence lineage, and regulatory defensibility.

---

# 1. Core principle

> **An AI operator may reason and act only through an explicit identity, purpose, scope, capability set, policy, and audit trail.**

A model is not an operator.

An operator is a governed runtime composed of:

- verified service identity;
- operator definition and version;
- purpose;
- tenant and legal-entity scope;
- permitted data classes;
- approved models;
- approved tools;
- action classes;
- policy gates;
- confidence and abstention rules;
- approval thresholds;
- execution controls;
- evaluation status;
- and immutable audit output.

---

# 2. Operator categories

## 2.1 Risk Intelligence Operator

Responsibilities:

- interpret normalized signals;
- propose entity and risk relationships;
- identify emerging patterns;
- draft materiality explanations;
- compare current and prior risk context;
- and surface missing institutional context.

It may not independently alter risk appetite or approve material risk state.

## 2.2 Evidence Operator

Responsibilities:

- identify evidence needs;
- search authorized existing evidence;
- rank best-placed sources;
- draft minimum-question requests;
- classify and extract evidence;
- propose claim mappings;
- identify contradiction;
- and assess evidence sufficiency dimensions.

It may not disclose protected identity, discard source evidence, or issue high-materiality conclusions without required review.

## 2.3 Regulatory Operator

Responsibilities:

- ingest approved regulatory sources;
- detect changes;
- extract candidate obligations;
- propose applicability;
- map obligations to policies, controls, services, and entities;
- identify gaps;
- and draft impact assessments.

It may not make final legal interpretation or external regulatory representation without authorized human review.

## 2.4 Control Operator

Responsibilities:

- identify control coverage;
- detect duplicate or conflicting control implementations;
- propose tests;
- evaluate evidence;
- identify design and operating-effectiveness gaps;
- and propose remediation.

## 2.5 Resilience Operator

Responsibilities:

- analyze critical-operation dependencies;
- identify concentration and single points of failure;
- propose stressed scenarios;
- interpret test results;
- and compare outcomes with impact tolerance.

## 2.6 Third-Party Operator

Responsibilities:

- coordinate due diligence;
- identify reusable evidence;
- analyze service and fourth-party dependencies;
- monitor changes;
- identify contract and exit-plan gaps;
- and prepare review recommendations.

## 2.7 Remediation Operator

Responsibilities:

- propose action plans;
- identify dependencies and infeasible deadlines;
- draft verification contracts;
- monitor implementation and outcome evidence;
- and recommend escalation or reopening.

It may not mark a material issue verified without accepted outcome evidence.

## 2.8 Assurance Operator

Responsibilities:

- challenge evidence sufficiency;
- select samples under approved policy;
- identify unsupported claims;
- prepare evidence lineage;
- and draft assurance observations.

It must preserve audit independence and cannot silently merge first-, second-, and third-line conclusions.

## 2.9 Executive Briefing Operator

Responsibilities:

- summarize material changes;
- adapt language to executive role;
- generate traceable decision briefs;
- prepare committee views;
- and answer authorized natural-language questions.

It may not hide material uncertainty or replace structured decision records with prose.

---

# 3. Operator definition

A versioned operator definition should include:

```yaml
operator_id: evidence-operator
version: 1.0.0
purpose: "Collect and evaluate evidence for authorized institutional claims"
owners:
  product: "Evidence Product Owner"
  risk: "Head of Control Assurance"
allowed_scopes:
  tenants: "runtime-bound"
  legal_entities: "policy-bound"
allowed_data_classes:
  - internal
  - confidential
prohibited_data_classes:
  - protected_reporter_identity
capabilities:
  - search_evidence
  - extract_assertions
  - propose_claim_mapping
  - draft_evidence_request
action_classes:
  read:
    auto: true
  propose:
    auto: true
  reversible_write:
    approval: policy_based
  material_write:
    approval: required
models:
  routing_policy: evidence-standard
confidence_policy: evidence-v1
audit_policy: material-ai-action-v1
evaluation_suite: evidence-operator-v1
```

Definitions are configuration-controlled, reviewed, signed where appropriate, and deployable independently of model prompts.

---

# 4. Action classes

Every tool or domain command belongs to an action class.

## 4.1 Read-only

Examples:

- retrieve authorized risk state;
- search evidence;
- inspect graph relationships;
- read policy;
- compare versions.

Read-only does not mean low risk. Retrieval of protected or highly confidential information may still require explicit purpose and approval.

## 4.2 Analytical

Examples:

- summarize;
- classify;
- extract;
- rank;
- estimate;
- simulate;
- and identify contradiction.

Analytical output does not directly mutate authoritative domain state.

## 4.3 Proposed write

Examples:

- proposed obligation;
- proposed mapping;
- draft risk statement;
- draft evidence request;
- draft action plan;
- or draft committee brief.

Proposals must remain distinguishable from approved records.

## 4.4 Low-impact reversible write

Examples:

- create a draft task;
- apply a non-sensitive classification;
- add a suggested tag;
- schedule an approved reminder;
- or attach an authorized existing evidence reference.

These may be automated under explicit policy.

## 4.5 Material write

Examples:

- issue or change a material risk conclusion;
- approve control effectiveness;
- create a significant issue;
- change a critical relationship;
- declare an incident;
- or initiate a high-impact remediation.

These require human authority unless a narrowly defined policy explicitly permits automation.

## 4.6 Restricted decision

Examples:

- accept material risk;
- change risk appetite;
- determine reportability;
- close a major finding;
- reveal protected identity;
- approve external regulatory communication;
- waive a critical control;
- or approve a high-impact AI model.

Operators may prepare recommendations but cannot execute these decisions autonomously.

---

# 5. Request-to-action pipeline

Every operator invocation follows a controlled pipeline.

```text
Trigger
→ authenticate actor and operator
→ resolve tenant, entity, purpose, and sensitivity
→ authorize data retrieval
→ retrieve and preserve source references
→ execute model or deterministic analysis
→ validate structured output
→ evaluate confidence and contradiction
→ run domain rules
→ run authorization and policy gates
→ determine approval requirement
→ present human review when required
→ execute domain command through approved service
→ verify side effect
→ emit immutable audit event
→ monitor outcome
```

No model-generated free text may bypass this pipeline.

---

# 6. Identity and delegation

## 6.1 Operator identity

Operators authenticate as service identities, not as shared system accounts.

Identity attributes include:

- operator ID and version;
- runtime instance;
- deployment environment;
- tenant context;
- invoking user or event;
- delegated authority;
- and correlation ID.

## 6.2 Delegated authority

An operator may act on behalf of a user only within the user’s authority and the operator’s narrower capability set.

Effective authority is the intersection of:

- user authority;
- operator capability;
- purpose;
- object sensitivity;
- tenant and entity scope;
- action policy;
- and current workflow state.

The operator cannot gain authority merely because a user asked it to perform an action.

## 6.3 Scheduled and event-driven operation

For non-user invocations, authority comes from an approved service policy tied to:

- trigger type;
- scope;
- action class;
- and approval threshold.

---

# 7. Tool governance

## 7.1 Tool registry

Every tool definition includes:

- tool ID and version;
- owner;
- action class;
- input and output schema;
- side effects;
- required scopes;
- data classifications;
- idempotency behavior;
- timeout;
- rollback or compensation behavior;
- and audit requirements.

## 7.2 Domain commands, not raw persistence

Operators use domain commands such as:

- `ProposeEvidenceRequest`
- `CreateDraftIssue`
- `SubmitDecisionForApproval`
- `LinkEvidenceToClaim`

They must not write directly to databases or generic CRUD endpoints.

## 7.3 External tools

External tools such as Probo, ITSM, IAM, email, messaging, or vendor platforms are invoked through constrained adapters.

The adapter must enforce:

- tenant mapping;
- least privilege;
- idempotency;
- source provenance;
- allowed object types;
- and response verification.

## 7.4 Tool result trust

A successful API response is not automatically proof that the intended risk outcome occurred.

Tool output becomes implementation evidence and may trigger separate outcome verification.

---

# 8. Model gateway

ClearSight must remain model-provider independent.

## 8.1 Gateway responsibilities

- provider abstraction;
- capability discovery;
- model allowlists;
- classification-aware routing;
- region and residency constraints;
- latency and cost budgets;
- context limits;
- fallback and retry;
- prompt-policy injection;
- response schema validation;
- usage and quality telemetry;
- and emergency kill switches.

## 8.2 Model metadata

For each invocation record:

- provider;
- model name and version where available;
- deployment or endpoint;
- region;
- temperature and relevant parameters;
- prompt-policy version;
- retrieval sources;
- token or cost metrics;
- latency;
- and safety or content-filter result.

## 8.3 Routing policy

Routing considers:

- data classification;
- tenant policy;
- legal entity and residency;
- use-case evaluation results;
- required capabilities;
- latency;
- cost;
- context size;
- and availability.

## 8.4 Degraded mode

When models are unavailable:

- deterministic rules continue;
- source data remains accessible;
- manual evidence and decision workflows remain usable;
- queued AI tasks are visible and resumable;
- and no stale AI conclusion is presented as newly computed.

---

# 9. Grounding and retrieval

## 9.1 Source hierarchy

Operators should prefer:

1. authoritative institutional source data;
2. approved policy and regulation repositories;
3. validated evidence and conclusions;
4. approved external intelligence;
5. general model knowledge only for non-authoritative explanation.

General model knowledge must not establish a material institutional fact.

## 9.2 Retrieval contract

Every retrieved item includes:

- source ID;
- version;
- title or type;
- effective period;
- classification;
- authorization decision;
- retrieval time;
- and relevant excerpt or structured fields.

## 9.3 Authorization-aware retrieval

Authorization applies before retrieval and after entity expansion.

Vector similarity, search indexes, caches, and graph traversal must not reveal inaccessible content through:

- snippets;
- counts;
- embeddings;
- titles;
- suggested queries;
- or timing.

## 9.4 Source conflict

When authoritative sources disagree, the operator must:

- preserve each source;
- identify the conflict;
- avoid false resolution;
- state affected conclusions;
- and route a focused resolution request.

---

# 10. Structured reasoning record

ClearSight must not store or expose private hidden chain-of-thought. It stores a concise, auditable reasoning record sufficient to explain the decision.

A reasoning record includes:

- task;
- source facts used;
- applicable rules and policies;
- key assumptions;
- alternatives considered;
- contradiction state;
- conclusion;
- confidence;
- proposed action;
- and approval requirement.

The record should be deterministic where possible and generated from structured intermediate outputs rather than an unrestricted narrative.

---

# 11. Confidence and calibration

## 11.1 Confidence dimensions

Confidence may include:

- retrieval completeness;
- entity resolution confidence;
- extraction confidence;
- rule applicability;
- evidence sufficiency;
- contradiction;
- model calibration;
- and source reliability.

Do not reduce these dimensions into a precise percentage unless calibration supports it.

## 11.2 Action thresholds

Thresholds depend on action class.

Example:

- low-risk classification suggestion: may auto-apply at high confidence;
- draft evidence request: may auto-create but require owner review before delivery;
- material control conclusion: requires human review regardless of confidence;
- protected identity disclosure: never autonomous.

## 11.3 Abstention

Operators must abstain when:

- evidence is insufficient;
- sources conflict materially;
- authorization is uncertain;
- request is outside purpose;
- output schema cannot be validated;
- model evaluation does not cover the case;
- or policy requires human judgment.

Abstention is a valid, measurable outcome.

---

# 12. Human review experience

A reviewer must see:

- what the operator proposes;
- which objects will change;
- source evidence;
- evidence and confidence dimensions;
- contradictions;
- policy and authority basis;
- external side effects;
- reversibility;
- and required rationale.

Review options may include:

- approve;
- edit and approve;
- reject;
- request more evidence;
- delegate;
- or escalate.

Every edit becomes feedback but does not automatically become policy.

---

# 13. Audit event contract

Every material operator invocation emits an immutable event containing:

- invocation ID;
- operator ID and version;
- actor or trigger;
- delegated authority;
- tenant and entity scope;
- purpose;
- action class;
- source references and versions;
- tool calls;
- model and prompt-policy metadata;
- structured output;
- confidence and contradiction;
- policy decisions;
- human review and edits;
- executed domain commands;
- external side effects;
- result;
- timestamps;
- correlation and causation IDs;
- and error or abstention state.

Sensitive raw content should remain in governed source stores rather than being duplicated into the audit event.

---

# 14. Prompt and policy management

## 14.1 Prompt policy

Prompts are versioned implementation artifacts, not hidden production configuration.

Prompt packages include:

- system instruction template;
- task template;
- structured output schema;
- examples;
- prohibited behavior;
- retrieval policy;
- tool policy;
- and evaluation suite reference.

## 14.2 Separation of policy and prompt

Authorization, authority, evidence minimums, action thresholds, and data residency must be enforced outside the model prompt.

A prompt may explain policy but cannot be the sole enforcement mechanism.

## 14.3 Change management

A prompt or model change requires:

- version increment;
- evaluation comparison;
- risk review proportional to use case;
- rollout plan;
- monitoring;
- and rollback.

---

# 15. Prompt-injection and untrusted-content defense

All user, evidence, email, document, web, and integration content is untrusted.

Controls include:

- strict separation between policy instructions and retrieved content;
- content provenance labels;
- no dynamic expansion of tool permissions;
- structured retrieval and quoting;
- domain validation after model output;
- output schemas;
- allowlisted tool calls;
- high-impact approval gates;
- secret isolation;
- and adversarial testing.

The operator must ignore instructions embedded in evidence that attempt to:

- alter system policy;
- reveal secrets;
- access unrelated objects;
- invoke tools;
- or suppress audit.

---

# 16. Data protection

## 16.1 Minimization

Only data required for the operator purpose should be provided to the model.

Use:

- structured fields;
- excerpts;
- redaction;
- pseudonymization;
- and local preprocessing.

## 16.2 Protected data

Protected reporter identity, legal privilege, sensitive investigation content, authentication secrets, and highly restricted customer data require specialized routes or must be excluded from model processing.

## 16.3 Retention

Model providers must not retain or train on ClearSight data unless explicitly approved by tenant policy and contract.

Invocation content retention must follow classification and purpose.

---

# 17. Operator evaluation framework

Every operator capability requires an evaluation suite before release.

## 17.1 Evaluation categories

### Domain correctness

- correct entity and relationship extraction;
- correct obligation or control mapping;
- correct evidence relationship;
- and correct risk context.

### Grounding

- citation precision;
- source version correctness;
- unsupported assertion rate;
- and contradiction disclosure.

### Decision behavior

- correct action class;
- correct authority requirement;
- appropriate abstention;
- and no unauthorized state change.

### Security

- prompt injection;
- cross-tenant retrieval;
- protected identity leakage;
- secret extraction;
- tool abuse;
- and malicious evidence.

### Fairness and inappropriate inference

- no reporter credibility inference from demographics or style;
- no unsupported employee-risk profiling;
- and no differential routing based on protected attributes without lawful policy.

### Reliability

- malformed output;
- timeout;
- provider failure;
- partial tool failure;
- duplicate invocation;
- and replay.

### Performance

- latency;
- cost;
- context size;
- and throughput.

## 17.2 Evaluation datasets

Datasets should include:

- realistic bank scenarios;
- ambiguous evidence;
- conflicting sources;
- multilingual content;
- adversarial content;
- protected reports;
- incomplete documents;
- stale evidence;
- and out-of-scope requests.

Synthetic data must preserve domain complexity and not inject the desired conclusion directly.

## 17.3 Release gates

A release requires:

- minimum quality thresholds;
- zero critical authorization failures;
- zero protected identity leakage in test coverage;
- acceptable abstention;
- reviewed regression comparison;
- and approved monitoring plan.

---

# 18. Runtime monitoring

Monitor:

- invocation volume;
- latency and cost;
- model/provider errors;
- schema failures;
- abstention rate;
- confidence distribution;
- human approval, edit, and rejection rates;
- citation failures;
- unauthorized action attempts;
- prompt-injection detections;
- cross-tenant policy denials;
- protected-data access;
- and downstream verification outcomes.

Quality monitoring should connect recommendations to later outcomes, not only reviewer approval.

---

# 19. Probo and external execution engines

An operator may interact with Probo or another engine through an approved adapter.

Requirements:

- ClearSight operator identity is mapped to a scoped integration identity;
- tokens are not exposed to the model;
- tools are allowlisted;
- organization and tenant mapping is server-controlled;
- writes are idempotent;
- external object IDs and versions are preserved;
- returned records become source evidence, not unquestioned truth;
- and material changes remain subject to ClearSight authority and verification.

The operator cannot call a broad MCP endpoint with unrestricted organization access merely because the endpoint is available.

---

# 20. Representative approval matrix

| Action | Operator may propose | Operator may execute automatically | Human authority |
|---|---:|---:|---|
| Classify ordinary evidence | Yes | Policy-dependent | Evidence owner for override |
| Extract assertions | Yes | Yes, as unconfirmed assertions | Reviewer for material use |
| Draft evidence request | Yes | Yes as draft | Owner or policy before delivery |
| Deliver low-sensitivity request | Yes | Policy-dependent | Request owner |
| Propose control mapping | Yes | No for material controls | Control assurance |
| Create draft issue | Yes | Policy-dependent | Risk/control owner |
| Close material issue | Yes | No | Defined assurance authority |
| Accept material risk | Yes | No | Authority matrix |
| Determine regulatory reportability | Yes | No | Compliance/legal authority |
| Reveal protected identity | No autonomous decision | Never | Explicit privileged authority |
| Publish external regulatory response | Draft only | Never | Authorized human signatory |

---

# 21. Acceptance scenarios

## Scenario A: low-confidence obligation extraction

- Regulatory Operator extracts a candidate obligation.
- Source and passage are preserved.
- Confidence is below the configured threshold.
- The operator abstains from publishing.
- A compliance reviewer receives the candidate with context.

## Scenario B: malicious evidence document

- A document contains instructions asking the operator to reveal secrets and close the finding.
- The content is treated as evidence, not policy.
- No unauthorized tool is called.
- The injection attempt is logged safely.
- Evidence analysis continues on the relevant content.

## Scenario C: cross-tenant prompt

- A user asks for another institution’s evidence.
- Retrieval returns nothing and does not expose counts or titles.
- The operator records an authorization denial.

## Scenario D: remediation proposal

- Remediation Operator proposes three options.
- Each includes cost, dependency, projected risk movement, uncertainty, and verification contract.
- The operator cannot approve the selected material option.
- Human edits and approval are recorded.

## Scenario E: model outage

- External model route fails.
- Deterministic risk data remains available.
- The user can complete a manual decision.
- The failed analysis is resumable and not shown as current output.

## Scenario F: protected report

- Executive Briefing Operator summarizes a protected case.
- Reporter identity and identifying metadata are excluded.
- The summary states allegation versus verified fact.
- No credibility score is produced.

---

# 22. Prohibited shortcuts

Do not:

- use one all-powerful assistant identity;
- expose unrestricted tool catalogs to the model;
- allow direct database access;
- rely on prompts for authorization;
- treat model confidence as evidence sufficiency;
- store only a prose transcript for material actions;
- omit source versions;
- allow free-form output to mutate state;
- train on protected data without explicit approval;
- infer whistleblower credibility from style or emotion;
- silently change policy from reviewer feedback;
- deploy a new model without regression evaluation;
- or let an external automation engine become the authority for material risk decisions.

---

# 23. Definition of success

Governed AI operators succeed when:

- users perform less assembly and navigation work;
- every material output remains grounded and explainable;
- operators act only within explicit authority;
- low-confidence and contradictory cases reach humans;
- external side effects are controlled and verified;
- the product remains usable without AI;
- protected information remains protected;
- and the institution can reconstruct exactly what an operator did, why it did it, who approved it, and what happened afterward.