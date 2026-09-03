# Program collection renewal and tabs issue log

**Status:** In review — production external delivery remains a release dependency

**Opened:** 2026-09-03

**Design:** `../specs/2026-09-03-program-collection-renewal-and-tabs-design.md`

**Implementation plan:** `../plans/2026-09-03-program-collection-renewal-and-tabs.md`

## Outcome tracker

| ID | Work item | Status | Depends on | Acceptance evidence |
| --- | --- | --- | --- | --- |
| PCR-01 | Versioned form collection-policy contract | Complete | Approved written spec | `df8201a1`; policy tests cover required expiry, 30-day/3-reminder defaults, ranges, form/source separation and governed revision behavior |
| PCR-02 | Durable schema, lineage and bounded indexes | Complete | PCR-01 | `c906fa93`, `40c35e7b`, `15220bd8`; schema/repository tests cover policy/cycle persistence, shared capture-origin reuse, tenant-scoped origin uniqueness, leases and bounded reads |
| PCR-03 | Submission-driven collection-cycle lifecycle | Complete | PCR-01, PCR-02 | `f26e0238`; consumer tests cover inbox replay, expiry/month-end calculation, one current cycle and safe event handling |
| PCR-04 | Attributed renewal prefill | Complete | PCR-02, PCR-03 | `55e2e874`, `85902b60`; service/browser tests cover scalar provenance, governed-source priority and file/signature exclusion |
| PCR-05 | Leased renewal worker | Complete | PCR-02, PCR-03, PCR-04 | `faad72f6`; worker tests cover origin reuse, lease fencing, retries, stop conditions and internal assignment receipt |
| PCR-06 | Reminder assignment and delivery | Blocked by adapter confirmation | PCR-05 | Internal queue receipt and configured external provider receipt; 1–5 deterministic reminders; truthful terminal failure |
| PCR-07 | Program collection-summary read model and API | Complete | PCR-02, PCR-03 | `ab5741aa`; actor-scoped API tests cover latest submission, safe respondent label, expiry, reminder progress, delivery and freshness |
| PCR-08 | Accessible tabbed Program workspace | Complete | Written UI brief | `40471f91`; Vitest/axe plus rendered evidence cover six sections, one panel, URL state, keyboard tabs and selector replacement |
| PCR-09 | Unified Monitoring collection experience | Complete | PCR-01, PCR-07, PCR-08 | `b0338914`, `17250882`; workflow tests/fixtures cover schedule setup, unified records, last submission and all currency states |
| PCR-10 | Full acceptance, copy, documentation and recovery proof | Complete | PCR-01…PCR-09 | `17250882`, `15220bd8`, `6fe5abef`; full Go, vet, 921 frontend tests, type/build/copy and 138/138 rendered states pass; live PostgreSQL cases explicitly skipped without `TEST_DATABASE_URL` |

Status values are `Planning`, `Ready for planning`, `In progress`, `Blocked`, `In review` and `Complete`. An item becomes complete only when its listed acceptance evidence has been run on the final change and recorded below.

## Issue details

### PCR-01 — Versioned form collection-policy contract

**Scope**

- Add `validity_months`, `renewal_window_days` and `reminder_count` to form Monitoring Check revisions.
- Require an explicit validity period for new form checks.
- Default renewal window to 30 days and reminders to 3.
- Accept 1–120 months, 1–90 days and 1–5 reminders, subject to the cross-field shortest-validity rule.
- Keep collection-policy fields absent for source checks.
- Add a governed policy-update command for existing checks.
- Preserve existing active checks without inventing a policy.

**Non-goals**

- No form-template-global schedule.
- No arbitrary recurrence expression.
- No silent update of an active check.

### PCR-02 — Durable schema, lineage and bounded indexes

**Scope**

- Add nullable policy columns and constraints to `monitoring_checks`.
- Add generic immutable origin and predecessor fields to Evidence Requests.
- Add a tenant-scoped unique origin index.
- Add a focused monitoring collection-cycle table with leases, retry state and due-action index.
- Update durable-schema ownership documentation.
- Provide reversible migration behavior that retains every current request and check.

**Security constraints**

- No raw external address, answer, artifact content or invitation token in monitoring tables, events, jobs or logs.
- Tenant and Program scope must lead every operational lookup.

### PCR-03 — Submission-driven collection-cycle lifecycle

**Scope**

- Consume the existing `EvidenceResponseSubmitted` event with inbox deduplication.
- Load the exact request and submission through bounded reads.
- Calculate expiry and renewal opening from the exact approved policy.
- Close the predecessor cycle and create one next cycle.
- Retain exact policy/check version and submission lineage.

**Edge cases**

- 29 February and month-end timestamps.
- Duplicate event delivery.
- A check paused or retired between request creation and submission.
- A superseded check revision with no approved replacement.

### PCR-04 — Attributed renewal prefill

**Scope**

- Map predecessor scalar answers only when field IDs and compatible types still match.
- Store field-level predecessor submission provenance.
- Label prefill and respondent corrections distinctly in capture and review.
- Leave new required fields empty.
- Keep predecessor file/signature artifacts reachable to authorized review without copying them into the new response.

**Stop conditions**

- Incompatible or retired form with no current approved revision.
- Missing predecessor submission.
- Cross-tenant or wrong-subject lineage.

### PCR-05 — Leased renewal worker

**Scope**

- Add one bounded work class to the existing worker process.
- Claim due collection cycles with lease fencing and retry budgets.
- Re-read current Program, check, form, route and authority state.
- Create or reuse the exact origin-keyed successor request.
- Record the successor link and next reminder action transactionally with the cycle update.
- Cancel future work on submission, cancellation, pause or retirement.

**Recovery cases**

- Crash before request creation.
- Crash after request creation but before cycle update.
- Reclaimed lease and stale worker completion.
- Terminal failure requiring operator attention.

### PCR-06 — Reminder assignment and delivery

**Scope**

- Assign internal successor requests to the current eligible principal and verify queue visibility.
- Resolve external destinations from an opaque current contact reference.
- Issue a new request-scoped invitation for each external successor request.
- Record provider reference, status and timestamp without raw address or token.
- Distribute configured reminders evenly and stop once submitted or no longer relevant.

**Current blocker**

The repository has recipient hashing, invitations and outbox delivery foundations but no confirmed production external delivery adapter or generic durable external contact resolver in this branch. Planning must identify the exact executable adapter boundary. The UI and API must fail closed rather than claim automatic external delivery when that boundary is absent.

**No-bloat resolution rule**

Add only the provider-neutral resolver/delivery interface needed by this workflow. Do not build contact management, an email platform or a notification center inside Monitoring.

### PCR-07 — Program collection-summary read model and API

**Scope**

- Add one Program-bounded summary query keyed by Monitoring Check.
- Return latest request/submission, safe respondent label, expiry, renewal opening, currency state, active deadline, reminder progress, delivery state and freshness.
- Use indexed exact lineage rather than loading broad request populations.
- Return partial/unavailable state without hiding other Program content.

### PCR-08 — Accessible tabbed Program workspace

**Scope**

- Preserve the current Program detail as the before-state baseline.
- Add Overview, Requirements & controls, Monitoring, Evidence & results, Issues & actions and History.
- Render one labelled panel at a time.
- Store exact section in the URL.
- Support the keyboard tabs pattern on desktop/tablet.
- Replace tabs with a labelled native selector on mobile and at the 200% zoom breakpoint.
- Keep Program identity and calculated state visible across sections.

**Non-goals**

- No new global navigation.
- No nested dashboard.
- No horizontally scrolling mobile tab strip.

### PCR-09 — Unified Monitoring collection experience

**Scope**

- Replace duplicate form/check cards with one Program-linked collection record.
- Add the schedule step after approved-form selection.
- Show questions, status, validity, renewal window, reminder count, latest respondent submission, expiry and current request state.
- Show **Response expiry is not configured** for migrated checks without policy.
- Retain connected-data checks with source freshness language.
- Show policy read-only when starting an initial collection.
- Add current, renewal-due, potentially-expired, awaiting-response and blocked fixtures.

### PCR-10 — Full acceptance, copy, documentation and recovery proof

**Scope**

- Run all affected Go and React tests plus full required repository gates.
- Run copy-quality regression and review the entire affected workflow.
- Render desktop 1440×900, tablet 1024×768, mobile 390×844, 320×800 and 200% zoom proxy.
- Cover light/dark where supported, focus, long labels, no-policy, current, renewal-due, expired and delivery-failure states.
- Fix the highest-impact rendered defect and rerun the affected evidence.
- Update README, implementation ledger, architecture/schema ownership, API contract and rendered-evidence manifest.

## Acceptance matrix

| Scenario | Expected result | Planned evidence |
| --- | --- | --- |
| New form check | Expiry is required; 30 days and 3 reminders are offered | PASS — policy service/component tests |
| Existing form check | Remains active; shows no invented policy | PASS — migration and sample UI state |
| Latest response | Shows exact submission time, permitted respondent label and expiry | PASS — read-model and CollectionRecord tests |
| Renewal opens | Creates one successor request before expiry | PASS — worker unit and PostgreSQL-build tests; live PostgreSQL integration pending environment |
| Worker replay | Reuses the origin-keyed request | PASS — origin/retry tests |
| Previous answers | Scalar values are attributed; files/signatures are not copied | PASS — evidence and CapturePanel tests |
| Reminder count | Sends exactly 1–5 configured reminders at deterministic times | PASS — policy/scheduler tests |
| Submitted successor | Stops remaining reminders and schedules from new submission | PASS — consumer/worker tests |
| Check paused/retired | Cancels future renewal actions and retains history | PASS — lifecycle/worker tests |
| Recipient changed | Re-resolves current eligible recipient or blocks safely | PASS — recipient/worker tests |
| External delivery absent | Shows renewal blocked or link created but not sent | PASS — fail-closed API/UI recovery; real adapter remains PCR-06 blocker |
| Program tabs | One section visible; keyboard and URL behavior work | PASS — Vitest/axe and browser render |
| Mobile Program | Native section selector replaces tabs with no overflow | PASS — 390px, 320px and 200% reflow renders |
| Partial Monitoring failure | Other Program sections remain usable | PASS — component isolation tests |

## Risk register

| ID | Risk | Mitigation | Status |
| --- | --- | --- | --- |
| R-01 | Expiry could be mistaken for a compliance conclusion | Use collection-attention states and keep assessment/status paths separate | Controlled by design |
| R-02 | Retry creates duplicate requests or invitations | Unique request origin, exact lookup, inbox dedupe and provider idempotency | Controlled for requests; external adapter remains blocked |
| R-03 | Prefill appears newly attested | Field provenance labels, explicit review/submit and no copied file/signature answer | Controlled by PCR-04/09 tests |
| R-04 | External destination cannot be safely retained/resolved | Opaque contact reference, dispatch-time resolution and fail-closed activation | Open; PCR-06 blocker |
| R-05 | Existing checks gain an arbitrary validity policy | Nullable migration and explicit governed update | Controlled by design |
| R-06 | Tabs hide important current state | Keep record identity/state visible and default Overview to next action | Controlled by PCR-08 renders |
| R-07 | Mobile tabs overflow or become inaccessible | Native selector replacement and rendered 320px/200% evidence | Controlled by PCR-08/10 renders |
| R-08 | Reminder load produces unbounded work | Maximum five reminders, due index, bounded claims, leases and retry budget | Controlled by PCR-01/05 tests |

## Decision log

| Date | Decision | Reason |
| --- | --- | --- |
| 2026-09-03 | Store collection policy on Program-linked form Monitoring Check | Reusable forms may need different schedules per Program |
| 2026-09-03 | Default to 30-day renewal window and 3 reminders; allow 1–5 | User-approved default and bounded burden |
| 2026-09-03 | Open renewal before expiry | Gives the respondent time to prevent a stale-response gap |
| 2026-09-03 | Do not count initial renewal request as a reminder | Keeps configured reminder count understandable |
| 2026-09-03 | Always create a new Evidence Request | Preserves immutable submissions and exact lineage |
| 2026-09-03 | Show latest respondent and submission time | User explicitly requested last-submitted visibility |
| 2026-09-03 | Use six Program sections with mobile selector replacement | Removes endless scrolling while preserving responsive usability |
| 2026-09-03 | Do not invent policy for existing active checks | Avoids unsupported automated action and status claims |
| 2026-09-03 | External renewal requires an opaque resolvable contact route and delivery receipt | Raw destinations do not belong in monitoring jobs/events and unsent work cannot be labelled sent |

## Activity log

| Date | Entry |
| --- | --- |
| 2026-09-03 | Reviewed root product/interface contracts, documentation map, monitoring design, Program operational design, capture/reminder semantics, application architecture, governance runtime and rendered-evidence gate. |
| 2026-09-03 | Inspected current Monitoring form/check models, APIs, Program detail composition, collection creation and supplied before-state screenshot. |
| 2026-09-03 | User approved Program-linked policy, pre-expiry renewal, 30-day/3-reminder defaults, six Program sections, unified collection display and latest respondent submission details. |
| 2026-09-03 | Written design and issue tracker created. No production code or migration changed. |
| 2026-09-03 | User approved the written specification. A test-first implementation plan was added and linked; production code remains unchanged by the planning step. |
| 2026-09-03 | Inline execution started on `codex/vendor-management`. Baseline passed `go test ./... -count=1` and 125 frontend tests across 28 files. Node 22 is below the repository's Node 24 requirement, but the baseline completed; dependency upgrades are outside this feature. |
| 2026-09-03 | Preserved the unchanged supplied Monitoring screenshot and added the approved compact UI decision brief. The pre-existing submitted-evidence diff spans 10 overlapping files with 284 insertions and 25 deletions; it remains user-owned and must not be reset or staged wholesale. |
| 2026-09-03 | Implemented the policy, cycle schema, origin/predecessor lineage, submission consumer, renewal worker, bounded reminders, fail-closed delivery state and Program collection-summary API in `df8201a1` through `ab5741aa`. |
| 2026-09-03 | Implemented unified collection records, expiry/reminder/last-respondent display, six URL-addressable Program sections and attributed previous-response capture in `b0338914`, `40471f91` and `85902b60`. |
| 2026-09-03 | Added deterministic sample states and 51-state rendered acceptance in `17250882`. The first render review found low-contrast light-theme state colors; semantic theme tokens were applied and the full render gate passed. |
| 2026-09-03 | Final verification passed all memory/PostgreSQL-build Go packages, `go vet ./...`, 33 frontend files/150 tests, typecheck, normal/static builds and the 51-state UI review. PostgreSQL integration-tag packages passed, with live-database cases explicitly skipped because `TEST_DATABASE_URL` was absent. |
| 2026-09-03 | Replayed the feature onto the updated `main` line, consolidated its schema into migration `000076_program_collection_renewal`, and reused the existing text-based capture origin/version contract in `15220bd8`. |
| 2026-09-03 | Updated exact-target and rendered-evidence automation to navigate the intentional **Issues & actions** Program section. No product composition was moved back to the default Overview section. |
| 2026-09-03 | Post-integration verification passed 144 frontend files/921 tests and the complete 138-state managed UI review. Direct inspection covered desktop current, dark blocked, 390px selector, 320px long-content and 200% reflow states. |

## Verification log

- Baseline backend: `go test ./... -count=1` — PASS on 2026-09-03.
- Baseline frontend: `npm test -- --run` — PASS, 28 files and 125 tests on 2026-09-03.
- Environment note: `npm install` reported Node v22.16.0 against the package requirement `>=24.0.0` and one high-severity audit item. No dependency remediation is included in this scoped change.
- Focused collection backend/API tests: PASS for memory and PostgreSQL build variants; live PostgreSQL integration was skipped because `TEST_DATABASE_URL` was not set.
- Focused frontend: PASS for collection policy/records, Monitoring setup, Program sections, route state, accessibility, capture provenance, copy quality, typecheck and production build.
- Rendered UI: `npm run review:ui` — PASS, 51/51 flow records and screenshots, 8 accessibility route states, no browser errors or horizontal overflow, 133 KiB JavaScript gzip and 21 KiB CSS gzip.
- External delivery: BLOCKED for production acceptance. Deterministic internal/failure adapters were exercised; no real provider adapter or receipt was available.
- Final backend: `go test ./... -count=1`, `go test -tags postgres ./... -count=1`, `go test -tags "postgres postgresintegration" ./... -count=1` and `go vet ./...` — PASS. Live PostgreSQL tests reported `TEST_DATABASE_URL is not configured` and were SKIPPED.
- Post-integration frontend: `npm test` — PASS, 144 files and 921 tests; `npm run typecheck`, `npm run build`, `npm run check:runtime-truth` and `npm run check:ui-contracts` — PASS. The copy-quality regression is included in the full test run.
- Post-integration rendered acceptance: `npm run review:ui` — PASS, 138/138 flow records, 138/138 screenshots, 72/72 governed Forms capabilities, 8 accessibility/touch route states and bundle budgets. Direct PNG inspection confirmed current/renewal/expired/awaiting/blocked collection states, last-submission attribution, mobile selector replacement and long-content reflow.
