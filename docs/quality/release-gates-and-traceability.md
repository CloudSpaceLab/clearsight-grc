# Release Gates and Use-Case Traceability

This document defines the minimum cross-cutting proof required before a ClearSight capability can be released.

Domain-specific acceptance tests remain under `docs/quality/`. This document ensures every advertised use case is traceable and that responsibility, invitations, data architecture, performance, and degraded operation are tested consistently.

## 1. Traceability rule

Every capability must have a row containing:

| Field | Required value |
|---|---|
| Use-case ID | Stable ID from `product/use-case-catalogue.md` |
| Maturity | Foundation, Pilot, Expansion, or Enterprise |
| Product spec | Canonical behavior and boundary |
| Actors | Performer, owner, reviewer, challenger, authorizer, signatory, escalation owner |
| State contract | Happy, exception, degraded, prohibited, closure, reopening |
| UX reference | First-use and repeat-use flow or golden screen |
| Architecture/ADR | Data, authorization, workflow, invitation, and performance decisions |
| Implementation phase | Planned delivery slice |
| Acceptance evidence | Automated tests, usability study, security tests, and benchmark |

A missing row blocks implementation-ready status.

## 2. Pilot traceability baseline

| Use case | Required specifications | Required acceptance coverage |
|---|---|---|
| UC-CFG-01 | authority routing; system architecture | policy simulation, conflict, absence, escalation, rollback, performance |
| UC-SRC-01/02 | source/evidence architecture | authority, freshness, degradation, recovery, dependent-state propagation |
| UC-IMP-01/02 | Respond/Capture; system architecture | first/repeat import, 1M-row workload, partial success, idempotent retry |
| UC-EVID-01 | Respond/Capture; authority routing | minimum question, redirect, delegation, source reuse, timing |
| UC-PROG-01 | continuous compliance model | incremental state, source degradation, review by exception |
| UC-PRIV-01/02/03 | NDPA journey | ROPA, DPIA, breach timing, authority, invitation, filing |
| UC-REG-01 | regulatory intelligence | exact source, applicability, affected scope, amendment, performance |
| UC-AUTH-01 | regulatory intelligence; Respond/Capture | protected routing, subject resolution, request links, response package, leakage |
| UC-FIND-01 | Matter/verification model | import, assignment, action, evidence, failed verification, reopening |
| UC-REPORT-01 | report and export UX | point-in-time version, approval, manifest, download authorization |
| UC-HIST-01 | temporal architecture | reconstruction of source, policy, authority, decision, action, and outcome |

## 3. Actor and authority gates

For every workflow:

- [ ] The correct performer, owner, reviewer, challenger, authorizer, signatory, and escalation owner are resolved from policy—not fixture users.
- [ ] The UI explains why each actor was selected.
- [ ] One dominant next action is correct for each actor and state.
- [ ] Parallel actors can progress without unsafe serialisation.
- [ ] Self-approval and conflicting role combinations are blocked.
- [ ] Delegation cannot exceed the delegator’s authority.
- [ ] Substitution, handoff, and delegation preserve different semantics.
- [ ] Leave, departure, entity transfer, role revocation, and stale directory data are handled.
- [ ] Circular delegation and unresolved routing become explicit failure states.
- [ ] Material actions re-evaluate current authority at execution.
- [ ] Committee quorum, recusal, dissent, conditional approval, and signatory are reconstructable.
- [ ] Break-glass authority is narrow, time-limited, notified, expired automatically, and retrospectively reviewed.

### Required scenarios

1. routine performer → reviewer flow;
2. material decision with independent challenge and authorizer;
3. owner unavailable with approved substitute;
4. employee or role changes while work is in progress;
5. declared and automatically detected conflict;
6. no valid actor can be resolved;
7. policy changes before final approval;
8. emergency authority and retrospective review.

## 4. Configure gates

Configuration is a high-impact workflow and must pass:

- [ ] source-backed role and organization selection;
- [ ] versioned draft and immutable approved version;
- [ ] representative scenario simulation;
- [ ] conflict, missing-role, circular-route, broad-scope, and self-approval detection;
- [ ] affected active workflow and decision impact preview;
- [ ] maker-checker approval;
- [ ] scheduled activation and effective time;
- [ ] rollback or supersession;
- [ ] audit and point-in-time reconstruction;
- [ ] no administrator privilege amplification through configuration alone;
- [ ] policy resolution performance at reference workload.

## 5. Invitation and Respond/Capture gates

For every invitation-based or focused request:

- [ ] The request is linked to an exact purpose, scope, Claim, Evidence Contract, or directive.
- [ ] Existing authorized evidence is searched first.
- [ ] Only unresolved facts are requested.
- [ ] Prefilled values show source class and correction behavior.
- [ ] The recipient sees only the minimum context.
- [ ] Invitation token is opaque, short-lived, audience-bound, revocable, and exchanged for a bounded session.
- [ ] Token and sensitive content are absent from logs, analytics, referrers, previews, and page titles.
- [ ] Identity assurance is proportional to data and consequence.
- [ ] Forwarded, wrong-recipient, expired, revoked, replayed, and already-used invitations fail safely.
- [ ] Draft, resume, schema change, amendment, follow-up, and cancellation are safe.
- [ ] Upload, submission, accepted Observation, evidence sufficiency, and closure remain distinct.
- [ ] Redirect, delegate, partial response, concern, and wrong-recipient paths preserve accountability.
- [ ] External participants cannot browse tenant data.
- [ ] Notification content is minimized.
- [ ] Ordinary request median active effort is under three minutes and p90 under five minutes unless justified.

### Required scenarios

1. internal SSO request;
2. external vendor request with email verification;
3. multi-contributor external organization response;
4. forwarded or leaked link;
5. revoked/cancelled request;
6. network interruption and idempotent resume;
7. customer capture with step-up identity;
8. file scan or extraction failure;
9. request changed while draft exists;
10. source evidence arrives elsewhere and reminders cancel.

## 6. Protected reporting gates

- [ ] Protected identity and report content use separate access boundaries.
- [ ] Anonymous reporting does not require ordinary account creation.
- [ ] Two-way communication works through a protected mailbox or recovery code.
- [ ] Conflict-aware investigator routing is tested.
- [ ] Reporter identity is excluded from ordinary search, queue, analytics, embeddings, notifications, AI context, and exports.
- [ ] Identity reveal requires a separate privileged decision.
- [ ] Allegation, observation, and verified fact remain distinct.
- [ ] No credibility inference uses protected traits, language style, grammar, emotion, or channel.
- [ ] Retaliation concern and investigator recusal paths work.
- [ ] Approved minimized systemic signals do not permit subject or reporter inference.

## 7. Review-by-exception gates

A review-by-exception flow must expose:

- [ ] total denominator;
- [ ] included, omitted, unauthorized, stale, contradicted, and sampled counts;
- [ ] reason unchanged items were omitted;
- [ ] source health and policy version;
- [ ] last full review;
- [ ] sampling policy;
- [ ] trigger for complete re-review;
- [ ] ability to inspect the full authorized population;
- [ ] no higher missed-contradiction or incorrect-approval rate than the full-review baseline.

A model, mapping, source-authority, policy, or schema change must be able to force broader review.

## 8. Matter lifecycle gates

Every Matter subtype must define and test:

- [ ] creation, classification, triage, and ownership;
- [ ] scope, deadline, and required authority;
- [ ] parallel evidence, review, decision, and action paths;
- [ ] redirect, delegate, conflict, escalation, and routing failure;
- [ ] duplicate detection;
- [ ] merge, split, cancellation, and supersession;
- [ ] action implementation versus outcome verification;
- [ ] closure contract;
- [ ] failed verification and reopening;
- [ ] source amendment or changed context;
- [ ] point-in-time reconstruction.

Merge or split must not widen access, combine incompatible purposes, lose deadlines, duplicate actions, or blur protected boundaries.

## 9. AI and automation gates

### AI

- [ ] exact source/version lineage;
- [ ] structured validated output;
- [ ] explicit versus inferred values;
- [ ] contradiction and abstention;
- [ ] correct action class and authority requirement;
- [ ] zero critical tenant, protected-identity, or tool-authority failures;
- [ ] lower median active effort than non-AI baseline;
- [ ] no increase in material omission or missed contradiction;
- [ ] provider outage and manual fallback;
- [ ] outcome-linked monitoring.

### Automation

- [ ] eligibility and prohibited-action policy;
- [ ] simulation/dry run;
- [ ] affected population and blast-radius preview;
- [ ] canary or staged activation where appropriate;
- [ ] approval, effective period, and expiry;
- [ ] idempotent execution;
- [ ] rollback or compensation;
- [ ] suspension and kill switch;
- [ ] partial failure reconciliation;
- [ ] implementation evidence and outcome verification;
- [ ] no broad external-tool identity or unrestricted tool catalogue.

## 10. Data, consistency, and concurrency gates

- [ ] Authoritative state resides in relational domain records; projections are rebuildable.
- [ ] Material versions preserve valid and record time.
- [ ] Commands use optimistic concurrency or equivalent state-version checks.
- [ ] External and repeated commands use idempotency keys.
- [ ] Outbox/inbox handles duplicate and reordered delivery.
- [ ] Partial failure and compensation are tested.
- [ ] Cache keys include tenant, purpose, policy, authorization, and object versions where relevant.
- [ ] Material execution does not rely on stale cached authority.
- [ ] Projection lag is observable and does not produce false current state.
- [ ] Retention, legal hold, deletion, and offboarding propagate to derivatives.
- [ ] No raw evidence, token, protected identity, or secret enters logs/events.

## 11. Performance and capacity gates

Every feature must declare:

- cardinality and growth;
- common and worst-case queries;
- read/write ratio and burst pattern;
- index, partition, pagination, cache, and materialization strategy;
- consistency requirement;
- async threshold;
- SLO and failure budget;
- benchmark fixture and production monitor.

Minimum cross-cutting targets follow `architecture/system-data-and-performance.md`.

Required load tests:

- [ ] 25,000-user/2,500-session reference identity and queue workload;
- [ ] routing resolution with deep scope and conflict rules;
- [ ] invitation issue/redeem/revoke bursts;
- [ ] 1-million-row import with partial errors;
- [ ] 100-million-observation/year partition and query profile;
- [ ] large Program state invalidation without full synchronous recompute;
- [ ] large response/report package generation;
- [ ] search and queue behavior under tenant skew;
- [ ] worker restart, duplicate events, and retry storms;
- [ ] cache loss, search lag, model outage, and source outage;
- [ ] restore and RPO/RTO exercise.

A feature that meets median latency but fails p95/p99, noisy-neighbor, authorization, or recovery behavior does not pass.

## 12. Usability and quality gates

Each golden journey requires:

- timed first-use and repeat-use tests;
- comprehension check;
- manual versus prefilled field count;
- transition count;
- redirect/delegation/correction rate;
- abandonment and resume behavior;
- keyboard and screen-reader completion;
- mobile/low-bandwidth test where relevant;
- AI/source unavailable fallback;
- material omission, incorrect approval, missed contradiction, and later reversal rate.

Speed alone cannot pass a flow. A faster flow fails if it reduces comprehension, evidence quality, correct routing, or decision accuracy.

## 13. Security and privacy gates

- tenant and legal-entity isolation;
- relationship and purpose authorization;
- counts, autocomplete, cache, graph, vector, timing, and export inference;
- protected case and reporter isolation;
- object-storage authorization and pre-signed URL expiry;
- invitation abuse and rate limiting;
- malicious files and prompt injection;
- external-party least privilege;
- support and break-glass access;
- offline revocation and device loss;
- report/export authorization at generation and download;
- no sensitive usability telemetry.

## 14. Release evidence package

A release candidate must produce:

- completed traceability rows;
- approved product/UX/architecture documents and ADRs;
- automated test results;
- representative user-testing results;
- security and privacy findings;
- AI evaluation results where used;
- load, latency, capacity, and recovery reports;
- unresolved risks and approved exceptions;
- rollback and operational runbooks;
- version and change summary.

## 15. Final release standard

ClearSight passes only when the target actor can accomplish the governed outcome with minimum reasonable effort, the correct authority and escalation are resolved, external collection is narrowly secured, the evidence and state are defensible, and the workflow remains correct and usable under realistic scale, interruption, degradation, and historical reconstruction.
