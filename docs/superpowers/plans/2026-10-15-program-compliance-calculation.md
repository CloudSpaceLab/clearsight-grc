# Program compliance calculation — implementation plan

Date: 2026-10-15  
Tracking: [#433](https://github.com/CloudSpaceLab/clearsight-grc/issues/433)  
Status: design/implementation plan; no runtime code changed  
Parent/adjacent work: #265, #268, #274, #344, #346, #140

## 1. Outcome and non-goals

Allow each Program to configure, simulate and approve a **versioned compliance calculation**. Evaluate current eligible requirements/controls, assessed internal/vendor/third-party forms, due evidence and linked issues against that policy, then expose (a) a calculated 0–100 percentage **when defensible**, (b) weighted assessment coverage/quality, (c) existing categorical Program compliance state, (d) critical blockers and (e) an exact contribution breakdown. Support organizational filters, as-of history, Insights and governed reports from the *same* calculation. Do not imply a numerical score is a legal determination.

Do **not** add another form evaluator, generic rules interpreter, metric engine, scheduling infrastructure, policy administration shell, source registry, vendor questionnaire store, report generator, chart library or dashboard. Do not duplicate requirements or create a synthetic list of compliance items from forms. No implicit write or remediation on a score calculation. Form submission does not approve evidence, accept remediation, or close a Matter.

### Current implementation to reuse (verified in main)

| Owner | Existing contract / important limitation |
|---|---|
| `internal/continuity/state.go` and `current_state_refresh.go` | Deterministic Program status with 10 dimensions, reasons, `OpenMatterCount`, deduplicated projection jobs and event/outbox recording. No percent; `chooseOverallState` is a priority ordering, **not** a weighted average. |
| `internal/continuity/requirement_coverage.go` | Requirement → applicable control → current sufficient evidence chain. MUST reconcile its treatment of partially applicable requirements and retired links against the primary state evaluator before using it to count eligible obligations. |
| `internal/continuity/program_state_visibility.go` and `summaries_postgres.go` | Actor-facing Program state recomputes visible Matter impact. A global weighted score cannot bypass these rules or indirectly expose restricted Matter penalties. |
| `internal/formcontract/advanced_scoring.go` and `internal/evidence/response_assessment.go` | Weighted response scores, explicit direction, adverse bands, provisional/final bank assessment and immutable form/profile/response versions. A form score is an observation, not a compliance finding. |
| `internal/evidence/vendor_forms.go` | Current submitted vendor forms, bank-assessed scores, review state and gaps. Existing `web/src/components/ProgramVendorFollowUp.tsx` uses capped client pagination: do NOT use it for numeric Program aggregation. |
| Existing canonical Vendor Assessment and relationship links | Relationship-scoped assessments and conclusions; Program↔relationship links. Need a server-side, exact, governed expected-vendor/assessment population — completed distributions alone are not the denominator. |
| `internal/continuity` Matters and links | Canonical open/closed/verified issue state, priority and requirement/control link. Use for penalties only through an explicit, deduplicated Program scoring rule. |
| `internal/metricview` | Canonical definition/observation, coverage/freshness, snapshot membership, bounded trends and exact member drills. Current `MetricUnit` implementation seen in `model.go` supports count/money; percentage/ratio delivery must coordinate with planned shared native-measure extension in #344/#346, not preempt it. |
| `internal/autonomy`, Program authority/commands, reporting | Existing maker-checker/approval patterns, audit/events, runtime projection work and report generation. Configuration is *not* an autonomous action, so do not repurpose `AutomationPolicy.ActionClass` as a calculator. |

## 2. Definition and denominator

**Scoring unit** is a current, uniquely identified, eligible obligation/assessment or independently scored exception — never an arbitrary row. Required identities:
- Requirement: `(tenant, entity, program, requirement_id, effective interval)`.
- Vendor obligation: `(tenant, entity, program, relationship_id, approved obligation/form reference, assessment period)`; current approved revision, one assessed outcome per expected obligation per period. A vendor with three requested forms contributes three *only if policy declared three distinct obligations*. A resend/amendment does not increase the population.
- Evidence: the configured Evidence Contract/requirement chain; the same missing artifact must not be counted independently as a requirement failure and an extra missing-evidence penalty unless the policy explicitly declares distinct control objectives.
- Matter: distinct `(tenant, program, matter_id)`, current open state and exact Requirement/Control/response origin. One Matter linked to several requirements may affect the mapped units but must not generate a second independent 'open issue' penalty for the same underlying failure.
- Optional unmatched issue category: only Matters without a scored originating obligation/response, with one severity-adjusted penalty per distinct eligible Matter.

**Policy categories:** requirements/controls, required internal/third-party assessed responses, evidence currency, unmatched issues. Configurable nonnegative category weights with total exactly 100 for applicable configured categories; within categories use bounded item/default weights or priority-based penalties. The editor exposes category weights first, optional overrides for *specific existing* requirements/forms/issue priority. Never allow arbitrary executable expressions or freeform API field selectors.

Applicability is an authorized effective-time decision. Exclude only approved not-applicable objects with a recorded reason. Potential/partial applicability, missing link, expired source and unassessed required response remain separate quality states, not silently excluded. Effective scope and Program period are explicit. If no eligible policy items exist, return **Not assessed**, not 100%.

### Outcome contract

Return typed fields:
- `status`: NOT_CONFIGURED | NOT_ASSESSED | PROVISIONAL | FINAL | STALE | UNAVAILABLE;
- `percent`: nullable decimal, 0–100, precision fixed and rounded only for presentation;
- `coverage_percent`: known assessed weighted mass / total eligible weighted mass, independent of compliance percentage;
- `weighted_earned`, `weighted_assessed`, `weighted_eligible` (decimal for exact reconstruction);
- scoped `breakdown[]` with category label, weight, score/coverage, known/unknown/excluded counts, and contribution/drill reference;
- `blocking_conditions[]` for critical failing requirement, mandatory unknown, expired mandatory evidence, overdue action or unsatisfied minimum coverage (depending on *approved* policy), never automatically changed into a numeric favourable score;
- Program status/dimensions from the existing evaluator, policy ID/revision/checksum, source versions/high-water, entity/organization scope, assessment period, calculated time and integrity/freshness state.

For scored known units `s_i∈[0,100]` and approved positive effective weights `w_i`:

```
eligible_mass   = Σ w_i for all applicable expected units
assessed_mass   = Σ w_i for FINAL, current, usable units
earned_mass     = Σ (w_i × s_i / 100) for these assessed units
coverage_pct    = 100 × assessed_mass / eligible_mass
assessed_pct    = 100 × earned_mass / assessed_mass   # only when assessed_mass > 0
```

A composite category percentage is weighted by the **approved fixed category weight**; category weights must not be silently redistributed when that category's required data are missing. A truly not-applicable category is excluded only through the approved policy's explicit applicability rule, with normalization recorded. Score quality is PROVISIONAL below minimum coverage or while mandatory units are unknown. The API must not supply a FINAL percentage from a known-only subset that could be mistaken for full compliance. A policy may require full coverage to publish `FINAL`; represent partial estimates separately, expressly labelled.

**Missing data vs failed compliance:** required form not submitted, assessment still pending, stale/unknown origin, conflicting evidence and result-evaluator failure default to UNKNOWN and reduce weighted coverage, never success. Support the *explicitly approved* exception 'overdue required data counts as failure from its due date' only where the missing input is itself an obligation and the effective due date is proved. Distinguish missing response from a submitted negative answer and a bank assessor's failed conclusion. Do not apply missing penalty twice.

**Risk score direction:** use the existing response `Mode` and `Direction`; a risk score is not a compliance percentage. Convert only by a Program-policy-approved mapping from finalized `AdverseScore` to contribution, retaining origin direction and explanation. Prefer eligible bank-assessed final result over unreviewed automatic score if policy requires bank assessment. A vendor assessment conclusion and form score must not each create duplicate contribution for the same obligation.

**Issues:** map an open Matter to the scoped source requirement/form whenever deterministic links exist; its effect is the failed/unresolved status of that *single unit*. For unmatched issues, optional fixed-category penalty `max(0, 100 - min(cap, Σ severity_penalty(priority)))` with approved priority points and cap. A linked loss is not itself a compliance failure without an explicit policy/obligation link; preserve loss gross/recovered/net in the Loss Ledger rather than assigning a guessed compliance penalty.

**Critical gates:** an approved legal/regulatory critical requirement failure or policy disqualifier may cap the display score and set critical condition, but does not revise the raw earned points or Program legal conclusion. Always show raw vs capped if different, rule ID and affected record.

### Sanity cases that must pass

- All five vendor requests completed but only two assessed: denominator is the *five expected* applicable assessments, not two submissions; three are unknown coverage.
- One adverse vendor form triggers a linked Matter: no double penalty.
- Existing Matter closes after verified outcome: recompute affected input, but never assume requirement now passes without current supporting evidence.
- 10 known passing units + 90 unknown units: **no '100% compliant' headline**.
- Two Programs share a vendor: link/scope/period decide eligibility separately, without copying a response's score into an unrelated Program.
- A reviewer loses access to a restricted Matter: neither score nor breakdown leaks the hidden penalty; authorized enterprise aggregate presentation, if ever needed, requires a separately reviewed permission/policy.
- Changing a policy does not silently recalculate historical scores under a new version.

## 3. Minimal state/ownership changes

**Policy owner:** continuity Program. Define a typed `ProgramCalculationPolicy` value with revision/checksum, scope/period, category weights, item/priority rules, missing-data rules, result-basis requirements, score caps, minimum coverage, effective dates and maker/checker/approval state. Use existing Program version, event log and authority commands to prepare, preview, submit, approve, activate, supersede/revert; changing active policy never mutates past effective revisions. Prefer a narrowly bounded JSONB current/pending policy on the existing Program row plus append-only Program events over a new generic policy table. Verify current Program event replay and Postgres projection invariants at T0 before choosing storage. Do not grow a single unbounded JSON document with every historical revision; history belongs to existing events.

**Calculation owner:** continuity. Extract a pure deterministic evaluator consuming typed, already normalized *facts* and an immutable approved policy; use the same effective date, coverage and confidence semantics as Program state. Form scoring stays in formcontract and vendor bank review stays in evidence. Avoid direct cross-domain imports into continuity: introduce the smallest needed repository-level aggregate read interface coordinated by the existing Program projection worker; no HTTP loopbacks or browser-based folding.

**Snapshot:** extend existing `program_state_snapshots` with one bounded typed calculation summary, policy/source revision and eligibility/quality fields; retain old snapshots with nullable calculation and `NOT_CONFIGURED` for previously unconfigured Programs. Reuse `continuity_projection_jobs` queue/lease/dedupe and state-equivalence check, but expand freshness fingerprint beyond ProgramVersion to include vendor assessment/source/Matter high-water and policy revision. A new source event must invalidate/queue only affected Programs through existing relationship/contract lookups. Store enough immutable source identifiers and versioned member contributions to reproduce the exact snapshot; prefer extending existing `metricview` retained membership records if its supported target types, per-member weighted values and privacy controls are adequate. T0 must prove this. If impossible, authorize **one minimal typed, indexed source-contribution relation** rather than a large JSON blob or an unprovable historical drill. Never manufacture exact-history claims from mutable current records.

**No parallel calculation route:** initially add optional `calculation` to existing Program/detail/list projections, plus a scoped bounded `GET /api/v1/programs/{id}/calculation/contributions` *only if needed* for server-paginated exact breakdown. Existing Program operations/commands should expose policy editing and approvals, with precise authority. No new top-level navigation/independent UI.

**Metrics/reporting:** read the retained Program snapshot as source of truth. Once the coordinated native percent/ratio metric extension (#344 T4b/#346 S1) exists, register a Program compliance metric observation with exact (weighted) numerator/denominator and scope/period/quality/revision. Do not encode a 0–100 rate as a COUNT or average Program percentages for Group: roll up compatible, disjoint earned/eligible masses using policy compatibility and permissions, otherwise 'Not comparable'. Trends compare same scope, policy revision and same population rules or explicitly show a definition break. Reports reuse governed definition, runs, artifacts and download authorization.

## 4. UX contract (no extra dashboard)

Existing Program Overview: keep current state label and one dominant next action. Add **Compliance · 78%**, **Assessment coverage · 92%**, **Updated · date** on one compact row only when an approved policy and adequate data are present; mark provisional/unknown/critical clearly. Clicking opens a compact breakdown and existing filtered requirement, vendor, response or issue workspaces. Do not repeat sections' entire tables in Overview.

Existing Program setup / Requirements & controls: one **Calculation rules** action, using established sheet/panel, numeric/select inputs and maker-checker affordances. Provide a reusable default policy as an optional starter and allow explicit per-Program overrides. Editor has category weights, item/priority overrides, missing/overdue rules, critical gates and **Preview calculation**. Show changed score, coverage, affected records and approval impact before submit; approval does not imply all evidence is accepted.

Portfolio: one optional sortable Compliance column **only after** reliable server-side sorting/filtering by calculated snapshot is in place. Insight/report rendering reuses existing `MetricCard`, `RankedBarList`, `MetricTrend`, `DataTable` and report datasets; no new card family, dashboard, standalone visualization package, side navigation entry, chart engine or decorative copy.

Visible copy: 'Compliance', 'Coverage', 'Calculation rules', 'Missing evidence', 'Assessment pending', 'Critical requirement', 'Not assessed', 'Review calculation'. Do not say 'Compliant' on the basis of score alone.

## 5. Security, time and correctness

- Require signed-in actor, exact tenant + legal-entity + organization-scope and Program read authority. UI is not an authorization boundary. Same policy/score projection read must be safe for direct Program, lists, Insights and reports.
- **Critical design proof:** `program_state_visibility.go` explicitly recomputes open-Matter counts and status from visible Matters. A shared score influenced by restricted issue records must not be returned unchanged to unauthorized actors. Choose (a) actor-visible calculation with corresponding matching weighted member set, or (b) a privileged governance-aggregate permission and withheld numeric score for lesser readers; reject mixed aggregate/detail that leaks hidden cases. Regression test changes in visibility and revocation.
- Vendor identities/response answers are restricted. Do not expose respondent PII, artifacts or unreviewed answers through aggregate contribution read. Drill target revalidates current authorization even for retained snapshots.
- An as-of score is evaluated against exact effective policy, recorded applicability, source/response/Matter versions and assessment period; never mix a historical numerator with the current denominator. Inaccessible, unavailable or expired history is marked unavailable, not reconstructed by assumption.
- Late-arriving events, amendments, supersession, revoked access and policy activation must enqueue bounded idempotent recompute; retries do not duplicate snapshots/notifications. Derived failure cannot roll back a successful material command.
- Snapshot version + policy revision + source high-water must be in response and consistency token. `FINAL` only when dependencies, required facts and projection freshness are valid; stale value shown as stale/last-known with timestamp.

## 6. Implementation order (small PRs, hard acceptance gates)

### T0 — factual reuse and benchmark (no schema/UI)
1. Record the exact current Program status/requirement coverage predicates, including partially applicable, retired/superseded requirement/control links, fresh evidence, and status visibility; add failing alignment tests where mismatched.
2. Trace Program ↔ vendor-relationship ↔ expected request/assessment ↔ current finalized bank score, and Matter ↔ Program/requirement/form origin. Prove bounded scoped server lookup and exact expected denominator for one real fixture.
3. Inspect `metricview` source snapshot/retained-membership support for weighted/typed member target and permission filtering; determine minimal extension. Check current shared native ratio work in #344/#346 and open PRs before implementing.
4. Measure Program count, requirement/linked-vendor/finding cardinality, per-event affected-Program fanout, source/approval freshness and preferred retention; document target query plan.
**Gate:** signed-off truth table, shared-source keys, audit-ready score semantics, no parallel owner and non-empty benchmark fixtures.

### T1 — versioned rule and preview
1. Typed contract, validation (weights sum/zero, duplicate IDs, precedence, denominator, critical override, effective time, invalid result basis).
2. Program commands/events/replay/authority; maker-checker; preview against scoped typed facts; deterministic fixture for critical/missing/unknown and invalid-rules errors.
3. Existing Program setup sheet with direct labels and impact preview. No new admin workspace.
**Gate:** draft→preview→approval→activation→reversion; replay and past revisions unchanged; concurrent updates conflict safely.

### T2 — real scoped calculation
1. Pure evaluator and one backend reader for requirement coverage, vendor assessment population, evidence and linked Matters.
2. Deduplication, missingness, origin precedence, version identity and category calculation. Snapshot integration + targeted invalidation in current Program projection worker.
3. PostgreSQL integration proof on amendments, unsubmitted/overdue responses, reviewer changes, stale sources, critical/partial and verified issue closure.
**Gate:** manual independently calculated fixture = stored weighted numerator/denominator/coverage/status at same revision; no false green and no cross-entity leakage.

### T3 — drill parity and privacy
1. Retain exact contribution IDs/weights/revisions in existing metric membership architecture where feasible; otherwise justify minimal typed relation. Cursor pagination and bounded SQL.
2. Same-snapshot total = sum of accessible contribution values using same population, and explicit restriction/missing state where full population cannot be shown.
3. Actor-visible/privileged-boundary behavior; revoked permissions, restricted Matters and vendors, replay and as-of.
**Gate:** no row leaks, actor-visible count/score integrity, exact current/revision drill, no browser-side totals.

### T4 — in-context UI
1. Compact Program overview and one breakdown sheet or expandable section; link directly to existing Program requirements, vendor responses, evidence and issue records.
2. Shared form/select/metric components; properly identified calculating/stale/missing/restricted/error/N/A states; no empty placeholder percentage.
3. Rendered desktop 1440/1280, tablet 768, mobile 390/320, light/dark/keyboard and copy regression.
**Gate:** one-click explanation from each weighted contribution, one dominant next action, no extra dashboard.

### T5 — shared metric, Portfolio, Insights, Reports
1. Integrate only with the canonical percentage/ratio extension on the #344/#346 path; exact definition/source revisions and quality.
2. Server-side Program-list/Portfolio metric read; compatible numerator/denominator Group rollups where authorized, never arithmetic average of percentages.
3. Existing Insights and governed reports read the *same* snapshot. Definition changes break trend comparison unless valid re-baselining is approved.
**Gate:** Program/Portfolio/Insights/Reports parity for same scope/as-of/revisions; export provenance and download authority verified.

### T6 — operational proof and release
- Normal and edge-case PostgreSQL tests, partial/unavailable/cross-entity, worker replay/lease/race/recovery; benchmark baseline and p95 query/scoring time at expected production cardinality without unbounded scans; indexed keyset drill and bounded fanout.
- Live/hosted integration with a representative Program, at least two linked vendors (one missing, one assessed), a scored response causing an open Matter, a critical control gap, an expired evidence source, a repaired finding and policy revision. Manually verify weighted math and source links.
- Opt-in rollout: no calculation policy = no percentage; preserve current Program state everywhere. Rollback must preserve facts, revisions and event history. No automatic historic re-scoring; backfill only from verified reconstructable history.

## 7. Change budget and non-regression rules

Target changes concentrate in `internal/continuity` (typed policy/evaluation/projected snapshot), small scoped queries in `internal/evidence` and linkage repository, `internal/metricview` (when common percent support exists), and current Program surfaces. Reuse existing authority, events, outbox, queue, UI building blocks and reporting. Avoid standalone UI components; prefer reusing existing panel/sheet and test fixtures.

**No additional tables by default.** A proven performance/reconstruction/authorization need for one typed membership store is an explicit T0-reviewed exception. No raw-score JSON hidden in UI state. No open-ended configuration DSL. No all-vendor pagination from React. No blanket historical backfill or rework of form scoring. No merging a docs-only tranche with unrelated UI/seed changes.

Review implementation in small independent PRs at each gate; every PR includes affected backend/web tests and cross-domain contract verification, followed by hosted truth/visual proof where applicable.
