# Governed form scoring and response-policy release evidence

Date: 2026-09-01

Branch: `codex/oversight-hierarchy`

Release state: pre-merge verification complete; deployed-commit acceptance is recorded separately after deployment.

## Outcome proved

The reference vendor-certification form stores a versioned compliance score profile. A completed poor response is scored Critical and disqualified, matches a governed response policy, creates one linked Matter, reuses that Matter for replay and a second poor response in the same adverse episode, closes the episode only after the Matter outcome is recorded, and creates a new Matter for a later adverse episode.

The journey uses the same form, response, policy, Matter, event, outbox, inbox and maintenance repositories as other records. There is no static API response or browser-only score calculation.

## Automated verification

| Gate | Result |
| --- | --- |
| `go test ./... -count=1` | Pass |
| `go test -tags postgres ./... -count=1` | Pass |
| `go vet ./...` | Pass |
| `go vet -tags postgres ./...` | Pass |
| `npm test` | Pass: 136 files, 887 tests |
| `npm run typecheck` | Pass |
| `npm run check:ui-contracts` | Pass: 10 checks |
| `npm run build` | Pass |
| Reference poor-response release journey | Pass |

The unconfigured workstation run of the `postgresintegration` suites skipped because `TEST_DATABASE_URL` was absent. That skip was not treated as database proof.

## Configured PostgreSQL proof

The exact branch migrations through revision 67 were applied to a temporary, isolated PostgreSQL database with memory-backed storage on the approved test host. The repository currently assumes a `uuidv7()` database function; the disposable PostgreSQL 15 image used a test-only `pgcrypto` compatibility function before migrations. Tests use explicit record identifiers, so this compatibility function does not affect score ordering, transaction, scope or deduplication assertions.

Server-local compiled tests from this branch produced:

| Package/proof | Result |
| --- | --- |
| Complete `internal/evidence` suite with `postgres postgresintegration` | Pass |
| Complete `internal/formpolicy` suite with `postgres postgresintegration` | Pass |
| 10,000 completed-response population, bounded index plan, legal-entity isolation, stable keyset pagination and warm query under 500 ms | Pass |
| Concurrent poor responses | Pass: one Matter, one open episode, newest response remains current |
| Execution, outbox, inbox, Matter, episode and outcome-check transaction | Pass |
| Replay, rollback compensation, authority-failure recovery, blast limits and maintenance lease recovery | Pass |

The database run found and fixed release-blocking defects that skipped tests had concealed:

- timestamp parameters used in interval arithmetic now have explicit PostgreSQL types;
- multi-statement fixtures use simple-protocol execution and valid revision ancestry;
- out-of-order concurrent responses cannot regress Matter or episode freshness;
- score indexes match the API's `DESC NULLS LAST` ordering;
- recipient reads scan and restore `scoring_mode` and `score_profile`;
- rollback tests compare transaction state with the existing event baseline.

The temporary container, database, copied migrations and compiled test binaries were removed after verification. No customer or production database was used.

## UI and workflow evidence

Builder scoring and the policy workspace were inspected in light and dark themes at desktop width and at 390 px. The review covered contribution/rule/band authoring, exact-revision preview, empty and unavailable policy states, effective dates, simulation, maker/checker actions, receipts and Matter links. Opening selects did not move the document layout, the builder panes retained independent scrolling, unavailable primary actions were not shown as enabled, and narrow inspector fields collapsed to one column.

Automated component and UI-contract tests cover the 320 px document-overflow contract, shared-control boundaries, focus-sheet stacking, primary-button contrast, typed scoring inputs, policy lifecycle states and sanitized errors. Hosted acceptance after deployment must recheck the materially affected workspaces at 320, 390, 768, 1280 and 1440 px in both themes before production promotion.

## Operational meaning

- `raw_score` is the profile's business score; `adverse_score` is normalized so higher always means more concern.
- A score is authoritative only for the exact stored form revision and profile checksum recorded on the completed response.
- Hidden fields are excluded and incomplete required evidence remains provisional or indeterminate rather than receiving a persuasive final number.
- Policy execution is current-response, legal-entity and authority-route scoped. Maker/checker separation, simulation freshness, effective dates, blast limits, shadow mode, rollback and outcome checks remain required.
- Repeated poor responses reuse an open adverse episode. A later response creates a new episode only after the earlier Matter has an independently recorded closure outcome.

## Remaining release gate

Merge and deploy this verified branch, record the deployed commit, run the hosted viewport/theme matrix, then execute the seeded certification journey against the deployed API and UI. Any failure keeps policy enforcement unavailable; a skipped database or hosted check is not a pass.
