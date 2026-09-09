# Finding follow-up proposal extension

The existing authenticated document form-proposal routes retain verified actor and legal-entity scope. No route, recipient-access policy or material authority is added.

`POST /api/v1/document-imports/{id}/form-template-proposals` accepts the existing `expected_document_version` and optional base revision. Omitting `finding_assessment_id` retains ordinary V2 generation. Supplying a source-derived assessment ID opts into one complete finding follow-up and cannot be combined with a base revision.

Available choices appear as `provenance.finding_assessments` with `id`, `label`, `sheet`, `row_start`, `row_end` and `finding_count`. At most 200 are returned. These are source assessment identifiers, not bank vendor, relationship or recipient identifiers. Unsafe grouping leaves the default proposal available with a source-correction warning; it does not infer missing identity.

The response remains an asynchronous 202 proposal receipt. The existing proposal GET exposes generation, review, failure, rejection and accepted draft results. Each exact source version/digest/assessment has its own retry-safe receipt. A selected proposal has `finding_assessment_id` and provenance version `FINDING_FOLLOW_UP_V2`.

`POST /api/v1/forms/proposals/{id}/accept` continues to require `expected_version` and `change_ids`. Selected-assessment proposals additionally require `assessment_confirmed: true` and every generated change ID. Partial, mixed or unknown IDs are rejected. General proposals retain selective acceptance. Accepted retries return the same draft; creation does not activate or send the form.

Rejection applies to the selected receipt. Independent assessments have no parent proposal whose rejection could race child acceptance. Exact source changes, proposal rejection, scope mismatch and lost draft-creation authority prevent a new draft through the existing command path. PostgreSQL acceptance keeps draft, form event, proposal update and outbox in one transaction.
