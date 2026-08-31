# Governed work handoffs and staff notification design

**Date:** 2026-08-31

**Status:** Approved direction under the operator's standing instruction to use the recommended approach

**Scope:** Matter current handoff, Matter and Action reassignment, staff assignment email, vendor-work request composition, response authorization/signatory presentation, demo identity fixtures and rendered workflow evidence

## Outcome

ClearSight must show the exact next responsibility for an issue or change, let the currently authorized owner reassign eligible work, notify the newly assigned staff member through a durable protected channel, and present vendor requests and response/signing actions with the shared component system. A person must never see an unrelated action promoted merely because their role can perform it.

## Evidence and root causes

The deployed CRO journey at `#work/matters/01a04fd2-3f67-7157-ab5e-1f3fd3b5194f` was inspected on 31 August 2026.

1. The record said **Confirm scope and owner**, but `MatterCurrentHandoff` selected the first operation with `can_act=true`. For the CRO this was `matter.transition`, so the dominant action became **Authorize issue status** even though the stored accountable owner remained Program Owner and the next work was not an authorization.
2. The generic **Assigned to Chief Risk Officer** chip described the selected authorization operation, not Matter ownership. It visually contradicted **Accountable owner: Program Owner** below.
3. Matter reassignment is implemented and authority-gated in `MatterDetailsPanel`, but it is disconnected from the current handoff and still uses raw native controls. An actor who does not hold the current stored-owner route correctly cannot reassign, but the interface does not make that distinction clear.
4. `responsibility_labels_complete=false` is not a transient browser fault in the demo. `seed-demo-foundation.sh` creates principals and routing selectors but no active organizational positions. `access.PostgresResolver` intentionally resolves principal labels only for a current position or governed active directory membership, so the warning is expected from the seeded data.
5. `MATTER_OWNER_CHANGED` and `ACTION_ASSIGNED` already share the material command transaction, append-only event and outbox. The worker updates Workflow projections but has no staff-contact resolver or assignment delivery consumer. The previously approved vendor-email acceptance specification explicitly left authenticated staff notification channels open.
6. `VendorWorkPanel`, `MatterDetailsPanel`, `MatterDecisionResponsePanel` and `MatterOutcomePanel` use raw buttons, selects, textareas and inputs despite the closed contracts in `components/ui`. The vendor request shown in the deployed UI therefore retains browser-default menus and visually weak date fields.
7. Response signing is executable through `matter.response.transition` and `ResponseLifecyclePolicy`, but the UI collapses review, signatory approval, transmission and acknowledgement into the generic action **Update response status**. This weakens role distinction and makes the signing path difficult to discover.

## Approaches considered

### Cosmetic component sweep

Replace raw controls and adjust labels without changing operation selection or delivery. This is fast but leaves the CRO handoff wrong and leaves reassignment email absent. Rejected.

### Isolated bug patches

Patch the handoff selector, seed positions and add a direct email call after assignment. This would improve the screenshots but a post-commit direct send would not be durable or idempotent and would violate the command/outbox boundary. Rejected.

### State-first governed handoff — selected

Select the dominant action from canonical next-work state, keep ownership/authorization/signatory labels distinct, migrate the affected forms to shared contracts, and consume existing assignment outbox events through one durable staff-notification adapter. This fixes both meaning and interaction without creating a second workflow or authorization model.

## Design

### 1. Select the current handoff from canonical work

`MatterCurrentHandoff` uses a pure selector with this priority:

1. an active Matter Action explicitly named by `aggregate.next_action`;
2. a response-package or outcome-check operation that corresponds to the current compiled work packet;
3. a state-specific record operation:
   - **Confirm scope and owner** → `matter.assign`, falling back to `matter.details.update`;
   - decision work → `matter.decision.record`;
   - work execution → `matter.action.transition`;
   - outcome work → `matter.outcome.record`;
4. the first operation only when it is the sole executable or visible operation.

The selector does not prefer `can_act` globally. Read-only users see who owns the actual next step. A separate lower-priority action may remain available in its relevant panel but cannot replace the current handoff.

The handoff fact names the responsibility: **Accountable owner**, **Assigned performer**, **Reviewer**, **Authorizer**, **Signatory**, **Transmitter** or **Acknowledgement recorder**. It never uses a generic **Assigned to** label when responsibilities differ.

### 2. Keep reassignment operable and authority-correct

Only `matter.assign` and `matter.action.assign` candidates returned by the current authority operation may be selected. The request body supplies a candidate ID, but the material command re-resolves the current stored-owner route, scope, visibility and candidate eligibility before execution. The actor ID continues to come from verified request context.

The ownership form moves into a `FocusedSheet` opened by **Change issue owner**. It shows the current owner, eligible replacement, required rationale and the notification consequence. The existing expected Matter version is retained. Loading, disabled, conflict, no-route and no-candidate states remain explicit. The form uses `SelectField`, `TextArea`, `Button` and `Notice`.

Action reassignment receives the same control and copy treatment. An unavailable assignment route is explanatory read-only state, never an enabled inert control.

### 3. Send staff assignment notifications from existing outbox events

No email is sent from the HTTP request. `MATTER_OWNER_CHANGED` and `ACTION_ASSIGNED` remain the canonical triggers already committed with the material change. A worker consumer:

1. ignores unrelated events;
2. decodes only the safe assignment payload;
3. resolves the new principal's current staff contact from an active SCIM user whose `user_name` is a valid email address and whose source is active;
4. loads the exact Matter title, action title, due date and legal-entity display name from bounded current records;
5. builds an HTTPS deep link to the exact issue record from configured application base URL;
6. renders a shared ClearSight multipart message naming the assigned work, responsibility, assigning actor label when available, due date and next action;
7. records a redacted delivery receipt keyed by outbox event, principal and notification kind before the event is considered published.

The delivery ledger stores no recipient plaintext, message body or link token. The deep link contains only the Matter identifier and requires normal staff authentication. Missing contact data is a terminal **not deliverable** receipt that does not roll back the assignment. Temporary SMTP failure returns a retryable error; a recorded delivered or terminal receipt makes replay idempotent.

This tranche supplies event-backed email for Matter owner and Action performer assignment. It does not claim a generic notification centre, preferences, digests, bounce processing or arbitrary messaging.

### 4. Repair demo identity through seeded operational records

The demo foundation seeds active role templates, legal-entity organizational positions and position-role bindings for every demo principal. Those rows are the same records the access resolver uses in normal operation; the API does not special-case demo names.

The optional demo staff email is a deployment input. When present, the seed creates or updates one active demo SCIM source and the selected staff principal's `scim_users.user_name`. No real recipient address is committed. When absent, assignment remains fully functional and the notification consumer records **contact unavailable**.

The seed remains idempotent and fails on incompatible fixed identifiers. Tests prove that the seeded principal is resolvable through `access.PostgresResolver` and that no warning is emitted for the complete demo fixture.

### 5. Migrate vendor request composition to shared controls

The **Request vendor work** composer opens in a wide `FocusedSheet` so the issue record remains readable and the form has a predictable desktop position and a full-height mobile replacement. The form uses:

- `SelectField` for request type, relationship, approved form and layout;
- `TextField` for recipient email and native date semantics;
- `TextArea` for purpose and instructions;
- `Notice` for scope, unavailable data, partial delivery and validation;
- `Button` for cancel, load more and the one primary **Prepare and send request** action.

Required validation is shown at the relevant field after attempted submit or blur. The selected vendor/service, exact active form revision, presentation, recipient, due date and submission/acceptance distinction remain visible. Preparing and sending retains the truthful partial-outcome recovery path.

### 6. Make authorization and signing actions explicit

Matter status authorization, decision authorization, response review, response sign-off, transmission and acknowledgement retain distinct responsibilities and commands. `MatterDecisionResponsePanel` derives action copy from the operation responsibility and permitted target:

| Responsibility | Visible action | Confirmation result |
| --- | --- | --- |
| Reviewer | Review response | Records the reviewed or rejected response state |
| Signatory | Review and sign response | Records institutional approval; does not claim transmission |
| Transmitter | Record transmission | Records transmission evidence/state; does not claim acknowledgement |
| Acknowledgement recorder | Record acknowledgement | Records receipt/acknowledgement separately |
| Authorizer | Authorize issue status | Changes only the permitted governed Matter state |

The focused form shows current state, next state, role, consequence and rationale before submission. Generic **Update response status** copy is removed. Existing server authority and lifecycle policy remain the enforcement boundary.

### 7. Shared fields and date presentation

`TextField` accepts string `min`, `max` and `step` values required by native date/time controls as well as numbers. The shared field CSS owns date-indicator color, padding, focus and disabled/read-only contrast in light and dark modes. No feature CSS overrides component fill, border, radius or foreground.

The migrated files are added to `web/ui-contract-migrations.json`, the adoption ledger and deterministic state fixtures. `DESIGN.md` gains the responsibility-specific handoff and staff-notification presentation rules; no new palette or typeface is introduced.

## State and failure fixtures

Rendered fixtures must cover:

- CRO viewing **Confirm scope and owner** without seeing authorization promoted as the dominant action;
- Program Owner opening a populated reassignment sheet;
- no eligible replacement candidate;
- stale Matter version preserving entered rationale;
- assignee labels complete and partially unavailable;
- assignment email delivered, contact unavailable, temporary failure and replay;
- vendor request sheet in light and dark modes at 1440px, 768px, 390px and 320px/200% reflow;
- vendor form options open in dark mode;
- response review, signatory approval, transmission and acknowledgement as four distinct states;
- authorization unavailable because the current route cannot be resolved.

## Acceptance

Completion requires:

- failing tests first for wrong handoff selection, demo label resolution, assignment delivery replay/contact failure, shared-field rendering and responsibility-specific response actions;
- Matter/Action assignment still fails closed for forged actor, wrong tenant/entity, hidden candidate, stale version and changed authority route;
- the assignment row, event, outbox event and Workflow projection remain transactionally consistent;
- SMTP and contact data never appear in event payloads, logs, API responses, screenshots or committed fixtures;
- the new delivery ledger is additive, rollback-safe, bounded and registered in durable schema ownership;
- copy-quality, UI-contract, accessibility, TypeScript, build, default Go, PostgreSQL-tagged and migration gates pass on the exact head;
- the affected live workflows are rendered, the highest-impact defect is fixed, and the exact deployed SHA and readiness result are recorded before completion.

## Explicit remainder

This tranche does not complete a generic notification inbox, user preferences, mobile push, digests, bounce/complaint handling, directory contact governance beyond active SCIM email, qualified electronic signatures, document-signing providers, transmission adapters, representative-user testing or the broader third-party lifecycle in issue #80. Those remain separate work after the governed handoff is usable end to end.
