# ClearSight Contributor Rules

The words **MUST**, **MUST NOT**, **SHOULD** and **SHOULD NOT** are normative.

## Mission

Every change must help a bank stakeholder understand what must be done, what proves it, what changed, who is responsible, who must review or authorize, and whether the outcome was achieved—with minimum reasonable effort.

## Required reading

Read the root README, docs map, relevant product specification, application architecture, implementation plan and acceptance tests before changing behavior.

## Core rules

- Programs maintain continuity; Matters handle change and exception.
- Signals are observations, not incidents or conclusions.
- Drift changes attention or uncertainty; it does not silently change material risk or legal status.
- Responsibility, ownership, review, challenge, authorization, signatory and escalation remain distinct.
- One dominant next action is per current actor and workflow state.
- Existing evidence and authoritative sources are searched before asking a person.
- Task completion, upload, implementation and verified outcome remain separate.
- Material records are versioned and reconstructable.
- The product remains usable without AI or a live integration.

## Authority and workflow

Do not hard-code approvers or reduce governance to one assignee. Routing must use versioned policy, scope, organizational position, role, materiality, delegation, conflict and current workflow state.

Material execution re-evaluates current authority. Configuration requires simulation, impact preview, maker-checker approval, effective dating, rollback and audit.

## Continuous autonomy

Deterministic policy detects evidence aging, source degradation, requirement change, control failure, routing gaps and failed verification. AI may explain or propose handling but is not required for these controls.

Automated actions require an Automation Policy with purpose, action class, eligibility, blast-radius limit, rollback/compensation, monitoring, kill switch, expiry and Verification Contract.

## Guided experience

Important first-run and empty states require a designed experience. Premium illustrations may support comprehension but cannot carry status, conceal an error or replace actionable content. Guides are role-specific, skippable, resumable, accessible and non-blocking.

## Enterprise copy and content

- UI copy MUST read like bank operating software, not a product pitch.
- Use concrete objects, states, sources, owners, deadlines and actions. Prefer “3 approvals due this week” to “work that needs your judgment.”
- Do not use slogans, anthropomorphic claims, urgency theatre, vague reassurance or unverifiable language such as “continuously prepared,” “everything handled,” or “automatically maintained” without a defined population and timestamp.
- Status labels and counts MUST come from stored or explicitly labelled sample data. Unknown denominators display as unknown; they never fall back to a persuasive number.
- Empty states state the population or query checked, the current result and the next valid action. They do not imply enterprise-wide completeness.
- Demo copy MUST be operationally plausible and clearly identified as sample data.
- Icons and illustrations support orientation but never replace labels, status, evidence, errors or required actions.
- Copy changes require the same review as workflow changes because wording can alter authority, risk interpretation and user action.

## Invitations and protected data

Invitation access is opaque, request-scoped, purpose-bound, short-lived, audience-bound and revocable. Tokens never appear in logs, analytics or previews. Protected reporting uses a separate identity-isolated mechanism.

## Data and performance

- relational authoritative state;
- versioned object storage for artifacts;
- durable workflow, outbox/inbox and idempotency;
- strong consistency for material commands;
- explicit eventual consistency and freshness for projections;
- keyset pagination and bounded queries;
- tenant/purpose-bound caching;
- no broad data load followed by application-memory authorization;
- high-volume features require cardinality, partition, index, retention and load-test plans.

## Definition of done

A feature is complete only when:

- its use case and maturity are documented;
- actor routing and authority work under conflict, delegation, absence and revocation;
- source, evidence and contradiction semantics are correct;
- routine/checkpoint effort targets are met without quality regression;
- accessibility and degraded paths work;
- performance and recovery targets pass;
- point-in-time reconstruction is possible;
- documentation, ADRs, code and tests are synchronized.
