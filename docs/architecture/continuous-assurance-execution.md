# Continuous assurance and connected-data execution

**Status:** T0 execution kernel implemented; durable population/rule/run configuration remains future work under #57.  
**Scope:** deterministic schema/profile/condition semantics that future connected-source execution must reuse.

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

`internal/assurance` owns only deterministic, side-effect-free execution semantics:

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

T0 intentionally has:

- no database tables;
- no API routes;
- no worker class;
- no credentials;
- no AI dependency;
- no domain-specific Account/Server/Vendor monitor types.

This keeps the first contract cheap to change before persistence or product UI depends on it.

## Schema and profiling semantics

Logical types are deliberately small: string, number, boolean, time and unknown.

Schema fingerprints are stable across column ordering but change when a named field's logical type or nullability changes. Reordering a source SELECT therefore does not create false schema drift, while a material type/nullability change does.

Profiles are summaries, not stored samples. Hard ceilings cap fields, rows, distinct values, top values, display bytes and cell bytes. Oversized or incompatible cells are counted as invalid rather than retained.

Top values are exposed only for lexically safe categorical fields. Fields resembling account numbers, identity numbers, secrets, tokens, email or phone data never expose sampled values through the profile.

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

T0 `NUMBER` evaluation is deliberately limited to finite IEEE-754 values. Integer inputs outside the exact ±2^53 range fail closed as `UNKNOWN`; the kernel does **not** claim exact-decimal or monetary arithmetic. A future rule may not use approximate `NUMBER` semantics where exact decimal/money comparison is material — that requires a separate explicit decimal contract before activation.

Hard ceilings cap AST depth, node count, IN cardinality and literal size even when a caller requests higher limits.

`CompileCondition` deep-copies the accepted condition and extracts its exact field dependencies so later execution can project only the required source columns.

## Tri-state truth

Per record:

- `MATCH` means the condition is demonstrably true;
- `CLEAR` means demonstrably false;
- `UNKNOWN` means a required value is null, absent or incompatible.

Boolean composition preserves three-valued semantics:

- `AND`: any clear child is clear; otherwise any unknown child is unknown;
- `OR`: any matching child matches; otherwise any unknown child is unknown;
- `NOT`: unknown remains unknown.

This distinction is mandatory because missing/invalid source data may not be represented as a clean result.

## PostgreSQL pushdown contract

The PostgreSQL compiler produces two parameterized predicates from the same validated AST:

- `MatchSQL` selects demonstrable matches;
- `UnknownSQL` selects null-dependent unknown rows.

Field identifiers come only from the validated schema and are quoted as PostgreSQL identifiers. Rule values remain positional parameters.

This compiler does **not** make arbitrary SQL safe. Future Population Definitions still require approved read-only source identities, separate source connection pools, timeouts, concurrency limits and one-statement query governance.

The purpose of predicate compilation is performance and truth parity: capable sources should filter/project data before transfer instead of sending every clear row to ClearSight for application-memory evaluation. PostgreSQL integration tests execute nested Boolean and cross-field predicates and require their MATCH/CLEAR/UNKNOWN result to equal the pure evaluator, rather than treating generated SQL text shape as sufficient evidence.

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

External source connections must be isolated from ClearSight's own authoritative PostgreSQL pool. Credential material is resolved from opaque secret references at the executor boundary and never enters Source endpoints, run records, events, logs or browser state.

Ordinary current-state rule storage should scale with runs plus configured affected detail, not full source population multiplied by run history.

Transition and window/KRI rules are separate execution tiers because they require CDC/prior values or explicit denominator/window semantics. They must not silently cause complete source-row replication.

## Product rule

Successful routine checks should produce no Today noise. Human work appears only through existing canonical downstream records when a material episode, source/schema failure, evidence need, decision, remediation or outcome check requires intervention.

Configure may later expose a focused Sources & checks flow; Programs may expose current check coverage/state; Matters/Work may expose exact originating rule/run context. None of these presentation surfaces becomes a second state or workflow system.

## T0 acceptance corpus

Permanent tests prove the same kernel works across materially different shapes:

- customer/account status and KYC review fields;
- server patch/support/certificate fields;
- vendor tier/assurance fields;
- resilience target versus observed RTO fields;
- null, invalid, oversized, schema-change and identifier-quoting adversarial cases;
- out-of-range integer values fail closed rather than being compared imprecisely;
- PostgreSQL pushdown and pure evaluation agree on nested Boolean and cross-field tri-state cases.

No domain-specific evaluator is permitted unless a later use case demonstrates semantics that cannot be represented safely by the shared contract.
