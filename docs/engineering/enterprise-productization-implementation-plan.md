# ClearSight Enterprise Productization Implementation Plan

## 1. Purpose

This plan converts the enterprise productization design into sequenced, testable implementation work.

It assumes the existing foundation on `main`:

- verified actor context and command authorization foundations;
- Programs, issues/changes, evidence and outcome semantics;
- authority resolution, governance policies and delegations;
- durable workflow, timers, outbox and inbox;
- Today, Programs, Work, Explore and Configure surfaces;
- role-labelled onboarding state;
- Nigerian-bank reference journeys;
- PostgreSQL repositories, migrations and integration tests.

The plan is intentionally stricter than a feature checklist. Every phase includes domain, API, UI, migration, security, recovery, performance and acceptance work.

## 2. Delivery principles

1. **No visual-only completion.** A control is not complete until the underlying command, authority check, audit record and recovery path exist.
2. **No authentication-only RBAC.** Enterprise identity establishes the actor; ClearSight still resolves scoped responsibility and authority.
3. **No silent directory authority.** Imported groups and positions do not automatically grant material decision rights.
4. **No notification leakage.** Subjects, previews and channel payloads are purpose- and sensitivity-minimized.
5. **No MFA theatre.** Step-up assurance is bound to a real policy and material command.
6. **No placeholder premium claim.** Light/dark assets, component states and rendered evidence must exist before visual completion is claimed.
7. **No broad in-memory authorization.** Tenant, legal-entity, purpose and restricted-record filters remain in bounded repository queries.
8. **No irreversible migration without dry run, observability and rollback.**

## 3. Target architecture additions

```text
Enterprise IdP / Directory / HR
        │
        ├── OIDC or SAML authentication
        ├── SCIM lifecycle sync
        └── LDAP/AD sync agent or controlled import
                    │
                    ▼
Identity source records → principal/position/group projections
                    │
                    ▼
Responsibility + authority + delegation + conflict policies
                    │
                    ▼
Actor-scoped workflow and command authorization
                    │
                    ├── role-aware onboarding
                    ├── notification policy and template registry
                    ├── email/messaging publishers
                    ├── step-up assurance service
                    └── premium web application and external capture
```

New services should remain modules in the modular monolith unless measured isolation, residency or throughput requires extraction.

## 4. Sequencing and dependency map

```text
P0 baseline and traceability
 ├─ P1 design system and UI finishing
 ├─ P2 identity-source and organization sync
 │    └─ P3 responsibility, authority and escalation administration
 │         └─ P4 role-aware onboarding
 ├─ P5 notification and template platform
 ├─ P6 MFA, session and step-up assurance
 ├─ P7 illustration, iconography and state completion
 └─ P8 pilot hardening and production release gates
```

P1, P5, P6 and P7 may progress in parallel after P0. P3 depends on stable source-backed principals and positions from P2. P4 depends on P2/P3 role resolution.

## 5. Phase P0 — Productization baseline and traceability

### Objective

Create an authoritative baseline of current screens, capabilities, gaps and release evidence before changing the product.

### Tasks

#### P0.1 Capability maturity register

Create a versioned register for each advertised capability:

- design specified;
- foundation implemented;
- pilot implemented;
- production gated;
- production validated.

Store owner, source docs, code modules, tests, known boundaries and next gate.

#### P0.2 UI inventory and before-state evidence

Capture every primary page at representative desktop, tablet and mobile widths in dark theme. Record:

- current information hierarchy;
- enabled and non-functional actions;
- component variants;
- missing states;
- hard-coded colors or styles;
- copy inconsistency;
- accessibility failures;
- dense-data limitations.

#### P0.3 Productization acceptance suite skeleton

Add test groupings for:

- identity/directory;
- authority/RBAC/escalation;
- onboarding;
- notifications;
- MFA/step-up;
- UI state and visual evidence;
- pilot performance and recovery.

#### P0.4 Documentation traceability

Every task in this plan must link to:

- use case;
- design section;
- data/authority contract;
- implementation issue or PR;
- behavioral tests;
- rendered evidence;
- release gate.

### Exit criteria

- no significant page or enterprise feature is unclassified;
- current screenshots and state fixtures are retained;
- the implementation backlog uses stable task IDs from this document;
- release gates are executable or have an explicit test-construction task.

## 6. Phase P1 — Design system, UI cleanup and operational finishing

### Objective

Turn the current coherent prototype into a complete reusable enterprise interface system without changing canonical domain meaning.

### P1.1 Token architecture

Implement theme-independent semantic tokens for:

- canvas/surfaces;
- text/borders;
- navigation/governance/attention/verified/blocking/unknown;
- focus and selection;
- charts;
- elevation;
- motion;
- density;
- responsive breakpoints.

Requirements:

- dark and light themes;
- no component-level hard-coded semantic colors;
- theme-safe SVG illustration variables;
- automated contrast checks for token combinations;
- print/export token subset.

### P1.2 Component library extraction

Create documented components for:

- application shell;
- page header/operating brief;
- status and reason block;
- work row and dense table;
- decision brief;
- authority explanation;
- timeline/history;
- side panel and mobile full-screen flow;
- form field, validation and source-prefill indicator;
- command confirmation and receipt;
- skeleton, empty, stale, unavailable, unauthorized and conflict states;
- matrix and sequence builder primitives;
- email preview and notification item;
- MFA challenge and security history.

Use Storybook or an equivalent isolated state gallery only if it remains part of CI and does not duplicate production markup.

### P1.3 Screen-by-screen finishing

Refactor Today, Programs, Issues and changes, Work, Explore and Configure to use the shared system.

For each screen:

- remove dead controls and generic labels;
- establish one dominant actor-specific next action;
- preserve filters and safe return context;
- implement long-content and unknown-population behavior;
- provide complete empty/degraded/unauthorized states;
- ensure keyboard and mobile replacement behavior;
- add compact and comfortable density where justified.

### P1.4 Complete operational write interfaces

Implement missing UI for canonical commands already supported by APIs. Where APIs do not exist, create a separate domain task rather than simulating completion in the browser.

Priority flows:

1. Program setup and controlled configuration changes.
2. Applicability and requirement review.
3. Decision proposal, review, approval, rejection and conditions.
4. Action assignment, update, block and implementation.
5. Response-package preparation, review, signatory approval and transmission record.
6. Verification contract/result and closure.
7. Evidence request creation, redirect, delegation and review.

### P1.5 Responsive and accessibility hardening

Automate:

- axe checks;
- keyboard journeys;
- focus order/return;
- 320px and 200% zoom reflow;
- reduced-motion fixtures;
- theme contrast;
- screen-reader landmark/name snapshots where practical.

### Exit criteria

- all primary screens use the production token/component system;
- complete light/dark coverage exists;
- no enabled visible control lacks a real action;
- priority write flows are executable from the UI;
- state-gallery and accessibility tests pass in CI;
- rendered baselines are reviewed at target viewports.

## 7. Phase P2 — Enterprise identity, LDAP/AD and organization synchronization

### Objective

Provide source-backed principals, groups, positions and organization changes suitable for bank deployment.

### P2.1 Data model

Add authoritative records:

- `identity_sources`;
- `identity_source_credentials` or external secret references;
- `external_principals`;
- `external_groups`;
- `external_positions`;
- `external_memberships`;
- `external_manager_links`;
- `identity_mappings`;
- `identity_sync_runs`;
- `identity_sync_changes`;
- `identity_reconciliation_findings`;
- `identity_source_events`.

Required fields include tenant, legal entity, source, immutable external ID, normalized attributes, source version/timestamp, first/last seen, status, mapping state and record time.

Secrets must not be stored in ordinary JSON configuration or returned through APIs.

### P2.2 Authentication federation

Implement provider abstraction for:

- OIDC;
- SAML 2.0;
- development/signed-gateway mode retained for testing and gateway deployments.

The resulting actor envelope must include:

- tenant and legal entity;
- principal external/internal IDs;
- authentication method;
- assurance level;
- session ID;
- issued/expiry times;
- selected role/delegation context where applicable.

### P2.3 SCIM service

Implement SCIM 2.0 endpoints or client sync, depending on deployment model, for users and groups.

Requirements:

- stable external IDs;
- create/update/deactivate;
- group membership;
- pagination and filtering;
- idempotency;
- source-version conflict handling;
- deletion policy that deactivates rather than destroys history;
- rate limiting and audit.

### P2.4 LDAP/Active Directory connector

Implement an on-premises sync agent or connector with:

- LDAPS and certificate validation;
- least-privilege bind account;
- configurable base DN and filters;
- paged searches;
- user, group, nested-group, manager and position mapping;
- source-side deletion/deactivation detection;
- incremental sync using supported change mechanisms with scheduled full reconciliation fallback;
- outbound-only agent option for restricted bank networks;
- encrypted secret storage and rotation;
- signed sync envelopes and replay protection.

Do not allow browser-to-LDAP connectivity.

### P2.5 Controlled file import

Provide CSV/XLSX import fallback with:

- saved approved mappings;
- schema/change detection;
- dry-run preview;
- duplicate and unresolved identifier handling;
- partial error isolation;
- maker-checker activation;
- rollback to prior accepted snapshot.

### P2.6 Reconciliation and downstream effects

On accepted identity changes:

- re-evaluate effective assignments;
- flag orphaned responsibilities;
- re-route open work only under approved policy;
- invalidate expired sessions/authority where required;
- preserve historical identity and decision attribution;
- create routing-integrity work for unresolved changes.

### APIs and UI

Add Configure endpoints and screens for source connection, mapping, dry run, activation, sync history, change review, findings and manual reconciliation.

### Tests

- tenant isolation;
- LDAPS/certificate failure;
- nested groups and cycles;
- stable external IDs across rename;
- deactivation and rehire;
- manager/position changes;
- duplicate identities;
- partial sync retry;
- stale source and rollback;
- authority impact without silent grant.

### Exit criteria

- a representative bank directory can be imported and reconciled;
- identity changes produce visible governed consequences;
- no imported group directly bypasses ClearSight authority policies;
- sync is idempotent, observable and recoverable;
- authentication and lifecycle tests pass for at least one OIDC and one directory path.

## 8. Phase P3 — RBAC, responsibility, authority and escalation administration

### Objective

Complete the runtime and Configure experience for role eligibility, scoped responsibility, decision authority, segregation, delegation and escalation.

### P3.1 Capability bundles and role templates

Add versioned capability definitions and role templates. Capabilities are coarse application actions; effective authority remains scoped.

Example capability groups:

- view Programs;
- maintain Program configuration;
- respond to evidence requests;
- review evidence;
- propose decisions;
- authorize decisions;
- sign external responses;
- configure routing;
- administer identity sources;
- manage notification templates;
- manage security policy.

### P3.2 Responsibility assignments

Implement versioned assignments by:

- object/object class;
- legal entity;
- function, branch, service, application or population;
- responsibility type;
- principal, group, position or role template;
- valid period;
- source and approval state.

### P3.3 Decision authority

Implement authority grants and decision policies including:

- decision type;
- materiality/amount/duration limits;
- required evidence state;
- review/challenge prerequisites;
- quorum and signatory;
- authentication assurance requirement;
- emergency path;
- expiry.

### P3.4 Conflict and segregation engine

Add reusable checks for:

- maker/checker;
- performer/reviewer/authorizer overlap;
- reporting-line conflict;
- evidence submitter independence;
- investigation subject relationship;
- protected-report conflict;
- prior decision participation;
- declared conflict.

Checks run at assignment and command execution.

### P3.5 Escalation policies and timers

Add structured escalation definitions and runtime materialization:

- reminder cadence;
- operational escalation;
- authority escalation;
- deadline/materiality escalation;
- routing failure;
- fallback/substitute/queue;
- business calendar and time zone;
- terminal unresolved behavior.

Timers must be deduplicated, leased, cancelable and re-evaluated when policy or ownership changes.

### P3.6 Configure UI

Implement:

- role catalogue;
- responsibility matrix;
- decision-authority matrix;
- routing/escalation sequence builder;
- delegation and substitution management;
- simulation and candidate explanation;
- impact preview;
- maker-checker submission/approval;
- scheduled activation and rollback;
- history and comparison.

### P3.7 Full mutation authorization matrix

Inventory every material API command and bind it to:

- required capability;
- responsibility/relationship;
- decision authority where relevant;
- legal-entity and object scope;
- assurance requirement;
- conflict checks;
- audit event.

Fail closed in production. Audit-only modes must be explicit and unavailable for protected high-risk commands.

### Tests

- candidate resolution and explanation;
- no eligible route;
- stale directory source;
- delegation expiry/revocation;
- nested/circular delegation;
- authority threshold changes;
- quorum and abstention;
- self-approval rejection;
- reassignment after deactivation;
- timer cancellation after completion;
- policy activation impact and rollback;
- authorization coverage test ensuring every material route declares a policy.

### Exit criteria

- administrators can configure and simulate common bank responsibility and authority structures without code;
- every material command is covered by the authorization matrix;
- conflicts, missing actors and escalation failures become visible work;
- policy changes are versioned, approved, effective-dated and reversible.

## 9. Phase P4 — Role-aware onboarding and adoption

### Objective

Replace the single hard-coded guide with a role-, permission- and tenant-configuration-aware onboarding platform.

### P4.1 Guide definition model

Add versioned guide definitions with:

- code and role/responsibility selectors;
- tenant/global ownership;
- required capabilities;
- applicable product maturity/configuration;
- steps and anchors;
- meaningful-task target;
- fallback behavior;
- locale;
- approval state;
- effective dates.

### P4.2 Guide resolution

At sign-in or guide restart:

1. resolve actor, responsibilities and capabilities;
2. remove guides targeting inaccessible features;
3. prioritize urgent operational work over tutorials;
4. select the smallest relevant guide set;
5. offer resume/restart without blocking critical work.

### P4.3 Initial guide packs

Implement and test guides for:

- executive;
- Program owner;
- reviewer/challenger;
- authorizer/signatory;
- evidence respondent;
- Configure administrator;
- auditor/read-only observer.

Each guide ends in a real task or explicit configured sandbox task.

### P4.4 Contextual guidance

Add:

- coach marks tied to stable semantic anchors;
- inline “why this matters” explanations;
- setup checklist;
- changed-feature reintroduction by guide version;
- help/restart entry point;
- permission-safe fallbacks.

### P4.5 Telemetry

Capture privacy-minimized events only:

- started/skipped/resumed/completed;
- time to first meaningful action;
- step abandonment;
- help opened;
- first-task result.

Never capture sensitive field content or infer competence.

### Tests

- multi-role user selection;
- no-permission anchors;
- urgent-work bypass;
- guide version upgrade;
- dismissal/resume;
- keyboard/focus trapping and restoration;
- translated long copy;
- mobile behavior;
- real-task completion.

### Exit criteria

- no role relies on the single Control Assurance guide;
- every major role has a tested, permission-aware first-run path;
- guide administration is governed and versioned;
- adoption telemetry excludes sensitive content.

## 10. Phase P5 — Notification, email and delivery platform

### Objective

Provide versioned, safe and observable communication across in-product, email and approved integration channels.

### P5.1 Notification domain

Add:

- `notification_events`;
- `notification_policies` and versions;
- `notification_templates` and versions;
- `notification_recipients`;
- `notification_deliveries`;
- `notification_preferences`;
- `notification_suppressions`;
- `notification_digests`;
- `notification_failures`.

A notification event references the authoritative domain event; it does not copy unrestricted record content.

### P5.2 Policy evaluation

Policy resolves:

- recipient by current responsibility/authority;
- channel;
- sensitivity/minimization profile;
- immediate versus digest;
- quiet hours;
- reminder/escalation cadence;
- cancellation when the need is satisfied;
- fallback on delivery failure.

Recipient and access are re-evaluated before generating a sensitive deep link.

### P5.3 Template registry

Templates include:

- subject and safe preview;
- HTML and plaintext bodies;
- approved variables;
- locale;
- sensitivity class;
- deep-link type and expiry;
- owner, maker/checker and active version;
- sample fixture and rendered preview.

Reject unknown variables and raw arbitrary HTML/scripts.

### P5.4 Delivery adapters

Implement interfaces and at least one production-capable email adapter. Additional adapters may include Microsoft Teams, Slack, ITSM or SMS only where approved.

Requirements:

- idempotent send;
- retry/backoff;
- delivery status;
- provider message ID;
- bounce/complaint handling;
- rate limits;
- tenant branding;
- content logging prohibition;
- dead-letter operating flow.

### P5.5 Notification centre and preferences

Implement actor-scoped notification APIs and UI with categories, grouping, direct actions, mark-read, delivery state and policy-constrained preferences.

### P5.6 Template packs

Create approved templates for:

- assignment;
- evidence request;
- reminder;
- overdue escalation;
- review/approval;
- returned/rejected/conditioned;
- failed outcome check;
- source unavailable/evidence aging;
- response milestone;
- invitation/expiry/revocation/wrong recipient;
- configuration approval;
- identity/MFA/security event;
- role digest.

### Tests

- restricted-content minimization;
- subject/preview safety;
- recipient role change before send;
- duplicate event deduplication;
- cancellation after completion;
- quiet hours and immutable regulatory deadlines;
- localization and plaintext parity;
- provider failure/retry/dead letter;
- malicious template variable;
- forwarded/expired link behavior.

### Exit criteria

- critical workflows can deliver safe notifications through a production email adapter;
- templates are versioned, previewed, approved and auditable;
- delivery failures create visible work;
- restricted details do not leak through subject, preview, logs or analytics.

## 11. Phase P6 — MFA, sessions and step-up assurance

### Objective

Support enterprise assurance claims and standalone high-assurance authentication where required.

### P6.1 Assurance policy

Add versioned policies mapping commands and record classes to:

- minimum assurance level;
- accepted authentication methods;
- maximum authentication age;
- device/session restrictions;
- step-up reason and expiry.

### P6.2 Enterprise step-up

For OIDC/SAML deployments, support redirect or challenge flows that request stronger/fresher authentication and validate returned assurance claims.

### P6.3 Standalone authentication factors

Implement:

- WebAuthn/passkey registration and challenge;
- TOTP enrolment compliant with RFC 6238;
- encrypted TOTP secret storage;
- recovery codes stored as one-way hashes;
- factor naming and revocation;
- rate limiting and lockout policy;
- dual-controlled administrator reset;
- session revocation.

### P6.4 Security settings UI

Provide:

- enrolled factors;
- add/verify/remove flow;
- recovery-code acknowledgement;
- active sessions/devices;
- recent security events;
- tenant MFA policy and enforcement status;
- safe recovery workflow.

### P6.5 Command integration

Before a protected material command:

1. resolve authority;
2. evaluate assurance policy;
3. return structured `step_up_required` without committing changes;
4. complete challenge;
5. retry with a bounded proof linked to actor, session, command class and expiry;
6. re-evaluate authority and record version;
7. commit and audit.

### Tests

- TOTP replay/time-window handling;
- passkey challenge origin/RP validation;
- recovery code single use;
- rate limiting;
- factor removal with insufficient assurance;
- expired step-up proof;
- actor/command mismatch;
- authority change during challenge;
- session revocation;
- IdP assurance downgrade;
- administrator recovery audit.

### Exit criteria

- high-risk commands enforce fresh assurance;
- TOTP and passkey flows are complete where standalone auth is enabled;
- recovery and reset are dual-controlled and auditable;
- no MFA secret or code appears in logs, analytics or ordinary exports.

## 12. Phase P7 — Illustration, iconography and complete state system

### Objective

Replace the initial four-variant illustration and small icon set with a production visual asset system.

### P7.1 Asset architecture

Create:

- token-driven SVG illustration package;
- light/dark variants;
- semantic icon package;
- asset metadata and accessibility labels;
- bundle-size checks;
- preview/state gallery;
- versioning and deprecation rules.

### P7.2 Required illustration families

Implement the families listed in the design plan, including onboarding, no work, no change, unavailable source, expired invitation, protected reporting, MFA, import reconciliation and notification failure.

### P7.3 Empty-state integration

Audit every significant list, panel, wizard and configuration area. Replace generic empties with explicit absence/no-result/not-configured/unauthorized/unavailable/stale/completed states.

### P7.4 Icon coverage

Create icons for all recurring object, responsibility, action and security classes. Add lint/test rules ensuring visible icon-only buttons have accessible names.

### P7.5 Visual regression

Capture and compare:

- light/dark;
- desktop/tablet/mobile;
- long copy;
- compact/comfortable density;
- empty/loading/error/stale/restricted/completed/overdue;
- reduced motion.

### Exit criteria

- no major state uses a generic placeholder illustration;
- all assets are theme-aware, optimized and accessible;
- iconography is consistent across primary screens;
- visual-regression evidence is retained and reviewed in CI.

## 13. Phase P8 — Pilot hardening and production release gates

### Objective

Prove the product can operate safely and credibly in a representative bank pilot.

### P8.1 Security and data

Complete or validate:

- production object storage and malware scanning;
- encryption-key policy and rotation;
- retention, deletion and legal hold;
- database row-level security or equivalent defense-in-depth;
- secrets management;
- gateway signing-key rotation;
- audit export and tamper evidence;
- vulnerability and dependency scanning;
- penetration test remediation.

### P8.2 Reliability and recovery

Validate:

- backup/restore and point-in-time recovery;
- worker retry and dead-letter operations;
- directory resync/reconciliation;
- notification-provider outage;
- IdP outage and safe degraded behavior;
- projection rebuild;
- object-store interruption;
- tenant offboarding/export.

### P8.3 Performance

Retain evidence for representative populations:

- 100,000+ Programs/issues where applicable;
- identity/group/position scale representative of a bank;
- bounded authority resolution;
- large responsibility matrices;
- Today and notification volumes;
- directory sync throughput;
- p95/p99 API and database plans;
- sustained worker backlog and recovery;
- frontend bundle and interaction budgets.

### P8.4 Pilot configuration

For each pilot bank:

- approve legal/regulatory reference configuration;
- configure legal entities and organization sources;
- map initial roles/responsibilities/authority;
- approve notification templates;
- configure authentication/MFA policy;
- select initial Programs/journeys;
- define support and incident process;
- train administrators and role champions;
- complete go-live review and rollback plan.

### P8.5 Operational telemetry

Provide dashboards and alerts for:

- authentication and assurance failures;
- identity sync freshness/findings;
- routing integrity;
- timer/outbox/notification backlog;
- source freshness;
- projection lag;
- evidence request completion;
- failed verification;
- authorization denial anomalies;
- onboarding first-action completion;
- frontend/API error budgets.

### Exit criteria

- all production boundaries advertised as release gates have evidence;
- no critical/high unresolved security finding;
- recovery exercises pass;
- pilot roles complete representative end-to-end scenarios;
- performance targets are retained with query plans and environment metadata;
- release approval is recorded through a governed checklist.

## 14. Cross-cutting database and migration rules

Every phase that adds persistent state must include:

- forward migration;
- rollback or compatibility strategy;
- tenant-leading indexes;
- effective and record time where history matters;
- append-only event/outbox behavior for material changes;
- idempotency keys;
- retention classification;
- RLS policy impact;
- representative cardinality and query plan;
- migration test on a populated fixture database.

Do not overload arbitrary JSON where fields are required for authorization, filtering, uniqueness or lifecycle state.

## 15. Cross-cutting API rules

- OpenAPI is updated in the same change.
- Verified actor scope overrides body/query actor fields.
- Sensitive not-found behavior avoids existence disclosure.
- Commands require optimistic version or equivalent conflict protection.
- Material commands produce authoritative state, append-only event, outbox event and required maintenance work in one transaction.
- Error responses use stable codes and human working-language messages.
- Step-up, conflict, missing route and source-unavailable states are structured and actionable.
- List endpoints are bounded and pagination-safe before authorization filtering.

## 16. Cross-cutting frontend rules

- Runtime context is loaded from verified server context.
- No hard-coded tenant, principal, role or legal entity in production paths.
- Visible enabled controls call real endpoints.
- A user sees actions based on server-returned eligibility and receives authoritative denial on execution.
- Optimistic UI never invents material success.
- Drafts and return context survive safe interruption.
- Every significant component has state fixtures and rendered evidence.
- Themes and density use tokens, not local color duplication.

## 17. CI and acceptance matrix

The final CI matrix must include:

### Backend

- format;
- race-enabled unit tests;
- PostgreSQL-tagged composition;
- migrations;
- PostgreSQL integration;
- vet/static analysis;
- authorization coverage;
- identity/directory sync integration;
- notification rendering/minimization;
- MFA/step-up security tests;
- recovery/idempotency tests.

### Frontend

- strict TypeScript;
- unit/component tests;
- role-based interaction journeys;
- axe accessibility;
- keyboard/focus journeys;
- visual regression for required state matrix;
- light/dark and responsive coverage;
- production build and bundle budget.

### End to end

At minimum:

1. directory user → mapped position → responsibility → assigned Today work;
2. evidence respondent → focused request → submission → reviewer handling;
3. Program owner → gap → issue/change → action;
4. independent reviewer → outcome check → closure readiness;
5. authorizer → step-up → approved decision;
6. overdue task → escalation → fallback owner;
7. identity deactivation → authority removal → safe reassignment;
8. restricted record → minimized notification → authorized deep link;
9. notification failure → retry/dead letter → operator resolution;
10. first-time role guide → meaningful real action.

## 18. Rollout strategy

### Stage 1 — Internal/demo

- development identity allowed;
- reference data labelled;
- non-production delivery adapter;
- broad telemetry for test environments only.

### Stage 2 — Controlled pilot

- enterprise SSO and directory source required;
- selected legal entities/Programs;
- approved role/authority configuration;
- production email adapter;
- enforced MFA/step-up for selected commands;
- support and rollback plan;
- no unsupported protected-report or direct regulator integration claims.

### Stage 3 — Production expansion

- release gates passed;
- complete monitoring and recovery;
- approved additional Programs and channels;
- capacity evidence retained;
- configuration changes governed through maker-checker lifecycle.

## 19. Recommended issue/PR decomposition

Keep changes reviewable. Suggested epics and PR boundaries:

- EP-P0 baseline and traceability;
- EP-P1A tokens/themes/components;
- EP-P1B Today/Programs/Issues finishing;
- EP-P1C Work/Explore/Configure finishing;
- EP-P2A identity source schema and sync core;
- EP-P2B OIDC/SAML/SCIM;
- EP-P2C LDAP/AD agent and reconciliation UI;
- EP-P3A capabilities and responsibility assignments;
- EP-P3B decision authority/conflict engine;
- EP-P3C escalation runtime and Configure builders;
- EP-P3D full command-authorization coverage;
- EP-P4 role-aware onboarding;
- EP-P5A notification domain/templates;
- EP-P5B email adapter/notification centre;
- EP-P6A assurance policy/enterprise step-up;
- EP-P6B passkey/TOTP/recovery/session UI;
- EP-P7 asset system and empty states;
- EP-P8 security/recovery/performance/pilot gates.

Each PR should contain one coherent vertical outcome rather than only schema or only mock UI.

## 20. Definition of complete

ClearSight may be described as a completed pilot-ready enterprise product only when:

- all P0–P7 exit criteria pass;
- the selected pilot scope passes P8 release gates;
- every major role can enter, understand and complete representative work;
- identity lifecycle, authority, escalation and notification behavior are proven under change and failure;
- light/dark premium UI and all required states are implemented and tested;
- MFA/step-up protects material actions;
- LDAP/AD or equivalent enterprise organization synchronization is operational;
- documentation, OpenAPI, migrations, code, rendered evidence and tests remain synchronized;
- remaining exclusions are explicit deployment boundaries rather than missing core product flows.
