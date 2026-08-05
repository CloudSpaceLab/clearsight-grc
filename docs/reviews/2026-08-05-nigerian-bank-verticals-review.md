# Nigerian bank verticals implementation review

Date: 5 August 2026  
Status: supersedes the initial PR #10 review after the post-merge audit in issue #11.

## Corrected findings

### Tenant scoping was not actor-bound

The web client previously hard-coded `bank-demo`, while the journey endpoint trusted a query tenant independently of the verified actor. The remediation derives runtime context from verified identity and rejects conflicting tenant, principal or legal-entity query scope without confirming whether the requested scope exists.

### Restricted reads could fail open or distort pagination

Malformed access metadata previously behaved as unrestricted, wildcard legal-entity scope bypassed the allow-list, and Matter summaries were filtered after pagination. The remediation introduces a typed fail-closed policy and applies visibility in memory/PostgreSQL summary repositories before keyset pagination. Linked evidence-request visibility is derived from the Matter before PostgreSQL limits.

### Today support was not present in PostgreSQL

PostgreSQL composition previously initialized Today with an empty static slice. Memory Today was also a startup snapshot. Both compositions now calculate actor-visible journey work at request time and carry exact action targets.

### Seed idempotency was overstated

The original seed returned as soon as the reference Program existed, so an interrupted installation could leave missing requests or Matters permanently. The new explicit non-production installer reconciles each stable reference object and repairs partial Program and Matter states. It rejects collisions with non-reference bank records.

### Journey completion accepted overly broad historical evidence

The original projection relied on counts and “any historical record” checks. The corrected projection requires named approved requirements, active evidence checks, implemented linked safeguards, current assessments, current decisions/responses/actions and independently recorded passing outcome results. Retired, superseded, cancelled, withdrawn, unrelated and non-independent records cannot satisfy a stage.

### Explore was informational rather than operational

The original cards displayed passive linked-record badges and next-action text. Explore now opens the exact Program, issue or evidence request, displays material reasons and blockers, and provides permission-aware next-action launchers. Program and Matter detail no longer silently truncates material facts, contradictions, requirements, evidence checks or closure blockers.

### Accessibility and hierarchy needed explicit acceptance

The remediation adds semantic progress bars, journey-specific accessible control names, live/busy loading states, keyboard focus visibility, stronger restricted-record contrast, responsive 320-pixel reflow and reduced-motion behavior. State, reason, next action and deadline receive priority over supporting metadata.

## Implemented scope

- verified actor-derived context and query-scope conflict rejection;
- fail-closed restricted Matter policy;
- pre-pagination Matter visibility and pre-limit linked evidence visibility;
- exact current-record journey evaluation;
- independent outcome-check enforcement;
- actionable-request prioritization;
- dynamic Today projection in memory and PostgreSQL;
- recoverable opt-in reference installer;
- actionable Explore connected-record inspector;
- accessible progress, focus, loading, error, restricted and mobile states;
- full material fact/reason/blocker display without raw JSON;
- synchronized README, documentation map, implementation plan and focused OpenAPI contract;
- adversarial security, partial-recovery and historical-record tests.

## Explicit boundaries

This remediation still does not claim:

- a complete Nigerian banking regulatory library;
- automatic legal interpretation;
- direct NDPC or CBN source ingestion;
- production document classification or legal privilege controls;
- external authority-channel transmission;
- enterprise identity and restricted-group synchronization;
- database row-level security;
- subject-scoped authorization across every Matter and evidence mutation;
- complete write UI for every lifecycle command;
- automated visual-regression/accessibility release evidence;
- production-scale journey benchmarking.

These remain release-gate work rather than hidden assumptions.
