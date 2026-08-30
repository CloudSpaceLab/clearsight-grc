# Governed Forms Task 22 Closure Design

**Date:** 2026-08-30

**Status:** Proposed for implementation

**Tracking:** [GitHub issue #94](https://github.com/CloudSpaceLab/clearsight-grc/issues/94)

**Maturity target:** Release evidence and operational-acceptance closure for the governed Forms foundation already delivered on `main`

## 1. Decision summary

Close the remaining governed Forms Task 22 evidence gap with three coordinated proof layers:

1. expand PostgreSQL integration coverage to exercise the documented scale, bounded-query, maintenance-claim, revision-history and point-in-time reconstruction contracts;
2. replace the small fixed screenshot list with a deterministic, capability-tagged Forms state registry whose representative renders cover every required state across desktop, mobile, reflow, light and dark evidence; and
3. add a non-mutating hosted release verifier that proves the deployed revision, runtime readiness, authenticated bounded reads and opaque denial behavior without polluting demo data or exposing invitation tokens.

The change will synchronize the implementation plan, product specification, acceptance tests, rendered-evidence index and performance/retention ownership. It will not claim that production is bug-free, and it will keep external operational dependencies explicit.

## 2. Why this work remains

The core governed Forms workflow is implemented and deployed, but the current Task 22 proof is narrower than the documented acceptance contract:

- the scale integration test creates 1,000 templates and 400 distributions, but does not populate representative recipient and response-revision depth or prove exact response lookup, bounded maintenance claims, reconstruction and retention ownership;
- the rendered evidence covers only five Forms states, leaving material workflow, degraded, responsive, theme and accessibility states unproven as a coherent matrix;
- deployment proves that an image was deployed and that health/read checks succeed, but the public verification path does not independently report the exact running release revision;
- hosted checks must not create durable demo records, leak opaque access tokens or pretend that automated checks replace delivery-provider, object-storage, malware-scanning and human accessibility evidence; and
- the implementation, product, acceptance, evidence and operational documents do not yet present one synchronized completion statement.

This is a release-evidence closure. It does not redesign Forms or create a second workflow model.

## 3. Scope

### 3.1 Included

- PostgreSQL-backed scale fixtures for templates, distributions, recipients and immutable response revisions.
- Assertions for keyset pagination, exact indexed lookup, tenant/legal-entity isolation and bounded maintenance work claims.
- Point-in-time reconstruction checks for the material Forms record families covered by the implementation plan.
- Explicit retention, partitioning, maintenance-job ownership and alert thresholds in the system-performance documentation.
- Deterministic Forms UI scenarios covering the complete Task 22 state contract.
- Representative rendered evidence across desktop, mobile, 200%-reflow approximation, light and dark modes.
- A manifest validator that fails when a required capability or presentation dimension has no evidence.
- Exact deployed-revision reporting and a non-mutating hosted verification script/workflow.
- Documentation and traceability updates tied to issue #94.

### 3.2 Excluded

- A new form-builder, response model, invitation model or authorization route.
- Mutating a shared production/demo tenant as part of a deployment smoke test.
- Seeding or logging plaintext invitation tokens.
- Replacing provider delivery receipts, versioned object storage, malware scanning, real assistive-technology testing or human timing studies with automated approximations.
- Closing the separate vendor lifecycle, AI governance UX, continuous-assurance or GA-readiness work tracked by issues #80, #74, #57 and #13.

## 4. Alternatives considered

### 4.1 UI evidence organization

**Alternative A: exhaustive Cartesian screenshots.** Render every state at every viewport and theme. This is mechanically complete but produces a slow, noisy artifact set whose duplicate images obscure real regressions.

**Alternative B: capability-tagged representative matrix — selected.** Define deterministic scenario fixtures with tags for workflow state, degraded behavior, viewport, reflow, theme and accessibility-relevant behavior. The capture plan selects a compact cover, while the manifest fails unless every required capability and presentation dimension is represented. Behavioral tests continue to prove interactions that screenshots cannot.

**Alternative C: component tests only.** Fast, but insufficient for the required rendered workflow and responsive proof.

### 4.2 Scale and reconstruction proof

**Alternative A: one monolithic mega-fixture.** Easy to describe, but slow to diagnose and prone to coupling unrelated assertions.

**Alternative B: modular deterministic fixture builders and focused subtests — selected.** Reuse one seeded population where safe, expose exact cardinalities, and separate pagination, lookup, claim, isolation and reconstruction assertions. Failure output identifies the broken contract.

**Alternative C: SQL-plan assertions without behavior checks.** Useful as a supplement, but an index plan alone does not prove correctness, isolation or bounded work.

### 4.3 Hosted release verification

**Alternative A: run the entire mutating Forms journey against the shared hosted demo.** This provides end-to-end mutation proof but contaminates durable records, complicates cleanup and increases token-disclosure risk.

**Alternative B: layered proof — selected.** PostgreSQL integration tests exercise all material mutations at the exact commit. The hosted verifier independently proves that the same revision is running, the service is ready, authentication works, bounded Forms reads succeed and invalid/unauthorized opaque access reveals no protected record. A future dedicated acceptance tenant may add safe hosted mutation testing.

**Alternative C: trust the deploy log and health status alone.** This cannot independently prove that the responding runtime matches the expected commit.

## 5. Detailed design

### 5.1 PostgreSQL scale, bounded work and reconstruction

Refactor the existing Forms scale test into deterministic builders and focused subtests. The test remains opt-in through `TEST_DATABASE_URL` and must run in the repository's PostgreSQL integration gate.

The seeded population will include:

- at least 1,000 form templates spanning tenant and legal-entity scopes;
- at least 400 distributions with enough recipients per distribution to exercise recipient pagination and maintenance selection;
- response revisions that prove immutability and historical reads rather than only the latest response; and
- records eligible and ineligible for reminder and refresh-maintenance claims, including leased and retryable cases.

The assertions will prove:

- stable keyset pagination with no duplicates or skips at page boundaries;
- exact lookup by indexed identifiers for known templates, distributions, recipients and responses;
- tenant and legal-entity filtering in the repository query rather than post-load filtering;
- bounded, deduplicated and lease-aware reminder and maintenance claims;
- normalized `EXPLAIN` plans use the intended indexes for high-cardinality paths; and
- reconstruction at a chosen timestamp returns the version that was material then, while the current read returns the latest version.

Reconstruction coverage will follow the material record families named in the governed Forms plan: template versions, distribution revisions, recipient state, response revisions, evidence/artifact references and refresh/sign-off outcomes. If an existing family has no historical contract, the test must expose that as a product gap rather than manufacture history in test-only code.

Performance assertions will use bounded-work properties and explain-plan shape, not fragile wall-clock thresholds on shared CI hardware. Operational latency and throughput targets remain documented load-test thresholds.

### 5.2 Deterministic UI scenario registry

Extend the existing `fixture` query mechanism into a Forms scenario registry. Each scenario will have:

- a stable fixture key;
- a business-readable evidence name;
- one or more required capability tags;
- an intended viewport class;
- a theme; and
- setup assertions that confirm the expected dominant action or limitation is visible before capture.

Required capability tags will cover the Task 22 state matrix:

- template library, search/filter, empty and error states;
- template creation/editing, version history and publish/retire constraints;
- distribution setup, recipient selection, send confirmation and sent state;
- recipient access, expired/revoked/invalid access and recovery guidance;
- response drafting, validation, submission, revision/history and reviewer follow-up;
- refresh, amendment, evidence, sign-off and outcome-check states;
- loading, offline/degraded, permission-denied and stale/freshness states;
- desktop, mobile, 200%-reflow approximation, light and dark presentation; and
- keyboard focus, labels, error association and non-blocking guide/notice behavior where visible evidence is meaningful.

The capture script will render a minimum representative cover instead of every combination. The manifest validator will calculate coverage from tags and fail when a required capability or presentation dimension is absent. Component and workflow tests will continue to own interaction semantics, focus movement, validation, authorization behavior and state transitions that a still image cannot prove.

Every customer-facing fixture string remains subject to `copyQuality.test.ts` and the repository's bank-operating-language rules.

### 5.3 Exact-revision hosted verification

Expose the running release revision through a safe readiness field or dedicated public revision endpoint. The value will come from an immutable deployment input such as `RELEASE_SHA`, will contain no secrets and will not infer identity or tenant context.

The deployment workflow will pass the expected commit and the verifier will assert:

1. readiness reports PostgreSQL mode and the exact expected release revision;
2. demo/session authentication succeeds through the supported test path;
3. bounded Today, Forms and Vendors reads return valid envelopes without broad population replay;
4. an invalid or unauthorized opaque Forms access request fails with the documented indistinguishable response and discloses no record metadata; and
5. all output redacts credentials, session material and opaque invitation values.

The hosted verifier will be read-only. Revoked, expired, submitted and replay behavior remains mutation-tested in the PostgreSQL integration suite at the same commit. A full hosted mutation journey remains an external acceptance item until a dedicated isolated acceptance tenant, safe token handoff and deterministic cleanup contract exist.

### 5.4 Retention, partitioning and job ownership

Update the performance and operations documentation with a Forms-specific table naming:

- authoritative table or record family;
- expected cardinality and growth driver;
- retention or archival rule;
- partitioning/index strategy and trigger threshold;
- maintenance job and owning service/team;
- lease, retry and dead-letter behavior; and
- observable alert and recovery action.

Documentation must distinguish implemented enforcement from a production policy dependency. No record will be deleted merely because a retention duration is documented.

### 5.5 Documentation and traceability

Synchronize:

- `docs/implementation-plan.md`;
- `docs/product/governed-forms.md`;
- the relevant acceptance-test document;
- `docs/quality/rendered-ui-evidence.md`;
- the system-performance/operations documentation; and
- issue #94.

The completion statement will identify the exact automated population and timestamp. It will explicitly list external evidence still required and will not imply enterprise-wide completeness.

## 6. Security and data boundaries

- Verified request identity remains the source of actor, tenant and legal-entity scope.
- Hosted verification is non-mutating and never accepts actor or tenant identifiers as authority from request bodies.
- Opaque access values are generated only inside tests that need them, are never committed, and are redacted from command and workflow output.
- Invalid, expired, revoked and unauthorized invitation access remains indistinguishable to the caller except for the documented recovery path.
- Scale fixtures use isolated test tenants and transaction-safe cleanup.
- No broad data load followed by application-memory authorization is acceptable.
- Exact revision reporting exposes only a source-control identifier and runtime mode; it exposes no environment secrets or customer data.

## 7. Failure handling

- Fixture builders fail with the record family and expected cardinality that could not be created.
- Query-plan checks normalize volatile cost and row estimates while retaining node and index identity.
- UI capture setup fails before screenshot generation when the intended state or dominant action is absent.
- Manifest errors name the missing capability or presentation dimension.
- Hosted verification stops on revision mismatch before interpreting later checks as evidence for the expected release.
- A post-commit derived-read failure must not be reported as though the material command failed; mutation tests assert the committed result separately from derived projections.

## 8. Verification gates

The implementation is complete only when fresh evidence shows:

- backend unit and integration tests pass, including PostgreSQL race/integration coverage;
- the expanded Forms scale, claim, isolation and reconstruction subtests pass;
- web unit, workflow, copy-quality and accessibility checks pass;
- the capability-tagged render manifest is complete;
- every selected rendered artifact is visually inspected at its intended viewport/theme and the highest-impact defect is corrected and rechecked;
- Compose runtime verification passes;
- deployment reports and serves the exact expected commit;
- the read-only hosted verifier passes without emitting protected values; and
- all named documents agree on what is automated, what is externally dependent and what remains open.

## 9. Explicit external remainder

Even after this design is implemented, the following do not become automated completion claims:

- production email/SMS provider delivery, bounce and complaint evidence;
- production versioned object storage, malware scanning and retention-policy approval;
- human-measured authoring, completion, review and sign-off effort targets;
- real browser 200% zoom and assistive-technology testing by representative users;
- disaster-recovery exercises and sustained production-scale load evidence; and
- a safe full hosted mutation journey, unless an isolated acceptance tenant and cleanup contract are separately approved.

These items remain named dependencies with owners/evidence expectations in the acceptance documentation. They do not block proof that the repository implementation satisfies the automated Task 22 contract.

## 10. Exit criteria

Task 22 may be marked complete when all verification gates pass on the exact merged commit, the hosted runtime independently reports that commit, issue #94 contains links to the evidence, and the synchronized documents distinguish repository completion from the external remainder above.

Issues #13, #57, #74 and #80 remain open because their residual work is separate from this Forms closure.
