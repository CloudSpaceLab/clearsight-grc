# ClearSight Contributor Rules

The words **MUST**, **MUST NOT**, **SHOULD** and **SHOULD NOT** are normative.

## Mission

Every change must help a bank stakeholder understand what must be done, what proves it, what changed, who is responsible, who must review or authorize, and whether the outcome was achieved—with minimum reasonable effort.

## Required reading

Read the root README, root `DESIGN.md`, docs map, relevant product specification, application architecture, implementation plan and acceptance tests before changing behavior.

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

## Identity, authority and commands

- Production command actors MUST come from a verified request identity, not JSON fields, query parameters or browser state.
- Actor, approver, assessor, reviewer and signatory IDs supplied in request bodies MUST be ignored or overwritten from verified context.
- Tenant and legal-entity scope MUST match the verified identity before any material command runs.
- Material execution re-evaluates the current versioned authority route. Missing identity, missing route, tenant mismatch or authority-service failure MUST fail closed in production.
- Do not hard-code approvers or reduce governance to one assignee. Routing uses scope, organizational position, role, materiality, delegation, conflict and current workflow state.
- Configuration requires simulation, impact preview, maker-checker approval, effective dating, rollback and audit.
- A material command, authoritative row changes, append-only event, outbox event and required maintenance job MUST share one transaction.
- Do not return a command failure after the command has committed because a derived calculation failed later.

## Continuous autonomy

Deterministic policy detects evidence aging, source degradation, requirement change, control failure, routing gaps and failed outcome checks. AI may explain or propose handling but is not required for these controls.

Automated actions require an Automation Policy with purpose, action class, eligibility, blast-radius limit, rollback/compensation, monitoring, kill switch, expiry and an outcome-check contract.

## Guided experience

Important first-run and empty states require a designed experience. Premium illustrations may support comprehension but cannot carry status, conceal an error or replace actionable content. Guides are role-specific, skippable, resumable, accessible and non-blocking.

## Human-friendly enterprise copy

- UI copy MUST read like bank operating software used by a busy person, not a product pitch, legal memo or database console.
- Primary screens use the words a business owner, reviewer or executive would naturally use. Internal codes remain available in APIs, audit history and specialist detail.
- Translate technical domain terms where they do not help the immediate task. Prefer “Does this apply?” to “Applicability determination,” “Evidence incomplete” to “Evidence insufficiency,” “Outcome check” to “Verification contract,” and “What needs to happen next” to “Required handling.”
- Programs may remain a primary product noun because they represent ongoing obligations. Matters SHOULD normally appear as “Issues and changes,” “Findings,” “Requests” or the specific matter type unless the specialist context needs the canonical term.
- Use concrete objects, states, sources, owners, deadlines and actions. Prefer “3 approvals due this week” to “work that needs your judgment.”
- Avoid noun chains, unexplained abbreviations and status codes in visible copy. `DECISION_REQUIRED` is an API state; “Decision needed” is UI copy.
- Buttons begin with a familiar verb and describe the result: “Review obligations,” “Confirm account owners,” “Send for approval,” “Check the outcome.”
- Supporting text explains why the item is shown and what will happen next; it does not repeat the heading.
- Do not use slogans, anthropomorphic claims, urgency theatre, vague reassurance or unverifiable language such as “continuously prepared,” “everything handled,” or “automatically maintained” without a defined population and timestamp.
- Status labels and counts MUST come from stored or explicitly labelled sample data. Unknown denominators display as unknown; they never fall back to a persuasive number.
- A calculated status MUST identify whether it reflects the latest material version. Stale status may remain visible, but it cannot be labelled current.
- Empty states state the population or query checked, the current result and the next valid action. They do not imply enterprise-wide completeness.
- Demo copy MUST be operationally plausible and clearly identified as sample data.
- Icons and illustrations support orientation but never replace labels, status, evidence, errors or required actions.
- Copy changes require the same review as workflow changes because wording can alter authority, risk interpretation and user action.
- A visible enabled control MUST perform a real action. Disabled controls explain why they are unavailable.

### Customer-facing copy gate

- This gate applies to every customer-visible string from React components, server and API responses, onboarding, demo fixtures, empty states, errors, notifications, tooltips, labels, accessibility text and illustration descriptions.
- Every customer-facing sentence MUST identify a business object or task; state a condition, source, owner, deadline or freshness; explain why the current bank role must act; state the next action and result; or explain a limitation, consequence or recovery step. If it does none of these, remove or rewrite it.
- Copy MUST address the bank user at the point of work. It MUST NOT compare ClearSight with another product category, defend a product or design decision, narrate internal architecture, or teach implementation terminology that is unnecessary for the task.
- Product-review commentary is prohibited, including references to a “generic dashboard,” an “exact record,” canonical or bounded views, projections, authoritative servers, internal resolution behavior, implementation guarantees, and equivalent rewordings. This is a semantic rule; passing a fixed phrase scan is not sufficient.
- Headings name the task, record, state or decision. Buttons use a direct verb and name the immediate result. Supporting text adds status, context, consequence or recovery information instead of repeating the heading.
- Familiar role and governance acronyms retain their established casing, including CRO, CCO, CISO and GRC.
- Guides orient users to work in concise business language. They MUST remain optional, dismissible, accessible and non-blocking, including when progress cannot be saved.
- Simpler wording MUST NOT weaken authority, evidence, legal-scope, uncertainty or compliance limitations.
- A copy change MUST review the complete affected workflow and every relevant source, not only the edited phrase. Phrase-by-phrase substitution is insufficient when equivalent commentary remains elsewhere.
- When a new class of product narration can be detected reliably, extend `web/src/copyQuality.test.ts` with a semantic pattern that avoids broad false positives. The pattern list is a regression aid, not the complete writing standard.
- Before completion, run the copy-quality regression and affected workflow tests, render every materially affected workspace at relevant viewport sizes, and confirm that guides, notices and errors do not block primary actions.

## UI design proof

Significant screen, workflow or component changes require a compact decision brief, required state fixtures and rendered evidence. Redesigns preserve a before-state baseline. Responsive work defines replacement behavior rather than merely shrinking desktop composition. Inspect the render, fix the highest-impact failure and re-check it before claiming visual completion. New tokens, variants, density modes, motion patterns or illustration styles update `DESIGN.md` and the relevant state fixtures in the same change.

## Invitations and protected data

Invitation access is opaque, request-scoped, purpose-bound, short-lived, audience-bound and revocable. Tokens never appear in logs, analytics or previews. Protected reporting uses a separate identity-isolated mechanism.

## Data and performance

- relational authoritative state;
- versioned object storage for artifacts;
- durable workflow, outbox/inbox and idempotency;
- strong consistency for material commands;
- explicit eventual consistency and freshness for projections;
- command versions and derived-projection versions remain separate;
- projection work is deduplicated, bounded, leased, observable and recoverable;
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

## Bank vertical and jurisdiction rules

- Jurisdiction-specific fixtures must identify their official source and date and must be labelled as reference data, not legal advice.
- A vertical is not complete merely because records were created. Acceptance must reach its required decision, action, response, verification or closure outcome.
- Restricted records must be filtered by the API and repository scope; hiding a row in the browser is not an access control.
- Reference-journey reads must use exact indexed identifiers or bounded summaries. Do not replay broad Program or Matter populations to locate a known journey.
- Sample fixtures must use human working language, realistic owners and deadlines, and must never imply that the connected bank is compliant.
