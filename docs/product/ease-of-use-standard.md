# ClearSight Ease-of-Use and Workflow Efficiency Standard

This document defines the mandatory usability, flexibility, workflow-efficiency, and human-effort standard for every ClearSight screen, component, flow, Program, Matter, capture journey, review, approval, and administrative experience.

It applies to product design, implementation, testing, AI behavior, integrations, channel packs, jurisdiction packs, and customer-specific configuration.

The governing principle is:

> **ClearSight should do the assembly work before asking a person to act.**

The product must use available institutional context, authoritative sources, prior evidence, approved defaults, and governed AI recommendations so that users can complete routine work in a few clear steps without sacrificing evidence, authority, security, or auditability.

---

# 1. The five-minute active-effort budget

## 1.1 Routine work

A routine, authorized, well-scoped task should normally be completable in **less than five minutes of active user effort**.

Examples:

- confirm or correct a prefilled asset record;
- answer a focused evidence request;
- approve or reject a low-complexity control review with context;
- upload and map a familiar spreadsheet using a saved mapping;
- review an AI-proposed obligation extraction;
- assign or redirect a Matter;
- confirm a vendor certificate or policy version;
- review a small set of unresolved accounts;
- acknowledge a Program change;
- submit a branch observation;
- approve a reversible, low-impact action.

The five-minute budget includes reading the immediate context, entering or confirming information, reviewing the proposed result, and submitting or routing it.

## 1.2 Complex work

Some activities cannot responsibly finish in five minutes:

- legal interpretation;
- regulatory applicability decisions;
- material risk acceptance;
- incident or breach investigation;
- suspicious-activity or suspicious-transaction determination;
- major remediation design;
- large evidence-package review;
- point-in-time examination reconstruction;
- high-impact authority response;
- multi-party approval;
- observation periods required for verification.

For these activities, ClearSight must still enable the user to reach a **clear, saved, correctly routed next state within five minutes**.

Examples:

- accept the initial case assignment;
- confirm scope and deadline;
- review the AI-produced first summary and source anchors;
- identify missing evidence;
- assign specialist reviewers;
- save a draft decision;
- request targeted clarification;
- approve the next investigation step;
- create a governed implementation workstream.

The system must preserve progress so that the user never has to reconstruct context on return.

## 1.3 Active effort is not elapsed time

The budget measures the time a person actively spends operating ClearSight. It does not include:

- waiting for an external source;
- an observation period;
- regulator acknowledgement;
- asynchronous approval;
- scheduled testing;
- a long-running import;
- model processing;
- external execution;
- human investigation outside the product.

ClearSight must show background progress and notify the user only when meaningful intervention is required.

---

# 2. Mandatory workflow rules

## 2.1 Prefill before asking

Before presenting an editable field, ClearSight must search authorized sources for an existing value.

Possible sources include:

- institution profile;
- asset, application, branch, merchant, vendor, customer, account, service, or process inventory;
- HR and organization directory;
- IAM and access systems;
- ITSM and project platforms;
- policy and document repositories;
- prior approved submissions;
- current Program evidence;
- prior Matter records;
- regulatory and jurisdiction packs;
- APIs, managed imports, telemetry, and approved spreadsheets.

Known values should be:

- prefilled;
- clearly sourced;
- read-only when the user is not authorized to change them;
- editable through a correction workflow when appropriate;
- accompanied by age or freshness where material.

A user must not repeatedly enter a branch, service, asset, vendor, owner, legal entity, control, or requirement that ClearSight can resolve from an approved inventory.

## 2.2 Ask only unresolved facts

Forms and requests must be generated from unresolved claims or missing fields.

Do not show a complete questionnaire merely because the underlying template contains many questions.

The request engine should:

1. establish the exact purpose;
2. identify the affected scope and population;
3. retrieve existing authorized information;
4. identify missing, stale, contradictory, or insufficient facts;
5. ask the smallest useful question;
6. select the least burdensome approved response type;
7. stop when the evidence need is satisfied or no longer relevant.

## 2.3 One clear next action

Every primary state must present one obvious next action.

Secondary actions may be available, but must not compete visually with the expected handling path.

Avoid generic actions such as:

- Continue;
- Submit;
- View details;
- Update;
- Process.

Prefer specific actions such as:

- Confirm four account owners;
- Review proposed CBN obligations;
- Resolve two terminal matches;
- Send for DPO approval;
- Request current vendor certificate;
- Approve temporary exception;
- Start 30-day verification;
- Prepare authority response package.

## 2.4 Minimize navigation

A routine flow should not require users to move through multiple module homepages.

A user handling a Matter should ordinarily remain in one workspace with progressively disclosed sections.

A user maintaining a Program should be able to move directly from a requirement or gap to the relevant control, evidence, owner, Matter, or filing without returning to a generic navigation hub.

## 2.5 Preserve context across steps

ClearSight must preserve:

- active institution and legal entity;
- Program or Matter;
- channel, service, branch, vendor, customer, account, asset, or population;
- period and effective date;
- filters and selections where safe;
- current evidence and contradictions;
- user role and delegated authority;
- unsaved drafts and completed steps.

Users must not repeatedly reselect scope during one coherent flow.

## 2.6 Progressive disclosure

Show only what is necessary for the current decision or response.

Advanced details remain available through inspect, compare, explain, history, or source-lineage views.

Progressive disclosure must not hide:

- material uncertainty;
- contradictory evidence;
- legal or authority limits;
- irreversible side effects;
- required approval;
- sensitive-data handling;
- scope or population;
- important exclusions.

## 2.7 Save and resume

Every multi-step or potentially interrupted workflow must support safe save and resume.

The return experience must show:

- what was completed;
- what changed since the last visit;
- what remains;
- current blockers;
- the next recommended action.

Do not require the user to reread the full source or reconstruct prior selections.

---

# 3. Integration-led usability

## 3.1 Inventories are workflow sources

Existing bank inventories must be usable as governed sources for workflow scope and controlled selection.

Examples:

- applications from CMDB or enterprise architecture inventory;
- assets from asset-management systems;
- branches from the institution directory;
- employees and owners from HR or directory systems;
- vendors and contracts from procurement;
- merchants and POS terminals from acquiring systems;
- ATMs from channel inventory;
- accounts and customers from approved core systems;
- projects and changes from ITSM, Jira, or Azure DevOps;
- processing activities from ROPA;
- critical services and dependencies from BIA;
- policies and certificates from document repositories.

The system must display source authority, freshness, and limitations without forcing the user to understand integration mechanics.

## 3.2 Progressive integration

A workflow must remain usable whether its source is:

- a manually maintained controlled list;
- an approved spreadsheet;
- a scheduled import;
- an API;
- an event stream;
- a real-time system query.

The interaction model should remain stable while automation increases.

## 3.3 Source failure must not create user confusion

When a source is unavailable, delayed, or stale, the interface must explain:

- what information is affected;
- the last known value and age;
- whether manual confirmation is allowed;
- what cannot safely proceed;
- the recommended fallback.

The system must not silently replace authoritative data with an unqualified human value.

---

# 4. Governed AI assistance

## 4.1 AI should propose, not make users start from blank pages

Where approved, AI should provide a useful first draft for:

- regulatory obligation extraction;
- applicability questions;
- control mappings;
- evidence requests;
- risk and Matter summaries;
- remediation options;
- verification criteria;
- policy changes;
- response-package indexes;
- review plans;
- questionnaire answers based on existing evidence;
- source reconciliation;
- assignment suggestions.

Users should review a grounded proposal rather than assemble ordinary content manually.

## 4.2 Recommendations must be actionable

An AI recommendation must include:

- recommended action;
- why it is recommended;
- sources and versions;
- assumptions;
- affected scope;
- important uncertainty or contradiction;
- estimated effort or complexity where useful;
- required authority;
- editable structured fields;
- safe alternatives.

Avoid generic recommendations such as “review this control” without identifying who, what, why, and expected evidence.

## 4.3 AI must not add interaction burden

Do not require a chat prompt for actions that can be represented directly.

AI should appear through:

- better defaults;
- prefilled fields;
- suggested mappings;
- concise explanations;
- highlighted gaps;
- focused questions;
- ready-to-review drafts.

A chat or command surface is optional, not mandatory.

## 4.4 Review by exception

When policy permits, reviewers should focus on:

- low-confidence fields;
- new mappings;
- contradictions;
- material changes;
- exceptions;
- unsupported recommendations;
- high-impact actions.

Repeatedly approved, high-confidence, low-impact patterns may be automated under explicit policy, monitoring, and rollback.

---

# 5. Component and interaction requirements

## 5.1 Program page

A Program page should immediately show:

- current overall position;
- material gaps and exceptions;
- evidence becoming stale;
- filings, reviews, or tests due next;
- recent regulatory or scope changes;
- Matters requiring attention.

The default view must not expose hundreds of controls as an undifferentiated list.

## 5.2 Matter workspace

A Matter workspace should combine:

- summary;
- scope and affected objects;
- evidence and source lineage;
- decisions and approvals;
- actions and dependencies;
- outcome or response verification;
- history.

Routine Matter handling should not require module hopping.

## 5.3 Evidence request

An evidence request must show:

- why the recipient was selected;
- what is already known;
- exactly what remains unresolved;
- acceptable response forms;
- estimated active effort;
- deadline and consequence;
- redirect, delegate, not-applicable, and sensitivity options.

Target: ordinary requests should require no more than a few inputs and less than five minutes of active effort.

## 5.4 Review and approval

A reviewer should see the proposed result, source evidence, important changes, uncertainty, authority, side effects, and recommended action in one view.

For low-complexity reviews, approval or rejection should require only:

- inspect exceptions;
- optionally edit;
- provide rationale where policy requires;
- confirm.

## 5.5 Imports

Repeat imports should reuse approved mappings and show only:

- changed columns or structure;
- errors;
- duplicates;
- unresolved identifiers;
- material variance;
- records requiring review.

Do not force users to remap a stable file every cycle.

## 5.6 Population worklists

Large populations require:

- saved views;
- recommended filters;
- clear denominators;
- bulk actions with authorization;
- keyboard efficiency;
- inline review;
- exception-focused modes;
- remembered column preferences;
- direct movement to the next unresolved item.

## 5.7 Mobile and field capture

Field users should receive pre-scoped requests and perform only the required action.

Examples:

- confirm one asset;
- capture one identifier;
- select a discrepancy reason;
- attach one approved document;
- report one incident observation.

The flow should tolerate poor connectivity, save drafts safely, and avoid requiring desktop-only context.

---

# 6. Flexibility without arbitrary complexity

ClearSight must support different bank sizes, jurisdictions, vocabularies, control frameworks, approval structures, source maturity, and workflow preferences.

Flexibility should be achieved through:

- Programs;
- Matter types;
- channel and jurisdiction packs;
- evidence contracts;
- source profiles;
- institution profile;
- controlled workflow policies;
- configurable authority matrices;
- reusable templates and mappings;
- role-specific views.

Avoid flexibility through:

- arbitrary database-schema builders;
- ungoverned custom scripts;
- per-customer forks;
- unlimited dashboard configuration;
- large generic form builders as the primary model;
- hundreds of optional fields shown to every user.

Configuration must preserve upgradeability, cross-bank semantics, accessibility, and testability.

---

# 7. Safety boundaries

Ease of use must not bypass:

- legal or regulatory interpretation approval;
- material risk authority;
- segregation of duties;
- evidence sufficiency;
- customer-data protection;
- protected-case isolation;
- account-action authority;
- suspicious-reporting authority;
- export and disclosure approval;
- immutable history;
- verification requirements.

Fewer clicks do not justify hidden scope, unchecked defaults, or irreversible action.

For material or irreversible actions, the interface must add only the confirmation necessary to make scope, authority, consequence, and evidence clear.

---

# 8. Quantitative usability targets

Each key workflow must define and measure:

- median active completion time;
- 90th-percentile active completion time;
- number of screens or major workspace transitions;
- number of fields entered manually;
- number of known fields prefilled;
- duplicate facts requested;
- abandonment rate;
- redirect and delegation rate;
- correction rate;
- reviewer edit and rejection rate;
- time to resume after interruption;
- accessibility completion rate;
- mobile completion rate where relevant.

Initial targets:

- routine focused request: median under 3 minutes, 90th percentile under 5 minutes;
- routine approval with complete context: median under 2 minutes;
- familiar recurring import using saved mapping: active effort under 5 minutes excluding processing;
- assignment or redirection: under 60 seconds;
- executive understanding of one material item: under 60 seconds;
- return to an in-progress complex Matter: next action understood within 30 seconds;
- no routine flow should require more than three major workspace transitions without documented justification.

These are initial product targets and must be validated with representative bank users.

---

# 9. Required usability testing

Every golden journey must include:

- a timed first-use test;
- a timed repeat-use test;
- keyboard-only completion;
- screen-reader or assistive-technology review where applicable;
- low-bandwidth or degraded-source behavior;
- mobile review for field tasks;
- interruption and resume;
- wrong-scope prevention;
- AI unavailable fallback;
- source unavailable fallback.

A workflow fails usability acceptance when:

- ordinary users need to understand internal GRC architecture;
- users re-enter known information;
- required context is distributed across module homepages;
- AI creates more review work than it removes;
- a routine task exceeds five minutes without clear justification;
- a complex workflow cannot reach a safe saved next state within five minutes;
- accessibility users require materially more steps;
- the fastest route bypasses governance or evidence.

---

# 10. Design review questions

Before approving any screen, component, or flow, ask:

1. What outcome is the user trying to achieve?
2. What does ClearSight already know?
3. Why is each editable field still necessary?
4. Can an approved source prefill or eliminate it?
5. Can AI create a grounded first draft?
6. Can the user complete the routine path within five minutes?
7. Can a complex path reach a clear saved next state within five minutes?
8. Is there one obvious next action?
9. Can the user stay in one coherent workspace?
10. Are scope, authority, evidence, uncertainty, and consequence still clear?
11. Does the workflow remain usable without AI or a live integration?
12. Can the result be reconstructed later?

A feature is not complete merely because it is visually polished or functionally possible. It is complete when the intended user can accomplish the governed outcome with the minimum reasonable effort.