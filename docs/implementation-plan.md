# ClearSight implementation ledger

**Status date:** 2026-09-02
**Current execution:** Stored runtime truth, oversight history completeness and hosted vendor email acceptance

## Stored runtime truth — implemented, integration verification in progress

Actor display context now resolves the exact verified tenant, legal entity and principal through an injected runtime directory resolver. Missing directory scope returns an unavailable response; the API does not invent a bank, person or role label. The Today package no longer exposes a callable hardcoded work list, and dead hardcoded context handlers have been removed. An architecture regression scans non-test runtime packages for these fallback paths.

Deterministic browser evidence now has its own HTML/TypeScript entry and `dist-evidence` build. The customer `main.tsx` import graph cannot reach fixture pages, the static HTTP interceptor or evidence-only build switches, and both CI workflows enforce that boundary. Seeded non-production environments must therefore exercise ordinary repositories and routes; static request interception is retained only for isolated rendered-evidence generation.

## Oversight history completeness — implemented, hosted installation pending

The reference installer now creates five comparable vendor reviews and two additional issue types through canonical Matter, decision, action, assignment, verification, reopening and closure commands at deterministic event times. The cohort includes on-time and overdue completion, reassignment, a returned decision and reopening, and remains explicitly labelled sample reference data. Aggregate-only oversight calculates only measures available from stored aggregate timestamps; PostgreSQL additionally derives reassignment and return counts and records the continuity-event high-water mark. Resolution ranges remain suppressed below five comparable closures.

The Oversight workspace exposes the checked population, exclusions, unknown scope, generation time, projection version and source freshness, plus workload, completed/measured work, cycle range, SLA, blocked, reopened, reassigned and returned facts. It does not produce a composite employee score or rank. Remaining work is to install the exact merged reference history on the non-production host, refresh the projection, confirm exact Matter drilldowns and record production-volume/performance limitations separately.

## Governed work handoffs — implementation verification

The current change corrects the Matter handoff selector, resolves seeded demo staff through the normal position/role path, delivers issue and Action assignment email through an idempotent outbox consumer, and moves the highest-risk vendor request, reassignment, response-signing and issue-authorization interactions onto shared focused sheets and fields. The handoff action now targets and opens the exact governed control it names; ambiguous generic next-work states remain read-only instead of choosing an arbitrary subresource. Action reassignment identifies the current performer from the performer transition route rather than the accountable owner who is authorized to reassign, excludes that performer from the alternatives, and requires an explicit replacement and reason before submission. Shared list overlays use a fixed application overlay root and a bounded internally scrolling listbox. A short opening-position guard restores incidental document movement caused while focusing the selected option, while later user page scrolling still dismisses the list. This prevents the menu from closing or changing sticky Form Builder geometry; the earlier measured sticky-toolbar jump was 363px. Date controls now inherit the shared light/dark field treatment. Vendor request sheets become temporarily non-dismissible during prepare/send and use an immediate submission guard so Escape, backdrop or repeated activation cannot discard or duplicate in-flight work.

Assignment delivery now persists a tenant-scoped `DELIVERY_STARTED` claim before SMTP. Because SMTP provides no portable reconciliation or provider idempotency key, a worker crash after that claim is treated as an unknown delivery outcome and is not automatically resent; this prevents duplicate staff mail at the cost of requiring operator reconciliation for the stranded receipt. An SMTP disconnect after the server accepts the DATA payload is recorded as `DELIVERY_OUTCOME_UNKNOWN`, including failure of the later QUIT exchange, and is also terminal pending reconciliation. Final delivery receipts update the same claim, and only a failure known to have occurred before provider acceptance is eligible for a new claimed attempt. Composite tenant foreign keys cover the legal entity, outbox event and principal.

Focused browser and integration verification remains required before release: render the affected desktop/light, desktop/dark, 390px and 320px states; run the full UI review and backend gates; deploy the exact merged revision; then execute one controlled non-production reassignment and verify only redacted delivery status plus manual inbox receipt. PostgreSQL migration rollback/reapply and the hosted email acceptance journey cannot be inferred from unit tests.

## Governed Forms UX #103 — code closure in verification

The accepted Forms UX direction now preserves selected-template context across filters and saved views, supports server-backed newest/oldest updated sorting, uses the shared focus-managed and scroll-locked sheet for template detail, replaces narrow-screen table overflow with labelled stacked records, and keeps every material builder action visible at 320–390px. Library lifecycle actions now come from bounded exact-template authority reads and fail closed when responsibility cannot be checked. Question duplication, safe pointer reordering, keyboard Move up/Move down and exact Review Fix focus are executable. The deterministic Forms evidence contract includes populated mobile library, mobile/reflow builder, real desktop pointer drag and 120-question performance fixtures.

The local exact-head matrix passes 127/127 rendered flows and 72/72 governed Forms capabilities; its 120-question Chromium fixture became usable in 342ms and applied a question edit in 57ms on this development host. Issue #103 must remain open until representative bank-user creation/operation timing, actual 200% browser zoom, normal-enterprise-network search/filter p95, hosted exact-commit journeys and production-shaped PostgreSQL/load evidence are recorded. Local fixtures, tagged compilation and unit tests cannot truthfully satisfy those external acceptance gates.

## Vendor email acceptance journeys — implemented, hosted acceptance pending

The approved [vendor email acceptance design](superpowers/specs/2026-08-30-vendor-email-acceptance-journeys-design.md) and [implementation plan](superpowers/plans/2026-08-30-vendor-email-acceptance-journeys.md) add a shared email-client-safe invitation/OTP presentation, task-specific registration messages, persisted `GENERAL` and `CERTIFICATION_REFRESH` Vendor Work kinds, and governed address/certification forms. Vendor registration and vendor-work collection now use canonical form distributions and expiring access routes; vendor-work initial and change requests use direct email OTP, and retry or cancellation revokes the exact route/session chain. The bank UI selects the correct recipient role and form, shows staff/vendor response states, and records evidence acceptance without claiming the linked Matter is resolved.

Remaining release work is external: merge the exact green head, configure protected recipient and SMTP values on the host, deploy that merged revision, pass the redacted STARTTLS readiness check, and complete the controlled inbox/click/submission/review/closure sequence with the approved test recipients. SMTP provider acceptance alone does not prove inbox receipt. Object inspection must report submitted PDFs as available before evidence acceptance.
**Umbrella pilot/GA catalogue:** #13

This file is the authoritative implementation ledger. Code, migrations and executable tests remain final capability truth. Completed detail belongs in focused architecture documents, PRs and tests rather than parallel planning frameworks.

## 1. Completed executable tranches — do not rebuild

| Capability | Completion |
| --- | --- |
| P0 route / identity / transaction / worker / authority integrity | PRs #25, #30 / #26 |
| P1 effective current-state, lifecycle, bounded reads and durable document imports | PRs #34–#39 / #32 |
| Intervention-first UI/UX foundation | PR #31 |
| Low-effort typed Capture + request-bound artifacts | PR #40 |
| P2 durable-schema ownership and dead compatibility removal | PRs #41, #42 / #33 |
| Today actor work queue + canonical Matter authority/materiality | PR #43 |
| Deterministic lifecycle work compiler | PR #45 |
| Governed lifecycle sequencing | PR #46 |
| Canonical Evidence Request recipient truth | PR #49 |
| Evidence Request recipient lifecycle + Today projection | PR #50 |
| Initial governed Program/Work mutation UX | PR #51 |
| Actor Program review baseline + review-by-exception | PR #53 |
| Protected Matter read parity + Program review explanation delta | PR #55 |
| Enterprise identity/access EIA-0…5 | PR #59 |
| Reusable connected-source T0…T2 | issue #61 through PR #70 |
| Stateless AI gateway transport T3 | issue #61; PR #71 |
| Governed AI workload/policy runtime T4 + durable receipts/response grants T5 | issue #61; PR #73 |
| Governed document-proposal reviewer/authorizer handoff | issue #72; PR #75 |
| Program monitoring setup | Program and requirement creation, reusable forms, connected public status endpoints, maker-checker form/check activation, on-demand collection and immutable results |
| Issue and change creation | Inline authority-checked Matter creation, business work types, actor ownership, optional Program linking and immediate in-workspace handoff |
| Complete Program operating record | Versioned details and ownership, requirement supersession, applicability, safeguards and eligible performers, evidence expectations/results, lifecycle, monitoring and exact linked issues |
| Complete issue/change operating record | Facts and gaps, eligible assignment, decisions, actions, responses, outcome checks and closure through actor-scoped UI commands |
| Governed Forms and vendor refresh | Direct Forms navigation; reusable/scored templates; document and AI proposals; multi-recipient distribution; revocable magic-link/OTP access; draft recovery; immutable response revisions; rich communication templates; confirm/correct/replace held vendor facts; field-level bank application receipts |

## Governed Forms and vendor refresh — implemented, release verification in progress

The implementation reuses Monitoring form revisions, document imports, Evidence Requests, capture sessions, artifacts, authority, outbox delivery and vendor assessment records. It adds a direct Forms workspace for searchable reusable templates, exact active revisions, saved views, manual/starter/document/AI authoring, Sent Forms, Responses, Imports and Communications. Distributions support multiple internal or external To/CC recipients, three bounded access policies, customizable future deadlines and expiries, amendments, lock/reopen/revoke and previewed supersession with optional compatible-answer carry-forward.

Recipient work supports optimistic server drafts plus encrypted, workspace-bound browser recovery. Immutable response revisions retain sign-off assurance and scoring. Vendor refresh forms can request only selected held fields, ask for confirmation/correction/replacement, and route each proposal through bank review with conflict detection and a durable application receipt. See [`product/governed-forms.md`](product/governed-forms.md).

Release verification requires the exact-head unit, tagged PostgreSQL, copy, accessibility, bundle and rendered Forms gates plus hosted smoke testing. Generic Forms distributions and vendor assessment requests remain deliberately distinct origins; the UI must not substitute one for the other or falsely advance vendor review.

## Program and Matter operational foundation

The approved [`Program and Work operational-completeness design`](superpowers/specs/2026-08-25-program-work-operational-completeness-design.md) closes the remaining API-only operating gaps without creating parallel workflow, assignment, authority or review state. Delivery follows the detailed [`Matter plan`](superpowers/plans/2026-08-25-matter-operational-completeness.md) first, then the [`Program plan`](superpowers/plans/2026-08-25-program-operational-completeness.md). PR #51 remains a completed initial mutation tranche; it did not make every supported command reachable or every record maintainable.

Domain expansion must build on these operation/participant reads and dedicated record workspaces. It must not reimplement generic Matter facts, assignment, Actions, Decisions, evidence assessment, outcome checks, Program requirements or linked-issue UI inside a vertical-specific module.

## Premium first-run and vendor identity presentation — implemented

The approved [`premium first-run and vendor-branding design`](superpowers/specs/2026-08-26-premium-first-run-and-vendor-branding-design.md) adds surface-aware Today and Vendors guides, a shared optional cinematic panel and stored vendor icons without creating a separate vendor dashboard or treating an icon as identity evidence. Guides remain actor-, tenant-, surface- and version-scoped; they can be skipped, resumed and restarted, and a guidance failure leaves the workspace available.

Vendor identity and service relationship use distinct resources and versions. `/api/v1/vendor-identities/{vendor_id}` owns shared legal-identity facts, including an optional normalized website hostname. `/api/v1/vendors/{relationship_id}` continues to own the legal-entity-scoped service, owner and due-diligence context. The protected brand subresource returns validated stored PNG bytes only. Browser image markup never uses the vendor website URL.

Website icon retrieval is durable, bounded and independent of vendor workflow availability. The worker uses no ambient proxy or credentials, validates DNS and connected destinations, revalidates redirects, bounds response size and converts accepted PNG, JPEG, WebP or ICO input to a canonical PNG. Discovery is disabled by default in production until outbound-network policy explicitly enables it. An approved upload overrides a discovered icon; removing the override restores the latest safe icon matching the current hostname. Upload reservation, command receipt, brand events and cleanup preserve replay and orphan recovery.

The integrated tranche passes the exact-head unit and tagged-build gates, the rendered Today/Vendors matrix, reduced-motion and responsive inspection. The 1600×900 presentation cover is retained under `docs/presentation-assets`. PostgreSQL integration tests still require `TEST_DATABASE_URL`; a compile-and-skip result is not production database evidence.

## 2. Mobile-channel monitoring — implemented application slice

An authorized user can create a channel Program and requirements without API or JSON work. Reusable collection forms support weighted Yes/No scoring and critical answers. Active forms create dated Evidence Requests on demand; a submitted response is evaluated automatically against the exact active form and Monitoring Check versions. A GRC administrator can connect a public HTTPS JSON status endpoint, select an observed field and expected value, and create a connected-data Monitoring Check. Active source checks run on demand and store the source receipt with score, band and coverage.

Form and Monitoring Check activation requires a different approver from the submitter. A result is an observation and does not create an approved Evidence Assessment or compliance conclusion. Recurring form-request generation, credential entry in the browser, automatic Matter creation and a general-purpose integration catalogue remain outside this slice.

The Work workspace also supports inline creation of user-reported risk issues, control gaps, regulatory changes, findings, requests, exceptions and incidents. New items begin as Draft work, default to internal access and the signed-in accountable owner, and can be linked to a scoped Program at creation. System-derived types remain reserved for their originating checks and observations.

## 2A. Program and issue/change operating completeness — implemented

An exact Program now opens a dedicated actor-scoped record. Authorized users maintain Program scope and owner, add source-anchored requirements, replace current requirements without overwriting history, decide applicability, define safeguards and select eligible performers, link coverage, retire incorrect coverage relationships with a reason, define evidence checks from named current sources, record reviewer results and change operating status with a rationale. Issue owners can likewise retire an incorrect Program relationship without deleting its history. Monitoring observations remain separate from approved evidence assessments. Open issues are read through an exact Program filter; users can open an existing issue or create linked work with the Program already selected.

The operations response explains the current owner, authorizer, reviewer and eligible assignment candidates. If authority routing is unavailable, stored values remain readable and mutations fail closed. A command-to-UI coverage gate accounts for all Program and Matter material commands; the deterministic Program-trigger command is explicitly automation-only rather than presented as a user control.

No supported individual Program or Matter operating command requires JSON, direct API use or database access. Bulk migration, enterprise configuration builders and production integrations remain separate productization work.

The production-hardening follow-up binds Program and issue/change reads, history, links, trigger deduplication and material commands to the verified legal entity. Legacy entity scope is migrated only when deterministic; migration aborts before enabling scoped commands when a Program, issue or cross-entity link cannot be resolved safely. Exact records load their aggregate, responsibility route and review position independently, so available business records remain readable while stale or unavailable authority data disables only material mutations.

## 3. AI governance gateway — T3–T5 runtime implemented

The repository has an isolated stateless gateway process with OpenAI-compatible Chat Completions and Responses ingress, truthful SSE translation, OpenAI and Anthropic pilot adapters, SHA-256 workload authentication, model aliases, weighted routing, pre-output fallback, route circuit breaking, request/token/cost/concurrency budgets and content-free logs/metrics.

The runtime scope tracked by #61 is complete. T4/T5 add governed AI workload registration, Automation Policy lifecycle, deterministic decisions and obligations, reusable Source Binding resolution, maker-checker-controlled SHADOW/ENFORCE activation, durable decision receipts, response controls and approval/execution grants. The gateway still does not retain raw prompts, responses or source values merely to enrich presentation.

The remaining AI-governance work is productization, not a second runtime stack. Issue #74 owns the bank-native workload/policy authoring, revision, maker-checker, SHADOW→ENFORCE promotion, bounded decision exploration/simulation, Matter handoff and safe credential-rotation UX. Reuse `internal/aigateway`, `internal/aigovernance`, Source Access, Matter and authority foundations; do not add a parallel inventory, task, approval, source registry or event console. See [`architecture/ai-gateway-transport.md`](architecture/ai-gateway-transport.md).

## 4. Enterprise identity/access — implemented on PR #59

The completed architecture is intentionally smaller than the earlier greenfield P2/P3 proposal:

```text
OIDC sign-in
→ local principal
→ current position and/or governed directory group-role binding
→ legal-entity-wide or exact-department capability eligibility
→ existing route registry / visibility / command authority

SCIM provisioning
→ source-owned User + direct Group facts
→ explicit ClearSight group → existing role binding
→ same local capability resolver

OVERDUE escalation
→ existing routing-policy sequence
→ next workflow timer only
→ current authority/delegation/grant/segregation/visibility
→ optional current source-role + target role/group guard
→ same Workflow Task escalation overlay
```

LDAP/AD/SAML/Kerberos/password/MFA infrastructure remains upstream in authentik, an existing bank Keycloak deployment, or another standards-compliant identity layer.

### EIA-0 — department and escalation contract

- hierarchical `org_positions.department_path` without a department subsystem;
- bounded `role_templates.capabilities`;
- strict multi-level escalation definitions inside existing routing-policy versions;
- duplicate triggers, trailing/unknown JSON, invalid responsibility, excessive depth and non-monotonic thresholds fail closed.

### EIA-1 — OIDC + server session

- Authorization Code + state + nonce + PKCE S256;
- immutable tenant-bound issuer + subject → principal correlation;
- unknown subjects fail closed;
- one identity maps to one principal rather than one legal entity;
- maintained SCS pgx server sessions;
- no OIDC token or cached authorization truth in browser session state;
- trusted-origin redirect/CORS/cross-origin protections;
- signed-gateway and development compatibility retained.

### EIA-2 — local capability evaluation

- each OIDC request re-resolves current local access state from PostgreSQL;
- IdP group/role/permission claims are not ClearSight authorization truth;
- empty department path = legal-entity-wide capability;
- non-empty path = exact-department grant;
- no inferred parent/child capability inheritance;
- current principal/position/source/group/binding/role changes take effect without IdP reauthentication;
- material commands still require current visibility + `commandauth.Guard` + authority/delegation/segregation.

### EIA-3 — SCIM Users + Groups

- exact-pinned OSS SCIM transport behind `internal/scimapi`;
- isolated `/scim/v2` machine edge with tenant-scoped bearer digests;
- bounded Users/Groups CRUD, PATCH, filtering and pagination;
- source-owned canonical `PERSON` principals;
- no mutable-email takeover of existing principals;
- direct same-source group membership only;
- SCIM cannot write group-role mappings;
- explicit ClearSight group → existing role bindings feed the same EIA-2 resolver.

### EIA-4 — executable multi-level OVERDUE escalation

- no new escalation table, scheduler or task model;
- existing timer → outbox → inbox-idempotent consumer path;
- current Matter/legal entity/materiality/decision type/visibility/authority is re-read when a level fires;
- exact department ancestry narrows already-eligible authority candidates;
- optional `source_roles` validates the current originating task principal;
- optional `targets.roles` / `targets.groups` further narrow already-authorized target candidates with OR semantics;
- directory-group guards use current direct membership and require an active SCIM source;
- target-role guards use current effective position-role or governed group-role state;
- guards never manufacture responsibility, authority or protected-record visibility;
- no-route/ambiguity/multi-candidate/hidden/source-role/target-guard mismatches fail closed;
- replay is idempotent;
- only the next level is scheduled;
- completion/cancellation cancels pending escalation;
- escalation assignment remains work routing, not material command authority.

`OVERDUE` is executable. Other schema-valid triggers remain deferred until their canonical domain event source exists.

### EIA-5 — compact Identity & Access Configure surface

Implemented inside the existing Configure workspace rather than creating a second IAM application.

- `IDENTITY_READ` and `IDENTITY_CONFIGURE` are explicit route capabilities;
- current sign-in mode/issuer/assurance is visible;
- SCIM source state and bounded user/group counts are visible;
- source create/rotate returns a reveal-once high-entropy token while storing only its digest;
- source revoke disables provisioning and source-derived group access on the next request without deleting principal history;
- bounded read-only People & Groups inspection;
- effective-dated group → existing role mappings with optional exact department path;
- duplicate current mappings are prevented;
- mapping retirement preserves history;
- escalation runtime health shows active escalations, pending levels, recent unresolved events and failed timers;
- hierarchy/guard preview uses the same runtime semantics and never predicts the future actor;
- operators can select one escalation level, optional originating role(s), and allowed target role(s)/directory group(s);
- escalation guard changes require both `IDENTITY_CONFIGURE` and governance `CONFIG_WRITE`;
- saving a guard creates the next **unapproved** `routing_policy_versions` row rather than mutating the active policy or creating an override table;
- `routing_policies.current_version` remains on the approved version while a proposal is pending, so current routing stays live;
- the proposal maker can preview/update their proposal but cannot approve it;
- another maker cannot overwrite the pending proposal;
- a different authorized checker supplies rationale and activates the latest revision;
- approval revalidates the whole policy, existing authority conflicts and current role/group references;
- stale/missing roles or groups block activation without changing the approved current version;
- activation effective-dates the old version out and the new version in atomically and refreshes the existing effective authority projection;
- source/token/mapping and guard activation history reuse existing governed decision/outbox infrastructure;
- all nine EIA-5 staff routes live in the canonical `route_registry.go` and `api/runtime.openapi.json` projection;
- `deploy/reference-identity/authentik/README.md` documents the optional AD/LDAP/SAML → authentik → OIDC/SCIM bridge without bundling or forking authentik.

### Demo stakeholder login — non-production fixture

Demo mode no longer silently injects one configured actor. When `CLEARSIGHT_DEMO_MODE=true` with development identity mode:

- the login page supplies CRO, CCO/Compliance Officer, CISO, GRC Administrator, System Administrator, Internal Auditor, Program Owner and Evidence Respondent accounts;
- all supplied accounts use the explicit fixture password `demo`;
- a successful selection creates an HttpOnly, SameSite=Lax, HMAC-signed eight-hour demo session;
- the process-local signing key means restart invalidates existing demo sessions;
- tampered and expired sessions fail closed;
- a bounded bearer/capture token is never interpreted as staff identity;
- `Switch demo role` logs out and unmounts the entire application before another role is selected so role-scoped UI state cannot leak between identities;
- the three `/api/v1/demo/*` routes use the same canonical route registry with `DEMO_ONLY` classification and are absent when demo mode is off;
- production OIDC/signed authentication is unchanged and production configuration rejects demo mode.

The fixture contract is documented in [`engineering/demo-role-login.md`](engineering/demo-role-login.md). It is not an enterprise password-authentication subsystem.

### Final review bugs closed

- revoked SCIM source previously could retain group-derived access;
- EIA-4 escalation overlay could survive a material routing-semantic change;
- an early EIA-5 browser error path could resend a mutation;
- identity-admin routes were initially placed in a parallel mini-registry before being consolidated into the one canonical route inventory;
- an early EIA-5 audit implementation targeted deliberately removed `audit_events`; governed history now reuses `governance_decisions`;
- the overlay regression fixture was made transaction-local so it cannot contaminate the global escalation maintainer;
- role-only/group-only candidate guards originally encoded the absent selector side as JSON `null`; evaluation now normalizes absent selectors to empty arrays before PostgreSQL filtering;
- demo mode previously auto-authenticated one actor without a real role-selection/login flow; the top-level demo auth gate now requires an explicit supplied role and resets the whole app on role switch;
- demo auth endpoints were briefly implemented as a mini route registry during review and were consolidated into the canonical route inventory before merge.

The exact final PR head must pass the normal release gates below before merge; do not rely on an older green head after documentation or demo-login changes.

## 5. Productization still required outside the identity tranche

### Third-party risk management — #80 (collection and core review implemented; lifecycle incomplete)

The executable boundary now includes tenant-scoped vendor organizations, legal-entity-scoped service relationships and versioned assessment episodes. Vendor creation captures an optional HTTPS website and registered address; safe icon discovery is queued in the same transaction but cannot block the new relationship. An authorized owner can start onboarding for a proposed relationship, restart cancelled onboarding without losing its history, or start an explicitly referenced periodic/event-driven reassessment for an active, restricted or suspended relationship. A stable kind-and-trigger key makes retries idempotent, while a new trigger preserves the prior episode and repository locking prevents concurrent live episodes. Each episode uses an exact active form revision; the existing worker creates or reuses one canonical `VENDOR_REVIEW` Matter; and the owner can issue an origin-keyed Evidence Request through the protected invitation boundary. The external form supports typed fields, conditional sections, Classic and Wizard presentation, source-aware prefills, uploads, autosaved drafts and request-scoped magic-link sessions. The first-class **Vendors** workspace shows the relationship and its current assessment with one state-specific action.

Reference runtimes install the standard vendor review form through draft, maker submission and distinct-checker activation. In other tenants, a missing active form leads to an in-product governed setup sheet tied to a selected Program; it does not require API or database access and does not bypass maker-checker approval. Program and Issue or Change summaries use bounded server filters for state, jurisdiction, type, priority, due condition and verified-actor assignment. Filter queries remain in the route across exact-record navigation. Vendor linking uses an accessible focused sheet with bounded delayed search and contextual vendor rows.

Assessment setup and submission reactions use the existing runtime with stable dedupe identities. Terminal setup can requeue the same job without duplicating the assessment or review Matter. Recipient addresses remain inside the Evidence boundary, invitation replacement revokes prior invitations and redeemed sessions, and cancellation commits a safe outbox event before immediate best-effort revocation so the worker can retry request capability revocation after an outage. Safe assessment events/outbox facts exclude answers, recipient addresses and tokens. A response submission proves receipt only; it does not approve the vendor relationship or complete the review.

The internal review read is limited to the assessment starter, current relationship owner or current authorized reviewer and composes the exact submitted answers, provenance, coverage, artifact scan state, evidence classification, provisional form result and visible linked deficiency Matters. Verified reviewer commands start review, record typed document decisions, request targeted clarification, create or reuse distinct canonical deficiency Matters and record the conclusion. Completion fails closed when required answers, artifacts, document decisions or the current response state are unresolved. The completed assessment remains distinct from relationship approval or activation.

The vendor use cases remain Expansion maturity because contract and obligation metadata, policy-selected privacy and risk decisions, fail-closed activation/continuation, governed restriction and verified exit are not executable yet. The Vendors workspace does not present a proposed relationship as approved, active or compliant.

The implementation must reuse, rather than fork:

- Programs for continuing third-party assurance requirements and evidence expectations;
- Matters, Decisions, Actions and outcome checks for onboarding/review episodes and vendor deficiencies;
- Evidence Requests, submissions and artifacts for vendor collection;
- Evidence Sources plus Source Connection/View/Binding for procurement and vendor-system access;
- Monitoring Checks and the connected-assurance execution path tracked by #57 for expiry, missing-evidence and source-led conditions;
- the existing worker, timer, outbox/inbox, authority, delegation and route registries;
- document import/reconciliation for spreadsheet migration;
- the generic notification centre and governed report/export capability when those catalogue items are delivered.

A vendor risk register is therefore a bounded projection over those canonical records, not another independently editable register. Assessment review and deficiency associations now reference the same vendor relationship–Matter link used by Programs, issues and changes, while retaining the assessment-specific REVIEW or DEFICIENCY role for audit. If a new deficiency Matter cannot be linked, the command reconciles an ambiguous committed link or cancels the newly created Matter before reporting failure. Vendor-supplied comments remain communications; uploaded evidence remains distinct from sufficiency; completed remediation remains distinct from independently verified outcome.

### Capture / Imports

- provenance classes for materially sourced, prefilled and respondent-entered values;
- draft/resume and amendment/supersession where durable semantics exist;
- production malware/content inspection, quarantine and retry;
- governed multi-file collection only when the request contract requires it;
- recurring mapping/schema-change detection and governed canonical conversion;
- PDF/OCR provider isolation when introduced.

### Configure / enterprise administration

- responsibility and decision-authority matrices where existing backend configuration is not yet operable from the UI;
- governed delegation/substitution/absence where the current product surface is insufficient;
- security/session/notification/integration policy surfaces tied to real backend capability;
- issue #74 bank-native AI-governance workload/policy/decision authoring, maker-checker approval, SHADOW→ENFORCE promotion, bounded decision exploration and safe credential rotation over the existing T3–T5 runtime.

Do not reopen generic IAM/directory scope merely because adjacent enterprise administration remains.

### Enterprise shell / acceptance

- production scoped Explore/reconstruction without restricted-existence leakage;
- actor-scoped notification centre with exact-action launch;
- representative bank-user timed usability;
- real browser 200% zoom/reflow and assistive-technology review;
- production-scale resilience/security/backup/restore evidence;
- pilot CRO/CCO/CISO, owner, reviewer, signatory and evidence-respondent validation.

### Escalation adapters

Add `NO_ROUTE`, `AUTHORITY_INSUFFICIENT`, `MATERIALITY_INCREASE`, `RECIPIENT_UNAVAILABLE`, or `CONFLICT` runtime adapters only when the corresponding canonical domain event/timestamp exists. Do not fabricate generic trigger polling.

## 6. Canonical invariants

- Program = ongoing obligation/compliance continuity.
- Matter = bounded change, exception, finding, decision, action, response or verification case.
- Matter Action ≠ Workflow Task.
- Signal ≠ conclusion.
- Submission ≠ sufficient evidence.
- Implementation ≠ verified outcome.
- Recommendation ≠ approval.
- Automation Policy ≠ execution receipt.
- WorkRequirement ≠ authoritative state.
- WorkAmbiguity ≠ actor assignment.
- Lifecycle sequence policy selects responsibility, not outcome or actor.
- Lifecycle sequence rule ≠ authority route.
- Escalation sequence selects responsibility + scope; optional role/group guards only narrow current candidates.
- Escalation candidate guard ≠ authority grant.
- Target role/group guard = OR filter over candidates already admitted by authority/visibility/department constraints.
- Pending routing-policy revision ≠ active routing truth.
- Escalation assignment ≠ material command authority.
- Escalation timer lineage ≠ authorization truth.
- Department scope narrows eligibility; it never broadens tenant/legal-entity/object visibility.
- Department-scoped role/capability ≠ legal-entity-wide role/capability.
- Client-supplied department scope ≠ authorization scope.
- Directory group membership ≠ role mapping, responsibility or material authority.
- Directory group → role binding ≠ responsibility or material authority.
- SCIM source ACTIVE state is part of source-derived eligibility.
- SCIM lifecycle ≠ generic IAM administration inside ClearSight.
- OIDC assertion ≠ ClearSight role/capability authority.
- OIDC correlation ≠ legal-entity authorization.
- Demo role credential ≠ production authentication mechanism.
- Identity configuration history belongs to governed decision history, not a resurrected generic audit table.
- Evidence Request recipient is canonical request state; Workflow work is rebuildable actor projection.
- Workflow command packet is an executable projection; mutations are revalidated by domain service.
- Program review checkpoint = actor acknowledgement, not Program state or approval.
- Protected Matter visibility must fail closed before actor-facing search/pagination/limit.
- Saved Work view ≠ assignment or authorization truth.
- Source document ≠ extracted proposal ≠ accepted candidate ≠ canonical Requirement/Control draft ≠ reviewer conclusion ≠ authorizer approval ≠ active governance object.
- Accepted document proposal ≠ active Requirement or Control.
- Document-proposal Workflow task ≠ reviewer/authorizer authority; authority remains resolved from the canonical authority model at action time.
- Schema/spec existence ≠ capability.

Do not add parallel authorization, task, workflow, event, worker, receipt, review, preference, document, directory or dashboard stacks that duplicate existing foundations.

## 7. Current executable flow truth

```text
enterprise sign-in
OIDC issuer + subject
→ provisioned local principal
→ selected legal entity
→ current position or ACTIVE-source governed group-role mapping proves eligibility
→ server session
→ current local roles/capabilities resolved each request
→ identity middleware
→ canonical route permission / object visibility / command authority

enterprise provisioning
SCIM bearer source
→ source-owned User / principal
→ direct Group membership
→ explicit ClearSight group → existing role binding
→ legal-entity / exact-department capability eligibility
→ normal OIDC/session application path

identity administration
IDENTITY_READ / IDENTITY_CONFIGURE
→ canonical route registry
→ source/token or group-role mutation
→ existing durable identity/access tables
→ governance_decisions history
→ current access resolver on next request

governed escalation guard change
IDENTITY_CONFIGURE + CONFIG_WRITE
→ select active policy / sequence / level
→ optional source roles + target roles/groups
→ create unapproved routing_policy_versions revision
→ approved current_version stays live
→ independent checker + rationale
→ revalidate policy / authority conflicts / current references
→ atomically activate new current_version

stakeholder demo login
DEMO_ONLY supplied role catalogue
→ explicit non-production credential
→ signed HttpOnly demo session
→ normal identity middleware / role-scoped demo work
→ switch role logs out and remounts application

OVERDUE Matter work
canonical due date + pinned escalation lineage
→ next workflow timer only
→ timer fire / outbox / inbox-idempotent consumer
→ current source-role qualification
→ current legal entity + materiality + decision + visibility
→ current authority/delegation/grant/segregation
→ exact department ancestry boundary where configured
→ optional current target role/group guard
→ same Workflow task escalation overlay or unresolved receipt
→ next timer only
→ material action still requires current command authority

document proposal governance
source document
→ extracted proposal
→ intake acceptance + durable idempotent handoff/outbox
→ current authority resolves reviewer
→ exact Workflow task appears in reviewer Today
→ reviewer can refine, return, reject or submit for authorization
→ current authority resolves authorizer
→ exact authorization task appears in authorizer Today
→ authorizer approval invokes the canonical domain service
→ Requirement / Control Objective materializes exactly once
```

Presentation/projection/session/provisioning/escalation/admin/demo state never substitutes for canonical domain or material authority truth.

## 8. Release gates

A tranche is not complete until relevant gates pass on its **exact final head**:

- `gofmt` and `go vet`;
- race-enabled Go tests;
- PostgreSQL composition, migrations and latest rollback/reapply;
- serialized PostgreSQL integration/adversarial tests;
- TypeScript strict checking;
- rendered-state/axe tests;
- production Vite build;
- deterministic UI evidence when a visual/user-flow surface changes;
- identity/tenant/authority/degraded/replay tests where relevant;
- representative query/performance/recovery proof when cardinality or durability changes.

Never claim a branch or PR is green from an older commit.
