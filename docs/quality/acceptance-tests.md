# ClearSight Acceptance and Non-Regression Tests

This document defines the minimum product, usability, source-quality, evidence, security, AI, visual, accessibility, operational, and historical tests required for ClearSight.

A feature is not accepted because CRUD, upload, API, workflow, or AI output works. It is accepted when representative bank users can complete the governed outcome with minimum reasonable effort under realistic positive, negative, ambiguous, stale, partial, unauthorized, offline, degraded, and historical conditions.

It conforms to:

- [`../../README.md`](../../README.md)
- [`../../AGENTS.md`](../../AGENTS.md)
- [`../product/ease-of-use-standard.md`](../product/ease-of-use-standard.md)
- [`../product/continuous-compliance-operating-model.md`](../product/continuous-compliance-operating-model.md)
- [`../product/experience-principles.md`](../product/experience-principles.md)

---

# 1. Test philosophy

Tests must prove:

- Programs for continuing obligations and Matters for change or exception;
- routine active effort under five minutes where responsibly possible;
- a clear saved next state within five minutes for complex work;
- prefill before asking;
- approved inventories before manual re-entry;
- existing evidence before requests;
- grounded AI first drafts before blank-page work;
- review by exception;
- one clear next action;
- minimal workspace transitions;
- source authority before automated trust;
- progressive integration;
- data-quality transparency;
- evidence before confidence;
- contradiction before false certainty;
- decisions before dashboards;
- verification before closure;
- human authority for material judgment;
- institutional memory.

Fixtures MUST NOT inject desired conclusions, mappings, owners, authority, materiality, verification outcomes, or pre-completed workflow context.

---

# 2. Required test dimensions

## 2.1 Domain

- Program and Matter lifecycles;
- Requirement and applicability versions;
- Control Objective and Implementation separation;
- Evidence Contract policy;
- source authority and limitation;
- Observation provenance;
- contradiction propagation;
- Compliance State dimensions;
- decision and authority;
- response packages;
- verification and closure;
- temporal semantics.

## 2.2 Usability

Every golden journey requires:

- timed first-use test;
- timed repeat-use test;
- active user effort measurement;
- major workspace transition count;
- manually entered and prefilled field counts;
- duplicate-fact request detection;
- abandonment and correction rate;
- interruption and resume;
- user comprehension check;
- representative role testing.

## 2.3 Source and integration

- controlled lists;
- spreadsheet import;
- saved mapping reuse;
- scheduled import;
- API or event source;
- source degradation;
- mapping changes;
- partial success;
- deletion and revocation;
- stale data;
- conflicting sources.

## 2.4 AI

- extraction and mapping correctness;
- exact source lineage;
- explicit versus inferred fields;
- usefulness of first draft;
- reviewer edit and rejection rate;
- human effort saved;
- appropriate abstention;
- contradiction disclosure;
- authorization and action class;
- prompt injection and malicious content;
- provider outage and fallback;
- latency and cost.

## 2.5 Security and privacy

- tenant and legal-entity isolation;
- wrong-scope action;
- relationship and purpose authorization;
- bulk-operation authorization;
- protected authority and reporting cases;
- search, count, cache, graph, embedding, and timing inference;
- restricted exports;
- offline capture;
- integration credentials;
- malicious files and prompts.

## 2.6 Visual, accessibility, and localization

- light and dark parity;
- comfortable and compact density;
- desktop, tablet, and mobile where relevant;
- keyboard-only journey;
- screen-reader journey;
- 200% zoom;
- reduced motion;
- local date, time, currency, and number formats;
- long translated labels;
- low bandwidth;
- no materially longer assistive-technology path.

## 2.7 Performance and recovery

- first meaningful deterministic content;
- Program and Work queue latency;
- population queries;
- spreadsheet and media processing;
- import backlog recovery;
- workflow resume;
- source and AI outage;
- projection rebuild;
- large response and assurance packages.

---

# 3. Quantitative usability release gates

Initial targets:

- focused routine evidence request: median under 3 minutes, p90 under 5 minutes;
- routine approval with complete context: median under 2 minutes;
- assignment or redirect: under 60 seconds;
- repeat import with unchanged mapping: active effort under 5 minutes;
- executive comprehension of one material item: under 60 seconds;
- resume complex Matter: next action understood within 30 seconds;
- no routine workflow above 3 major workspace transitions without approved exception;
- zero repeated entry of fields available from approved sources unless correction is the intended action;
- no accessibility path with materially higher field count or required navigation.

A flow that misses the target requires documented cause, product review, and remediation plan before general release.

---

# 4. Golden Journey A — Continuous NDPA Program and targeted ROPA update

## Setup

- institution has existing ROPA spreadsheet;
- application inventory, vendor register, organization directory, and project/change source are available;
- several processing activities have missing lawful basis, retention, or recipient data;
- one new vendor and one system change affect existing activities.

## Required path

1. import ROPA with row-level provenance;
2. match applications, vendors, departments, and owners from inventories;
3. surface unresolved mappings rather than asking owners to rebuild all entries;
4. create NDPA Program Requirements and Evidence Contracts;
5. trigger only affected processing activities after the vendor and system change;
6. prefill known fields;
7. generate focused owner requests;
8. owner confirms or corrects unresolved facts;
9. DPO reviews changed and exceptional fields;
10. Program state updates;
11. Material gap creates a Matter.

## Assertions

- [ ] routine owner update median under 3 minutes and p90 under 5 minutes;
- [ ] no full annual questionnaire is presented;
- [ ] applications, vendors, departments, and owners are prefilled;
- [ ] request contains only unresolved facts;
- [ ] changed source values are visibly sourced;
- [ ] one correction does not silently overwrite authoritative inventory;
- [ ] DPO reviews exceptions rather than every unchanged field;
- [ ] Program state derives from evidence rather than manual RAG selection;
- [ ] AI unavailable fallback remains usable.

---

# 5. Golden Journey B — DPIA screening from a new project

## Setup

- new project exists in ITSM or project system;
- application, vendor, processing purpose, data categories, and project owner are partly known;
- project may involve sensitive data and automated decisioning.

## Required path

1. change event creates privacy screening Matter;
2. known project, owner, system, vendor, and data context is prefilled;
3. AI proposes risk indicators and whether full DPIA appears necessary;
4. project owner answers only unresolved questions;
5. DPO reviews recommendation, sources, and uncertainty;
6. DPO decides whether full DPIA is required;
7. conditions and remediation are created before go-live;
8. Program and project state update.

## Assertions

- [ ] project owner screening targets under 5 minutes;
- [ ] no duplicate project or vendor entry;
- [ ] AI recommendation is editable and source-grounded;
- [ ] DPO remains authority;
- [ ] complex full DPIA reaches saved next state within 5 minutes of initial review;
- [ ] go-live approval cannot bypass unresolved required remediation.

---

# 6. Golden Journey C — Annual compliance filing assembled continuously

## Setup

- Program has year-long evidence, reviews, DPIAs, vendor records, training, incidents, and exceptions;
- several evidence items are stale or incomplete;
- filing deadline approaches.

## Required path

1. system maintains filing manifest throughout year;
2. deadline trigger shows readiness by dimension;
3. only stale, missing, contradictory, or unapproved items create work;
4. AI drafts filing index and summaries from approved records;
5. reviewers inspect exceptions and material narrative;
6. authorized signatory approves;
7. package is transmitted and acknowledgement recorded.

## Assertions

- [ ] no year-end manual search through folders is required;
- [ ] filing status is distinct from underlying compliance state;
- [ ] AI cannot invent evidence or source references;
- [ ] unchanged approved sections do not require full re-review;
- [ ] submission does not automatically prove operating effectiveness;
- [ ] complete point-in-time package can be reconstructed.

---

# 7. Golden Journey D — CBN regulatory change

## Setup

- official final circular and a prior related circular exist;
- publication changes several obligations affecting digital channels;
- bank inventories identify applications, channels, vendors, owners, and controls.

## Required path

1. source authenticity and document status are verified;
2. provisions are segmented with stable anchors;
3. AI extracts candidate Requirements and compares prior version;
4. reviewer sees source text beside proposed structured Requirement;
5. only low-confidence, changed, ambiguous, or material items require detailed review;
6. applicability is proposed using institution profile and inventories;
7. compliance/legal approves applicability;
8. affected Programs, systems, controls, vendors, and owners are suggested;
9. implementation Matters and Evidence Contracts are created;
10. Program state updates after verification.

## Assertions

- [ ] exposure draft cannot be treated as final;
- [ ] every Requirement has exact source anchor;
- [ ] routine source triage and routing under 5 minutes;
- [ ] AI reduces blank-page extraction work;
- [ ] human remains final interpretation authority;
- [ ] existing controls and owners are prefilled;
- [ ] no duplicate control is created without reconciliation;
- [ ] amendment preserves prior versions.

---

# 8. Golden Journey E — Protected authority request

## Setup

- authority request concerns named customers/accounts and a defined period;
- legal instrument state requires review;
- approved customer, account, KYC, address, transaction, and records sources are available;
- identities may be ambiguous.

## Required path

1. source and attachments are preserved in protected boundary;
2. legal reviewer confirms authority and disclosure scope;
3. subjects are matched with exact, provisional, unresolved, and contradictory states;
4. known KYC/account/address data is prefilled;
5. directives are decomposed;
6. focused tasks are routed to legal, records, KYC, address, AML, fraud, branch, or technology teams;
7. AI drafts response-package index from approved evidence;
8. missing directives remain visible;
9. authorized signatory approves;
10. transmission and acknowledgement are recorded.

## Assertions

- [ ] initial assignment and scope confirmation under 5 minutes;
- [ ] users do not re-enter known customer/account identifiers;
- [ ] ambiguous identities are not merged automatically;
- [ ] request does not imply guilt, account restriction, or suspicious reporting;
- [ ] protected subjects do not leak through search, counts, notifications, or analytics;
- [ ] response package reconciles every directive;
- [ ] AI cannot disclose or file without authority;
- [ ] complex case resumes with next action understood within 30 seconds.

---

# 9. Golden Journey F — Legacy exception import and future maintenance

## Setup

- legacy exception spreadsheet contains findings, affected applications, owners, dates, status, comments, and standards references;
- application, owner, control, and vendor inventories exist;
- comments contain informal redirects and commitments.

## Required path

1. import using mapped columns and row provenance;
2. match canonical applications, owners, controls, and vendors;
3. preserve comments as communication observations;
4. create Exception Matters;
5. unresolved assignments enter focused queue;
6. users redirect or accept through explicit workflow;
7. evidence is submitted and reviewed;
8. action implementation remains separate from verification;
9. future dashboard and workplan views derive from Matters.

## Assertions

- [ ] repeat import with stable mapping under 5 minutes active effort;
- [ ] no module-specific duplicate application or owner;
- [ ] comments do not silently change assignment or state;
- [ ] no green at evidence upload or task completion;
- [ ] legacy spreadsheet can be retired for future maintenance;
- [ ] export remains available for transition users.

---

# 10. Golden Journey G — Minimum-question access review

## Setup

- IAM population, approvals, HR, and exception records exist;
- only a small subset lacks business-need evidence.

## Required path

1. exact population and period are resolved;
2. existing evidence is evaluated first;
3. request contains only unresolved accounts;
4. manager sees known account, role, and employee context;
5. manager confirms, rejects, redirects, or reports lack of authority;
6. contradiction creates action;
7. verification observes no unauthorized reactivation.

## Assertions

- [ ] no request for resolved population;
- [ ] median response under 3 minutes;
- [ ] business language before control IDs;
- [ ] manager cannot view other scope;
- [ ] human attestation does not replace technical evidence;
- [ ] action completion remains awaiting verification.

---

# 11. Golden Journey H — Repeat spreadsheet import and review by exception

## Setup

- source has approved mapping from prior month;
- new file has same columns, several changed values, two duplicates, and one new unmapped identifier.

## Required path

1. user selects existing source or system detects it;
2. saved mapping is applied;
3. schema comparison shows no structural change;
4. only changed, duplicate, invalid, and unresolved rows require review;
5. AI suggests match for new identifier;
6. user confirms or rejects;
7. import runs asynchronously;
8. reconciliation result appears.

## Assertions

- [ ] active effort under 5 minutes;
- [ ] no full remapping;
- [ ] unchanged rows do not require review;
- [ ] AI match reason and confidence dimensions are visible;
- [ ] partial success is explicit;
- [ ] retry is idempotent.

---

# 12. Golden Journey I — Routine approval from recommendation

## Setup

- complete source-grounded recommendation proposes a reversible low-impact action;
- reviewer has authority;
- no contradiction exists.

## Required path

1. one view shows recommendation, sources, changes, scope, effect, authority, and next state;
2. reviewer inspects exception summary;
3. reviewer optionally edits rationale;
4. reviewer approves;
5. action and audit event are created.

## Assertions

- [ ] median completion under 2 minutes;
- [ ] approval is not context-free;
- [ ] no mandatory chat;
- [ ] keyboard path is equivalent;
- [ ] AI unavailable path supports manual draft;
- [ ] high-impact variant invokes stronger review rather than reusing low-impact flow.

---

# 13. Golden Journey J — Interruption and resume

## Setup

- user begins complex regulatory or authority Matter;
- source or approval arrives while user is away.

## Required path

1. draft is saved automatically or explicitly;
2. return view summarizes completed work;
3. system highlights changes since last visit;
4. blocker and next owner are visible;
5. recommended next action is shown;
6. user continues without reconstructing prior context.

## Assertions

- [ ] next action understood within 30 seconds;
- [ ] no repeated scope selection;
- [ ] prior edits and source versions preserved;
- [ ] changed source invalidates affected recommendation visibly;
- [ ] notification routes directly to relevant step.

---

# 14. Golden Journey K — Source and AI degraded mode

## Setup

- key inventory source is stale;
- primary model provider is unavailable;
- time-sensitive work continues.

## Required path

1. deterministic context appears;
2. stale source age and affected fields are visible;
3. approved fallback is offered;
4. AI state shows unavailable;
5. manual workflow remains usable;
6. unsafe action is blocked;
7. queued work resumes after recovery without duplication.

## Assertions

- [ ] no stale value shown as current;
- [ ] no stale AI answer shown as new;
- [ ] fallback does not silently gain source authority;
- [ ] user can reach safe saved next state within 5 minutes;
- [ ] recovery preserves idempotency.

---

# 15. Golden Journey L — Population and bulk action

## Setup

- user has mixed read/write authority over filtered population;
- action affects many records.

## Required path

1. scope, filter, denominator, and exclusions are visible;
2. selection summary shows authorized, excluded, and failed records;
3. exact side effects and reversibility are shown;
4. server authorizes each object;
5. proportional confirmation occurs;
6. post-action reconciliation shows result.

## Assertions

- [ ] bulk UI cannot bypass policy;
- [ ] routine selection and action uses few steps;
- [ ] unauthorized record details do not leak;
- [ ] partial success is explicit;
- [ ] action is idempotent and auditable.

---

# 16. Golden Journey M — Vendor work for a Program or issue

## Setup

- one existing vendor relationship supplies the service in scope;
- a Program or issue needs a vendor response, document or bounded attestation;
- the bank owner has current authority and a purpose-appropriate form revision;
- the vendor contact has no broader bank access.

## Required path

1. the bank owner finds the existing vendor relationship without creating a duplicate;
2. the relationship is linked to the exact Program or Matter;
3. known vendor, service and bank context is shown without asking the vendor to re-enter it;
4. the owner selects Classic, Wizard or form-controlled presentation, sets the deadline and sends a short-lived request-scoped invitation;
5. the vendor completes typed fields or uploads the required document and receives a submission receipt;
6. the bank reviewer opens the exact current answers and AVAILABLE documents;
7. the reviewer accepts the response with rationale or requests specific changes through a new capture in the same history;
8. acceptance leaves Program status, Matter action completion and outcome verification unchanged;
9. cancellation revokes active access, and the vendor link cannot end while active work depends on it.

## Assertions

- [ ] tenant, legal-entity, request, submission and artifact scope fail closed;
- [ ] current authority is re-evaluated for each material bank command;
- [ ] invitation tokens and recipient addresses do not enter logs, events, URLs or recovery records;
- [ ] invitation issuance is preceded by a durable work/request reservation, and a failed finalization plus failed revocation remains recoverable without an untracked active capability;
- [ ] partial delivery and ambiguous preparation failures recover without duplicate requests;
- [ ] Classic and Wizard use the same fields, limits, validation and draft state;
- [ ] a prior response remains reconstructable after changes, cancellation or relationship-link end;
- [ ] vendor submission, bank acceptance, implementation and verified outcome remain separate.

---

# 17. AI usefulness release gate

An AI capability may not ship solely because its outputs appear plausible.

It must demonstrate:

- source-grounded accuracy;
- acceptable abstention;
- no critical authorization or leakage failure;
- lower median active effort than the non-AI workflow;
- acceptable reviewer edit/reject rate;
- no increase in missed contradictions;
- safe provider-outage fallback;
- structured output and rollback.

A capability that adds review burden without proportional correctness benefit must be redesigned or removed.

---

# 18. Visual regression and golden screens

Maintain light/dark and relevant breakpoint references for:

- Today;
- Program overview;
- Requirement and evidence table;
- Program exception view;
- Work queue;
- Matter workspace;
- recommendation panel;
- regulatory source review;
- authority case;
- response package;
- focused request;
- mobile capture;
- population worklist;
- first and repeat import;
- reconciliation;
- routine and material approval;
- verification;
- ROPA update;
- DPIA screening;
- breach Matter;
- source degradation;
- AI unavailable;
- offline and resume;
- no material change;
- unknown/no-data;
- protected reporting.

Reject uncontrolled density, decorative effects, control walls, module hopping, hidden actions, and green before verification.

---

# 19. Final release standard

ClearSight passes only when it can:

1. maintain a Program from approved sources;
2. detect meaningful change or exception;
3. create the correct Matter;
4. assemble known context from bank inventories;
5. request only missing facts;
6. provide grounded recommendations;
7. route correct authority;
8. execute or respond safely;
9. verify outcome or acknowledgement;
10. update all derived views;
11. reconstruct the complete history;
12. achieve the governed outcome with minimum reasonable effort.

A workflow that is correct but unnecessarily cumbersome is not release-ready.
