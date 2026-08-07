# Authority Routing and Escalation

This document defines how ClearSight assigns responsibility, resolves reviewers and authorizers, enforces segregation of duties, and moves work when time, authority, conflict, or availability prevents completion.

The goal is:

> **Make complex bank responsibility and authority easy to configure, easy to understand, and difficult to misuse.**

A single assignee field, a static RBAC table, or a hard-coded approval chain is not sufficient.

## 1. Actor distinctions

ClearSight keeps these responsibilities distinct:

- **Performer** — carries out work or provides information.
- **Accountable owner** — owns the obligation, Matter, control, service, or outcome.
- **Proposer** — prepares or resubmits a position, response, exception, or other governed recommendation without gaining approval authority.
- **Reviewer** — checks evidence, recommendation, response, or work quality and may return/reject work where policy allows.
- **Independent challenger** — provides second- or third-line challenge without inheriting ownership.
- **Authorizer** — may approve, reject, expire, supersede, close, cancel, or reopen a defined decision/Matter within authority.
- **Signatory** — may approve an external institutional representation within mandate.
- **Transmitter** — may send an approved external response through the governed channel; transmission authority is distinct from signatory approval.
- **Acknowledgement recorder** — may record authoritative receipt/acknowledgement after transmission; this does not create or approve the response.
- **Escalation owner** — receives work when deadlines, materiality, authority, or routing failures require intervention.
- **Observer or consulted party** — receives visibility without blocking progression unless policy says otherwise.

One person may hold several institutional roles but must not perform conflicting steps in the same workflow instance where segregation of duties applies.

## 2. Canonical model

### Principal

A human, team, queue, external party, committee, or service identity.

### Role template

A reusable functional role such as DPO, Branch Manager, Application Owner, Control Reviewer, Legal Signatory, or Operational Risk Approver.

A role template describes expected capabilities; it does not grant universal access.

### Organizational position

A source-backed position in the institution hierarchy, optionally occupied by a person and linked to a manager, deputy, legal entity, function, branch, or service.

### Responsibility assignment

A versioned relationship binding a principal or role to an object and responsibility type for a scope and period.

Examples:

- accountable owner of Retail Payments;
- performer for branch physical-security checks;
- proposer for a regulator-response package;
- reviewer for privileged-access evidence;
- transmitter for an approved authority response;
- DPO for one legal entity.

### Authority grant

A versioned permission to make a defined decision or representation under stated limits.

Authority may depend on:

- tenant, legal entity, jurisdiction, Program, Matter type, service, branch, or population;
- materiality, financial amount, duration, customer impact, reversibility, and evidence quality;
- decision type;
- valid period;
- required challenge or co-approval.

### Decision policy

Defines the authority, quorum, sequence, independence, rationale, and confirmation required for one decision class.

### Routing policy

Resolves who should propose, perform, review, challenge, authorize, sign, transmit, record acknowledgement, or receive a request at a specific workflow state.

### Escalation policy

Defines reminders, overdue behavior, materiality escalation, unavailable-recipient fallback, and terminal failure handling.

### Delegation and substitution

- **Delegation** transfers permission to act within a narrow scope and period but does not silently transfer accountability.
- **Substitution** activates a pre-approved deputy or role when the primary actor is unavailable.
- **Handoff** transfers current workflow ownership with context and explicit acceptance.

Each has different semantics and audit requirements.

## 3. Effective authority

Effective authority is the intersection of:

```text
principal identity and status
∩ role and position
∩ scoped responsibility assignment
∩ authority grant
∩ tenant, entity, purpose, and sensitivity
∩ object relationship
∩ workflow state and decision type
∩ materiality and limits
∩ delegation or substitution
∩ conflict and segregation-of-duties rules
∩ current policy version and time
```

A broad application role must never override a narrower domain restriction.

## 4. Routing expression

A routing rule should be expressible without code using source-backed selectors.

Example:

```yaml
step: authorize_exception
resolve:
  role: operational_risk_approver
  relationship: legal_entity_of_matter
conditions:
  residual_rating: [high, critical]
  duration_days: "> 30"
sequence:
  - proposer: control_owner
  - reviewer: control_assurance
  - challenger: second_line_risk
  - authorizer: entity_cro
fallback:
  - active_substitute
  - functional_queue
  - group_cro
segregation:
  authorizer_not_in: [performers, proposers, evidence_submitters]
time:
  due: 2_business_days
  escalate_after: 1_business_day
```

The UI may present this as a matrix and visual sequence; the stored policy must remain structured, versioned, testable, and portable.

### 4.1 Lifecycle-specific runtime routing

A material HTTP route is not itself an authority role. ClearSight first loads the current record, validates the requested transition, and then resolves the responsibility required for that exact lifecycle step.

Decision lifecycle:

```text
PROPOSED                     → PROPOSER
IN_REVIEW / RETURNED         → REVIEWER
CHALLENGED                   → INDEPENDENT_CHALLENGER
APPROVED / CONDITIONALLY_APPROVED
REJECTED / EXPIRED / SUPERSEDED
                             → AUTHORIZER
```

External-response lifecycle:

```text
prepare / resubmit           → PROPOSER
review / reject / return     → REVIEWER
approve                      → SIGNATORY
transmit approved response   → TRANSMITTER
record acknowledgement       → ACKNOWLEDGEMENT_RECORDER
```

Matter transitions into `DECISION_REQUIRED`, `CLOSED`, or `CANCELLED`, and reopening a closed Matter, require current authorizer responsibility.

The verified actor is recorded in the append-only event envelope and in the appropriate lifecycle field on the current Decision/Response record. A client-supplied actor identifier never determines authority.

## 5. Supported sequence patterns

- sequential approval;
- parallel review;
- any-of approval;
- all-of approval;
- quorum;
- weighted committee vote where legally supported;
- independent challenge before authorization;
- veto role;
- conditional approval;
- return for evidence;
- emergency authorization followed by mandatory retrospective review;
- non-blocking consultation;
- external signatory after internal approval;
- governed transmission after signatory approval;
- acknowledgement recording after transmission.

Every sequence must define what happens on propose, review, challenge, approve, reject, return, transmit, acknowledge, abstain, conflict, timeout, delegation, unavailable user, and policy change.

## 6. Escalation semantics

### Reminder

A nudge to the current owner. It does not change ownership or authority.

### Operational escalation

Moves visibility or work to a manager, queue, or substitute because the responsible actor has not progressed the task.

### Authority escalation

Routes the decision to a higher mandate because the current actor lacks authority or the matter became more material.

### Risk or deadline escalation

Raises priority and required governance because delay changes exposure, customer impact, legal duty, or commitment status.

### Routing-failure escalation

Used when no valid actor can be resolved, source identity is stale, delegation is circular, conflict rules exclude all candidates, or required quorum cannot be formed.

Routing failure must become visible work; it must never silently assign an arbitrary administrator.

## 7. Time rules

Policies must support:

- business and calendar time;
- tenant, entity, or recipient time zone;
- working calendars and holidays;
- acknowledgement, action, review, challenge, authorization, signatory, and transmission deadlines;
- pause rules for approved blockers;
- regulatory deadlines that cannot be paused;
- reminder cadence;
- escalation thresholds;
- maximum delegation period;
- authority expiry.

The UI must show the actual deadline, current owner, next escalation, and whether the clock is paused.

## 8. Conflict and segregation of duties

Conflict rules may use:

- same person or reporting line;
- proposer/performer/reviewer/challenger/authorizer overlap;
- signatory/transmitter overlap where policy requires separation;
- ownership of affected service, vendor, control, or evidence;
- investigation subject or related party;
- prior decision participation;
- audit independence;
- protected-report subject or manager relationship;
- financial or procurement interest.

Required behavior:

- detect conflict before assignment and again before action;
- allow the user to declare an unmodelled conflict without revealing protected details;
- remove the conflicted actor from content and notifications where required;
- resolve an independent substitute;
- preserve the conflict decision and policy basis;
- prevent self-approval and privilege amplification.

## 9. Delegation, absence, and organization change

ClearSight must respond to HR, directory, or governance changes:

- leave or out-of-office;
- employee departure;
- manager change;
- position vacancy;
- role revocation;
- entity transfer;
- emergency substitution;
- temporary project authority.

Rules:

- drafts and current assignments remain preserved;
- authorization is re-evaluated before material action;
- expired authority blocks approval but offers the safe route;
- a substitute sees only the required context;
- accountability history is not rewritten;
- delegation must be accepted where policy requires;
- circular delegation is rejected;
- stale HR data is visible and may trigger routing review.

## 10. Configuration experience

Configure should provide five focused views.

### Role catalogue

Create role templates from familiar banking responsibilities. Show current holders, scopes, sources, gaps, and conflicts.

### Responsibility matrix

Rows represent objects or object classes; columns represent performer, owner, proposer, reviewer, challenger, authorizer, signatory, transmitter, acknowledgement recorder, and escalation role.

The matrix must support filters, inheritance, exceptions, effective dates, and source-backed suggestions without becoming an unrestricted spreadsheet.

### Decision-authority matrix

Define decision type, scope, thresholds, authority, quorum, challenge, signatory, expiry, and emergency path.

### Routing and escalation sequence builder

Use plain-language steps such as:

```text
Ask the scoped control owner to propose
→ review by Control Assurance
→ if high impact, challenge by Operational Risk
→ authorize by entity CRO
→ if external, approve representation by signatory
→ transmit through Regulatory Affairs
→ record authority acknowledgement
→ if overdue for 24 hours, notify function head
→ if unresolved at deadline, escalate to committee secretary
```

### Simulation and impact preview

Before activation, test representative scenarios:

- who receives each step;
- why they were selected;
- what authority they have;
- whether any conflict or missing role exists;
- expected timers and escalations;
- number of active workflows affected;
- whether current decisions or invitations become invalid.

Configuration changes require versioned draft, maker-checker approval, scheduled activation, rollback, and audit.

## 11. Runtime work experience

Every assigned user should immediately see:

- why they were selected;
- their current responsibility;
- scope, deadline, and authority;
- what has already been done;
- what they may and may not do;
- the one dominant next action for them;
- delegate, redirect, conflict, or escalate routes where allowed;
- the next owner after their action.

Parallel actors may have different dominant actions in the same Matter.

## 12. Committee and quorum handling

A committee is not a shared user account.

A committee decision must record:

- mandate and version;
- eligible members;
- quorum;
- chair or signatory;
- conflicts and recusals;
- materials reviewed;
- votes, approvals, abstentions, dissent, and conditions;
- effective time and expiry;
- actions and verification.

Meeting mode may simplify presentation but must not weaken individual identity or authorization.

## 13. Break-glass and emergency authority

Emergency access or decision paths require:

- explicit emergency basis;
- narrow scope and prohibited actions;
- time limit;
- named accountable authority;
- rationale and evidence;
- immediate or prompt notification to independent oversight;
- automatic expiry;
- mandatory retrospective review;
- restoration, correction, or remediation where misuse occurred.

“Administrator override” is not an acceptable emergency model.

## 14. Data and architecture requirements

Authoritative records include:

- role templates and versions;
- positions and source references;
- responsibility assignments;
- authority grants and limits;
- routing and escalation policies;
- delegations, substitutions, conflicts, and recusals;
- Decision lifecycle actors (proposer, reviewer, challenger, authorizer);
- external-response lifecycle actors (preparer, reviewer, rejector/withdrawer, signatory, transmitter, acknowledgement recorder);
- approval instances and votes;
- effective and record time.

Runtime resolution should use a materialized assignment index keyed by tenant, scope, relationship, role, responsibility type, and valid period. Policy results may be cached only with tenant, purpose, policy version, object version, and authorization context in the key.

Material actions re-evaluate authority at execution even when routing was previously resolved. Current-record actor fields support explanation/audit; they do not substitute for execution-time authority resolution.

## 15. Performance targets

Initial service targets:

- common actor resolution: p95 under 100 ms uncached and 25 ms cached;
- material authorization decision: p95 under 150 ms excluding external identity lookup;
- sequence preview for a typical policy: p95 under 500 ms;
- policy simulation for 100 representative scenarios: under 10 seconds asynchronously, with progressive results;
- active-work reassignment after identity revocation: visible within 60 seconds of accepted source event;
- no N+1 directory or policy calls when rendering work queues.

These targets must be validated against the reference workload in the system architecture.

## 16. Acceptance scenarios

### A. Routine evidence review

The scoped control owner submits evidence. The owner cannot approve it. Control Assurance reviews. A low-impact accepted result proceeds without executive approval.

### B. Material exception

A high-impact, 90-day exception requires performer, control owner, independent risk challenge, entity CRO authorization, conditions, expiry, and verification. No actor may approve their own submission.

### C. Owner unavailable

The primary owner is on leave. An approved substitute receives the request. The original owner remains accountable and sees the handoff on return.

### D. Organization change

An employee changes entity after assignment. Existing drafts remain, but authority is re-evaluated and material action is blocked or rerouted.

### E. Conflict

An investigator is related to a protected-report subject. The actor declares conflict; content access is removed; an independent investigator is resolved without revealing identity to the ordinary queue.

### F. Routing failure

No valid authorizer exists for a new entity. The Matter enters a routing-failure state and escalates to governance configuration. It is not assigned to a global administrator by default.

### G. Emergency action

An authorized executive uses a narrow break-glass path during a critical incident. The grant expires automatically and triggers independent retrospective review.

### H. Regulator response

A preparer drafts the response, a reviewer checks it, an authorized signatory approves the institutional representation, a permitted transmitter sends it, and an acknowledgement recorder records receipt. Each step preserves the verified actor; no earlier actor is silently treated as having completed a later responsibility.

## 17. Prohibited shortcuts

Do not:

- use one assignee for responsibility and authority;
- equate application role with material decision authority;
- let users delegate broader powers than they possess;
- permit silent self-approval;
- treat signatory approval as proof of transmission or acknowledgement;
- hide escalation ownership in notification configuration;
- create customer-specific hard-coded approval chains;
- let configuration changes affect in-flight work without impact review;
- use stale cached authority for material execution;
- route protected work through ordinary queues;
- treat committee approval as a generic status change.

## 18. Definition of success

The model succeeds when a bank can define a complex authority structure in familiar terms, simulate it before activation, resolve the correct people at runtime, continue safely through absence and organization change, and explain exactly why each person was asked, what they were allowed to do, and how unresolved work escalated.
