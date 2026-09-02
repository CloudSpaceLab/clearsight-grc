# Real email, seeded runtime truth and oversight history design

**Date:** 2026-09-02

**Status:** Approved by the operator on 2 September 2026

**Scope:** Issue #128 email/link acceptance, elimination of hardcoded runtime/API demo truth, and stored oversight-history completeness

## Outcome

ClearSight will prove both vendor email/link journeys through the configured hosted SMTP transport and the canonical capture, Vendor Work, Matter, review, outcome and closure paths. Normal and demonstration runtimes will return only stored or verified identity data through ordinary repositories and projections; deterministic browser fixtures will remain available only through a test-only evidence entry that cannot ship in the application bundle. The non-production reference installer will also create enough explicitly labelled, realistically completed issue history through normal workflow transitions for the oversight projection to calculate explainable resolution ranges and performance populations.

Completion is based on the exact deployed commit, stored state, real inbox receipt and traversed links. Provider acceptance, seeded rows, a rendered email, a static UI fixture or a green unit test is not a substitute for the governed outcome.

## Confirmed baseline

The hosted release is `5d7a510f99eabdf9a0ce370d69c19e9d42d2af0d`.

- SMTP, STARTTLS, sender, external delivery, recipient keyring, active key, access HMAC and public capture URL are configured; the redacted readiness check passes.
- The hosted database has one delivered address-verification Vendor Work item, but no completed response revision and no completed certification-refresh response.
- Oversight has persisted snapshots, but only two closed Matters with near-zero sample durations. That population cannot support a truthful resolution outlook.
- `internal/today.DemoItems()` remains as hardcoded sample work even though current production composition no longer calls it.
- `web/src/main.tsx` can import the static API interceptor when `VITE_STATIC_DEMO` and `VITE_UI_EVIDENCE` are enabled. The interceptor contains product-looking API records and is too close to a shippable application entry.
- The hosted demonstration deployment is already PostgreSQL-backed and runs the non-production reference installer. The remaining problem is code-bound demo truth and incomplete stored acceptance state, not an absent database.

## PR #129 disposition

PR #129 is not a merge prerequisite and must not be merged in its reviewed state.

Useful direction:

- resolve workspace display labels from verified identity and tenant-scoped PostgreSQL rows in persisted composition;
- select exact approved form revisions rather than asking an operator to type identifiers;
- drive scored responses through the normal distribution, access, workspace and submission path.

Blocking findings:

- the branch fails formatting, does not compile, and misses two existing seeded-form regression updates;
- its form-policy selector includes active unscored forms even though policy creation rejects them;
- it introduces hardcoded demo organization and role labels in normal memory/demo API composition;
- its static-truth regression scans only non-test files in `internal/httpapi`, so it cannot detect the new `cmd/api` fixture, `today.DemoItems()` or the web static interceptor;
- its seed-to-Matter acceptance command is not executed by CI, and the repeat-run shortcut does not reconstruct or revalidate the claimed good, shadow and active-automation evidence.

This tranche may reproduce the PostgreSQL resolver idea with corrected tests. Scoring/policy delivery remains outside the operator's current priority unless a small correction is required to keep the current branch buildable.

## Decisions

### 1. One persisted runtime truth path

PostgreSQL demonstration and non-demonstration runtimes use the same actor-context, Today, Forms, Vendor, Matter and oversight handlers and repositories.

- tenant, legal-entity and principal labels come from the exact verified actor scope and current stored directory rows;
- operational counts, cards, tasks, responses, policies, hierarchy records, vendors and Matters come from bounded stored queries or projections;
- missing rows produce a scoped unavailable or empty response, never a friendly invented organization, person, count or task;
- demo mode may expose reference journeys and supplied non-production sign-in accounts, but it does not switch an API handler to static business data;
- all reference records carry stable IDs and visible `Reference data` or equivalent sample provenance.

The in-memory composition remains a deterministic development/test adapter, not the stakeholder demonstration. It may start empty or be populated through explicit fixture installation using domain services. It must not return business records from handler constants.

### 2. Test-only UI evidence entry

Deterministic visual and accessibility fixtures remain necessary, but they are isolated from the product entry.

- `web/src/main.tsx` never imports or activates a static API interceptor;
- a dedicated evidence entry and build command own fixture-only API interception;
- normal development, preview, demonstration and production builds cannot resolve that entry through environment flags;
- evidence fixtures are clearly synthetic and cannot be mistaken for hosted operational truth;
- CI fails if application entrypoints or production API handlers import fixture modules, call reserved demo-data helpers or contain registered business-record literals.

The gate uses explicit architectural boundaries as well as reserved-literal checks. A phrase scan alone is insufficient.

### 3. Canonical real-email journeys

Both journeys use the active Forms and Vendor Work contracts, not a demo renderer or one-off mail command.

#### Journey A — registration and address verification

1. Create a fresh, clearly labelled reference vendor with the approved vendor test recipient.
2. The verified compliance officer starts onboarding and sends the registration request.
3. The outbox worker delivers one STARTTLS message with one secure action, deadline, expiry, recovery text and equivalent plain text.
4. The vendor opens the received link, completes the required email-verification step and submits registration through the canonical response workspace.
5. The vendor relationship and exact Vendor Review Matter show `Address verification pending`; pending verification is not a deficiency or resolved outcome.
6. The compliance officer assigns the address-verification work to the approved staff test recipient. Assignment and notification delivery remain separate durable outcomes.
7. The staff recipient opens the purpose-bound link, verifies the invited address and submits result, method, date, source, PDF evidence and attestation.
8. The compliance officer reviews the exact current response and available artifact, then accepts it or requests a targeted change with rationale.
9. A current passed outcome, separate authorized sign-off and authorized closure resolve the Matter. The final UI identifies the evidence basis, signatory and resolution time.

#### Journey B — certification refresh

1. Use the registered reference vendor and create certification-refresh Vendor Work.
2. Send the active `VENDOR-CERTIFICATION-REFRESH` form to the approved vendor test recipient.
3. The received message and secure link name ISO 27001 and PCI DSS separately without assuming applicability.
4. The vendor verifies access, answers applicability/current-state questions, submits test-labelled PDFs where applicable and attests to the response.
5. The bank reviewer evaluates each item separately and can request replacement of one item without discarding an accepted item.
6. Submission, artifact availability, bank acceptance, passed outcome, sign-off and Matter closure remain separate visible states.

### 4. Link and delivery proof

Acceptance records only redacted identifiers and timestamps.

- verify SMTP provider acceptance and human-observed inbox receipt independently;
- verify the message sender, subject, HTML/plain parity, deadline, expiry and one dominant action;
- traverse the exact received link rather than copying a route from the database;
- prove wrong audience/OTP, expiry, revocation, replay, reissue and active-session invalidation fail without disclosing task metadata;
- verify reassignment notifies the new assignee once and does not grant Matter ownership, review, signatory or closure authority through email possession;
- do not print or persist SMTP credentials, recipient addresses, OTPs, opaque selectors or full secure URLs in logs, screenshots or acceptance documents.

### 5. Stored oversight history

The reference installer creates at least five completed issue histories through ordinary Matter commands and the current authority route.

The cohort is operationally varied and explicitly labelled reference data:

- several Matter types and categories;
- priorities representing ordinary and material work;
- on-time, overdue and reassigned examples;
- realistic creation, action, review, outcome and closure intervals;
- at least one returned or reopened example where the domain permits it;
- distinct accountable owners and reviewers without fabricating a composite employee rating.

The installer does not insert closed Matter rows or oversight metrics directly. It uses canonical creation, decision/action, verification and closure services so authoritative rows, append-only history, outbox events and projection maintenance retain normal semantics. Repeated installation is idempotent by stable reference identity and verifies the complete expected history before reporting `already seeded`.

Oversight acceptance proves:

- population and exclusions for the selected legal entity, period, category and organization scope;
- projection `generated_at`, version and source high-water marks;
- completed count, median and p75 cycle time, SLA attainment, reopen/return/reassignment facts and sparse-cohort behavior;
- deterministic resolution ranges only where the documented minimum sample is met;
- no employee ranking, invented completion date or persuasive fallback number;
- bounded drilldowns open the exact underlying Matters and reconstruct their history.

## Authority, identity and transaction boundaries

- Staff commands derive actor, tenant and legal entity only from the verified request context.
- External links grant one request-scoped response capability and never staff identity or broader tenant access.
- Assignment, response, evidence review, outcome, sign-off and closure re-evaluate the current route required for that command.
- Material commands commit authoritative state, append-only event, required outbox event and maintenance work in one transaction.
- SMTP delivery occurs asynchronously. A delivery failure does not roll back an already committed assignment or distribution.
- A derived projection failure after commit produces a stale/unavailable read with recovery; it does not turn a committed command into a reported failure.

## UI and copy contract

The affected bank and recipient workspaces use shared components and one dominant next action.

- Every screen names the vendor/request/Matter, current state, owner or recipient role, deadline/freshness and next result.
- Emails use conservative client-safe HTML, a plain-text alternative, no remote images or tracking, and one accessible action.
- Recipient pages work in light/dark modes, desktop and narrow mobile layouts, keyboard navigation and 200% reflow.
- Empty states name the stored population checked and valid next action.
- `Submission received`, `Evidence accepted`, `Outcome passed`, `Signed off` and `Resolved` remain distinct labels.
- Unknown, stale, unscanned, unavailable and partially accepted states remain explicit.

## Failure and recovery states

Required negative and degraded coverage includes:

- SMTP unavailable before acceptance and ambiguous delivery outcome after DATA acceptance;
- missing staff contact, reassignment, duplicate event replay and expired assignment link;
- wrong inbox, wrong OTP, expired/revoked/replayed route and reissued route invalidating the old session;
- artifact unavailable or unscanned, contradictory address, rejected certificate and targeted change request;
- stale outcome after a newer response, unauthorized sign-off and closure before a current passed outcome;
- empty or partially installed reference data, interrupted seed run, projection lag and oversight sparse-history suppression;
- unavailable directory label or operational source without a hardcoded fallback.

## Verification strategy

Implementation follows test-driven slices.

1. Add failing architectural tests that prove normal API/web entries cannot reach static fixture truth.
2. Add failing actor-context tests for stored labels, missing rows and strict tenant/legal-entity/principal scope.
3. Add failing vendor email/link workflow and SMTP adapter tests, including security and delivery ambiguity.
4. Add failing reference-history installer tests for canonical transitions, idempotency and complete reconstruction.
5. Add failing oversight projection tests for cohort sizes, exclusions, freshness, ranges and sparse suppression.
6. Run current-contract Go, tagged PostgreSQL, web, copy-quality, accessibility, bundle and migration gates.
7. Render every materially affected state at desktop/mobile, light/dark and reflow sizes; inspect, fix the highest-impact defect and re-render.
8. Deploy the exact green commit, run the redacted readiness check, then traverse both received-email journeys and verify the final stored UI/audit state.

## Deployment and rollback

- Preserve the protected host configuration and existing recipient keys; do not rotate keys that are required to decrypt active recipient records.
- Merge and deploy only an exact tested commit.
- Seed reference history before starting the new release, then verify counts and reconstruction before marking the deployment current.
- Stop the live acceptance run if sender, recipient, tenant or legal entity differs from the approved scope.
- If external delivery must be disabled, preserve committed distributions, delivery receipts and operator recovery state.
- Retain the prior image/configuration for application rollback. Never hide a post-migration incompatibility behind a legacy runtime path.

## Acceptance criteria

This tranche is complete only when:

- both approved inboxes receive the intended messages through the configured SMTP relay and the received links are traversed successfully;
- registration reaches pending address verification, staff evidence, bank acceptance, passed outcome, sign-off and verified Matter closure in the UI;
- certification refresh reaches secure vendor submission, per-item bank review and the intended unresolved/resolved state without conflating receipt and acceptance;
- expiry, revocation, audience, replay, reassignment and delivery-recovery behavior is proven without secret exposure;
- normal application/demo APIs and web entrypoints return stored/verified data only, with no static business-record fallback;
- deterministic UI fixtures are isolated to a non-shipping evidence entry;
- at least five completed reference Matters have reconstructable normal transition history and oversight shows explainable populations, freshness, exclusions and resolution ranges;
- all exact-head automated, rendered and hosted acceptance gates pass.

## Explicit remainder

This tranche does not remove unreleased compatibility paths, complete scoring/policy automation, build production object scanning/storage, add bounce/complaint monitoring, finish the broader third-party lifecycle, implement a generic staff notification platform, or prove representative production load and bank-user timing. Those remain separately tracked and must not be implied by these acceptance journeys.
