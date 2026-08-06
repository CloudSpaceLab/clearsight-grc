# ClearSight Enterprise Productization Design Plan

## 1. Purpose

This document defines the design required to move ClearSight from a strong working application foundation and reference MVP into a pilot-ready, premium enterprise product for banks.

It consolidates the remaining product experience work across:

- interface cleanup and finishing;
- role-aware first-time guidance;
- responsibility, authority, RBAC and escalation administration;
- enterprise directory and identity-source compatibility;
- illustrations, iconography and complete empty-state coverage;
- email, in-product and escalation-notification design;
- premium visual refinement;
- MFA, step-up authentication and security settings;
- accessibility, responsiveness, localization and release evidence.

This plan does not replace the canonical domain, authority, capture or security specifications. It translates those specifications into a finished product experience.

## 2. Completion target

ClearSight may be described as a **pilot-ready enterprise product** only when a configured bank can:

1. connect or import its people, positions, groups and legal entities;
2. define responsibility, decision authority, segregation and escalation without code;
3. give each major user role a safe, relevant first-run path;
4. execute the main Program, issue/change, evidence, decision, response and verification workflows in the interface;
5. deliver secure, minimized and actionable notifications;
6. enforce enterprise authentication and step-up requirements;
7. operate the product across desktop, tablet and supported mobile capture states;
8. demonstrate accessible, premium and consistent behavior across live, empty, loading, degraded, unauthorized and completed states;
9. explain configuration, ownership, authority and next action without requiring GRC-specialist knowledge;
10. pass the release gates defined in the implementation plan.

“Premium” means calm, exact, cohesive, fast and trustworthy. It does not mean decorative complexity.

## 3. Product experience principles

### 3.1 One role, one immediate purpose

After sign-in, the user must understand within one screen:

- why they are here;
- what requires their attention;
- what they own, review, authorize or observe;
- what changed;
- what they may do;
- what happens next.

The home experience is role-aware but not role-exclusive. A user with several responsibilities sees a prioritized operating brief rather than multiple disconnected dashboards.

### 3.2 Configuration is governed work

Role, authority, directory, escalation, template and security configuration must use the same governance principles as operational work:

- draft and preview;
- source and scope visibility;
- maker-checker approval;
- effective dating;
- impact analysis;
- rollback;
- immutable history.

### 3.3 Progressive disclosure, not hidden consequences

The default interface should remain minimal. Advanced details are available through inspect, compare, explain, history and simulation views. The interface must never hide:

- material uncertainty;
- authority limits;
- segregation conflicts;
- sensitive-data handling;
- irreversible side effects;
- missing evidence;
- next escalation;
- stale identity or directory data.

### 3.4 Identity and access are visible where they matter

Users should not need to understand identity infrastructure. They must still see:

- the active institution and legal entity;
- their current role or delegated capacity;
- why they can or cannot perform an action;
- when step-up authentication is required;
- when access is temporary, delegated or restricted;
- where to report an incorrect assignment or conflict.

## 4. Target roles and finished first-run experiences

The guide engine must resolve the user’s actual responsibilities and select one or more short guides. Guides are versioned, skippable, resumable, permission-aware and anchored to real controls.

### 4.1 Executive: CRO, CCO, CISO, DPO and equivalent leaders

First-run outcome:

- understand the executive operating brief;
- distinguish current status from unknown or stale status;
- identify material issues, overdue decisions and failed outcome checks;
- inspect why an item is escalated;
- open one real high-value record;
- configure digest preferences.

The executive guide must not teach routine data-entry mechanics.

### 4.2 Program owner

First-run outcome:

- open an assigned Program;
- understand current status and reasons;
- identify obligations, safeguards, evidence and open issues;
- see what is already known;
- request or redirect one focused piece of evidence;
- understand how status becomes current again.

### 4.3 Reviewer and independent challenger

First-run outcome:

- understand assigned review scope and independence requirements;
- compare proposed result with source evidence;
- inspect contradictions and exceptions;
- approve, return or challenge through a real workflow;
- declare a conflict or insufficient authority.

### 4.4 Authorizer and signatory

First-run outcome:

- understand current authority, limits and decision scope;
- inspect required review/challenge completion;
- identify side effects and external representation consequences;
- approve, reject, condition or return a real low-risk example;
- understand step-up authentication and audit evidence.

### 4.5 Evidence respondent or business owner

First-run outcome:

- understand why they were selected;
- review known facts;
- answer only unresolved questions;
- upload or confirm evidence safely;
- redirect, delegate or report wrong scope;
- receive a clear submission receipt.

### 4.6 Configure administrator

First-run outcome:

- connect or import organization data;
- review mapping and reconciliation results;
- define one role/responsibility assignment;
- simulate a routing rule;
- submit configuration for approval;
- understand identity, delivery and security health.

### 4.7 Auditor, legal reviewer and read-only observer

First-run outcome:

- understand point-in-time reconstruction;
- distinguish authoritative records, projections and samples;
- inspect source lineage and decision history;
- export only where policy permits;
- understand restricted and privileged boundaries.

## 5. Information architecture and navigation

### 5.1 Primary navigation

The primary navigation remains compact:

- **Today** — assigned work, decisions, evidence and escalations;
- **Programs** — ongoing obligations, controls and current status;
- **Issues and changes** — specific gaps, findings, requests, incidents and responses;
- **Work** — focused evidence, sources and operational queues;
- **Explore** — connected bank journeys and reference patterns;
- **Configure** — organization, roles, authority, routing, notifications, integrations and security.

### 5.2 Global context

The application shell must expose without clutter:

- institution;
- legal entity or bank-wide scope;
- active role/delegation where relevant;
- source freshness or degraded-mode indicator;
- notification centre;
- help and first-run guide restart;
- user/security menu.

Scope changes must be explicit, permission-checked and preserved only where safe.

### 5.3 Search and command access

Global search should support Programs, issues/changes, requests, sources and configuration objects with purpose-bound access. Search must not expose restricted existence through counts, suggestions or timing.

A command palette may accelerate navigation, but it must not become the only route to important functions.

## 6. Screen-level finishing plan

### 6.1 Today

Required finish:

- role-aware brief and dominant next action;
- grouped sections for action, review, authorization, evidence and escalation;
- clear owner, due date, reason and next escalation;
- completed/recently handled section without inflating open counts;
- safe bulk triage only for reversible, low-risk actions;
- saved views and density preferences;
- empty state that names the checked population and freshness.

### 6.2 Programs

Required finish:

- executive and owner variants of the same canonical data;
- current status, material reasons and freshness;
- obligations, applicability, safeguards, evidence and open issues in progressive sections;
- change-since-last-view summary;
- controlled setup and bulk import flows;
- direct actions for review, assignment, evidence request and governed rebuild;
- point-in-time comparison.

### 6.3 Issues and changes

Required finish:

- unified workspace for summary, scope, evidence, decisions, actions, responses, outcome checks and history;
- one dominant actor-specific action;
- visible authority and segregation explanation;
- draft/save/resume for complex decisions and responses;
- protected-record mode with minimized navigation and notification behavior;
- clear closure readiness and blockers;
- mobile replacement behavior for focused capture and review.

### 6.4 Work and evidence

Required finish:

- source health, evidence requests, invitations and review queues;
- clear distinction between request status, submission, evidence sufficiency and verified outcome;
- focused request composer based on unresolved facts;
- redirect/delegate/conflict flows;
- external response preview and safe invitation lifecycle;
- upload progress, validation, quarantine and retry states;
- evidence-package reuse where purpose and period permit.

### 6.5 Explore

Required finish:

- connected journeys as educational and operational entry points;
- production tenant journeys separated from reference data;
- exact linked records and permission-aware actions;
- journey setup checklist for administrators;
- comparison of configured, incomplete, current, blocked and completed journeys;
- no implication that four reference journeys represent full bank coverage.

### 6.6 Configure

Configure becomes a coherent enterprise administration product rather than a set of cards.

Required areas:

1. Organization and directory sources.
2. People, positions and groups.
3. Role catalogue and capability bundles.
4. Responsibility matrix.
5. Decision-authority matrix.
6. Routing and escalation builder.
7. Delegation, substitution and absence.
8. Notification templates and channel policy.
9. Integrations and source health.
10. Authentication, MFA and session policy.
11. Audit, configuration history and pending approvals.

Every area needs list, detail, create/edit, simulation/preview, approval and history states where applicable.

### 6.7 Authentication and account settings

Required surfaces:

- enterprise sign-in handoff;
- standalone sign-in where enabled;
- MFA enrolment and challenge;
- passkey/TOTP/recovery-code management;
- active sessions and device history;
- delegated roles and temporary authority;
- notification preferences;
- security-event history;
- safe account recovery.

## 7. Authority, RBAC and escalation design

### 7.1 Permission model

ClearSight should present simple roles but implement effective authority as the intersection of:

- broad capability bundle;
- tenant and legal entity;
- organizational position;
- responsibility assignment;
- object relationship;
- workflow state;
- decision type and materiality;
- delegation or substitution;
- segregation and conflict rules;
- policy version and effective time;
- authentication assurance.

A role grants eligibility, not universal access.

### 7.2 Role catalogue

The catalogue must show:

- role name and plain-language purpose;
- capabilities;
- prohibited combinations;
- source and owner;
- assigned positions and people;
- legal-entity and object scope;
- effective dates;
- pending changes;
- usage and unresolved gaps.

### 7.3 Responsibility matrix

Rows represent object classes or selected objects. Columns represent:

- performer;
- accountable owner;
- reviewer;
- independent challenger;
- authorizer;
- signatory;
- escalation owner;
- observer/consulted party.

The matrix supports inheritance, exceptions, source-backed suggestions, filters and effective dates. It must not become an unrestricted spreadsheet.

### 7.4 Decision-authority matrix

The matrix defines:

- decision class;
- scope;
- materiality or amount thresholds;
- required evidence quality;
- reviewer/challenger sequence;
- authorizer or quorum;
- signatory;
- expiry;
- emergency path;
- step-up authentication requirement.

### 7.5 Escalation builder

The builder uses plain-language steps and displays a visual sequence. It supports:

- reminders;
- operational escalation;
- authority escalation;
- risk/deadline escalation;
- routing-failure escalation;
- fallback queue or substitute;
- business-calendar rules;
- terminal unresolved handling.

The builder must show the next escalation, recipient, timer basis and consequence.

### 7.6 Simulation and impact preview

Before activation, administrators test representative scenarios and see:

- selected actor and explanation;
- rejected candidates and reasons;
- authority limits;
- conflicts and missing positions;
- timers and escalation path;
- active workflows affected;
- invitations, decisions or delegations invalidated;
- rollback path.

## 8. Directory, LDAP and organization-import experience

### 8.1 Supported source model

The design supports:

- OIDC or SAML for authentication;
- SCIM for user/group lifecycle where available;
- LDAP/LDAPS or Active Directory through an on-premises connector or sync agent;
- controlled CSV/spreadsheet import as a fallback;
- API and event-based HR/organization feeds.

Directory membership does not directly grant material decision authority. It provides source-backed principals, positions, managers and groups that governance assignments may reference.

### 8.2 Connection wizard

The administrator flow includes:

1. choose source type;
2. enter or upload connection metadata;
3. validate TLS, bind and permissions without exposing secrets;
4. select base DN, tenant/entity scope or SCIM domain;
5. map user, group, manager, department, branch, position and status attributes;
6. preview records and exclusions;
7. detect duplicates and unresolved identifiers;
8. choose reconciliation and deactivation policy;
9. run a dry sync;
10. submit for activation approval.

### 8.3 Reconciliation experience

Each sync displays:

- added, changed, deactivated and unresolved records;
- stale source data;
- authority assignments affected;
- workflows requiring reassignment;
- conflicts and orphaned positions;
- last successful sync and next attempt;
- rollback or correction route.

## 9. Notification and email design

### 9.1 Notification classes

- assignment;
- evidence request;
- reminder;
- overdue escalation;
- review or approval required;
- approved, rejected, returned or conditioned;
- source unavailable or evidence aging;
- failed verification;
- response-package milestone;
- invitation, expiry, revocation and wrong-recipient;
- configuration approval;
- identity, MFA and security event;
- daily or weekly role digest.

### 9.2 Channel hierarchy

Each notification policy defines:

- in-product notification;
- email;
- approved messaging channel;
- ITSM or collaboration integration;
- escalation fallback;
- quiet hours and digest eligibility;
- content-minimization level;
- delivery and read receipt requirements.

### 9.3 Email template anatomy

Every email contains:

- institution identity;
- safe, specific subject;
- minimized preview text;
- why the recipient is receiving it;
- one dominant action;
- deadline and consequence where safe;
- scope-safe support route;
- expiry or access notice;
- no protected details in subject or preview;
- plaintext and accessible HTML variants.

Templates are versioned, localized, previewable with safe fixture data and approved before activation.

### 9.4 Notification centre

The in-product centre supports:

- unread, due, escalated and security categories;
- grouping by record without hiding distinct obligations;
- mark read and dismiss where allowed;
- direct launch into the exact action;
- delivery status and failed-channel explanation;
- preference controls constrained by policy.

## 10. MFA and security experience

### 10.1 Enterprise-first assurance

Where the bank supplies enterprise SSO, ClearSight consumes authentication assurance and method claims. Sensitive actions may require a fresh or stronger authentication event.

### 10.2 Standalone factors

Where standalone authentication is enabled:

- passkeys/WebAuthn are preferred;
- TOTP is supported for compatibility;
- recovery codes are issued once and stored safely;
- SMS is not the default high-assurance factor;
- trusted-device behavior is policy-controlled;
- administrator recovery is dual-controlled and audited.

### 10.3 Step-up triggers

Examples:

- high-materiality approval;
- regulator or external signatory action;
- restricted or privileged record access;
- identity reveal for protected reporting;
- bulk export;
- authority-policy activation;
- MFA reset;
- break-glass action.

The challenge explains why it is required without disclosing protected details.

## 11. Illustration, iconography and empty-state system

### 11.1 Illustration families

Create theme-aware production assets for:

- first-run introduction;
- no assigned work;
- no material change;
- no search result;
- no configured Program or journey;
- source unavailable;
- invitation expired or revoked;
- response submitted;
- routing configuration;
- continuous readiness;
- protected reporting;
- secure authentication and MFA;
- import reconciliation;
- notification delivery failure.

Each family has light and dark variants, an accessible title/description contract, responsive behavior and bundle-size budget.

### 11.2 Icon system

Use a single semantic line-icon family for:

- Programs, requirements, safeguards and evidence;
- issues, findings, incidents, requests and changes;
- people, positions, groups, roles and delegations;
- review, challenge, approval, signatory and escalation;
- source, sync, import, notification and security;
- verified, incomplete, blocked, overdue, stale and unknown states.

Icons identify objects or actions. Color and icon alone never carry status.

### 11.3 Empty-state taxonomy

Every empty state identifies which of these applies:

- true absence;
- no assigned work;
- no change since last review;
- no result for the current query;
- not configured;
- unauthorized or wrong scope;
- source unavailable;
- data stale or status pending;
- completed population;
- sample/reference environment.

## 12. Premium visual system

### 12.1 Themes

Deliver complete dark and light themes. Both must preserve semantic contrast, density and executive credibility. Theme behavior includes illustrations, charts, focus indicators, print/export and email previews.

### 12.2 Tokens

Formalize tokens for:

- canvas and layered surfaces;
- text and muted text;
- borders and separators;
- navigation, governance, attention, verified and blocking semantics;
- typography scale;
- spacing and density;
- radii;
- elevation;
- focus;
- motion duration/easing;
- chart and data-visualization semantics.

Hard-coded component colors are prohibited except within versioned illustration assets.

### 12.3 Component library

Minimum production components:

- application shell and context switcher;
- page header and operating brief;
- work rows and dense tables;
- status/reason block;
- decision brief;
- actor/authority explanation;
- timeline and immutable history;
- side panel and full-screen mobile flow;
- command confirmation and receipt;
- import mapping/reconciliation;
- matrix and sequence builder;
- notification item and email preview;
- MFA challenge and security-event row;
- skeleton, empty, error, stale and partial states.

### 12.4 Density and executive presentation

Support comfortable and compact density where appropriate. Executive views prioritize materiality and exceptions. Operator views prioritize throughput and keyboard efficiency. Neither uses decorative cards for every object.

## 13. Accessibility, responsiveness and localization

Required standards:

- WCAG 2.2 AA target;
- keyboard operation for all core workflows;
- visible focus and logical focus return;
- semantic headings, landmarks, tables, forms and progress;
- screen-reader names that identify the record and action;
- 200% zoom and 320px reflow;
- reduced-motion behavior;
- sufficient contrast in both themes;
- no time-limited action without extension or clear policy basis;
- localization-safe layouts and long-text fixtures;
- date, number, currency and time-zone formatting by tenant/user policy;
- accessible HTML and plaintext notification variants.

## 14. Required state and evidence matrix

Every significant page or component must define and test:

- initial loading;
- live/default;
- empty;
- no search result;
- partial/stale;
- source unavailable;
- permission denied/wrong scope;
- validation failure;
- optimistic conflict;
- save/resume;
- success/receipt;
- completed/read-only;
- overdue/escalated;
- long translated copy;
- keyboard and focus;
- 200% zoom;
- reduced motion;
- light and dark theme;
- compact and comfortable density where supported.

## 15. Design deliverables

Each productization workstream must produce:

1. decision brief;
2. user and role journey;
3. information architecture or workflow map;
4. component/state inventory;
5. light/dark desktop designs;
6. tablet/mobile replacement behavior;
7. empty/error/degraded states;
8. copy and notification content;
9. authority and security implications;
10. rendered evidence and accessibility results;
11. implementation acceptance criteria;
12. post-release telemetry and support signals.

## 16. Design definition of done

A productization area is design-complete only when:

- all target roles and authority states are covered;
- every enabled control maps to a real permitted action;
- configuration preview, approval and rollback are designed;
- empty, degraded, unauthorized and recovery states are explicit;
- light/dark, responsive and accessibility behavior is specified;
- notification and email effects are included;
- sensitive content minimization is reviewed;
- first-run guidance leads to a meaningful real task;
- visual assets are production-ready rather than placeholders;
- acceptance fixtures and rendered evidence are ready for implementation.
