# Program and Work operational completeness

**Status:** Approved direction
**Date:** 2026-08-25
**Delivery order:** Work and Matters first on shared foundations, then Programs

## Decision

ClearSight will expose every supported Program and Matter operation through an authority-aware working interface. A bank user must not need the API, SQL, seed code or browser developer tools to maintain a current Program, resolve missing information, assign and complete work, record an independent outcome, or understand why another person must act.

The existing Program and Work lists remain portfolio-scanning surfaces. Opening a specific record launches a dedicated workspace for maintaining that record. The workspace shows one dominant next action, the responsible person or function, the deadline, the current blocker, and the complete governed context needed to act. Focused forms use ordinary banking language and object-specific commands; ClearSight will not expose generic JSON or database editors.

This work closes the current API-only product gap in two increments:

1. complete the Matter journey and the shared participant, authority and operation-discovery foundations;
2. use those foundations to complete ongoing Program maintenance.

Both increments belong to this design and acceptance contract. Work-first is sequencing, not a reduction in the final scope.

## Baseline and defect statement

The deployed stakeholder UI on 2026-08-25 proves the following failures:

- the regulatory-change Matter reports two missing facts but provides no way to supply or resolve them;
- recorded facts such as filing channel and filing deadline are read-only;
- the CRO sees an open action but cannot see that it is assigned to the Program Owner;
- the Program Owner can transition the assigned action but cannot correct the Matter information needed to complete it;
- the outcome-check contract is visible only as a collapsed count or result label, and result capture is available only when a narrowly shaped workflow packet is present;
- Draft and early-stage Matters can display a next action such as “Confirm scope and owner” without an executable control;
- existing Programs expose review acknowledgement, lifecycle status and monitoring setup, but not complete requirement, applicability, safeguard, ownership, evidence-expectation or assessment maintenance;
- the UI provides no systematic assurance that a supported material command is reachable from the product.

This is an operability defect, not a copy or styling defect. The repository already contains many domain commands, but the interface exposes only a subset and hides responsibility when the current actor is not the routed performer.

## Options considered

### 1. Add isolated buttons to the current accordions

This is the smallest visual change, but it preserves a cramped read-first layout, repeats permission logic in components and leaves future commands easy to omit. It does not provide a reliable end-to-end operating journey.

### 2. Add an object-specific operation layer and dedicated record workspaces — selected

The API derives actor-visible operations, routing and referenced participant labels from current domain state and authority policy. Dedicated Matter and Program workspaces render focused, typed forms for those operations. Commands still execute through the canonical services and are re-authorized at submission time.

This approach makes the system usable without weakening authority, avoids parallel workflow state and provides a durable way to prevent new API-only gaps.

### 3. Add a generic record/JSON editor

This could expose fields quickly, but it would leak internal schemas, allow invalid combinations, obscure authority distinctions and make audit history difficult to interpret. It is rejected.

## Shared operating contract

### Actor-visible operation projection

Each Program and Matter detail read will include, or be paired with, a bounded actor-scoped operation projection. The projection is derived from current aggregate state, lifecycle rules, referenced workflow tasks and current authority resolution. It is not authoritative state and does not replace the command guard.

Each operation describes:

- the canonical command name;
- the business action label;
- whether the current actor may act;
- the responsibility required;
- the currently responsible principal or function, with a display label;
- why the action is available, unavailable or assigned elsewhere;
- valid lifecycle targets or eligible assignment candidates where applicable;
- the aggregate and subresource versions the command will require;
- the resulting state or follow-up expected after success.

The operation read is exact and bounded to one aggregate. It must not load a broad principal or record population and filter it in the browser.

### Responsibility and participant display

Every visible Program, Matter, action, decision stage, response stage and outcome check shows the current responsible person or function when one is recorded or resolved. Raw principal IDs remain available in audit detail but are not the primary label.

Referenced participants are resolved server-side through exact tenant-scoped lookups. Assignment controls show only candidates returned by current routing and eligibility policy. A client-supplied actor, approver, reviewer or signatory is never trusted. The material command re-resolves authority, visibility, delegation and conflict immediately before execution and fails closed if the route changed or is unavailable.

Users who cannot perform an operation see a read-only explanation such as “Assigned to Program Owner” or “Internal Audit must record this outcome after the observation period.” The UI does not hide essential ownership merely because the viewer cannot act.

### Command and audit behavior

Every new material command uses optimistic concurrency and records, in one transaction:

1. the authoritative row change;
2. an append-only continuity event with the before/after meaning and rationale;
3. the outbox event;
4. any required workflow-task or Program-status maintenance work;
5. the new aggregate version.

A command returns success after this transaction commits. Projection or explanation work that fails later cannot turn the committed command into a reported failure.

## Matter workspace

### Structure

The Work list remains a searchable portfolio view. Opening a Matter route displays a dedicated workspace with:

1. **Current handoff** — state, reason, responsible person, deadline and one dominant action;
2. **Issue details** — type, title, summary, priority, due date, accountable owner, affected area, source and linked Programs;
3. **Information** — recorded facts, information still needed and contradictions;
4. **Decision and response** — the current decision or response stage, authority and history relevant to the Matter type;
5. **Actions** — work description, owner, deadline, state, blockers and completion evidence;
6. **Outcome checks** — expected result, linked action, independent reviewer, timing, threshold, evidence and latest result;
7. **Closure and history** — explicit closure blockers and reconstructable event history.

Sections irrelevant to the Matter type remain absent. Complete history is reachable but does not compete with the current handoff.

### Issue details and ownership

Authorized users can:

- correct the title, summary, priority, due date and affected area;
- assign or reassign the accountable owner from eligible current candidates;
- link the Matter to a scoped Program when the domain permits it;
- move the Matter through valid lifecycle stages with the required rationale;
- close or reopen only when the existing closure rules allow it.

Other users can see the same current owner, due date and route explanation without receiving an enabled command.

### Facts, missing information and contradictions

The information section uses task-specific controls rather than a JSON textarea.

Authorized users can:

- add a recorded fact with a business label and scalar value;
- correct a fact with a rationale and optional evidence references;
- mark a fact as no longer current without erasing its history;
- add an item to “Information still needed”;
- resolve a missing item by recording the answer atomically;
- record a contradiction and later resolve it with a rationale.

Resolving missing information removes it from the current missing list and records the supplied fact in the same command. Filing channel and filing deadline therefore become normal correctable facts, while the Matter due date remains a separate operational deadline.

The command event preserves the prior value, new value, reason, actor, time and evidence references. The current aggregate retains the existing `known_facts`, `missing_facts` and `contradictions` compatibility shape while reconstruction uses the events.

### Actions

The actions section always shows title, description, owner, due date and current state.

Authorized users can:

- add an action;
- edit its working description and deadline before closure;
- assign or reassign it to an eligible performer;
- start it, mark it blocked, resume it, mark implementation complete or cancel it when the lifecycle permits;
- record a concise rationale when the selected transition requires one.

Changing an action owner updates the authoritative action and the current projected workflow task transactionally. The old assignee must not retain an active task after reassignment, and the new assignee must not need an API or restart to receive it.

“Implemented” remains separate from “Outcome confirmed.” Action completion cannot silently close the Matter or produce a green verified state.

### Decisions and responses

The workspace exposes the existing decision and response lifecycles at the point of work:

- propose, review, challenge, authorize, reject or return a decision as allowed by the canonical lifecycle;
- create and progress response packages for Matter types that require them;
- show the current proposer, reviewer, challenger, authority or signatory at each stage;
- preserve prior decisions and responses without treating a historical approval or acknowledgement as current.

### Outcome checks

Authorized users can define an outcome check from the Matter workspace by selecting the action being checked and recording:

- the expected outcome;
- the observation scope and period;
- the success threshold in a typed, human-readable form;
- the independent reviewer or reviewer route;
- the failure response;
- any measurement source or required evidence.

Once the observation period is ready, the assigned independent reviewer can record **Outcome confirmed**, **Outcome not achieved** or **More evidence needed**, together with observations, evidence references and rationale. The action owner cannot self-verify when independence is required.

A failed result invokes the existing configured failure response atomically. Closure blockers refresh after every result and remain visible in ordinary language.

### Matter operation coverage

The completed workspace must expose usable paths for every supported Matter command:

| Command | UI location |
| --- | --- |
| `matter.create` | Work list — New issue or change |
| `matter.details.update` | Issue details — Edit details |
| `matter.context.change` | Information — Add, correct, resolve or contradict |
| `matter.assign` | Issue details — Change owner |
| `matter.transition` | Current handoff / Closure |
| `matter.link` | Issue details — Link Program |
| `matter.decision.record` | Decision and response |
| `matter.action.add` | Actions — Add action |
| `matter.action.update` | Action details — Edit |
| `matter.action.assign` | Action details — Change assignee |
| `matter.action.transition` | Current handoff / Action details |
| `matter.response.add` | Decision and response — Prepare response |
| `matter.response.transition` | Current handoff / Response details |
| `matter.outcome.define` | Outcome checks — Add outcome check |
| `matter.outcome.record` | Current handoff / Outcome check details |

The new update, context and assignment commands are object-specific domain commands. They do not authorize arbitrary field mutation.

## Program workspace

### Structure

The Programs list remains the ongoing-obligation portfolio. Opening a Program displays a dedicated workspace with:

1. **Current position** — calculated status, freshness, reasons and one dominant next action;
2. **Program details** — owner, authority, function, jurisdiction, scope and effective dates;
3. **Requirements and applicability**;
4. **Safeguards and requirement coverage**;
5. **Evidence expectations, monitoring and assessments**;
6. **Issues and changes affecting the Program**;
7. **Review history and reconstruction**.

### Program maintenance

Authorized users can perform the existing Program commands without API work:

- create a Program with separate server-authorized accountable-owner and approval-authority selections, then complete setup;
- correct Program working details and change eligible ownership or approval authority through separate versioned commands;
- add a requirement with its source anchor;
- record the current applicability decision and rationale;
- define a safeguard objective and implementation;
- link requirements to safeguards;
- define evidence expectations;
- record evidence assessments separately from submissions and monitoring observations;
- create and operate monitoring checks;
- create or open a linked issue from a gap or change;
- mark the current state reviewed;
- request activation, pause, reactivation or retirement when permitted.

Existing requirements, safeguards and evidence expectations show owner, status, source, effective version and available correction/supersession action. An authorized owner can remove an incorrect requirement-to-safeguard or issue-to-Program relationship only after recording a reason; the relationship stops contributing to current coverage, issue counts and status while its actor-backed event remains available for point-in-time reconstruction. Material historical versions are never overwritten or hidden.

### Program operation coverage

The implementation will maintain a command-to-UI coverage manifest for:

- `program.create`;
- `program.details.update`, `program.assign` and `program.approval-authority.assign`;
- `program.transition`;
- `program.requirement.add` and the supported supersession/correction command;
- `program.applicability.decide`;
- `program.safeguard.define`, `program.safeguard.link` and `program.safeguard.unlink`;
- `matter.link` and `matter.unlink`;
- `program.evidence.define`;
- `program.evidence.assess`;
- supported monitoring-check, form and collection commands;
- Program review acknowledgement;
- creation/opening of linked Matters.

## Preventing future API-only features

The repository will contain an executable operational-coverage manifest. Every production material command route must declare one of:

- a customer-facing UI entry point with applicable record states and roles;
- a service-only classification with a documented machine actor, purpose and non-UI justification;
- an intentionally unavailable status with a visible product limitation and tracked delivery item.

CI fails when a new material command is added without one of these declarations, when a customer-facing entry points to a missing component, or when an enabled control lacks a tested command path. A fixed phrase scan is insufficient; component and workflow tests must prove the operation in its relevant state.

## Interaction and error behavior

- Forms open in focused inline panels or sheets within the dedicated record workspace; routine actions do not require module hopping.
- Known values are shown as context and are not requested again.
- Each form has one primary submit action and a clear cancel action.
- Validation preserves entered values and focuses the first invalid field.
- A stale version reloads the affected record and explains that another person changed it; ClearSight never silently overwrites.
- A route or authority change fails closed, retains the form values where safe and identifies the newly responsible person or recovery step.
- A command-service failure cannot be presented as success.
- A post-commit projection delay shows the committed receipt and marks derived status as updating rather than returning a false command failure.
- Missing participant labels fall back to a clearly labelled recorded identifier; they do not become blank owners.
- Disabled controls explain the exact missing prerequisite. Controls with no behavior are not rendered as enabled.

## Responsive and accessibility behavior

- Desktop uses a narrow record summary rail only when it leaves sufficient width for the current form and evidence.
- Tablet and 200% zoom replace parallel layouts with a single content column.
- Mobile opens focused actions as full-screen steps and keeps the record title, responsible person and save/cancel actions visible.
- All controls meet the supported touch target, keyboard, focus, screen-reader and reduced-motion requirements.
- Facts and status are never encoded only by color or icons.
- Long titles, translated labels, unknown owners, no-data states and permission-limited states have explicit fixtures.

No new visual token, illustration style or motion pattern is required. Existing semantic tokens and restrained operational styling remain authoritative.

## State fixtures

Rendered fixtures must cover at least:

- CRO viewing work assigned to another person;
- accountable owner resolving missing information and correcting a fact;
- owner adding and reassigning an action;
- assignee starting, blocking, resuming and completing implementation;
- independent reviewer recording pass, fail and inconclusive outcomes;
- authority route unavailable or changed after form open;
- optimistic version conflict;
- Draft Matter with an executable initial-review action;
- closure blocked and ready-to-close states;
- active Program with incomplete evidence;
- Program requirement, applicability, safeguard and evidence maintenance;
- stale Program projection after a committed change;
- empty, loading, unavailable and long-content states;
- desktop, tablet/200% zoom and mobile layouts in light and dark presentation.

## Acceptance journeys

### Regulatory-change Matter

1. The CRO opens the annual-return Matter and sees that the action is assigned to Program Owner, with its deadline and current blocker.
2. The Program Owner changes the filing channel with a rationale, resolves both missing-information items and records the supplied facts.
3. The owner adds or updates the annual-return checklist action, assigns an eligible performer and records the due date.
4. The performer completes the action without closing the Matter.
5. The independent Internal Auditor sees the ready outcome check and records observations, evidence references and a result.
6. A passing result satisfies the outcome blocker; an authorized actor can close with a rationale.
7. The event history reconstructs every correction, assignment, action transition, outcome result and closure.

### Continuing Program

1. The Program Owner opens the Nigeria data-protection Program and sees current evidence gaps, the accountable owner and next action.
2. The owner adds or supersedes a source-anchored requirement, records applicability, defines and links a safeguard, and defines the evidence expectation.
3. A reviewer records an evidence assessment; the submission or monitoring result remains a separate observation.
4. A material gap opens or links to a Matter instead of being hidden by a manually selected status.
5. The Program status recalculates from the current version and displays freshness while the projection catches up.
6. Another authorized reviewer can mark the current state reviewed without changing the underlying Program conclusion.

## Test and release proof

Implementation is complete only when:

- domain tests prove each new command, validation rule, event, reconstruction and optimistic conflict;
- PostgreSQL integration tests prove the authoritative row, append-only event, outbox event and required workflow maintenance share one transaction;
- authority tests prove actor fields are server-bound, tenant/legal-entity scope is checked, assignments are limited to eligible current candidates and route failure is fail-closed;
- projection tests prove action reassignment removes stale work and creates the correct current work without duplication;
- component and browser tests prove both acceptance journeys across their actor handoffs;
- the operational-coverage manifest accounts for every production material command;
- copy-quality, accessibility, strict TypeScript, Go, PostgreSQL integration and production-build checks pass;
- the existing deployed screens are retained as before-state evidence;
- materially affected Work and Program workspaces are rendered at 1440px, 1024px/200% replacement and 390px in light and dark presentation;
- the highest-impact visual or interaction defect found during rendered review is fixed and rechecked;
- the implementation ledger, API contract, architecture notes, acceptance matrix and `DESIGN.md` remain synchronized.

## Out of scope for this design

This design does not add arbitrary customer-defined schemas, direct regulatory filing/transmission, legal interpretation, bulk population actions, a complete regulatory library, production object-storage upgrades or new AI decision authority. Those capabilities have separate product and safety contracts.

The absence of those later capabilities does not permit any currently supported Program or Matter operation to remain API-only.
