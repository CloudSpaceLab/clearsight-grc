> Historical design note. For the current implementation and limits, read `../DATA_AND_WORKFLOW_REVIEW.md` and `../TEST_REPORT.md`.

> Retained module reference from the preceding prototype. V2.4 workflow changes and actual delivery limits are documented in `../README.md`, `UX_REVIEW.md`, `PRIVACY_MODEL.md` and `TEST_REPORT.md`.

> V2.3 interaction update: see README.md and SCOPE_AND_UX.md. The data model and production integration boundaries below remain applicable.

# ClearSight V2.3 — Assessment instruments and source-backed fields

**Design handoff · 13 November 2026 · Proposed production architecture**

This document distinguishes the intended architecture from the runnable HTML prototype. All demo records, connectors and observations are synthetic. No production repository or tenant was modified.

## 1. Product decision

Build one assessment-instrument engine. Human, Integration and Hybrid are starting templates and filters, not separate response models or services. An assessment instrument may be attached to a program, included in a framework seed pack, or used for a standalone scoped assessment. Not every operational form creates a compliance obligation.

A field has independent dimensions: display/answer shape; input source; validation; interpretation rule; scope; evidence requirements; and optional mappings to requirement objectives. A boolean field is still a boolean whether an employee answers it, a connector observes it, or a formula calculates it. A mixed grid may contain all three.

Do not create another requirements table or duplicate the existing monitoring engine. Preserve ClearSight's established MonitoringCheck / MonitoringResult, Evidence, Matter / Actions and requirement-evaluation ownership. Forms present those records; they do not become a competing compliance truth.

## 2. Form capabilities

Human input should include short/long text, exact numbers, percentages, dates, currency with explicit currency code, single choice, multi-select, searchable dropdowns, entity pickers, attestations and evidence attachments. Operational assessments need single-choice matrices, multiple-choice matrices, mixed-type grids, repeating tables and repeating sections.

Mixed grids require stable field, row and column identities. A control table might contain control name, design effectiveness, test methods, likelihood, impact, derived score, accountable person, evidence and review comments. A technical service table might contain service identity, environment, measured uptime, measured latency, declared owner and remediation. Cell provenance must survive reordering, imports and template revisions.

The builder should use a single, directly editable document canvas. Edit question labels, choices, table rows and column headings in place. Open a focused source or column drawer only when requested; do not keep a schema inspector permanently visible. Expose field settings first; source binding, rules, conditional logic and data handling progressively. Offer a visual column inspector, not JSON as the standard bank-user interface. Large tables need keyboard traversal, copy/paste, row validation, bulk assignments, bounded imports and row virtualization. Small screens should offer a row-detail editor as well as deliberate horizontal table scrolling.

Conditional visibility must have an explicit retention policy: hidden answers may remain in audit history but must not silently affect active completion, scoring or rule evaluation. Repeating sections, entity pickers, advanced conditional branching, batch assignments, row virtualization and file-upload content handling are production follow-ons, not all implemented in this HTML revision.

## 3. Definition and binding layers

Suggested logical records, composed with existing services:

- **FormDefinition / FormVersion:** reusable instrument, ordered stable fields and groups, typed constraints, display rules and immutable published version.
- **FieldDefinition:** value type, renderer, required rule, source mode, guidance, sensitivity, retention and optional objective mapping.
- **CheckDefinition / CheckRevision:** reuse the canonical MonitoringCheck definition for source query/probe, extraction, typed comparison, observation window and quality policy.
- **Connection:** authorized connector endpoint, environment, runner, permitted operations, credential-vault reference and tenant/legal-entity boundary.
- **AssessmentBinding:** program, exact assessment target, form-version ID and connection-slot bindings. An API row may additionally bind a service/asset ID, not a free-text display name.
- **AssessmentRun / FieldResponse:** immutable submitted response set; human entries and references to source observations, not hidden overwrites of one another.
- **Observation / MonitoringResult:** append-only collected value, type, units, scope, source revision, source identity, observed/received times, window, sampling population, provenance and quality.
- **RequirementEvaluation:** conclusion from sufficient accepted evidence against the actual objective. A field-level pass is not a framework-wide pass.

Framework packs contain logical slots such as `bank-db`, `observability`, `payment-api` and `web-probe`. Never embed a customer's endpoint credentials or literal secrets in a shared framework. On adoption, map those slots to authorized connections for the bank or vendor engagement. Unbound slots remain visible as unconfigured/unknown.

Source references are reusable, but reuse must match tenant, legal entity, subject, environment, query parameters, authorization context, observation window and freshness policy. The same vendor name is insufficient grounds to reuse a bank's observation for another engagement.

## 4. Technical field configuration

A technical field should expose the following configuration in the UI:

**Source:** approved connector, asset/service, endpoint or approved query, authentication profile / secret reference, method, bounded request parameters and execution location.

**Extract:** JSON path, selected metadata property, scalar/aggregate query result or probe metric; expected type and unit. Distinguish absent properties, null, false, zero and empty arrays.

**Evaluate:** typed operator, threshold/expected value, quantifier (all/any/percentage of rows), objective mapping, severity and any policy-approved combination rule.

**Observe:** point-in-time or rolling/calendar window, cadence, freshness limit, required sample count, expected population/collection coverage and treatment of missing measurements. Window boundaries must include timezone and source timestamps.

**Govern:** owner, data classification, redaction, review/approval policy, notification routing, exception policy and retained evidence.

Publish configuration through the existing governed Draft → Simulate → Approve → Activate workflow in production. The HTML demonstrates draft/publish and local fixtures only, not secure independent configuration approval.

## 5. Four examples and their exact claims

### Database column encryption

A test of stored byte length greater than 32 is a length check. It cannot establish encryption or identify 3DES: 33 bytes of plaintext also pass. Treat it as a heuristic or a data-format invariant, not proof of an algorithm. Checking the database column's *declared maximum length* proves even less about stored values.

Prefer approved metadata queries to encryption configuration, application cryptography configuration and/or appropriately scoped KMS/HSM records. Match the claim to the encryption layer: application field encryption, database column encryption, whole-database encryption and transport encryption are different. An authorized query can return plaintext from a database protected at rest; seeing plaintext through that interface does not alone prove encryption-at-rest is disabled.

For SQL Server Always Encrypted specifically, `sys.columns` exposes `encryption_type_desc` and `encryption_algorithm_name`. This is source-specific configuration evidence, not a universal database test and not proof of key custody or every copy of the data. The prototype uses an illustrative approved-query reference rather than retrieving sensitive values.

NIST withdrew SP 800-67 Rev. 2 on 1 January 2024 and disallowed TDEA for applying new cryptographic protection under its guidance after 31 December 2023. Do not seed 3DES as a modern good-practice default. Applicable bank/payment requirements and legacy-use exceptions need their own reviewed policies; this is not a claim that NIST guidance automatically governs every bank.

Sources: https://learn.microsoft.com/en-us/sql/relational-databases/system-catalog-views/sys-columns-transact-sql?view=sql-server-ver16 ; https://csrc.nist.gov/pubs/sp/800/67/r2/final ; https://csrc.nist.gov/News/2023/nist-to-withdraw-sp-800-67-rev-2

### API availability > 90%

Define the service, probe location, success predicate and window first. A current status response proves only a current observation. A status endpoint can supply historical availability only when it actually returns auditable historical metrics with defined semantics.

For evenly scheduled synthetic probes, use successful valid probes / valid probes and report collection coverage separately as valid / expected probes. A request timeout or service error from a functioning probe is a valid bad service observation. A collector outage, missing credentials or failed monitoring import is a collection problem. Do not discard service failures from the denominator or silently convert collection gaps to success.

Request-based availability, duration-weighted availability and probe-success ratio are different metrics; label which one is used. Require minimum coverage and sample count. Preserve the user's >90% example as a configured threshold, not an endorsed bank-wide SLO.

### Website responds in < 5 seconds

Specify total HTTP transaction duration, time to first byte, or a real-browser page-load metric. They are not interchangeable. Configure a single-probe requirement or an agreed aggregate such as p95 over 24 hours, the probe location, timeouts, status-code predicates and TLS verification. A timeout must not become a missing value excluded from a success-only percentile. Track availability alongside latency, or use a combined good-event predicate.

The prototype demonstrates a labelled synthetic p95 total-HTTP-duration rule, in milliseconds, with `<5000` as the strict boundary. It is not measuring real browser page performance.

SLI sources: https://sre.google/workbook/implementing-slos/ ; https://sre.google/sre-book/service-level-objectives/ ; https://github.com/prometheus/blackbox_exporter/blob/master/CONFIGURATION.md

### Authenticated API capability

Bind an authorized connection, select a response property such as `$.capabilities.idempotency`, require a boolean, and compare to `true`. A `false` value is a real negative result; a missing property or authentication failure is unknown. A self-reported capability flag does not prove correct behavior. An active functional test requires a separate approved recipe and, where possible, an isolated test environment. Avoid uncontrolled state-changing tests against production banking APIs.

## 6. Execution and interpretation

Use a single existing monitoring execution pipeline:

`versioned check + authorized binding → scheduled/on-demand execution → typed observation → quality gate → check result → scoped assessment evidence → requirement evaluation`

Store three independent dimensions:

1. Execution: succeeded, failed, timed out, unbound, denied.
2. Quality: current, stale, incomplete, invalid, unavailable.
3. Rule outcome: pass, fail, unknown, or not applicable with governed rationale.

Keep previous accepted observations in history. A new collection failure must not make an old pass appear newly verified. Current posture can use a prior still-valid observation only under an explicit freshness/last-known-value policy; display its age and the current monitoring fault. A submitted assessment pins observation IDs and never mutates when continuous monitoring continues.

Production automatic acceptance may be appropriate for narrow, approved and well-instrumented objectives. Do not require manual sign-off for every machine observation unnecessarily. Conversely, a manual override must retain the raw value and rule outcome and create a distinct reasoned, approved exception/evaluation. No silent conversion of source failure or heuristic pass to compliance satisfaction.

This separation is consistent with OSCAL's distinct observations, supporting evidence and findings, without requiring ClearSight to adopt the whole OSCAL schema.

Source: https://pages.nist.gov/OSCAL/learn/concepts/layer/assessment/assessment-results/

## 7. Security and operating model

Run database and private API checks in bank-approved private runners/agents, using least-privilege read-only roles, approved parameterized queries or stored views, and network segmentation. Parameterize data values; identifiers require allowlisted query definitions rather than unsafe string interpolation. Do not permit arbitrary SQL or script execution in a form definition.

Use a central secret store. Connection configuration contains secret references and scoped access policy, not secrets inside form JSON, answers, browser storage, generated HTML or logs. Support required auth profiles through adapters (OAuth2, mTLS, API-key references, service identities); rotate and audit access.

An arbitrary endpoint field is an SSRF risk. Enforce approved destinations at connection time, validate DNS-resolved addresses, preserve TLS hostname verification, control redirects and retries, and prohibit cloud metadata/loopback destinations. Private addresses needed for bank assessment are allowed only through explicitly scoped private-runner policies, not a general bypass of SSRF controls. Apply bounded timeouts, concurrency, payload sizes, query cost limits and egress rules.

Keep sensitive records in the bank where possible; return metadata, counts, aggregates and redacted proof. Bound retention, evidence access and export permissions. Vendor-provided evidence sharing is explicit and engagement scoped.

Sources: https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html ; https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html

## 8. Performance and implementation sequence

Do not create one cron job or network call per form cell. Group compatible checks by connection, query/probe revision, exact target and observation window. Fetch once, extract multiple fields, and reference one underlying observation safely across consumers. Use idempotency keys, per-tenant/connector quotas, jitter, backoff and durable scheduling. Do not cache across authority boundaries. Reuse existing metric stores rather than writing every raw probe into every assessment response.

Keep high-cardinality time series separate from assessment summaries. Evaluate quality and requirement projections incrementally. Use stable keyed field/row components and virtualized large grids in the real application; the HTML intentionally performs full-view renders for simplicity and is not a production performance implementation.

Implement in small tranches: shared typed field/grid schema and renderer; independent source bindings and governed check recipes; private-runner execution and observation references; then automated review policies, bulk workflows and framework upgrade handling. Preserve existing assessment, RCSA and monitoring tests while adding source-error/coverage/version-pinning tests.

## 9. What this prototype actually implements

The local HTML provides 17 field shapes, three seeded form templates, human/integration/hybrid starting points, direct document editing, focused source drawers and visual mixed-column editing, matrices and multi-select matrices, repeating rows, typed TSV paste, conditional visibility, simple numeric calculations, source/rule configuration, five deterministic technical recipes, five fixture scenarios, run inspection, JSON export payloads, form versions, program/target attachments, framework-seeded assignments, connector-slot selection, explanatory submission of unknown results and locked submitted snapshots.

It does not perform real HTTP/SQL calls, collect credentials, schedule jobs, upload or inspect evidence bytes, authenticate users, enforce tenant authorization, independently approve form definitions, integrate these new form observations into the existing requirement-evaluation screen, provide a full question bank authoring workflow, implement repeating sections, support complex formula graphs end to end, or operate at production grid scale. The existing bank requirement assessment and RCSA demonstrations remain available unchanged.

Source and rules are stored inline in prototype form versions for portability. The production implementation should reference canonical MonitoringCheck revisions and durable observations rather than copy the prototype's inline structure into a second monitoring subsystem.
