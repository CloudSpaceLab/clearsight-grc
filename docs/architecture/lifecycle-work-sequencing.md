# Governed lifecycle work sequencing

**Status:** #27.2b-A implementation boundary  
**Issue:** #27  
**Implementation:** PR #46

This document defines how ClearSight may turn a multi-branch Decision or Response lifecycle state into actor work without pre-deciding the substantive outcome and without adding another workflow or policy engine.

## 1. Core rule

A routing policy may select the **next responsibility/gate**. It may not select the eventual Decision or Response outcome on behalf of the responsible actor.

For example:

```text
Decision IN_REVIEW
→ policy selects AUTHORIZER as the next responsibility
→ compiler derives the actions AUTHORIZER is currently allowed to perform
→ current authority resolves the actor
→ Today shows one decision packet
→ actor still chooses approve / conditional approve / reject / supersede as applicable
```

The policy does **not** write `APPROVED` merely to route work to an authorizer.

This preserves the existing invariants:

- Decision/Response rows remain canonical lifecycle state;
- `WorkRequirement` remains derived compiler output;
- Workflow Task remains a rebuildable actor projection;
- authority resolution remains the source of who may act;
- policy selection never becomes an execution or approval receipt.

## 2. Reuse the existing maker-checker RoutingPolicy

No lifecycle-sequence table is introduced.

The existing governed `RoutingPolicy.definition` rule opts into sequencing only when it explicitly declares:

```json
{
  "lifecycle_type": "DECISION",
  "lifecycle_state": "PROPOSED",
  "lifecycle_subtype": "EXCEPTION_APPROVAL"
}
```

`lifecycle_subtype` is optional. For Decisions it matches the Decision type; for Responses it matches the response audience.

Rules without `lifecycle_type` and `lifecycle_state` remain ordinary authority-routing rules and do not participate in lifecycle sequence selection.

Malformed partial declarations are rejected during normal RoutingPolicy validation before approval/activation.

## 3. Responsibility selection, not actor selection

Lifecycle sequence resolution ranks matching current ACTIVE/effective routing rules using the same explicit rule priority/specificity concepts already present in policy definitions.

The result is:

```text
responsibility
rule provenance
policy version provenance
```

It is not a principal assignment.

After a responsibility is selected, the existing authority service independently resolves the current eligible principal/candidate set using:

- tenant and legal entity;
- exact Matter;
- responsibility;
- command/decision type;
- materiality with Matter priority as its floor;
- current grants/assignments/delegations/routing/segregation state.

Canonical Matter visibility is then applied before a READY assignment may exist.

Lifecycle metadata therefore cannot smuggle a state-specific actor around the authority engine. Existing RoutingPolicy validation continues to reject same-rank authority routes with conflicting selectors.

## 4. Legal entity identity

Authority configuration may identify a legal entity by durable ID or code. Lifecycle sequence matching normalizes the current legal entity to both aliases before applying policy rules so sequencing cannot silently fail because one surface used a UUID and another used the institutional code.

Only currently effective legal entities participate.

## 5. Packet compilation

`CompileMatterWork` first returns ordinary deterministic requirements plus unresolved `WorkAmbiguity` values.

For each ambiguity with a governed sequence rule:

1. select the next responsibility;
2. re-evaluate every currently legal lifecycle target through the shared Decision/Response lifecycle policy;
3. retain only targets executable by the selected responsibility;
4. create one `WorkRequirement` packet;
5. preserve all retained targets in `allowed_targets`;
6. set `target_status` only when exactly one legal target remains.

When several legal outcomes remain, `target_status` is deliberately empty.

Example authorizer packet:

```text
primary action: Decide
responsibility: AUTHORIZER
allowed targets: APPROVED, CONDITIONALLY_APPROVED, REJECTED, SUPERSEDED
target status: <empty>
```

That is an actor decision packet, not a pre-approved transition.

## 6. Failure and ambiguity behavior

Fail closed:

- no lifecycle sequence rule → ambiguity stays unresolved; no actor Task;
- selected responsibility has no legal action from the current lifecycle state → ambiguity stays unresolved;
- equal-ranked sequence rules select different responsibilities → policy conflict/error; no guessed Task;
- authority route unavailable/ambiguous → existing unassigned/blocked routing behavior;
- authority service failure → operational error/retry, not fabricated business state;
- actor cannot read the Matter → no READY assignment.

A sequence policy becoming stale because the Decision/Response state changes is harmless: the compiler re-derives legal transitions from the current canonical row and will no longer apply a rule whose lifecycle state does not match.

## 7. Convergence

Matter events continue to trigger immediate lifecycle recompilation through the existing transactional outbox publisher.

Routing-policy changes may alter the selected next responsibility without creating a Matter event. The existing slower bounded `matter-work-projection` maintainer therefore includes Decision/Response candidates and re-runs current compilation/authority resolution.

Reconciliation retires obsolete actor Tasks and creates/reassigns only the current packet. Workflow remains projection state; policy and lifecycle records remain authoritative.

## 8. Evidence and acceptance

This boundary is complete only when exact-head CI proves:

- ordinary authority-only routing rules do not become sequence rules;
- malformed lifecycle rule metadata fails policy validation;
- equal-ranked conflicting next responsibilities fail closed;
- legal-entity UUID/code aliases match consistently;
- selected responsibility preserves all currently legal actor outcomes rather than choosing one;
- a responsibility with no legal action creates no Task;
- PostgreSQL projects a policy-selected packet through current authority;
- a routing-policy change converges the active actor packet without a new Matter event;
- the previous deterministic Response/Verification projection remains green;
- baseline web, accessibility and rendered-evidence gates remain green.

## 9. Explicitly outside this tranche

This sequencing boundary does not define ordinary Evidence Request recipient truth.

`why_you`, `created_by`, invitation prose and capture-session ownership remain insufficient to project an Evidence Request into actor Today work. #27.2b-B must separately define canonical internal/external recipient scope, delegation/redirect/wrong-recipient behavior, expiry/revocation and protected-record visibility before those requests enter the actor work queue.
