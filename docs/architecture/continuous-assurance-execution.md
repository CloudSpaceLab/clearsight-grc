# Continuous assurance and connected-data execution

**Status:** T0 deterministic kernel is merged; isolated PostgreSQL source execution is implemented as the next bounded execution tranche. Durable population/rule/run configuration and worker scheduling remain future work under #57.  
**Scope:** deterministic schema/profile/condition semantics plus an isolated, read-only PostgreSQL execution boundary that future continuous checks must reuse.

## Purpose

ClearSight needs to evaluate operational populations such as accounts, privileged identities, servers, vendors, branches, resilience records and KRIs without becoming a second data warehouse or specialist detection platform.

The reusable primitive is:

> an approved executable population, an explainable governed condition, and bounded execution semantics.

The bank/source system remains authoritative for the population. ClearSight retains only the configuration, execution receipt, affected-state detail, evidence and intervention context required for governance and reconstruction.

## Existing boundaries that remain authoritative

Continuous assurance does not replace:

- `internal/evidence` Source Registry and source-health observations;
- `internal/autonomy` Signal/Drift and Automation Policy semantics;
- `internal/continuity` Program, Matter, trigger, Decision, Action and outcome truth;
- `internal/workflow` rebuildable actor work projections;
- `internal/runtime` worker classes, outbox/inbox, leases and retry isolation;
- `internal/httpapi/route_registry.go` identity/access classification.

A future rule match must enter those existing seams rather than creating a second task, alert, incident, workflow or authorization model.

## T0 kernel

`internal/assurance` owns only deterministic execution semantics:

```text
native source fields
→ bounded logical schema
→ stable schema fingerprint
→ bounded sample profile
→ deterministic lexical/structural/statistical hints
→ typed condition compile
→ exact field dependencies
→ MATCH / CLEAR / UNKNOWN evaluation
→ parameterized PostgreSQL match/unknown predicates
```

T0 intentionally introduced no database tables, API routes, worker class, credentials, AI dependency or domain-specific Account/Server/Vendor monitor types. This kept the first semantic contract cheap to change before persistence or product UI depended on it.

## Schema and profiling semantics

Logical types are deliberately small: string, number, boolean, time and unknown.

Schema fingerprints are stable across column ordering but change when a named field's logical type or nullability changes. Reordering a SELECT projection therefore does not create false schema drift, while a material type/nullability change does.

Profiles are summaries, not stored samples. Hard ceilings cap fields, rows, distinct values, top values, display bytes and cell bytes. A separate total field×row cell budget prevents individually legal field and row limits from multiplying into an unexpectedly large in-process profile. Oversized or incompatible cells are counted as invalid rather than retained.

Top values are exposed only for lexically safe, demonstrably low-cardinality categorical fields. Candidate raw categorical values stop being retained as soon as the observed field ceases to satisfy that low-cardinality bound. Fields resembling account numbers, identity numbers, secrets, tokens, email or phone data never expose sampled values through the profile.

Hints identify useful candidate semantics from field names, logical types and bounded cardinality. They carry explicit provenance and confidence and are never treated as approved mappings.

## Condition semantics

The T0 AST supports bounded Boolean composition plus:

- exists / missing;
- equality / inequality;
- IN / NOT IN;
- numeric/time ordering and BETWEEN;
- bounded string containment;
- field-to-field comparison for compatible types.

There is no arbitrary scripting language or implicit string-to-number/type coercion.

T0 `NUMBER` evaluation is deliberately limited to finite IEEE-754 values within the bounded ±2^53 magnitude used by the current evaluator. Source values and numeric rule literals outside that domain are rejected or fail closed; the kernel does **not** claim exact-decimal or monetary arithmetic. A future rule may not use approximate `NUMBER` semantics where exact decimal/money comparison is material — that requires a separate explicit decimal contract before activation.

Hard ceilings cap AST depth, node count, per-`IN` cardinality and per-literal size. Aggregate budgets also cap total literal count and total literal bytes across the complete condition tree, so many individually legal nodes cannot combine into an unbounded rule payload.

`CompileCondition` deep-copies the accepted condition and extracts its exact field dependencies so later execution can project only the required source columns.

## Tri-state truth

Per record:

- `MATCH` means the condition is demonstrably true;
- `CLEAR` means demonstrably false;
- `UNKNOWN` means a required value is null, absent, incompatible or outside the bounded logical domain.

Boolean composition preserves three-valued semantics:

- `AND`: any clear child is clear; otherwise any unknown child is unknown;
- `OR`: any matching child matches; otherwise any unknown child is unknown;
- `NOT`: unknown remains unknown.

This distinction is mandatory because missing/invalid source data may not be represented as a clean result.

## PostgreSQL predicate pushdown contract

The PostgreSQL compiler produces two parameterized predicates from the same validated AST:

- `MatchSQL` selects demonstrable matches;
- `UnknownSQL` selects null, invalid, oversized or otherwise out-of-domain rows that cannot be safely classified by the T0 evaluator.

Field identifiers come only from the validated schema and are quoted as PostgreSQL identifiers. Rule values remain positional parameters. Logical strings are normalized to PostgreSQL text semantics, including UUID-backed source fields; T0 numbers are compared through bounded double-precision semantics so source-side execution does not silently become more precise than the pure evaluator. Unsafe numeric casts are guarded behind the bounded validity test. PostgreSQL pushdown rejects time literals finer than PostgreSQL's microsecond precision rather than silently changing their meaning.

This compiler does **not** make arbitrary SQL safe. Population Definitions require governed query ownership and a dedicated source identity. The purpose of predicate compilation is performance and truth parity: capable sources filter/project data before transfer instead of sending every clear row to ClearSight for application-memory evaluation.

## Isolated PostgreSQL source execution

`PostgresSourceExecutor` proves the first connected-source boundary without introducing persistent population/rule state or worker scheduling.

### Connection ownership

The executor creates its own pgx pool from an opaque secret reference. It cannot accept or reuse the authoritative ClearSight application pool.

The secret resolver must return an explicit PostgreSQL URI containing a host, user and database. Partial keyword/URI forms that could inherit process-environment defaults are rejected. Credential material is not retained in executor state, receipts, events or API objects.

Per-executor pool size, connection/query/lock/idle/ping timeouts, connection lifetime and idle lifetime have defaults and non-raiseable ceilings. `MinConns` and `MinIdleConns` remain zero so an inactive source does not reserve connections.

### Least privilege and read-only defense

Session startup forces:

- `default_transaction_read_only=on`;
- bounded `statement_timeout`;
- bounded `lock_timeout`;
- bounded `idle_in_transaction_session_timeout`;
- UTC timezone;
- a generic ClearSight source-execution application name.

DSN-level `options` overrides are discarded before connection creation.

Every inspection/evaluation also starts an explicit `REPEATABLE READ READ ONLY` transaction. PostgreSQL identities with superuser, create-role, create-database, replication or bypass-RLS attributes are rejected at executor creation. Production source accounts are still expected to be purpose-built read identities; transaction read-only is defense in depth, not permission to reuse administrative credentials.

A bounded client context wraps every operation independently of server-side GUCs. Server timeout, client cancellation and source-pool bounds therefore cooperate rather than relying on any one mechanism.

### Population query hygiene

A transient `PopulationDefinition` contains an ID, one bounded SELECT/WITH query and a stable subject key. The current hygiene check deliberately accepts only one SELECT/WITH statement, rejects NUL/multi-statement input and bounds query size.

This hygiene is **not** the SQL security boundary: conservative false negatives are acceptable, and approved source queries may legitimately contain complex joins/CTEs. Dedicated source privileges, explicit read-only transaction semantics and configuration governance remain the safety boundary.

The query text is hashed with the subject key into a population fingerprint. Evaluation receipts contain that fingerprint rather than the raw query.

### Schema inspection and drift

Schema inspection runs `LIMIT 0` inside the same read-only source transaction and uses pgx field OIDs/type metadata. Arbitrary derived queries are treated as conservatively nullable because base-table `NOT NULL` semantics do not necessarily survive joins/expressions.

Projected schema width is bounded. The configured subject key must be present. Unknown native types remain `UNKNOWN` rather than being guessed.

A compiled condition carries the logical schema fingerprint it was validated against. Evaluation re-inspects the current source schema in the same repeatable-read snapshot and fails with schema-change state before running the condition when the fingerprint differs. Schema drift therefore cannot appear as zero matches.

### Count-only evaluation

Current-state evaluation executes one aggregate source query using the T0 PostgreSQL predicates and returns only:

- total population count;
- MATCH count;
- UNKNOWN count;
- derived CLEAR count;
- population/schema fingerprints;
- evaluation time and completeness.

No clear-row population is transferred or stored. The executor also asserts that MATCH and UNKNOWN do not overlap before issuing a complete receipt.

Source failure, timeout, credential failure and schema change are errors; none produces a complete zero-match receipt.

## What remains deliberately absent

This tranche still does **not** add:

- durable Population Definition, Rule, Run or affected-subject tables;
- a credential store;
- global source-pool manager or worker composition;
- API/configuration UI;
- scheduling;
- CDC/transition state;
- KRI/window aggregation;
- affected-row persistence;
- Signal/Drift/Program/Matter integration;
- generic SQL/HTTP connector framework.

Those are separate decisions after this execution boundary is proven and benchmarked. In particular, a future source-pool manager must enforce a deployment-wide connection budget before multiple persistent source executors are composed into workers.

## Future connected-source execution

Later tranches under #57 may add the smallest durable model required for:

```text
existing Evidence Source
→ versioned Population Definition
→ versioned governed Rule
→ bounded worker/source execution
→ Run receipt + optional affected-subject projection
→ rule-level episode
→ existing Signal/Drift → Program Trigger → Matter/Work
```

Ordinary current-state rule storage should scale with runs plus configured affected detail, not full source population multiplied by run history.

Transition and window/KRI rules are separate execution tiers because they require CDC/prior values or explicit denominator/window semantics. They must not silently cause complete source-row replication.

## Product rule

Successful routine checks should produce no Today noise. Human work appears only through existing canonical downstream records when a material episode, source/schema failure, evidence need, decision, remediation or outcome check requires intervention.

Configure may later expose a focused Sources & checks flow; Programs may expose current check coverage/state; Matters/Work may expose exact originating rule/run context. None of these presentation surfaces becomes a second state or workflow system.

## Acceptance corpus

Permanent T0 tests cover account/KYC, server/security, vendor and resilience/BIA shapes; null/invalid/oversized/schema-change cases; numeric/string bounds; profile/condition aggregate budgets; and PostgreSQL/pure-evaluator parity.

The isolated PostgreSQL executor integration test additionally proves:

- privileged source credentials are rejected;
- a non-superuser source role with an otherwise-permitted DELETE still cannot mutate through the executor's read-only transaction;
- source session read-only/UTC/timeout settings are active;
- UUID/text/numeric schema normalization;
- a representative four-row population evaluates to one MATCH, two UNKNOWN and one CLEAR;
- schema drift fails closed;
- slow source execution is bounded and query text is not returned in errors.

No domain-specific evaluator or connector is permitted unless a later use case demonstrates semantics that cannot be represented safely by these shared contracts.
