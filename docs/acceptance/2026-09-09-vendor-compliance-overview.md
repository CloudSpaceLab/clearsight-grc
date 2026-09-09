# Vendor compliance overview acceptance

The vendor overview now lists failed configured requirements immediately after a custom form is submitted. Submission, response findings, independent review and the last due-diligence conclusion remain distinct. The default sample checklist permits declared gaps without requiring an invented upload. See the [decision](../design/2026-09-09-vendor-compliance-overview.md), [backend acceptance](2026-09-09-vendor-compliance-backend.md), [sample form acceptance](2026-09-09-third-party-compliance-form.md) and [rendered evidence](../evidence/2026-09-09-vendor-compliance-overview/README.md).

## Local verification

- Five affected web suites pass: 130 tests covering the overview, Vendors workspace, vendor Forms panel, application and copy-quality gate.
- TypeScript project build passes; the UI contract, flow-manifest and Forms scenario checks pass all 51 tests.
- Fresh `go test -tags postgres ./internal/evidence ./internal/bankverticals ./internal/monitoring ./internal/httpapi -count=1` passes.
- The complete `go test -tags postgres ./...` suite passes after updating the API demo boot test's expected sample-form set from three to four. The targeted boot test failed before that expectation change and passed afterward; the initial CI failure is retained in its run history.
- The backend and seed acceptance records include actual PostgreSQL tests and the custom distribution → OTP → response save → submission → vendor overview API workflow.
- Independent backend review cleared the source-attribution correction with no remaining P1/P2 findings.
- Visual evidence includes desktop, 390px and 320px, both themes, keyboard review handoffs, unknown/error states and existing evidence reuse. The broader Forms runner preserves an initial performance failure and subsequent verification separately; its evidence README records the precise outcome.

## Hosted Forms checkpoint

The earlier Forms fix deployed successfully as `b84ae49b7a7f484eb6dacf91c2122035bfab2ca0`, with API and worker revision matches and `/health/ready` reporting ready. In the hosted CRO session, the existing user-created Third Party Risk Compliance draft opens through the direct Edit draft table action. Back to Forms returns to the library, and Details exposes Edit draft and Send for approval at the top. No form content was saved during this check.

Vendor overview merge, exact-revision deployment and hosted default-form checks remain release steps. Local and fixture verification does not certify the hosted vendor data or a legal compliance conclusion.
