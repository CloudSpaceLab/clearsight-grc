# Issue #26 P0 closure review

**Reviewed:** 2026-08-07  
**Pre-close baseline:** `main@df98a7f66c28642637a45a10662abac042dcd144` after PR #25  
**Closure implementation:** PR #30

## Conclusion

The P0 seam-integrity findings in #26 are now implemented without a rewrite or parallel framework.

PR #25 closed:

- typed route/identity boundary;
- persisted capture consolidation/security;
- source-health reconciliation;
- worker work-class isolation and bounded terminal failure.

PR #30 closes the remaining P0 seams:

- P0.4 authoritative command/response truth;
- P0.5 executable runtime/OpenAPI contract parity;
- P0.6 bounded effective-authority convergence.

Still-valid P1/P2 findings from the original audit are intentionally **not** being marked fixed by the P0 closure. They are moved to linked follow-up issues before #26 closes.

## P0.4 — command and transaction truth

### Compound verification

`VerificationResultBundle` commits a failed verification result with its required REOPEN/ESCALATE or CREATE_MATTER/link consequence in one optimistic PostgreSQL transaction. Required event/outbox/projection work shares that boundary.

### Post-commit response truth

The previous systemic defect was that many material mutators committed and then reconstructed a full Program/Matter. A later read failure could return an HTTP failure for an already committed write.

PR #30 adds two narrow protections:

1. **existing aggregates:** a normalized current-version probe detects that an authoritative Program/Matter version advanced even if response reconstruction returns 5xx; the API then returns a `COMMITTED` degraded-response receipt with aggregate ID/version;
2. **create commands:** API PostgreSQL composition uses a short-lived in-process fallback containing only the just-committed create result until the first successful reconstruction/fallback response.

Neither mechanism is a second source of truth. PostgreSQL remains authoritative; no parallel durable receipt/orchestration store was introduced.

## P0.5 — executable transport contract

`api/runtime.openapi.json` now defines the production method/path and security/access inventory.

CI verifies exact parity with `internal/httpapi/route_registry.go` for:

- method/path;
- route access class;
- administrative permission.

The contract explicitly distinguishes signed authenticated routes, public health routes, bounded capture capability and authenticated-or-capability upload.

CI also uses `npm ci` with the lockfile rather than mutable install behavior.

Detailed domain schema/client generation remains a later code-deletion opportunity; P0 closes route/security drift mechanically without adding unnecessary dependencies.

## P0.6 — effective authority

Migration `000014_effective_authority_routes` compiles currently-effective approved routing rules into an indexed read model.

The production authority decision now combines, in bounded queries:

- compiled routing rules;
- current responsibility assignments;
- active scoped delegation chains;
- applicable authority grants/materiality limits;
- active segregation constraints.

Resolution ranks by explicit priority and specificity. Same-rank conflicting candidate sets fail closed.

ROLE/POSITION selectors expand current occupants. Multiple eligible humans remain an explicit candidate set. TEAM/QUEUE/COMMITTEE semantics are not collapsed to one arbitrary occupant.

Unresolved selectors remain excluded from command execution but are surfaced as integrity findings.

PostgreSQL integration tests cover active delegation, expired delegation, grant materiality, segregation, responsibility assignments and ambiguity rejection.

## Regression gates found during closeout

The closeout reran the full PostgreSQL integration surface and exposed two test-isolation/model-alignment defects that PR #25's earlier green run had not caught consistently:

1. source reconciliation fixture used a DRAFT Program even though operational dependency resolution intentionally targets ACTIVE/PAUSED Programs;
2. runtime poison-work fixture could claim unfinished global queue rows left by earlier serialized integration packages.

The fixtures now exercise the actual operational contract deterministically:

- source replay activates the Program before degradation/recovery assertions;
- runtime terminal-work test quarantines unrelated unfinished global queue rows before checking claim/reclaim semantics.

## Validation

A pre-documentation exact head cleared:

- `gofmt`;
- race-enabled Go unit tests;
- PostgreSQL composition;
- migrations through `000014_effective_authority_routes` on PostgreSQL 18;
- serialized PostgreSQL integration, including source replay, worker terminal work and effective authority tests;
- `go vet`;
- `npm ci`;
- TypeScript typecheck;
- rendered/axe tests;
- production Vite build.

The final documentation head must pass the same CI before PR #30 is merged; an older green SHA is not sufficient evidence for the final head.

## Remaining findings

P0 closure does not erase the original lower-priority audit findings. The next P1 sequence is:

1. Program-state temporal/currentness truth;
2. Matter closure current-record truth;
3. lifecycle-specific command responsibility;
4. bounded ordinary reads/work projections;
5. document-import durability/resource hardening.

Schema ownership/dead compatibility cleanup is retained as P2.

## Closure decision

Issue #26 may close after:

- [x] P0.4 implementation;
- [x] P0.5 implementation;
- [x] P0.6 implementation;
- [ ] linked P1/P2 follow-up issues preserve remaining acceptance criteria;
- [x] implementation and architecture docs are reconciled;
- [ ] final PR #30 exact-head CI is green;
- [ ] PR #30 is merged into `main`.
