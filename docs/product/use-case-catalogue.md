# Target Customers and Use-Case Catalogue

This document is the canonical inventory of who ClearSight serves, what each customer must be able to accomplish, and when each capability is expected to mature.

A capability is not implementation-ready until it has a use-case ID, actor and authority contract, workflow, architecture or ADR, release phase, and acceptance test.

## 1. Target institutions

### Regional and smaller banks

Typical starting condition:

- spreadsheet-led registers and evidence;
- limited integration capacity;
- small specialist teams wearing multiple roles;
- strong need for guided configuration and low administrative overhead.

ClearSight must work from controlled lists, files, focused requests, and a small number of high-value connectors without requiring a transformation programme.

### National banks

Typical condition:

- several legal entities, channels, business units, and control functions;
- established HR, IAM, CMDB, ITSM, procurement, core banking, and document systems;
- high-volume recurring reviews and regulatory work;
- formal committees, delegated authority, and segregation of duties.

ClearSight must provide source-led workflows, cross-domain context, protected cases, role routing, scalable populations, and board/regulatory reporting.

### Multinational and banking groups

Typical condition:

- multiple jurisdictions, licences, entities, residency constraints, shared services, and group policies;
- local and group-level authority;
- cross-border evidence and reporting restrictions.

ClearSight must support jurisdiction packs, entity-specific applicability, shared-control implementations, residency-aware data placement, group consolidation, and local override without forks.

## 2. Primary personas

| Persona | Primary need |
|---|---|
| Board and risk/audit committees | Understand material change, decide within mandate, track verified outcomes |
| CRO, CCO, CISO, DPO, General Counsel | Maintain accountable Programs, route decisions, manage exceptions and external obligations |
| Risk, compliance, security, privacy, resilience specialists | Interpret, assess, review evidence, challenge, recommend, and verify |
| Internal audit and assurance | Preserve independence, define samples, inspect lineage, issue conclusions |
| Business, channel, service, branch, system, control, and vendor owners | See only the responsibility relevant to owned operations and respond quickly |
| Evidence respondents and records custodians | Answer the smallest clear request with known context prefilled |
| GRC and platform administrators | Configure Programs, sources, routing, authority, retention, integrations, and automation safely |
| Vendors and other external parties | Submit scoped evidence without receiving broader bank access |
| Customers | Provide case-specific information through governed, minimal, privacy-aware capture |
| Protected or anonymous reporters | Report safely, communicate two ways, and remain identity-isolated |

## 3. Maturity labels

- **Foundation** — required before any business workflow is safe.
- **Pilot** — required for the initial four connected journeys.
- **Expansion** — documented and architecture-supported; delivered after pilot evidence.
- **Enterprise** — multi-entity, group, scale, residency, and advanced assurance maturity.

Documentation of an Expansion or Enterprise use case does not imply implementation in the first release.

## 4. Canonical use cases

| ID | Use case | Trigger → governed outcome | Primary authority | Maturity |
|---|---|---|---|---|
| UC-CFG-01 | Institution, role, authority, and escalation setup | organization/source import → simulated and approved routing policy | platform admin + control owner | Foundation |
| UC-CFG-02 | Program, Matter, evidence, and workflow configuration | approved template/policy change → versioned effective configuration | Program owner + maker-checker | Foundation |
| UC-SRC-01 | Source onboarding | new file/system/inventory → approved Source Profile and mapping | source owner + data steward | Foundation |
| UC-SRC-02 | Source degradation and recovery | stale/error/revoked source → affected state, safe fallback, recovery | source owner + dependent Program owner | Foundation |
| UC-IMP-01 | First spreadsheet import | source file → validated Observations and reconciliation queue | data steward | Pilot |
| UC-IMP-02 | Repeat import | recurring file → exception-only review and idempotent update | data steward | Pilot |
| UC-EVID-01 | Focused internal evidence request | unresolved Claim → sufficient evidence or explicit unresolved state | request owner/reviewer | Pilot |
| UC-EVID-02 | External evidence request | vendor/customer/other party needed → invitation wizard and governed response | request owner + data owner | Expansion |
| UC-CAP-01 | Mobile or field capture | branch/asset/event fact needed → validated Observation | scoped respondent + reviewer | Expansion |
| UC-REP-01 | Protected reporting | confidential/anonymous report → isolated Matter and two-way communication | protected-case intake authority | Expansion |
| UC-PROG-01 | Maintain a continuing Program | source/calendar/change trigger → recomputed state and targeted work | Program owner | Pilot |
| UC-PRIV-01 | ROPA update | processing change or stale facts → focused owner update and DPO review | DPO | Pilot |
| UC-PRIV-02 | DPIA screening and assessment | project/vendor/AI/data change → decision, conditions, and verification | DPO + project authority | Pilot |
| UC-PRIV-03 | Privacy breach | suspected breach → timed assessment, notification decision, response, verification | DPO/legal/incident authority | Pilot |
| UC-PRIV-04 | Data-subject request | verified request → scoped search, response, disclosure, and closure | privacy operations + DPO | Expansion |
| UC-REG-01 | Regulatory change | official publication → approved Requirements, affected controls, actions, and readiness | compliance/legal | Pilot |
| UC-SUP-01 | Supervisory finding | authority finding → management response, commitments, evidence, and effectiveness | accountable executive + compliance | Expansion |
| UC-AUTH-01 | Protected authority request | verified external request → legal scope, tasks, response package, acknowledgement | legal/compliance signatory | Pilot |
| UC-RISK-01 | Risk situation | signals/change → materiality, decision, action, and verified risk movement | risk owner/committee | Expansion |
| UC-EXC-01 | Exception or waiver | control/requirement cannot be met → bounded approval, conditions, expiry, revalidation | authority matrix | Expansion |
| UC-FIND-01 | Finding and remediation | review/test/imported finding → ownership, action, evidence, verification | issue owner + independent reviewer | Pilot |
| UC-INC-01 | Incident management integration | event or declaration → governed risk/compliance Matter and required decisions | incident authority | Expansion |
| UC-LOSS-01 | Operational loss | loss recognition → linked event, cause, recovery, control impact, reporting | operational risk | Expansion |
| UC-TPRM-01 | Vendor onboarding and due diligence | proposed relationship → evidence, risk decision, conditions | service owner + third-party risk | Expansion |
| UC-TPRM-02 | Vendor reassessment or deficiency | evidence expiry/change/finding → targeted review, remediation, continuation/exit | third-party risk + business owner | Expansion |
| UC-RES-01 | BIA and critical-service context | periodic/change trigger → current dependencies, tolerances, owners | resilience owner | Expansion |
| UC-RES-02 | Resilience test and recovery verification | scheduled/ad hoc test → result, gaps, decisions, verified improvement | resilience authority | Expansion |
| UC-RCSA-01 | RCSA cycle | scheduled or change trigger → prefilled assessment, challenge, Matters | business owner + second line | Expansion |
| UC-KRI-01 | KRI collection and breach | period/source event → derived indicator, focused residual questions, Matter | metric owner + risk authority | Expansion |
| UC-AUD-01 | Audit or assurance review | approved plan → scope, sample, evidence, independent conclusion | internal audit/assurance | Expansion |
| UC-POL-01 | Policy lifecycle | new/change/expiry trigger → draft, consultation, approval, publication, acknowledgement | policy owner + approver | Expansion |
| UC-COND-01 | Complaint or conduct concern | customer/staff/system input → assessment, action, remedy, reporting | conduct/compliance authority | Expansion |
| UC-DEC-01 | Material decision | Matter requires choice → options, challenge, approval, conditions, verification | authority matrix | Foundation |
| UC-AUTO-01 | Governed automation | repeated low-impact pattern → simulated, approved, staged, monitored automation | automation owner + risk authority | Expansion |
| UC-COM-01 | Executive/committee decision | reporting period or escalation → point-in-time brief, decision, dissent, action | committee mandate | Expansion |
| UC-REPORT-01 | Governed report or export | filing/committee/audit/request → approved point-in-time package and manifest | report owner/signatory | Pilot |
| UC-HIST-01 | Point-in-time reconstruction | audit/legal/review inquiry → exact state known and acted on at selected time | authorized investigator/auditor | Foundation |

Vendor-completed work for a Program or Matter composes `UC-EVID-02` with `UC-TPRM-01` or `UC-TPRM-02`. It is not a separate task, form, document or approval use case: Capture owns the external response, the vendor relationship owns the association and request history, and the Program or Matter retains bank ownership, authorization, verification and closure.
| UC-ADMIN-01 | Tenant support and break-glass | incident/support need → narrow time-bound access and retrospective review | security/platform authority | Enterprise |
| UC-GROUP-01 | Multi-entity/group operation | shared obligation/service/change → local and group state without scope leakage | group and entity authorities | Enterprise |
| UC-MIG-01 | Tenant migration or offboarding | deployment/change/end of contract → verified export, retention, deletion/hold | tenant owner + platform authority | Enterprise |

## 5. Required use-case contract

Every use case must document, concisely:

1. customer segment and personas;
2. business outcome and non-goal;
3. trigger, preconditions, scope, and period;
4. authoritative sources and known limitations;
5. data sensitivity, purpose, and retention;
6. performer, accountable owner, reviewer, challenger, authorizer, and escalation owner;
7. happy path;
8. ambiguity, contradiction, delegation, conflict, absence, and overdue paths;
9. source unavailable, AI unavailable, partial failure, and resume paths;
10. explicitly prohibited actions;
11. state transitions, parallel work, closure, cancellation, and reopening;
12. notifications and invitation channels;
13. AI contribution and deterministic fallback;
14. evidence, audit, and point-in-time requirements;
15. active-effort and comprehension target;
16. workload and performance profile;
17. acceptance-test and implementation references.

Do not duplicate common mechanics inside every specification. Reference the canonical routing, capture, evidence, architecture, and quality contracts and document only domain-specific differences.

## 6. Role-oriented journeys

### Responsible or respondent

```text
Receive one scoped responsibility
→ understand purpose and known facts
→ confirm, correct, provide, redirect, or raise concern
→ review final assertions
→ submit
→ see accepted, follow-up, or completed state
```

### Reviewer or challenger

```text
See proposed result and full denominator
→ inspect changed, uncertain, contradictory, sampled, or material items
→ accept, edit, reject, request evidence, delegate, or escalate
→ preserve independent conclusion
```

### Authorizer or signatory

```text
See scope, evidence, uncertainty, options, authority, consequences, and verification
→ approve, conditionally approve, reject, return, or escalate
→ record rationale and effective period
```

### Administrator

```text
Create versioned draft
→ configure from source-backed objects
→ simulate representative scenarios
→ inspect impact and conflicts
→ maker-checker approval
→ schedule activation
→ monitor and roll back if necessary
```

### Invited external party

```text
Redeem request-scoped invitation
→ complete required identity step
→ see minimal purpose and request context
→ provide approved evidence
→ review submission
→ receive safe receipt or follow-up
```

## 7. Capability boundary

The catalogue does not make ClearSight the specialist execution engine for every domain.

ClearSight governs:

- applicability and institutional context;
- responsibility, authority, and decision;
- evidence and contradiction;
- Matters and commitments;
- external response and reporting;
- verified outcomes and institutional memory.

Specialist systems may perform transaction monitoring, security detection, ITSM execution, IAM changes, project delivery, customer servicing, or commodity compliance automation. Their results enter ClearSight as governed observations and implementation evidence.

## 8. Change rule

A new feature name may not be added to the README, navigation, sales material, or implementation plan until this catalogue identifies:

- its use-case ID;
- maturity;
- primary actors and authority;
- product boundary;
- required acceptance coverage.
