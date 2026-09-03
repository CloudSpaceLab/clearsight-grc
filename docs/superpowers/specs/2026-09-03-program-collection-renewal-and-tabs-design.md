# Program collection renewal and tabbed workspace design

**Status:** Approved design awaiting written-spec review

**Date:** 2026-09-03

**Primary users:** Program Owner, Control Owner, Evidence Reviewer, GRC Administrator, internal respondent and invited external respondent

**Related design:** `2026-08-17-monitoring-setup-and-risk-scoring-design.md`, `2026-08-25-program-work-operational-completeness-design.md`, `2026-08-26-vendor-due-diligence-data-collection-design.md`

## 1. Outcome

ClearSight will let an authorized user define how long a response collected by a Program form remains current, begin renewal before that response becomes potentially expired, and send a bounded number of reminders to the current eligible respondent. The Program workspace will use sections that show one coherent part of the record at a time rather than requiring the user to scroll through the full Program.

The Monitoring section will show the configured validity period, renewal window, reminder count, latest respondent submission, calculated expiry and current renewal state. A previous response remains immutable. Renewal creates a new Evidence Request whose prefilled values identify the exact previous submission and require confirmation before submission.

This work does not turn response age into a compliance conclusion. It changes attention and creates governed collection work only.

## 2. Decisions and alternatives

Three storage approaches were considered:

1. Store expiry and reminder values on the reusable Form Template. This is small, but forces every Program that uses a form to use the same operating schedule.
2. Store a versioned collection policy on the Program's form-based Monitoring Check. This keeps the form reusable and lets each Program apply the schedule its owners need.
3. Build a generic recurrence engine. This would support more scheduling shapes but would introduce unnecessary configuration and workflow machinery.

The selected approach is **a versioned policy on the form-based Monitoring Check**. The existing worker, outbox, Evidence Request, recipient, invitation and monitoring boundaries remain authoritative. No second questionnaire, task, scheduler, notification or approval system is introduced.

## 3. Product semantics

### 3.1 Collection policy

A form-based Monitoring Check has this additional policy:

```text
validity_months: required for new form checks
renewal_window_days: default 30
reminder_count: default 3, allowed 1..5
```

`validity_months` uses calendar months and must be explicitly selected. The initial release supports 1–120 months. `renewal_window_days` supports 1–90 days and must be shorter than the shortest possible validity interval for the selected month count. For a one-month validity period, the maximum renewal window is therefore 27 days. Validation is enforced in the API and explained next to the fields.

The initial renewal request is not a reminder. The configured reminders are distributed as evenly as possible between renewal opening and expiry, ordered by exact timestamps. Rounding is deterministic and never creates two reminders at the same time.

Existing active form checks receive no invented policy during migration. They remain active without automatic renewal and display **No expiry period set**. Setting a policy creates a new governed Monitoring Check revision and follows the existing maker-checker lifecycle.

Source-based checks continue using `freshness_minutes`; collection-policy fields are absent and rejected for source checks.

### 3.2 Response currency

The latest successfully committed submission for the current collection lineage is the response-age anchor. Its calculated expiry is:

```text
submission timestamp + validity_months using UTC calendar arithmetic
```

The UI formats timestamps in the viewer's locale while retaining the exact UTC value in the record. Month-end and leap-year behavior is deterministic: adding one calendar month to 31 January uses the last valid day in February, and adding twelve months to 29 February uses the last valid day in February of the following year.

Response currency states are:

- **No response submitted** — no committed submission exists for the collection lineage;
- **Current** — the latest response is earlier than the renewal opening time;
- **Renewal due** — the renewal window has opened and the prior response has not expired;
- **Response potentially expired** — the expiry timestamp has passed without a replacement submission;
- **Awaiting response** — a current successor request exists and has not been submitted;
- **Renewal blocked** — the policy, recipient route, authority, delivery or source request cannot be safely resolved.

These are collection-attention states. They do not alter legal applicability, evidence sufficiency, Program status or material risk without the existing assessment and authority paths.

### 3.3 Latest respondent activity

Every Program-linked collection form shows:

- latest submission timestamp;
- respondent display label where the actor may see it;
- privacy-safe recipient hint or **External respondent** otherwise;
- calculated expiry timestamp;
- current renewal state;
- current request deadline and reminder progress when a successor request exists.

Example:

```text
Last submitted 14 Aug 2026 at 10:32 by Vendor security contact
Expires 14 Aug 2027 · Renewal starts 15 Jul 2027 · 1 of 3 reminders sent
```

The UI never exposes a raw external address merely to explain the schedule.

## 4. Program information architecture

Opening a Program presents six sections:

1. **Overview** — calculated state, freshness, reasons, owner, deadline, one dominant next action, Program details and lifecycle controls;
2. **Requirements & controls** — requirements, applicability, safeguards and coverage;
3. **Monitoring** — collection forms, connected-data checks, schedules and operational actions;
4. **Evidence & results** — evidence expectations, latest monitoring results, coverage and expired-response warnings;
5. **Issues & actions** — linked issues, changes, owners, deadlines and next actions;
6. **History** — versioned changes and reconstruction.

Only the selected section is rendered as the primary content region. The Program header and current state remain visible so section changes do not remove record identity.

Desktop and tablet use an accessible tab list. Keyboard behavior follows the tabs pattern: Left/Right changes the focused tab, Home/End moves to the first/last tab, and selection exposes one labelled tab panel. Mobile and 200% zoom use a native **Program section** selector rather than a compressed or horizontally scrolling tab bar. The selected section is represented in the URL so an exact Program section can be opened and browser Back/Forward remains meaningful.

Loading, unavailable and permission-limited states remain local to the selected section when the rest of the Program is available. A failure in Monitoring does not hide Overview or Requirements & controls.

## 5. Monitoring experience

### 5.1 Setup

The reusable form lifecycle remains separate from a Program's operating schedule:

1. create and approve the reusable questions;
2. choose **Add to Program** from the approved form;
3. enter expiry in months, renewal window in days and reminder count;
4. create the form-based Monitoring Check as a draft;
5. submit and approve the check through the existing maker-checker path.

The schedule step contains one primary action, **Add collection to Program**. It previews the first renewal rule in plain language before saving. Known Program and form information is read-only context.

### 5.2 Records

Program-linked forms and their form-based Monitoring Checks are presented as one collection record rather than duplicate cards. A record summary contains:

```text
Collection form
Vendor security and privacy review
8 questions · Active · Valid for 12 months
Renewal starts 30 days before expiry · 3 reminders
Last submitted 14 Aug 2026 at 10:32 by Vendor security contact
Expires 14 Aug 2027
```

Unlinked Form Templates appear only in setup selection. Connected-data checks remain separate monitoring records and show source freshness rather than response expiry.

The initial collection panel retains reporting-period and request-deadline inputs. It shows the approved validity and reminder policy as read-only context. The request recipient may be an eligible internal principal or a resolvable external recipient route.

### 5.3 Renewal capture

A renewal request uses the current approved form revision at renewal time. If no compatible current revision exists, renewal enters **Renewal blocked** and creates operator work; it does not silently send the retired schema or discard fields.

Compatible fields with a value in the predecessor submission are prefilled. Each such value is labelled **From the response submitted on [date]**. When changed, provenance becomes **Changed by you**. Removed fields are not copied. New required fields are empty. File and signature answers are never copied as if they were newly supplied; their prior artifacts remain reachable to authorized reviewers as predecessor evidence.

The respondent must review and explicitly submit the successor request. Prefill never auto-submits, attests or verifies an answer.

## 6. Durable model and data flow

### 6.1 Monitoring Check revision

The form-check record gains nullable collection-policy fields:

```text
validity_months
renewal_window_days
reminder_count
```

The database constrains the basic ranges. Domain validation enforces the cross-field window rule and that the fields occur together only for `FORM` checks.

### 6.2 Request origin and lineage

Evidence Requests gain a generic, immutable origin reference:

```text
origin_type
origin_id
origin_sequence
predecessor_request_id
```

The tuple `(tenant_id, origin_type, origin_id, origin_sequence)` is unique. For recurring Program collection, `origin_type` is `MONITORING_COLLECTION`, `origin_id` is the stable Monitoring Check ID and `origin_sequence` increments for every request. Exact-origin lookup makes renewal request creation idempotent after a worker crash.

The initial manually started request is sequence 1. Every successor links its predecessor. Historical request schemas, submissions, results and artifacts remain immutable.

### 6.3 Collection cycle

A focused monitoring-owned collection-cycle record contains:

- tenant, Program and stable Monitoring Check IDs;
- exact policy/check revision that scheduled the cycle;
- current and predecessor request IDs;
- latest submission ID and timestamp;
- expiry and renewal-opening timestamps;
- reminder count sent and next action timestamp;
- recipient-route type and stable reference;
- state, lease fencing, attempts, last error and terminal failure time;
- created and updated timestamps.

It contains no raw external address, answers, artifact content or invitation token. The cycle is operational authoritative state for renewal scheduling, not an evidence assessment or Program conclusion.

### 6.4 Recipient route

Internal routes store the current principal ID. External routes store an opaque stable contact reference resolved by the configured recipient/delivery adapter. Monitoring storage never contains the raw external address.

Automatic renewal cannot be activated for an external recipient unless the route can be resolved and the delivery adapter confirms it can issue future invitations. At dispatch time, the resolver rechecks scope, contact status and purpose. A missing, revoked or changed route produces **Renewal blocked** and operator work.

### 6.5 Submission to next schedule

The PostgreSQL evidence submission transaction already stores the immutable submission, advances the request and appends `EvidenceResponseSubmitted`. The monitoring consumer uses inbox deduplication to:

1. load the exact request, submission, origin and active form Monitoring Check;
2. calculate response expiry and renewal opening;
3. close any prior open cycle for the lineage;
4. create or update the next cycle idempotently;
5. record a safe monitoring event.

The event carries identifiers only. Answers are read through bounded exact-ID evidence access.

### 6.6 Renewal and reminders

One bounded worker class claims due collection cycles with leases and retry limits. At renewal opening it:

1. re-reads the current Program, Monitoring Check, current form revision and recipient route;
2. fails closed when the check is paused/retired, scope mismatches, the route is invalid or authority cannot be resolved;
3. creates or reuses the exact origin-keyed successor Evidence Request;
4. attaches attributed prefill provenance;
5. assigns the internal request or issues a new request-scoped external invitation;
6. records delivery receipt or delivery failure truthfully;
7. schedules the next reminder action in the same monitoring transaction that records the completed action.

Each reminder rechecks that the successor request remains open and still needs the response. Reminders do not change ownership. Submission, cancellation, recipient reassignment failure, form/check pause or retirement, or evidence sufficiency from an authoritative alternative cancels remaining reminder actions.

No state is labelled sent from an outbox or log entry alone. External delivery requires a provider receipt. Internal assignment requires the request to appear in the current eligible recipient's work queue.

## 7. API and read model

The browser uses the existing create/list/transition APIs plus focused additions:

```text
POST /api/v1/programs/{id}/monitoring-checks
POST /api/v1/monitoring-checks/{id}/collection-policy
GET  /api/v1/programs/{id}/collection-summaries
POST /api/v1/form-templates/{id}/collections
```

Creating a form check accepts the collection policy. Updating policy on an existing check accepts `expected_version`, creates the next revision and never mutates the active revision. Activation revalidates the exact policy, form revision, route and maker-checker conditions.

`collection-summaries` is an actor-scoped, bounded Program read model keyed by Monitoring Check ID. It returns only the latest collection state needed by the Program UI:

```text
monitoring_check_id
latest_request_id
latest_submission_at
respondent_label or recipient_hint
expires_at
renewal_opens_at
currency_state
active_request_deadline
reminders_sent
reminder_count
delivery_state
last_error_safe
projection_generated_at
projection_source_version
```

Unknown values remain absent and render as unknown. The read never loads a broad request population in application memory to find a known lineage.

All commands derive tenant, legal-entity and actor scope from verified identity. Body-supplied actor, approver, reviewer or sender identity is ignored. Policy changes and automatic activation re-evaluate current authority and fail closed.

## 8. Failure and recovery

- Existing checks without policy remain operational but never auto-renew.
- A stale expected version reloads the check and preserves safe form input.
- A missing current form revision blocks renewal and names the required operator action.
- A recipient route that cannot be resolved blocks delivery without exposing the raw destination.
- A created request with failed external delivery remains the same request on retry; a second request is not created.
- A worker crash after request creation reuses the origin-keyed request.
- A reminder-delivery failure retries within the bounded budget and then becomes visible terminal work.
- A potentially expired response remains visible with its collection date; it is not deleted or relabelled current.
- Partial Monitoring or collection-summary failure does not hide other Program sections.
- Pausing or retiring a check cancels future collection actions but preserves history.
- Re-enabling collection requires a new approved revision and does not replay missed reminders automatically.

## 9. Accessibility, responsive behavior and copy

Tabs have programmatic tab/tab-panel relationships, visible focus and keyboard navigation. The mobile selector has a persistent label and moves focus to the selected section heading. URL changes do not steal focus on ordinary pointer selection.

The design introduces no new tokens, illustration style or ambient motion. Section changes may use the existing short content transition and must honor reduced motion.

Customer-visible copy names the Program, collection form, response state, respondent, date or recovery action. Examples:

- **Valid for 12 months**
- **Renewal starts 30 days before expiry**
- **3 reminders**
- **Last submitted 14 Aug 2026 at 10:32 by Vendor security contact**
- **Response potentially expired**
- **Choose a current recipient before renewal can continue**

The product does not say that ClearSight emailed a person, obtained current evidence, verified a response or maintained compliance unless the relevant receipt exists.

## 10. Acceptance and rendered evidence

Backend coverage includes:

- policy validation and maker-checker revision behavior;
- month-end and leap-year expiry arithmetic;
- deterministic reminder distribution for counts 1–5;
- origin uniqueness and crash-safe request reuse;
- exact predecessor and prefill provenance;
- file/signature exclusion from prefill;
- tenant, legal-entity, recipient and authority isolation;
- pause, retirement, cancellation, reassignment and alternative-evidence stop conditions;
- internal assignment and external delivery receipts;
- delivery retry, terminal failure and recovery;
- summary boundedness, freshness and safe recipient labels.

Frontend coverage includes:

- six Program sections and one visible panel;
- keyboard tab behavior and mobile selector replacement;
- URL-backed exact-section opening;
- collection policy setup and validation;
- unified collection records without form/check duplication;
- last-submitted respondent, time and expiry display;
- current, renewal-due, potentially-expired, awaiting-response and blocked states;
- loading, empty, partial, conflict, permission and long-content behavior;
- copy-quality regression.

Rendered evidence preserves the supplied current Monitoring screenshot as the before-state baseline. Production components are rendered at 1440×900, 1024×768, 390×844, 320×800 and the 200% zoom proxy. Evidence covers light and dark presentation where supported, keyboard focus, long respondent/form names, no policy, current response, renewal due, potentially expired, delivery failure and mobile section replacement. The highest-impact defect found during review is fixed and the affected render is repeated.

## 11. Delivery boundary

This tranche intentionally excludes:

- generic RRULE or arbitrary recurrence authoring;
- business calendars, holidays or timezone-specific cutoff rules;
- a new notification center or arbitrary channel builder;
- provider-specific contact management inside Monitoring;
- reopening, overwriting or auto-submitting prior responses;
- copying file or signature attestations into a new submission;
- automatic evidence sufficiency, compliance, legal-status or material-risk conclusions;
- automatic Matters beyond an approved Automation Policy;
- a new generic workflow engine.

The feature is complete when an authorized Program user can configure a form collection's validity and reminders, see the latest respondent submission and expiry, and observe an idempotently created renewal request with truthful assignment/delivery state; the respondent can review attributed prefill and submit a new immutable response; and the Program workspace remains usable through accessible responsive sections without endless page scrolling.
