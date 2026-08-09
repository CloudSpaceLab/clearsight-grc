# ClearSight implementation ledger

**Status date:** 2026-08-09  
**Current execution issue:** #27  
**Umbrella pilot/GA catalogue:** #13

This file is the authoritative implementation ledger. Code, migrations and executable tests remain the final capability truth. Completed detail should live in focused architecture documents, PRs and tests rather than being copied into new planning frameworks.

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
| Governed Program/Work mutation UX | PR #51 |
| Actor Program review baseline + review-by-exception | PR #53 |
| Protected Matter read parity + Program review explanation delta | PR #55 |

### PR #55 — protected Matter read parity and review-digest correction

Selected from a fresh merged-code audit rather than backlog order.

- canonical `MatterVisibleTo` / `ParseMatterAccessPolicy` remains the policy reference;
- PostgreSQL Matter summary/search reads now fail closed on malformed access metadata before cursor/limit;
- `RESTRICTED` requires a string-only, non-empty allow-list and exact current principal membership;
- verified actor tenant must match the tenant being listed when actor context is present;
- production `CurrentPostgresRepository.ListMatters` applies actor visibility before `LIMIT` rather than relying on post-limit HTTP filtering;
- in-memory Matter lists apply the same canonical visibility before sorting/limit;
- internal worker/reconciliation reads without actor context keep their existing tenant-scoped behavior;
- hidden/malformed Matters therefore cannot consume page slots, alter cursors or appear in search for an unauthorized actor;
- a Program status-reason wording change now produces an `EXPLANATION` review delta without manufacturing a new/resolved exception identity;
- no RLS layer, visibility table, policy engine, preference system, task model or new authorization framework was introduced;
- exact final head `e245754bebea013475499a7fbdb0f6da0db62032` passed CI #717;
- squash-merged as `e9e61cafa5d6715b3e94bd72454b58b3ead87ff4`.

## 2. Current execution — re-derive again from merged code

There is deliberately **no preselected next tranche** after #55.

Before the next implementation, inspect the merged executable contracts again and rank the remaining gaps by user impact, correctness/security risk, and whether canonical domain support already exists.

Current candidates include:

1. actor-visible Work filtering / saved views;
2. protected-record focused mode beyond the now-correct list/search boundary;
3. durable draft/resume for genuinely complex Decision/Response work;
4. supported delegate/recuse/conflict/escalate flows where executable authority/domain commands exist;
5. Capture/Import lifecycle completion.

Do not select a candidate merely because it appears first. If a UI claim requires a missing domain contract, either establish only that narrow contract first or defer the claim.

## 3. Later productization still required

### Capture / Imports

- provenance classes for materially sourced, prefilled and respondent-entered values;
- draft/resume and amendment/supersession where durable semantics exist;
- production malware/content inspection, quarantine and retry;
- governed multi-file collection only when the request contract requires it;
- recurring mapping/schema-change detection and governed canonical conversion;
- PDF/OCR provider isolation when introduced.

### Configure / enterprise administration

- production directory/identity synchronization;
- responsibility and decision-authority matrices;
- routing/escalation configuration, simulation and candidate explanation;
- governed delegation/substitution/absence;
- maker-checker, effective dating, impact preview and rollback;
- security/session/notification/integration policy surfaces tied to real backend capability.

### Enterprise shell / acceptance

- production scoped Explore/reconstruction without restricted-existence leakage;
- actor-scoped notification centre with exact-action launch;
- representative bank-user timed usability;
- real browser 200% zoom/reflow and assistive-technology review;
- production-scale resilience/security/backup/restore evidence;
- pilot CRO/CCO/CISO, owner, reviewer, signatory and evidence-respondent validation.

## 4. Canonical invariants

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
- Evidence Request recipient is canonical request state; Workflow work is a rebuildable actor projection.
- Workflow command packet is an executable projection; every mutation is revalidated by the domain service.
- Program UI lifecycle choices are affordances only; server lifecycle/authority/version checks remain final.
- Program review checkpoint = actor acknowledgement of exact canonical versions; it is not Program state or approval.
- Program review digest = bounded presentation; full continuity history remains authoritative.
- Protected Matter visibility must fail closed before actor-facing search/pagination/limit.
- Saved Work view ≠ assignment or authorization truth.
- Schema/spec existence ≠ capability.

Do not add parallel authorization, task, workflow, event, worker, receipt, review, preference, document or dashboard stacks that duplicate existing foundations.

## 5. Current executable flow truth

```text
Matter
canonical state + canonical visibility
→ deterministic work compiler / canonical Action
→ current authority or accountable owner
→ existing Workflow projection
→ Today / Work
→ governed domain command
→ authoritative Matter aggregate
→ projection converges

Program
canonical current state
→ actor review checkpoint
→ exact baseline projection + bounded post-baseline history
→ review-by-exception digest
→ actor acknowledges exact current versions

Evidence Request
canonical recipient + subject visibility
→ recipient lifecycle
→ rebuildable Workflow projection
→ Today / Capture
```

Presentation/projection/acknowledgement state never substitutes for canonical domain truth.

## 6. Release gates

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
