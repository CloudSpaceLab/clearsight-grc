# Nigerian bank reference journeys

Status: repository reference implementation, reviewed 5 August 2026.

These journeys demonstrate how ClearSight's existing Programs, issues, sources, evidence requests, authority, decisions, actions, responses and outcome checks work together. They are sample operating records for product and acceptance testing. They are not legal advice, a substitute for the bank's regulatory inventory, or a claim that every Nigerian-bank obligation has been captured.

## Source discipline

The privacy fixture uses official Nigeria Data Protection Commission material as its starting point:

- Nigeria Data Protection Act 2023;
- NDP Act General Application and Implementation Directive 2025;
- NDPC audit-filing and licensed DPCO guidance;
- official authority correspondence when modelling a regulator request.

A bank deploying the fixture must confirm the current text, commencement dates, applicability, sector-specific overlays, internal policy interpretation and accountable owners. Source records carry freshness expectations so later changes can create review work instead of silently changing an approved position.

## Journey 1 — Nigeria data protection

The ongoing Program records five representative obligation families:

1. accountable personal-data processing records;
2. data-subject request handling;
3. privacy incident assessment and notification decisions;
4. privacy review for high-risk processing changes;
5. annual Compliance Audit Return readiness.

Each obligation is connected to:

- an applicability decision;
- a safeguard objective;
- an implemented safeguard;
- one or more authoritative sources;
- a population and coverage expectation;
- a freshness period;
- a recorded evidence assessment.

The reference state deliberately includes three high-risk changes without approved privacy review records. ClearSight therefore creates one focused evidence request containing only the missing change references and review records.

## Journey 2 — Regulatory change

The regulatory-change issue starts with an official source and records:

- the affected annual-return process;
- known and missing facts;
- an approved implementation decision;
- conditions attached to the decision;
- an assigned action;
- an outcome check that prevents closure until the revised process is evidenced.

The sample remains open at **Change in progress**. It does not imply that creating a task completes regulatory implementation.

## Journey 3 — Protected regulator request

The authority-request issue is marked `RESTRICTED` and carries an explicit principal allow-list. Read APIs omit it for other users; direct reads return not found. The related restricted evidence request uses the same Matter visibility check.

The sample journey records:

- receipt and due date;
- requested records;
- a restricted internal evidence request;
- response-package preparation;
- signatory approval;
- transmission;
- authority acknowledgement;
- closure only after acknowledgement.

Restricted access is enforced by the API. A hidden card or CSS rule is never treated as an access control.

## Journey 4 — Finding remediation

The legacy audit finding records the affected population and original deadline, then proceeds through:

- assignment;
- implementation;
- a defined outcome check;
- independent re-performance;
- a passed result;
- evidence-based closure.

The finding cannot close merely because the remediation owner marks an action complete.

## Explore workspace

Explore shows the four connected journeys using ordinary working language:

- current status;
- accountable function;
- completed stages;
- next recorded action;
- deadline;
- source names;
- linked Program, issue and evidence-request availability.

Sample data is visibly labelled. The workspace contains no inactive buttons and loads only when Explore is opened.

## Performance contract

The journey endpoint does not list and replay the bank's full Program or issue population. It uses:

- exact indexed Program-code lookup;
- exact indexed trigger-key lookup;
- exact subject-scoped evidence-request lookup;
- bounded source-code lookup.

The endpoint loads at most one Program, three issues, two evidence requests and the named sources required by the four journeys.

## Acceptance criteria

The reference implementation is accepted only when tests prove that:

- all four journeys are derived from canonical domain records;
- sample seeding is idempotent;
- the privacy Program has five obligations and five evidence checks;
- the regulatory-change issue has an approved decision, assigned action and pending outcome;
- the authority response is approved, transmitted, acknowledged and closed;
- the legacy finding has an implemented action, independent passing outcome and closure;
- unauthorized actors cannot read the restricted issue or request;
- Today includes open journey work but excludes completed journeys;
- PostgreSQL constraints and tenant boundaries remain intact.
