# Governed Form Advanced Scoring and Response Policies Design

**Date:** 2026-09-01

**Status:** Approved

**Primary users:** Form Owner, Program Owner, Reviewer, CRO, CCO, CISO, GRC Administrator

**Related specifications:** Governed Forms; Monitoring Setup and Risk Scoring; Governed Form Refresh and Sign-off

## 1. Outcome

ClearSight must calculate explainable, version-qualified scores for every eligible completed form response, let authorized users find completed responses by score or concern level, and apply governed policies that may create a canonical Matter when a poor result meets approved conditions.

The implementation extends the existing form contract, immutable response revision, monitoring result, outbox and Matter trigger foundations. It does not introduce a browser-only calculator, a parallel questionnaire store or a separate issue system.

Success means that a bank user can:

1. configure weighted questions and bounded cross-field scoring rules in the Form Builder;
2. preview the calculation against test answers before submitting the form revision for approval;
3. receive an immutable, server-calculated score and explanation when a response is completed;
4. search, filter and sort completed responses across distributions by score, concern level, form, subject and completion time;
5. configure and simulate a maker-checker-approved response policy; and
6. have one policy-qualified adverse episode create or update one deduplicated Matter with a complete audit trail.

Scores remain observations. A score does not by itself approve evidence, conclude compliance, change a vendor relationship, close a Matter or verify an outcome.

## 2. Existing foundation and gaps

The current form contract supports `NONE`, `RISK` and `COMPLIANCE` scoring modes. Choice questions may map answers to points and weights, compliance forms may weight sections, and completed response revisions can retain a compliance score, coverage, critical results and the scoring implementation version.

The remaining gaps are:

- advanced rules cannot combine answers or score numeric, percentage, currency and date conditions;
- risk-mode scoring is evaluated for Program monitoring but is not consistently retained on every completed response revision;
- the stored response score does not state its direction, concern-equivalent score or business band;
- the Forms Responses workspace first loads distributions and then revisions for one distribution, so it cannot perform bounded cross-distribution score queries;
- the existing adverse monitoring-result Matter path is reviewer-triggered;
- the generic Automation Policy lifecycle is not connected to typed form-response eligibility or Matter actions; and
- there is no durable receipt for policy non-match, match, suppression, failure or Matter linkage.

## 3. Chosen architecture

The chosen design is a native form score profile plus a typed Form Response Automation Policy. It reuses the generic Automation Policy lifecycle and canonical Matter trigger service.

Alternatives rejected for this tranche are:

- extending only Monitoring Checks, because generic Forms responses and non-Program subjects would remain outside the response workspace; and
- embedding a general-purpose rules engine, because arbitrary expressions add execution, security and reconstructability risks without improving the approved use cases.

The solution has five bounded units:

1. **Score profile contract** — validates deterministic contributions, predicates, effects and bands as part of an immutable form revision.
2. **Score evaluator** — calculates one reconstructable result from one form revision and one completed response.
3. **Completed response query** — exposes legal-entity-scoped, indexed score filtering and keyset sorting.
4. **Form response policy** — attaches typed response eligibility and Matter action configuration to one governed Automation Policy revision.
5. **Policy executor** — consumes response-scored events, records an execution receipt and idempotently applies the canonical Matter trigger.

Each unit exposes a typed interface and can be tested without the browser.

## 4. Score profile

### 4.1 Modes and direction

Every scored form revision declares one mode and score direction:

| Mode | Raw score meaning | Poor result |
| --- | --- | --- |
| `RISK` | 0 is lowest risk; 100 is highest risk | high raw score |
| `COMPLIANCE` | 0 is weakest; 100 is strongest | low raw score |
| `NONE` | no score is calculated | not applicable |

The evaluator stores both:

- `raw_score`, preserving the form's stated meaning; and
- `adverse_score`, where 0 is least concerning and 100 is most concerning.

For risk scoring, `adverse_score = raw_score`. For compliance scoring, `adverse_score = 100 - raw_score`. This prevents sorting, filtering and automation from treating an 80% compliance score as equivalent to an 80 risk score.

The user interface names the mode and never displays an unexplained number. It offers **Needs attention first**, **Highest score**, **Lowest score** and **Newest response** sorts.

### 4.2 Contributions

Existing scored choice questions remain supported. The score profile additionally supports bounded rule contributions for:

- text equality and bounded membership;
- yes/no and single-choice answers;
- multi-choice contains, contains-all and contains-any checks;
- integer, decimal, percentage and currency comparisons and inclusive ranges;
- date before, after and inclusive-range checks against literal governed dates; and
- answered or unanswered checks.

Text pattern matching, arbitrary regular expressions, scripts, SQL, network calls and AI evaluation are excluded.

Each contribution declares an ID, label, weight, predicate, match points, non-match points and missing-answer behaviour. Points and weights are whole numbers from 0 to 100. Missing-answer behaviour is explicit:

- `INDETERMINATE` — the contribution is uncovered and the result cannot be final when it is required;
- `EXCLUDE` — the contribution is excluded from the applicable denominator; or
- `ZERO` — the contribution is covered with zero points.

Required visible questions must still be answered before submission. Hidden questions are excluded from the applicable denominator. Optional unanswered questions are never silently scored as zero unless the approved contribution explicitly selects `ZERO`.

### 4.3 Advanced predicates and effects

An advanced predicate is a bounded tree using `AND`, `OR` and `NOT` over the supported comparisons. A profile is limited to:

- 200 questions inherited from the form contract;
- 100 scoring contributions;
- 100 advanced rules;
- nesting depth 8;
- 20 children per logical node; and
- 20 literal comparison values per condition.

An advanced rule may have one of four effects:

- `CONTRIBUTION` — adds a weighted score contribution;
- `FLOOR` — sets the least adverse score permitted when the predicate matches;
- `CAP` — sets the greatest adverse score permitted when the predicate matches; or
- `DISQUALIFY` — marks the result Critical regardless of the numeric score.

Calculation order is fixed:

1. resolve applicable visible questions;
2. evaluate field and advanced contributions;
3. calculate the weighted raw score and coverage;
4. convert the raw score to adverse direction;
5. apply the greatest matching floor and lowest matching cap;
6. reject the profile if any configured floor is greater than any configured cap, regardless of whether their predicates are expected to match together;
7. apply a matching disqualification; and
8. resolve the configured concern band.

Every matched and unmatched rule is retained in the explanation. Runtime evaluation never depends on map iteration order.

### 4.4 Bands

The profile defines non-overlapping, exhaustive adverse-score bands over 0–100. Stored severity codes are `LOW`, `MODERATE`, `HIGH` and `CRITICAL`; customer-facing labels depend on mode. For example, a compliance form may display **Below required level** for a Critical adverse band instead of implying that the raw compliance number is itself a risk score.

Activation fails if weights, bands, predicates, referenced question IDs, comparison types, effects or missing-answer behaviour are invalid. Profile changes create a new form revision and never reinterpret historical responses.

## 5. Immutable response score result

Completed response scoring runs on the server against the exact form revision carried by the distribution. The response transaction stores:

- response revision and submission IDs;
- form template ID and revision;
- score-profile checksum and version;
- scoring mode and direction;
- raw and adverse scores when calculable;
- concern band;
- weighted coverage;
- final, provisional or failed evaluation state;
- contribution and rule results;
- critical or disqualifying results;
- evaluator version and calculation time; and
- a bounded failure code when evaluation cannot complete.

The existing `compliance_score` field remains readable during migration. New reads use the generalized score result. Historical rows are not assigned invented score direction; they remain explicitly legacy until reconstructed from their stored form revision by an authorized migration job.

Invalid scoring configuration must be rejected before activation. If an unexpected evaluator failure occurs after a valid response is received, the response remains completed, the score is labelled unavailable, and an operational retry is scheduled. It never receives a favourable default.

The completed response, score result and `FORM_RESPONSE_SCORED` outbox event share one transaction. Reprocessing is idempotent by response revision and evaluator version.

## 6. Completed Responses workspace

### 6.1 Query contract

`GET /api/v1/forms/responses` becomes the bounded portfolio read for completed responses. It supports:

- legal entity from verified request identity;
- form template and revision;
- subject type and exact subject ID;
- scoring mode;
- raw score minimum and maximum;
- adverse score minimum and maximum;
- concern band;
- scored, unscored, provisional, final or evaluation-failed state;
- completion date range; and
- revision scope, defaulting to current response revisions with an explicit option to include superseded history; and
- keyset sort by completion time, raw score or adverse score.

The response contains safe summary rows only. Answers, protected recipient addresses and secure access data remain outside the list projection. Exact response detail continues through a scoped response endpoint. Restricted subjects and responses are removed by repository scope before pagination and counts.

Partial indexes cover current completed response revisions by legal entity and `(adverse_score, completed_at, id)`, `(raw_score, completed_at, id)` and `(form_template_id, completed_at, id)`. Queries are bounded and do not load a broad response population for browser-side sorting.

The existing per-distribution revision endpoint remains available for immutable audit history.

### 6.2 User experience

The Responses tab becomes a responsive table/list with:

- form and subject;
- completed time;
- score with its mode;
- concern label;
- coverage and score state; and
- one **Review response** action.

The default sort is **Needs attention first**, then newest. Filters use the shared SelectField, date and filter-chip contracts. Applied filters, result count and freshness remain visible. Mobile replaces the table with stacked response records rather than horizontal overflow.

Response detail shows the immutable answers, provenance, score explanation, critical rules, profile version, policy receipts and linked Matter. It distinguishes **Response received**, **Score calculated**, **Evidence reviewed**, **Matter created** and **Outcome verified**.

Required states are loading, live, explicitly empty, no score configured, provisional coverage, evaluation pending, evaluation failed with recovery, restricted, stale projection and linked-Matter replay.

## 7. Governed form response policies

### 7.1 Configuration and lifecycle

Forms gains a **Policies** view and each form detail links to policies that target it. A typed `FORM_RESPONSE_CREATE_MATTER` definition is stored one-to-one with a generic Automation Policy revision.

The definition contains:

- purpose and exact legal-entity scope;
- form template and optional exact revision;
- eligible subject types and required canonical Program-context resolution;
- response state, coverage, mode, band and score predicates;
- whether only the current response revision is eligible;
- Matter type, priority, bounded title/summary variables and requested handling;
- deduplication scope;
- blast-radius limits per run and per day;
- shadow or enforced rollout;
- effective and expiry dates;
- kill-switch and compensation instructions; and
- outcome-check contract.

Lifecycle states are Draft, Pending approval, Approved, Active, Suspended, Expired and Retired. The maker cannot approve their own revision. Activation rechecks the form revision, legal-entity scope, subject resolver, Matter route, service actor, limits and current authority configuration.

### 7.2 Simulation and impact preview

Before submission for approval, **Simulate policy** runs against a bounded, authorized historical response population. It reports:

- population and time range checked;
- matched, excluded, indeterminate and restricted counts;
- existing Matters that would be linked rather than duplicated;
- proposed Matter types and priorities;
- blast-radius blocks;
- subject-resolution or authority failures; and
- the score/profile versions represented.

Simulation never creates a Matter or task. Approval shows the saved simulation receipt and a current impact preview. The receipt records the normalized policy checksum, query window and response source high-water mark. Activation requires the same policy checksum and a preview created within the preceding 24 hours; it reruns the bounded impact query and requires a new approval preview when the eligible count, deduplicated-subject count or blast-radius result has changed.

### 7.3 Execution and deduplication

The policy executor consumes `FORM_RESPONSE_SCORED` after the response transaction commits. It loads the policy revision effective at the response completion time, revalidates tenant and legal-entity scope, evaluates the typed predicate, and appends one execution receipt with `NOT_MATCHED`, `MATCHED_SHADOW`, `SUPPRESSED`, `ACTION_PENDING`, `ACTION_APPLIED` or `ACTION_FAILED`.

Enforced execution uses a verified automation service principal. Human actor IDs are never accepted from the event payload or browser.

The default deduplication key is:

```text
form-response-policy:{policy-code}:{subject-type}:{subject-id}:{active-adverse-episode}
```

The first qualifying response in an episode creates a canonical Matter through the existing trigger service. Later qualifying responses link their score receipts and update the Matter's observed facts without creating another Matter. Once the Matter reaches verified closure, a later qualifying response starts a new episode and may create a new Matter.

If the subject cannot resolve to an authorized canonical Program/Matter context, the policy cannot activate for that subject type. A later runtime loss of route or scope records `ACTION_FAILED`, creates an operational exception for the policy owner, and retries safely; it never creates an unscoped Matter.

### 7.4 Compensation and rollback

Suspension and the kill switch stop new executions. Rollback activates a prior approved policy revision after impact preview and maker-checker authorization. Material records are not deleted. If a policy created an inappropriate Matter, compensation records the policy execution and routes the Matter for authorized review; it does not silently close or erase it.

## 8. Authority and security

- Form authors may draft scoring profiles only when the form route permits it.
- A distinct authorized checker approves the complete form revision, including scoring.
- Response list and detail reads require response authority and verified tenant/legal-entity scope.
- Policy creation, simulation, approval, activation, suspension and rollback use current configuration authority routes.
- Matter creation re-evaluates the current route and subject scope at execution time.
- The automation service principal, policy revision, maker, checker, score result and resulting Matter are recorded in the audit chain.
- Restricted records are filtered in the API/repository, not hidden in React.
- Secure invitation selectors, OTP material and protected recipient addresses never enter score or policy events.

## 9. API surface

The implementation adds or extends these authenticated routes:

```text
POST /api/v1/config/form-templates/{id}/score-preview
GET  /api/v1/forms/responses
GET  /api/v1/forms/responses/{response_revision_id}

GET  /api/v1/config/form-response-policies
POST /api/v1/config/form-response-policies
GET  /api/v1/config/form-response-policies/{id}
POST /api/v1/config/form-response-policies/{id}/simulate
POST /api/v1/config/form-response-policies/{id}/submit
POST /api/v1/config/form-response-policies/{id}/approve
POST /api/v1/config/form-response-policies/{id}/activate
POST /api/v1/config/form-response-policies/{id}/suspend
POST /api/v1/config/form-response-policies/{id}/rollback
```

Material commands ignore actor, maker, checker, approver and service-principal IDs supplied by request bodies. Cursor, sort and filter values are bounded and validated server-side.

## 10. Data and transaction boundaries

The form template revision retains the normalized score profile and checksum. Response revision storage gains generalized score columns plus a bounded JSON score explanation. Existing compliance fields remain compatible during migration.

A typed form-response policy definition references one `automation_policies` revision. Policy simulation receipts and execution receipts are append-only. The execution table has a unique key on tenant, policy revision and response revision, while Matter application retains the adverse-episode trigger key.

The response submission transaction includes the authoritative submission, response revision, score result, append-only response event, outbox event and required retry job. Policy execution is a separate inbox-idempotent material transaction containing the execution receipt, canonical Matter trigger/application, Matter event, outbox event and maintenance job. A derived projection failure after either transaction commits cannot turn the committed command into a reported failure.

Policy and response projections expose source high-water marks and freshness. High-volume response queries use keyset pagination, bounded filters and indexed database ordering. No API returns hard-coded score rows or demo metrics; reference environments use ordinary seeded templates, responses, policies and Matters marked as sample data.

## 11. Error and degraded behaviour

- A scoring preview validation error identifies the exact contribution or rule and makes no stored change.
- A score calculation failure preserves the completed response and exposes **Score unavailable** with a retry state.
- A policy service outage does not fail an already committed response.
- A policy action failure does not claim that a Matter was created.
- Replayed events return the existing receipt and Matter linkage.
- A suspended, expired or retired policy records why no action was attempted.
- Blast-radius exhaustion suppresses the action, records the affected response and alerts the policy owner.
- Authority, tenant, legal-entity or subject-route failure fails closed.
- Stale response projections remain labelled with their source high-water mark; detail reads can recover from authoritative state.

## 12. Testing and UI proof

### 12.1 Domain and service tests

- risk and compliance score direction;
- weighted choice, multi-choice, numeric and date predicates;
- nested logical rules and boundary limits;
- hidden, optional and required missing-answer behaviour;
- floors, caps, disqualification and invalid conflicts;
- exhaustive bands and boundary scores 0 and 100;
- deterministic explanation ordering and evaluator replay;
- immutable profile and response version reconstruction;
- response completion, score and outbox transaction rollback;
- policy maker-checker, effective dates, expiry, suspension and rollback;
- simulation counts, restricted-record exclusion and changed-population preview;
- policy non-match, shadow match, enforced match, deduplication and new adverse episode;
- blast-radius suppression, route loss, retry and idempotent Matter application; and
- cross-tenant, cross-entity and actor-field rejection.

### 12.2 Repository and performance tests

Tagged PostgreSQL tests cover migrations, transaction atomicity, legal-entity isolation, score indexes, stable keyset pagination and query plans across at least 10,000 completed responses and 1,000 active policies. Worker tests cover lease expiry, duplicate delivery, bounded batches and recovery after Matter creation conflicts.

### 12.3 Browser and accessibility tests

Rendered fixtures cover score authoring, advanced-rule validation, score preview, completed-response filtering, mobile stacked results, score-unavailable recovery, policy simulation, approval impact, shadow mode, enforced Matter linkage and restricted/empty states.

Verification runs at 320, 390, 768, 1280 and 1440 pixels in light and dark modes. The audit confirms keyboard operation, focus order, accessible names, non-colour score labels, no dropdown layout shift, no sidebar overlap and no document-level horizontal overflow. Customer-visible copy and affected workflow regressions pass before completion.

## 13. Delivery slices

1. **Score contract and evaluator** — normalized profile, advanced predicates, immutable generalized score result and compatibility migration.
2. **Completed response portfolio** — indexed API, score filters/sorts, detail explanation and responsive Forms UI.
3. **Policy governance** — typed policy configuration, simulation, impact preview, maker-checker lifecycle and policy UI.
4. **Matter automation** — scored-response outbox, executor receipts, authority revalidation, deduplicated adverse episodes and operational recovery.
5. **Proof and release hardening** — load tests, rendered states, copy gate, PostgreSQL integration, worker recovery and seeded reference journey.

Each slice must leave unavailable actions absent or explicitly disabled with a reason. No slice may substitute static browser data for an incomplete API.

## 14. Acceptance journey

An authorized Form Owner creates a vendor certification assessment, gives the compliance sections and questions approved weights, adds a cross-field disqualification for an expired required certification, previews several answer sets and submits the revision. A distinct checker reviews the score explanation and activates it.

The GRC Administrator creates a policy stating that a final response with full required coverage and either a compliance score below 65 or a disqualifying certification result should create a High-priority vendor control-gap Matter. Simulation shows the historical population, proposed actions, deduplicated subjects and blast-radius result. A distinct checker approves and activates the policy in shadow mode, reviews matched receipts, and then authorizes enforced rollout.

When the vendor completes the form, the response is stored with its immutable raw score, adverse score, band, coverage, profile version and rule explanation. The Responses workspace can place it first under **Needs attention first** and filter it by form, Critical concern and completion date. The active policy records its decision and creates one linked Matter under the current authority route. Replaying the event or receiving another poor revision during the same open episode does not create a duplicate. Review, remediation, outcome verification and closure remain separate governed actions.
