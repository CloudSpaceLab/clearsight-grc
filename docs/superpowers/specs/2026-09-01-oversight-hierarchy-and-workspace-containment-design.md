# Oversight, reporting hierarchy and workspace containment design

**Date:** 2026-09-01

**Status:** Approved by the operator on 1 September 2026

**Scope:** Form Builder pane containment, single owner-change action, manager-authorized reassignment, governed reporting hierarchy administration, assignee notification, and CRO/GRC operational oversight

## Outcome

ClearSight will keep long authoring workspaces stable while their contents scroll, expose one discoverable owner-change action per record, let the current owner and eligible reporting ancestors redirect work without inheriting execution or approval authority, let System Administrators maintain effective-dated reporting structure through governed change, and give CRO/GRC roles a freshness-labelled view of risk pressure, resolution outlook and operating performance.

## Evidence and root causes

The deployed Form Builder and Configure workspace were inspected from operator screenshots and traced to the exact `origin/main` implementation at `e113487624391ffff52e225fbb2d7a49a1ef8f7f`.

1. `.form-builder-outline` is sticky and scrollable, but `ReusableSectionPicker` and the example action are siblings outside it. Those siblings continue through the page scroll while the navigation is pinned, creating the observed overlap. The scroll boundary belongs to the whole outline pane.
2. `MatterCurrentHandoff` renders a dominant **Change issue owner** control which finds and clicks the second control in `MatterDetailsPanel`. Two visible controls therefore invoke one command.
3. `matter.assign` and `program.assign` currently validate the stored owner for owner-bound commands. They do not yet admit an active reporting ancestor solely for the narrow reassignment command.
4. The authoritative organization schema already contains effective-dated `org_positions.parent_position_id`, occupants, role bindings and department paths. The current Configure UI exposes people and escalation data as compact card text, but has no usable position, reporting-line or impact-preview workflow.
5. Department ancestry currently scopes governed escalation and does not itself grant authority. A reporting-line change must preserve that boundary.
6. Oversight facts exist across Programs, Matters, Actions, Workflow Tasks, escalation timers and outcome checks. Broad browser aggregation would produce inconsistent snapshots and unbounded reads. Oversight needs an authorization-scoped, bounded projection with explicit generation time and source coverage.

## Approaches considered

### Surface repair only

Repair CSS, remove the duplicate button and restyle Configure. This is insufficient because manager reassignment and authoritative reporting configuration remain absent. Rejected.

### Surface repair plus live dashboard queries

Add hierarchy editing and compute dashboard cards from several live endpoints in the browser. This avoids a projection but creates inconsistent counts, expensive fan-out and poor historical resolution estimates. Rejected.

### Layered operating model — selected

Land the interaction defects first, add effective-dated reporting administration and narrowly governed reassignment second, then build one bounded oversight projection and workspace. This keeps quick fixes shippable while establishing correct data and authority semantics for the larger experience.

## Design

### 1. Stable Form Builder panes

The desktop builder owns one scroll boundary per side pane:

- `.form-builder-outline-shell` becomes the sticky, height-bounded, overflow container;
- the outline navigation, add actions, reusable-section picker and example action remain in normal flow inside that container;
- the inner outline navigation is neither sticky nor independently scrollable;
- the inspector keeps an equivalent pane-level scroll contract;
- the central canvas owns document scrolling without moving controls through either side pane;
- opening a field type, condition or section selector must not lock or resize the document root;
- responsive sheets replace the desktop panes below the existing breakpoint and remain single-scroll surfaces.

Acceptance covers long outlines, expanded reusable-section content, keyboard focus, 200% zoom, light/dark modes and 1440px, 1024px, 768px, 390px and 320px widths.

### 2. One visible owner-change action

Each record renders at most one control for a given operation.

- When `matter.assign` is the selected dominant handoff, the handoff renders the only **Change issue owner** control and opens the shared owner sheet directly.
- When another operation is dominant, the owner control appears once beside the accountable-owner fact.
- The same suppression rule applies to `program.assign`, `matter.action.assign` and equivalent work-owner commands.
- A read-only handoff may explain who can redirect work, but it must not repeat a disabled version of the detail action.

The sheet remains populated only with server-returned candidates and requires a reason. The actor continues to come exclusively from verified request identity.

### 3. Reporting hierarchy semantics

ClearSight remains downstream of the directory rather than becoming a person directory.

Authoritative organization records are:

- source-backed principals;
- effective-dated organizational positions;
- position occupants and approved deputies;
- position-to-parent-position reporting edges;
- position-role bindings;
- scoped responsibility assignments, delegations and substitutions;
- versioned routing and escalation policy definitions.

`department_path` remains a governed scope and escalation input. It does not become a general authorization inheritance mechanism.

An eligible reporting ancestor for reassignment is the active occupant of an effective ancestor position of the current owner's active position, within the same tenant and legal entity, after conflict, visibility, active-principal and policy checks. The traversal is bounded, cycle-safe and uses current authoritative state at command time.

### 4. Narrow manager reassignment authority

For `matter.assign`, `program.assign` and approved owner-change command classes, execution is permitted when the verified actor is:

1. the current accountable owner;
2. an actor selected by the existing exact responsibility route when the record is unassigned;
3. an eligible active reporting ancestor under the current reassignment policy;
4. an active approved substitute/delegate for one of the above; or
5. a time-bounded emergency reassignment actor under an approved break-glass policy.

This permission authorizes only the owner-change command. It does not let the manager perform the work, approve it, sign, close it, change evidence or inherit the employee's other visibility.

Candidate selection still resolves the eligible replacement from current scoped authority. A manager cannot assign an arbitrary principal. Cross-tenant/entity, inactive occupant, stale hierarchy, cycle, conflict, hidden candidate, missing route or authority-service failure fails closed.

Every successful handoff records previous owner, new owner, verified initiating actor, reason category and rationale, policy/hierarchy version, command version and effective time in the existing material transaction and append-only event path.

### 5. Assignee notification

Manager-initiated and owner-initiated handoffs use the same existing assignment outbox events and staff-notification delivery path.

The new assignee receives one idempotent message containing:

- bank/legal-entity context;
- record type, title and reference;
- assigned responsibility;
- reason for the handoff;
- due date and current next action;
- an authenticated HTTPS deep link to the exact record.

The former owner may receive a non-blocking change notice when an active contact exists. Notification delivery never grants record access and does not roll back a committed reassignment. Receipts remain redacted and replay-safe.

### 6. Configure information architecture

`#configure/access` becomes a focused administration workspace with five views:

1. **People** — bounded, searchable, read-only directory inventory with source and activity state.
2. **Positions and roles** — position, occupant, department, scoped roles, vacancies and deputies.
3. **Reporting lines** — accessible hierarchy tree plus sortable table fallback, orphan/cycle/gap detection and focused editing.
4. **Escalation routes** — plain-language sequences, time thresholds, source/target guards, fallbacks and representative simulations.
5. **Change history** — pending, approved, scheduled, active, superseded and rolled-back revisions with actor and rationale.

Editing a reporting line creates a versioned proposal. The impact preview shows affected active assignments, reporting descendants, routing gaps, escalation paths and invalidated substitutions. A different authorized checker approves and effective-dates the proposal. The current version remains active until activation succeeds; rollback restores a prior approved version without rewriting history.

System Administrator may maintain identity, position and reporting proposals but does not gain risk-decision authority. GRC Administrator may draft governed routing where capability and policy permit. CRO may inspect organization-wide oversight but does not automatically configure identity or approve their own configuration proposal.

### 7. Oversight read model

The oversight API reads one bounded projection keyed by tenant, legal entity, period, category and organization scope. Projection rows are generated from authoritative record/event versions and contain no client-computed authorization.

The response includes:

- `generated_at`, projection version and source high-water marks;
- coverage population and excluded/unknown counts;
- intervention counts for critical/high issues, overdue work, due-soon work, routing failures, unassigned work and failing outcome checks;
- aging buckets and current owner/function/category dimensions;
- completed-volume, median and p75 cycle time, SLA attainment, return rate, reopen rate and reassignment rate;
- resolution estimates with historical cohort, sample size, median range and confidence class;
- bounded drilldown identifiers for exact records.

Unknown denominators remain unknown. Stale projections remain visible with a warning and cannot be labelled current.

Resolution estimates are deterministic in the first version: historical durations are grouped by legal entity, work type, category, priority and current state. Sparse cohorts widen to a documented parent cohort; if the minimum sample is still not met, no estimate is shown. The UI never invents a completion date.

### 8. CRO/GRC oversight workspace

The workspace is data-dense but intervention-first:

```text
Scope, period and freshness
┌ Critical/high ┐ ┌ Overdue ┐ ┌ Routing gaps ┐ ┌ Outcome failures ┐
What needs attention now — ranked record list with owner, reason and next action
Risk pressure — type × criticality/status
Resolution outlook — aging, due dates and estimate confidence
Operating performance — function/team metrics and workload context
Where to improve — deterministic exception clusters and exact drilldowns
```

Charts always show exact values and have a table alternative. Category comparisons use ordered bars; time trends use lines with non-colour differentiation; resolution forecasts use ranges and confidence rather than a single asserted date. Keyboard drilldown, reduced motion, light/dark contrast and narrow-layout replacements are required.

Individual performance drilldown is authority-scoped and contextual. It exposes completed volume, current load, median/p75 cycle time, SLA attainment, returned/reopened work and reassignments. It does not create a composite employee score or ranking. Leave, vacancy, blocked time and reassignment are shown so the bank does not mistake routing conditions for individual quality.

### 9. Role-accurate Today

Today is the verified actor's operational queue, not a shared executive summary and not a sample-data fallback.

Production Today combines only bounded, stored attention sources that the verified actor may act on:

- active Workflow Tasks assigned to the actor or to a governed queue/role the actor currently occupies;
- exact review, challenge, authorization, signatory, transmission and acknowledgement work resolved to the actor;
- evidence/vendor requests assigned to the actor;
- escalations explicitly routed to the actor;
- for identity/configuration administrators, actionable source failures, provisioning failures, routing gaps, failed timers, pending checker decisions and scheduled configuration activations that require their current capability and responsibility.

Every item carries a real target, current responsibility, reason, deadline/freshness and executable next action. Loading or source failure produces an unavailable state; the browser never substitutes persuasive static items. Demo environments use normal seeded authoritative rows and the same projection path. `fallbackItems`, `today.DemoItems()` in runtime composition and fixture-specific `/api/v1/today` responses are not permitted as an operational data source.

Role behavior is explicit:

- performer/owner sees assigned work and accepted delegations;
- reviewer/challenger/authorizer/signatory sees only the exact routed step currently requiring that responsibility;
- manager sees their own work plus separately labelled escalations routed to them, not every subordinate task by default;
- CRO/GRC oversight access does not turn all visible records into personal Today assignments;
- System Administrator sees actionable identity, routing and system-operation exceptions, not risk decisions merely because they administer the platform;
- an actor with no current work sees a scoped empty state naming the sources checked and generation time.

The server deduplicates work that appears through more than one source, sorts by overdue/materiality/deadline, rechecks current source visibility before returning it and caps results with a continuation token. Today shows `generated_at` and source health so a stale or partial queue is never presented as complete.

### 10. State fixtures and rendered evidence

Deterministic fixtures cover:

- long builder outline at top, middle and end scroll positions;
- expanded reusable-section picker without overlap;
- owner change dominant and non-dominant states with one control;
- current owner, direct manager, higher reporting ancestor, delegate and unauthorized peer;
- inactive owner, vacant position, cycle, conflict and missing hierarchy;
- new-assignee delivery, former-owner notice, contact unavailable and replay;
- populated, empty, stale and partially unavailable hierarchy workspaces;
- draft, impact preview, checker approval, scheduled activation and rollback;
- oversight current, stale, sparse-history, no-data and restricted-individual states;
- Today for performer, owner, reviewer, authorizer, CRO/GRC, System Administrator and an actor with no assigned work;
- Today with a routing failure, failed timer, pending checker decision, stale source and unavailable source;
- desktop/tablet/mobile and both themes.

## Acceptance

Completion requires:

- failing tests before each behavior change;
- no builder overlap or root-scroll shift under scroll, select-open and 200% zoom stress;
- no duplicate owner action in any rendered record state;
- exact tests for owner, ancestor, delegate, emergency, conflict, revocation, stale hierarchy, cross-entity and authority outage;
- reporting configuration simulation, maker-checker, effective dating, rollback and audit;
- reassignment notification remains outbox-backed, idempotent and redacted;
- oversight queries are bounded, tenant/purpose scoped and projection-freshness labelled;
- Today contains only stored actor-scoped work or capability-scoped operational exceptions, never a browser/server static fallback;
- every Today action rechecks current target visibility and authority before opening or executing;
- performance metrics preserve unknown/excluded populations and do not imply a compliance or employee-quality conclusion;
- copy-quality, UI-contract, accessibility, TypeScript, build, Go, PostgreSQL and migration gates pass on the exact final head;
- every materially affected workspace is rendered at the required sizes, inspected, corrected and re-rendered before deployment.

## Explicit remainder

This design does not add directory CRUD, payroll/HR performance appraisal, arbitrary managerial access to protected work, autonomous risk conclusions, a generic BI builder, qualified electronic signatures, or ungoverned administrator override. It does not change the rule that material execution re-evaluates current responsibility and authority.
