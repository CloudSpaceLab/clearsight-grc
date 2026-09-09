# Vendor compliance attention backend acceptance

Date: 2026-09-09. Scope: existing authorized vendor form list and aggregate; no material assessment, legal conclusion, risk rating or scoring policy is changed by these reads.

## Result contract

Each form row includes `attention_items` containing `label`, `state` (`MISSING`, `EXPIRED`, `GAP`) and `source` (`RESPONSE`, `REVIEW`), with optional `field_id` and `rule_id`. Labels come from the exact request fields or pinned scoring profile. Compound rules retain their configured label and rule ID. The browser translates `GAP` to “Not met”. No answer-language, form-name or sample-code heuristic determines failure.

`outdated` is true for expired current document evidence or a partially replaced submitted response; false requires known current document expiry dates; otherwise it is null. Expiry is compared with the UTC calendar date, inclusive of the expiry day. Invitation and request deadlines never determine response aging. Aggregate `outdated_forms` and `freshness_unknown_forms` count only authorized, current, submitted forms before pagination. Historical records do not affect these counts.

Missing required fields remain distinct from submitted negative answers. Reconciled current evidence satisfies the existing collection contract. A submitted questionnaire with admitted gaps remains submitted; independent review remains pending until the existing review workflow completes it.

## Scoring and review provenance

- Advanced contribution failures use the configured concern bands and score direction. A nonmaximum score does not automatically mean failure. Matched `FLOOR` rules operate on adverse score in both risk and compliance modes; `CAP` alone cannot establish a failure; contribution rules use raw configured points and direction. Disqualifying rules remain visible.
- Automatic findings appear immediately from the response result while review is pending. Only exact field IDs present in the current assessment snapshot's decisions can supply reviewed contribution findings. Partial review cannot relabel an unreviewed automatic finding as reviewed. Rules evaluated from immutable answers retain response provenance, including critical findings after favourable review.
- Reused vendor evidence is not described as reviewed merely because it was reused. Document review provenance requires a recorded validated, rejected or expired review state. Current document-review expiry overrides source expiry; an earlier expiry on the selected typed document still applies.
- Legacy due-diligence captures without a response revision derive requirement attention with the existing deterministic evaluator against their immutable submitted answers and exact stored profile. This does not populate a stored score or an assessed score. Their summary remains unassessed.
- Answer validation retains advanced automatic-review profiles. Existing collection requests that narrow an older scored template to document-only fields retain their prior field-validation behavior. Failed score configuration remains a scoring failure, rather than discarding an otherwise valid submitted response.

## Scope and query behavior

The existing current actor, tenant, legal entity, vendor relationship, workflow link and field-replacement predicates remain in force before pagination and aggregation. Document review joins use the exact tenant, entity, assessment, request and artifact. Reuse rechecks source currency and artifact identity/integrity with the existing collection predicate. The page retrieves facts and reviewed field IDs in its existing bounded query; it does not fetch response details once per row. Existing collection artifact/review refreshes remain unchanged. There are no new tables, caches, migrations, material writes or production authority defaults.

## Verification

Unit regressions cover missing fields, neutral unscored negative answers, configured concern thresholds, disqualification, compliance adverse floors, caps, contribution rules, date boundaries, unknown dates, partial replacement, real partial-review evaluation and legacy captures.

The custom-form workflow test creates and opens a distribution from an arbitrary active form revision, issues an OTP route, verifies access, saves negative answers, submits through the response workspace, then lists the vendor forms. It proves immediate response findings, submitted status, outstanding count zero and independent review pending. The governed form creation/approval lifecycle is separately covered by the default form installer tests.

Actual PostgreSQL 18.6 regressions cover page-size-one with a two-form population, current/expired/unknown dates, hidden expired fields, deadline separation, an arbitrary compliance rule before review, actor/tenant/entity/relationship isolation, current held evidence, held expiry, recorded expired/rejected review, and full/partial/supplemental replacement. The existing demo collection status/reminder regression also passes, preserving reuse and runtime artifact policy behavior.

Commands:

```text
go test -tags postgres ./internal/evidence ./internal/bankverticals ./internal/httpapi -count=1
go test -tags "postgres postgresintegration" ./internal/evidence -run "TestPostgresVendorForm|TestPostgresDemoCollectionStatus" -count=1
```

Receipts: `.codex-tmp/vendor-compliance-backend-unit-green.txt` and `.codex-tmp/vendor-compliance-backend-postgres-green.txt`. Independent backend review rechecked the provenance correction and reran the focused unit/workflow and actual PostgreSQL tests; no remaining P1/P2 finding was reported. UI rendering and hosted deployment evidence belong to the parent overview acceptance checkpoint.
