# Finding follow-up branch: verified reuse decisions

## Source revisions and merge result

Fetched origin on 9 September 2026. Main remains `676995b16b2e58b0dd217ba8845e3320a42a5da6`; `feature/finding-followup-proposals` remains `6ebe0746ec9ba5f64b6a172d634bca70a219bc9b`; common ancestor is `60a6a6065b7ca6de7f52b66dd24c36fb19fc0904`. The current user checkout has separate demo/document commits `98cc62b0` and `32864109` on PR #212. This work starts in an isolated checkout of fetched main.

`git merge-tree --write-tree origin/main origin/feature/finding-followup-proposals` exits 1. Its eight conflicts are XLSX extraction, PostgreSQL proposal acceptance, assessment review, VendorDueDiligence component/tests, VendorsWorkspace and FormProposalReview component/tests. There is also a migration collision: branch `000082_finding_followup_proposals` and main `000082_response_field_assessments`. A clean textual merge alone would not establish compatibility.

## What is already present

Main's V2 importer retains structured spreadsheet cells and proposes one question per recognized requirement/finding row. Existing acceptance records prove 47 checklist questions and five historical finding questions through real extraction. Exact row anchors, historical deadlines, unresolved scope warnings, existing vendor evidence reuse, bank field assessment and current proposal/draft recovery must survive this change. Those workflows are not missing merely because the feature branch implements an older alternative.

## Copy/adapt/exclude map

| Candidate | Exact source | Decision and required adaptation |
| --- | --- | --- |
| Vertical merged cells | `internal/documentimport/xlsx_merges.go`, `xlsx_extractor.go` | Adapt the bounded range approach. Preserve both display Sections and V2 `Elements.Values`; propagate only an explicit vertical range into existing rows. Validate coordinates, overlap and contradictory populated values within inherited vertical ranges; retain extraction budgets and increment parser provenance. Horizontal values remain explicit and unpropagated. |
| Correct source quote | `web/src/components/forms/FormProposalReview.tsx:sourceExcerpt` | Adapt matching to all retained row/location anchors. Main currently repeats the first row's quote for subsequent rows on the same worksheet. Preserve the current scrolling region, error recovery and null-array handling. |
| Competing generation completion | `internal/monitoring/form_proposal_service.go:Generate` | Adapt conflict recovery to return only the latest same-source, same-scope durable receipt. Failed generation remains failed; a failed reload is not a successful proposal. |
| Five response fields per finding | `internal/documentimport/finding_followup.go` | Rebuild using structured cells. Retain the useful response/action/owner/proposed-date/evidence pattern as an explicit optional follow-up mode. Do not copy its text parser or implicit blank-cell grouping. |
| One assessment per draft | `internal/monitoring/finding_followup_selection.go`, `FormProposalReview.tsx` | Retain the design: explicit assessment choice/confirmation and one complete group. Validate unknown IDs and partial/mixed selection at the server. Do not infer a vendor identity or sending recipient. |
| Separate reusable draft receipts | `form_proposal_service.go`, `form_proposal_memory.go`, `form_proposal_postgres.go`, branch migration | Adapt only with scoped generator/group identity, migration renumbering and real PostgreSQL concurrency tests. Parent rejection/source change must be checked at child acceptance; retry must return the same draft. Existing atomic draft/event/outbox acceptance stays intact. |
| Workflow access status | `distribution_access_service.go:ListActiveRouteMetadata`, invitation admin handlers/UI | Useful future integration. Determine workflow ownership independently of whether an active link exists, so expired/revoked links do not re-enable the wrong invitation path. Preserve the current authorized subject-read check. |
| Request submitted while amendments remain open | `response_workspace_memory.go`, `response_workspace_postgres_write.go`, `service.go` | Do not port the current implementation. Same-respondent amendment regression reproduced below. A future change must coordinate request status, collection revalidation, response revisions, bank review routing and request-version provenance in one transaction. |
| Tenant comparison relaxation | `form_proposal_service.go`, `form_proposal_postgres_accept.go` | Exclude. Current document and proposal PostgreSQL projections both expose `t.slug`. Main's atomic acceptance already resolves UUID/slug equality against the same tenant row. Deleting equality checks is not needed. |
| Invitation selector bridge | `internal/httpapi/evidence_handlers.go:attachDistributionAccessSelector` | Exclude. It issues a legacy invitation and then another access route, substituting `routes[0]` into the first receipt. This does not establish consistent recipient, expiry and revocation semantics or one atomic command. |
| Wholesale assessment/workspace changes | `assessment_review.go`, `response_workspace_*`, `service.go` | Exclude. Direct replacement drops current bank `field.Assessment`, held-evidence revalidation, subject visibility checks and scored-event request ID/version metadata. |

## Reproduced branch defects

The isolated feature-branch checkout was tested without changing its production code. New probes use synthetic fixtures and make assertions about actual branch functions, not a rewritten approximation. Reproduction sources are retained under `docs/evidence/2026-09-09-finding-followup-salvage`.

1. **Cell text misread as a column:** a finding containing a newline followed by `Column 10: quoted example inside the finding` loses that text. `retainedColumns` treats human source text as a structural delimiter. Use `Elements.Values` instead.
2. **Wrong vendor group after a heading row:** a new numbered assessment row with vendor/service/date but no finding is skipped before group context updates. A following finding with blank context is assigned to the preceding assessment. Handle every boundary row; require explicit source linkage or reviewer mapping.
3. **Wrong recorded owner:** a second finding with a populated different `RESPONSIBILITY` silently displays the first finding's responsibility. Preserve row-level responsibility; unknown and conflicting context must remain explicit.
4. **Same respondent cannot amend:** first submission changes the request to `SUBMITTED`; a subsequent amendment by the same recipient reaches the memory store's `requestOpenAt` check, which accepts only READY/IN_PROGRESS. The existing branch amendment test uses a different recipient with a different request and misses this. The identical added same-recipient regression passes on current main and fails on the branch.

Commands on the feature revision:

```text
go test ./internal/documentimport -run TestSalvageAudit -count=1
FAIL: ColumnMarkerInsideFinding, AssessmentHeaderWithoutFinding, RowSpecificResponsibility

go test ./internal/evidence -run TestSalvageAudit -count=1
FAIL: SameRespondentCanAmendSubmission (response workspace is unavailable)

go test ./internal/evidence -run 'Workspace|DistributionAccess' -count=1
PASS: existing branch tests, demonstrating the missing same-recipient case
```

Additional code-review concerns for grouped follow-up: limits apply across all groups before selecting one; long source context aborts the specialized generator instead of retaining an editable general proposal; detection replaces the current import mode automatically; child acceptance re-enters the service outside the parent's version/rejection transaction. These are review findings, not separately reproduced runtime failures.

## Follow-on implementation contract

Track these remaining items under the existing third-party lifecycle issue #80, particularly its legacy-register import/reconciliation scope; #200 remains closed. This release does not close #80 or establish grouped follow-up completeness.

- Keep current row-based drafts as the default. Offer “Prepare finding follow-up” only for recognized structured source rows. Missing extraction, ambiguous ownership, malformed or contradictory ranges must produce actionable review needs.
- Preserve source row, finding, recommendation, historical date/status and per-finding responsibility. Never derive structure by parsing displayed source quotes. Do not silently inherit unmerged blank cells.
- Select one assessment and confirm its vendor/service/source details. Preview the response/action/owner/date/evidence fields and retain the full source alongside bounded question help. Apply field/section limits to the selected assessment. Do not copy internal ratings into editable responses or add signatures unless the form requires them.
- Use existing proposal receipts and ordinary governed form drafts. Allocate the next migration only after a fresh fetch; verify migration up/rerun and historical rows. Cover duplicate acceptance, concurrent workers, source edits, parent rejection, lost authority and cross-entity attempts with PostgreSQL tests.
- Keep collection receipt, submitted response, document review, form activation, vendor approval and finding closure separate. Use the existing distribution sender to choose the relationship/recipient and send only after ordinary form approval.
- Repair workflow access ownership and submission-status integration separately, preserving main's collection checks, immutable response versions, typed event payload and revocation behavior. Cover same and different recipients, deadline/lock/cancellation and current bank review access.

## Release verification

See the [implementation plan](../superpowers/plans/2026-09-09-finding-followup-salvage.md). Final test, render and CI receipts are recorded when they finish; the failing feature-branch probes above are expected audit results, not failures in the release implementation.
