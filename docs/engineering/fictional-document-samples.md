# Persisted fictional document samples

This non-production demonstration lets a bank owner inspect the same submitted Northstar document occurrence from response detail, Forms Documents and Vendor Documents. It installs real governed records and immutable respondent submissions. It does not complete due diligence, authorize a vendor, validate document legitimacy or prove bank compliance.

## Explicit invocation

Deploy the ordinary API and worker first, including migration `000085_unbound_form_history_lookup`. Keep the default reference seeding invocation unchanged. Then run the existing API image's `/clearsight-seed-bank-reference` binary with its document-only flag (for example, through `docker exec` in the healthy API container):

```sh
/clearsight-seed-bank-reference -document-samples-only \
  -tenant <existing-tenant-UUID-or-slug> \
  -legal-entity <existing-entity-UUID-or-code> \
  -actor <current-owner-principal-UUID> \
  -owner <same-current-owner-principal-UUID> \
  -reviewer <distinct-current-checker-principal-UUID>
```

Use the deployment's existing protected environment and mounted artifact directory. Required configuration is non-production `CLEARSIGHT_ENV`, `CLEARSIGHT_DEMO_MODE=true`, `DATABASE_URL`, an explicit durable `CLEARSIGHT_ARTIFACT_ROOT` shared with the API/worker, `CLEARSIGHT_CAPTURE_PUBLIC_BASE_URL`, `CLEARSIGHT_RECIPIENT_ACTIVE_KEY_ID`, `CLEARSIGHT_RECIPIENT_KEYRING` and `CLEARSIGHT_DISTRIBUTION_ACCESS_HMAC_KEY`. Never pass key material on command lines. The normal command timeout is two minutes; pending assessment setup reports a retryable installer error, and the next invocation resumes its stored assessment after the worker is ready.

The command resolves active principals and current entity membership, canonicalizes scope, and enforces the existing effective authority routes. The maker must own vendor/form/assessment commands; the checker independently activates the form. It does not create principals, roles, authority routes or unrelated reference journeys, and it does not run shared projection or maintenance queues. Missing identity, authority, configured storage or recipient protection stops installation.

## Stored scenario

The dedicated source is `fictional_document_samples_v1`, vendor reference `northstar-infrastructure-documents-v1`, and unbound form code `SAMPLE-NORTHSTAR-DOCUMENTS-V1`. The service and form explicitly identify sample data. Existing `reference_data/vendor:managed-infrastructure`, scoring fixtures, operator forms and Program-bound forms sharing the code retain their separate identities.

The installer uses the normal draft → maker submission → distinct-checker activation commands, starts a canonical onboarding assessment, and waits for its exact worker-created review issue. Its request uses `THIRD_PARTY_ASSESSMENT` origin and the fictional `.invalid` audience. Direct delivery is absent; the existing secondary communication worker recognizes workflow-owned origin and records a skip without sending mail. Route/session credentials remain in memory and never appear in the JSON receipt.

Six reviewed embedded PDF/PNG/XLSX files are uploaded through an ordinary respondent session. The first immutable response contains the prior security declaration plus four other documents; the replacement response contains the current declaration and reuses those four documents. There are five current occurrences and ten historical occurrences. Typed reference and issue/review/expiry metadata agree with the authored files; actual uploads and submissions use current timestamps. The prior security declaration retains its expired date; the insurance intentionally names Brooklane Hosting Limited, and the register retains its three differing review dates. Archive-exercise evidence and independent address proof are omitted optional answers. No signature is invented.

All artifacts remain `STORED_UNSCANNED`. The separate exact-manifest demo preview exception displays **No antivirus scan was performed** and does not create a scan receipt or alter review eligibility. No clean inspection, review, acceptance or approval is synthesized.

## Recovery and limits

A tenant/entity-scoped advisory lock serializes installers. Exact source identities locate the vendor and relationship. Exact unbound form-code history reads inspect at most four revisions; migration 000085 adds the missing nonunique index for draft/pending history. Artifact, submission and response-revision reads stop at the expected population plus one. The installer compares full form/request contracts, known facts, typed answers, edit sequence, response chain, first-submission artifact membership and the actual stored bytes before resuming access.

Matching partial drafts, request persistence, uploads, first submission and second draft resume through existing services and optimistic versions. Two matching submitted revisions return an unchanged receipt population without another upload, route, session or submission. Unknown/extra rows, edited values, altered bytes, changed scope, paused forms, locked distributions and access revoked before expiry stop for inspection. Identical content alone does not establish who uploaded a historical draft artifact: the installer makes no uploader attribution beyond the existing capture model. An ambiguous access revocation is not silently reversed.

The normal workspace submit event currently does not advance the linked assessment consumer. Consequently these real responses remain readable while the assessment stays `COLLECTING`. Fixing event bridging, revision ordering, review freshness and activation is separate #139 work. The sample installer never rewrites assessment state to hide this limitation. This is locally integration-tested demonstration maturity; hosted release/recipient acceptance and production storage/scanning remain separate gates under #138/#139/#147. #200 remains closed.

## Acceptance evidence

Run only against a disposable migrated PostgreSQL database; the command package's integration fixture truncates tenants like existing repository integration tests:

```sh
go test -count=1 -p 1 -tags "postgres postgresintegration" ./cmd/seed-bank-reference
```

The suite uses real effective authority, distinct actors, normal assessment setup, respondent upload/save/submit, all three document entry points and current/history access. It covers ten interruption checkpoints (including an injected request-link transaction failure), concurrent installation, exact-row no-write rerun hashes, changed contracts/answers/bytes, unknown or excess records, missing authority, command-specific send-authority revocation after a partial installation, revoked access, non-demo/production refusal, unrelated-record preservation and the secondary worker's no-email receipt.

The index test applies, rolls back and reapplies migration 000085. With 1,500 other draft forms and the normal planner, the exact query used `monitoring_form_templates_unbound_history_idx`. This proves the lookup path on the fixture, not production-scale latency. UI preview and release evidence remain in the [sample plan](../superpowers/plans/2026-09-08-demo-document-samples.md) and [demo preview decision](../design/2026-09-08-demo-document-preview.md).
