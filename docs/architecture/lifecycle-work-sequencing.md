# Governed lifecycle work sequencing

**Status:** #27.2b-A implementation boundary  
**Issue:** #27  
**Implementation:** PR #46

This document defines how ClearSight may turn a multi-branch Decision or Response lifecycle state into actor work without pre-deciding the substantive outcome, granting hidden actor authority, or adding another workflow/policy engine.

## 1. Core rule

A routing policy may select the **next responsibility/gate**. It may not select the eventual Decision or Response outcome on behalf of the responsible actor, and a lifecycle sequence rule may not identify the actor.

For example:

```text
Decision IN_REVIEW
→ selector-free sequence rule selects AUTHORIZER as next responsibility
→ compiler derives the actions AUTHORIZER may currently perform
→ separate ordinary authority rule/assignment/grant/delegation resolves the actor
→ Today shows one decision packet
→ actor still chooses approve / conditional approve / reject / supersede as applicable
```

The sequence rule does **not** write `APPROVED`, and it does **not** grant a CRO or other principal authority merely to make routing convenient.

This preserves the existing invariants:

- Decision/Response rows remain canonical lifecycle state;
- `WorkRequirement` remains derived compiler output;
- Workflow Task remains a rebuildable actor projection;
- authority resolution remains the source of who may act;
- sequence policy selects responsibility only;
- policy selection never becomes an execution or approval receipt.

## 2. Reuse the existing maker-checker RoutingPolicy

No lifecycle-sequence table is introduced.

The existing governed `RoutingPolicy.definition` may contain two deliberately separate rule types.

### Lifecycle sequence rule

A sequence rule is **selector-free** and opts into sequencing only when it explicitly declares:

```json
{
  "id": "exception-authorizer-gate",
  "legal_entity_id": "BANK-NG",
  "object_type": "MATTER",
  "object_id": "*",
  "responsibility": "AUTHORIZER",
  "decision_type": "matter.decision.record",
  "priority": 100,
  "lifecycle_type": "DECISION",
  "lifecycle_state": "IN_REVIEW",
  "lifecycle_subtype": "EXCEPTION_APPROVAL"
}
```

`lifecycle_subtype` is optional. For Decisions it matches the Decision type; for Responses it matches the response audience.

A lifecycle rule with an actor `selector` is invalid and fails RoutingPolicy validation.

### Ordinary authority rule

Actor authority remains a normal routing rule with a selector and **without** lifecycle metadata, for example:

```json
{
  "id": "exception-authorizer-authority",
  "legal_entity_id": "BANK-NG",
  "object_type": "MATTER",
  "object_id": "*",
  "responsibility": "AUTHORIZER",
  "decision_type": "matter.decision.record",
  "priority": 100,
  "selector": {
    "kind": "ROLE",
    "ref": "CRO"
  }
}
```

Rules without lifecycle metadata remain ordinary authority-routing rules and do not participate in lifecycle sequence selection.

Malformed partial lifecycle declarations are rejected during normal RoutingPolicy validation before approval/activation.

## 3. Sequence rules cannot grant authority

The existing effective-authority materializer only creates an `effective_authority_routes` row when a rule has both `selector.kind` and `selector.ref`.

Because lifecycle sequence rules are validation-enforced to be selector-free:

- they produce **zero effective authority routes**;
- they cannot silently create state-independent actor authority;
- they cannot bypass current grants, assignments, delegation or segregation constraints.

The maker-checker selector-cardinality conflict check receives an authority-only projection of the policy definition. Sequence rules are validated independently and are excluded from actor-selector conflict evaluation.

PostgreSQL acceptance explicitly asserts that a sequence rule materializes zero authority routes while its separate ordinary authority rule materializes exactly one route.

## 4. Responsibility selection, then actor resolution

Lifecycle sequence resolution ranks matching current ACTIVE/effective selector-free sequence rules using explicit rule priority/specificity.

The result is only:

```text
responsibility
sequence rule provenance
policy version provenance
```

After a responsibility is selected, the existing authority service independently resolves the current eligible principal/candidate set using:

- tenant and legal entity;
- exact Matter;
- responsibility;
- command/decision type;
- materiality with Matter priority as its floor;
- current grants/assignments/delegations/ordinary authority routes/segregation state.

Canonical Matter visibility is then applied before a READY assignment may exist.

## 5. Legal entity identity

Authority configuration may identify a legal entity by durable ID or code. Lifecycle sequence matching normalizes the current legal entity to both aliases before applying policy rules so sequencing cannot silently fail because one surface used a UUID and another used the institutional code.

Only currently effective legal entities participate.

## 6. Packet compilation

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

## 7. Failure and ambiguity behavior

Fail closed:

- no lifecycle sequence rule → ambiguity stays unresolved; no actor Task;
- sequence rule contains a selector → policy validation/runtime sequence resolution error;
- selected responsibility has no legal action from current lifecycle state → ambiguity stays unresolved;
- equal-ranked sequence rules select different responsibilities → policy conflict/error; no guessed Task;
- authority route unavailable/ambiguous → existing unassigned/blocked routing behavior;
- authority service failure → operational error/retry, not fabricated business state;
- actor cannot read the Matter → no READY assignment.

A sequence policy becoming stale because the Decision/Response state changes is harmless: the compiler re-derives legal transitions from the current canonical row and will no longer apply a rule whose lifecycle state does not match.

## 8. Convergence

Matter events continue to trigger immediate lifecycle recompilation through the existing transactional outbox publisher.

Routing-policy changes may alter the selected next responsibility or current actor without creating a Matter event. The existing slower bounded `matter-work-projection` maintainer therefore includes Decision/Response candidates and re-runs current compilation/authority resolution.

Reconciliation retires obsolete actor Tasks and creates/reassigns only the current packet. Workflow remains projection state; policy and lifecycle records remain authoritative.

## 9. Evidence and acceptance

This boundary is complete only when exact-head CI proves:

- ordinary authority-only routing rules do not become sequence rules;
- lifecycle sequence rules are selector-free and cannot materialize effective authority routes;
- ordinary authority rules still require supported selectors;
- maker-checker selector-conflict checks ignore selector-free sequence rules but still validate authority rules;
- malformed lifecycle metadata fails policy validation;
- equal-ranked conflicting next responsibilities fail closed;
- legal-entity UUID/code aliases match consistently;
- selected responsibility preserves all currently legal actor outcomes rather than choosing one;
- a responsibility with no legal action creates no Task;
- PostgreSQL projects a policy-selected packet through **separate current authority**;
- a routing-policy change converges the active actor packet without a new Matter event;
- previous deterministic Response/Verification projection remains green;
- baseline web, accessibility and rendered-evidence gates remain green.

## 10. Explicitly outside this tranche

This sequencing boundary does not define ordinary Evidence Request recipient truth.

`why_you`, `created_by`, invitation prose and capture-session ownership remain insufficient to project an Evidence Request into actor Today work. #27.2b-B must separately define canonical internal/external recipient scope, delegation/redirect/wrong-recipient behavior, expiry/revocation and protected-record visibility before those requests enter the actor work queue.
