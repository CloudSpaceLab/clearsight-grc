# Nigerian bank reference journeys

Status: repository reference implementation, corrected after the PR #10 post-merge audit on 5 August 2026.

These journeys demonstrate how Programs, issues, sources, evidence requests, authority, decisions, actions, responses and outcome checks work together. They are sample operating records for product and acceptance testing. They are not legal advice, a substitute for the bank's regulatory inventory, or a claim that every Nigerian-bank obligation has been captured.

## Security and scope contract

Journey, Program, issue and evidence reads are bound to the verified actor:

- a conflicting tenant, principal or legal-entity query is rejected without confirming whether that scope exists;
- restricted records require valid access metadata and an explicit principal allow-list;
- malformed, unknown or empty restricted policy metadata fails closed;
- legal-entity wildcard values do not bypass a restricted allow-list;
- Matter visibility is applied before keyset pagination;
- linked evidence-request visibility is derived from the Matter before PostgreSQL limits, regardless of duplicated request sensitivity;
- direct unauthorized reads return not found.

This remains an HTTP/repository access-control foundation. Production still requires enterprise group synchronization, database row-level security and subject-scoped authorization for all protected mutations.

## Source discipline

The privacy fixture uses official Nigeria Data Protection Commission material as its starting point:

- Nigeria Data Protection Act 2023;
- NDP Act General Application and Implementation Directive 2025;
- NDPC audit-filing and licensed DPCO guidance;
- official authority correspondence when modelling a regulator request.

A deploying bank must confirm current text, commencement dates, applicability, sector overlays, internal interpretation and accountable owners. Internal system registers are displayed as supporting evidence sources; they do not satisfy the journey's official-law/source stage.

## Journey 1 — Nigeria data protection

The ongoing Program records five named obligation families:

1. accountable personal-data processing records;
2. data-subject request handling;
3. privacy incident assessment and notification decisions;
4. privacy review for high-risk processing changes;
5. annual Compliance Audit Return readiness.

Completion is based on the exact approved requirement and active evidence-check codes, implemented safeguards linked to those requirements, current assessments and an active Program. Unrelated records and broad counts cannot satisfy a stage.

The reference state includes three high-risk changes without approved privacy-review records. ClearSight creates one focused actionable request containing only the missing change references and review records.

## Journey 2 — Regulatory change

The regulatory-change issue records:

- the official source and affected annual-return process;
- known and missing facts;
- the current approved implementation decision;
- assigned, non-cancelled work;
- an active outcome check that prevents closure until the revised process is evidenced.

A historical approved decision does not satisfy the stage when a newer decision is rejected or returned. Cancelled actions and retired outcome checks are excluded.

## Journey 3 — Protected regulator request

The authority-request issue is `RESTRICTED` with an explicit allow-list. Its related evidence request inherits the Matter's visibility rule.

The sample records receipt, requested records, restricted evidence collection, response preparation, current signatory approval, transmission, authority acknowledgement and closure after acknowledgement. A historical acknowledgement cannot satisfy the journey when the current response is withdrawn or rejected.

## Journey 4 — Finding remediation

The legacy audit finding records the affected population and original deadline, then proceeds through assignment, implementation, an active outcome check, independent re-performance, a passing result and evidence-based closure.

The result must be recorded by the contract's reviewer and must not be recorded by the remediation action owner. Retired contracts and non-independent passing results do not satisfy closure.

## Explore workspace

Explore prioritizes:

1. current state and plain-language reason;
2. exact next action and deadline;
3. accountable function;
4. incomplete and complete stages;
5. supporting sources and linked records.

Each configured journey provides a permission-aware launcher to its exact Program, issue or evidence request. The connected-record inspector displays all material status reasons, facts, missing information, contradictions and closure blockers without silent truncation or raw JSON.

The workspace uses semantic progressbar attributes, specific accessible control names, focus-visible states, mobile reflow and reduced-motion behavior. Reference records are visibly labelled.

## Today work

Today is derived at request time from current actor-visible journey state in both memory and PostgreSQL compositions. It excludes closed/not-configured journeys and includes action-target metadata so a work item can open the relevant workspace. Changes to journey state no longer require an API restart to update Today.

## Recoverable installation

The reference installer is explicit and refuses `CLEARSIGHT_ENV=production`. It independently reconciles stable source, Program, requirement, safeguard, evidence, request and Matter identities. Re-running after an interrupted installation repairs missing stages without duplicating completed work.

```bash
go run -tags postgres ./cmd/seed-bank-reference \
  -tenant <tenant-uuid-or-slug> \
  -legal-entity <legal-entity-uuid> \
  -actor <installer-principal-uuid> \
  -owner <owner-principal-uuid> \
  -reviewer <reviewer-principal-uuid> \
  -signatory <signatory-principal-uuid>
```

An existing Program code or Matter trigger is reused only when it carries the ClearSight sample/journey provenance marker. A collision with a bank-owned non-reference record fails explicitly.

## Performance contract

The journey endpoint uses bounded exact lookups for one named Program, three trigger-key Matters, subject-scoped current requests and named sources. It does not replay the bank's full population. Restricted Matter summaries and linked evidence-request lists apply access predicates before pagination or limits.

## Acceptance criteria

The reference implementation is accepted when tests prove:

- all four journeys derive from canonical records;
- verified actors cannot read or infer another tenant's scope;
- malformed restricted policy fails closed;
- visibility is applied before pagination/limits;
- partial installation recovers without duplicate work;
- the privacy Program has the five named approved obligations and active evidence checks;
- the regulatory journey has a current approved decision, current work and pending outcome;
- the authority response is currently approved, transmitted, acknowledged and closed;
- the finding has current implemented work and an independent passing result;
- retired, superseded, cancelled, withdrawn, unrelated and non-independent records cannot satisfy stages;
- Today includes current open actor-visible work and excludes completed journeys;
- the web workspace opens exact connected records and exposes material blockers;
- PostgreSQL constraints and tenant boundaries remain intact.

## Remaining production boundaries

This reference phase does not claim a complete Nigerian regulatory library, automatic legal interpretation, direct NDPC/CBN source ingestion, production privilege classification, external authority-channel transmission, synchronized restricted groups, database row-level security, complete mutation authorization, or production-scale benchmarking.
