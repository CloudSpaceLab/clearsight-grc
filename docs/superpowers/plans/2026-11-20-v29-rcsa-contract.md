# RCSA 2.9: existing contracts and implementation design

**Status:** Partial contract investigation. Proposed behavior is not implemented, and this is not yet a complete code-ready plan. It supplements the [domain plan](2026-11-20-v29-full-rewrite-domain-plan.md).

## Existing API and ownership verified

`internal/httpapi/rcsa_routes.go` registers these exact routes:

| Method/path | Existing request | Current route policy |
| --- | --- | --- |
| `GET /api/v1/rcsa/cycles` | Optional `status`, `cursor`, `limit` | Authenticated read; bounded page. |
| `GET /api/v1/rcsa/cycles/{id}` | No body | Authenticated exact read. |
| `POST /api/v1/rcsa/cycles` | `code`, `name`, `trigger_kind`, `risk_ids` | `RCSA_CYCLE`, owner, materiality 3, legal entity bound; handler overwrites owner and actor from verified identity. |
| `POST /api/v1/rcsa/cycles/{id}/first-line-distribution` | `{ "expected_version": 1, "distribution_id": "uuid" }` | Cycle owner, materiality 3. |
| `POST /api/v1/rcsa/cycles/{id}/first-line-complete` | `{ "expected_version": 2 }` | Cycle owner, materiality 3. |
| `POST /api/v1/rcsa/cycles/{id}/challenge/start` | `{ "expected_version": 3 }` | Cycle reviewer, materiality 3. |
| `POST /api/v1/rcsa/cycles/{id}/challenge/complete` | `{ "expected_version": 4 }` | Cycle reviewer, materiality 3. |

Cycle state is `DRAFT`, `ASSESSMENT_OPEN`, `AWAITING_CHALLENGE`, `COMPLETED` or `CANCELLED`. `internal/rcsa/model.go` retains a population checksum, exact first-line distribution/response revision and challenge Matter. Migration `000103_rcsa_cycles.up.sql` retains frozen risk and control membership with versioned source references, immutable revisions and events. `CompleteFirstLine` resolves an actual submitted response revision; `StartChallenge` rejects the first-line owner and resolves a challenge Matter; `CompleteChallenge` validates a decision via the configured resolver. `web/src/rcsaApi.ts` currently exposes only list and detail readers.

The reference wants per-risk resume, pinned evidence and independent decisions guarded against changed control/evidence/rating context. The legacy response draft is keyed by tenant/request/session and `SaveDraftInput` is `{answers, presentation_mode, expected_version}` (`internal/evidence/draft.go` and `draft_postgres.go`). A distinct distribution workspace is uniquely keyed by distribution and records per-field edit provenance (`migrations/000054_form_distributions.up.sql`); its `PATCH /api/v1/evidence/session/workspace` body is `{expected_version, presentation_mode, edits:[{field_id,value,base_sequence}]}` and its submission body is `{expected_version, attestation_field_ids?}` (`internal/evidence/response_workspace.go`). Both use capability sessions, so neither observation establishes that a staff member can safely resume per-risk work after session replacement. Verify that path before adding another assessment store.

`configureRCSAFirstLine` in `cmd/api/services.go` requires a distribution with subject type `RCSA_CYCLE`, the exact cycle ID and an active internal first-line-owner recipient. Completion requires a completed distribution and a current FINAL response revision. `configureRCSAChallenge` reuses or creates a Matter with deterministic trigger key `rcsa-challenge:<cycle-id>` and ensures an `RCSA_CHALLENGE` decision is present. `internal/rcsa/postgres.go` opens its own transaction for `UpdateCycle` after that resolver returns. The code path therefore performs at least the resolver operations before opening the cycle update transaction; assess failure/retry and actual persistence behavior before asserting the full material operation is atomic. The trigger key aids duplicate recovery but does not itself prove atomicity.

## Contract decisions to implement

1. Preserve the existing cycle and frozen membership as the population owner. Exact risk and implementation versions must come from the cycle snapshot, not from a new broad search or current UI state. Keep source and legal-entity authorization at each read and command.
2. Inspect `internal/evidence/draft_postgres.go`, capture request/session/recipient ownership and `cmd/api/services.go` (`configureRCSAFirstLine`) to establish whether the existing draft can safely represent each cycle/risk, support staff session recovery, carry optimistic concurrency and preserve source versions. If it can, use the existing draft endpoints and capture record; document their real DTO and exact session scope. If it cannot, design the minimum versioned scoped extension to the existing capture owner with `(tenant, legal_entity, cycle, risk, recipient)` identity and do not introduce a second authoritative response.
3. Bind each submitted assessment to the frozen cycle/risk/control versions, exact submitted response revision and authorized evidence versions. At independent decision, recheck reviewer authority, segregation, current evidence access and any policy-required effective rating or control-version guard. A stale basis returns a conflict and does not mutate an approved rating.
4. Keep the existing challenge Matter and decision when it represents the intended decision. Determine whether the reference's per-risk review maps to one cycle-level decision or requires independently reviewable risk-level decisions. That granularity is not resolved by the reference-to-backend inspection so far. If additional durable records are necessary, specify their schema and route policy only after this comparison; keep a typed link to the cycle and source submission and avoid duplicate conclusions.
5. Audit `StartChallenge` end to end: `internal/rcsa/service.go` calls the configured challenge resolver before `repo.UpdateCycle`, and `cmd/api/services.go` can call `matters.CreateMatter` before the cycle is updated. Trace the actual transaction/context implementation and failure/retry behavior. Do not assume atomicity from a successful integration test. If both are independent commits, provide a single transaction boundary or an explicitly authorized recovery design satisfying the project command-integrity rule; test an injected failure after Matter creation and a concurrent start.
6. Keep existing API paths backward compatible. Proposed client actions must use the actual request DTOs above and never supply an actor. Add any new paths to the route registry, access descriptor and mechanically verified runtime OpenAPI in the same change. No new endpoint names or SQL migration are approved here.

## Required test additions and execution

- Add a regression in `internal/rcsa/service_test.go` for frozen population/version after a risk/control change, preserving the original assessment basis.
- Add tests for draft interruption/resume, revoked recipient, stale response revision, stale evidence, independent reviewer conflict and correction without overwriting an accepted rating. Use the actual capture/decision contract identified above.
- Extend `cmd/api/rcsa_challenge_postgres_integration_test.go` for failure between challenge creation and cycle binding, two concurrent starts, replay and point-in-time source preservation. Assert transactional business row/event/outbox behavior rather than only the final Matter ID.
- Update `web/src/components/rcsa/RCSAWorkspace.test.tsx` and `web/src/rcsaApi.ts` tests for actual transition DTOs, exact deep-link, reload, Back, denied/stale/source-unavailable and responsive review states. Use source screenshots and hashes at all approved widths; do not treat historical 2.7 screenshots as 2.9 evidence.

Run from repository root `go test ./internal/rcsa ./internal/evidence ./internal/continuity ./internal/httpapi ./cmd/api`. Run `go test -tags "postgres postgresintegration" ./internal/rcsa ./cmd/api` with a migrated isolated `TEST_DATABASE_URL`, and inspect skips. From `web`, run `npm test -- --run src/components/rcsa/RCSAWorkspace.test.tsx src/copyQuality.test.ts` and `npm run typecheck`; add browser evidence and source-to-Work checks. These are prescribed commands, not test results.

## Remaining work before this becomes code-ready

Trace capture draft DTOs and storage, challenge transaction boundaries and route resolution, reviewer decision granularity, evidence-version predicates and current schema beyond migration `000103`. Then write exact request/response schemas, any necessary paired SQL migration and tests before changing implementation. Bank policy must determine independent decision granularity and any required source-drift response; do not invent an approved policy or a historical reviewer.
