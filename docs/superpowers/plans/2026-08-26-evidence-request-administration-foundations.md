# Evidence Request Administration Foundations Implementation Plan

**Goal:** Deliver safe request creation and requester administration contracts, including one-use external invitation handling, so the Work UI can operate evidence requests without raw IDs or privileged back channels.

**Architecture:** Add an exact subject-scope resolver and canonical entity to requests, authorize creator and requester commands against current subject access, make capability changes transactional, expose sanitized bounded reads, then compose a separate requester administration workspace.

## Task 1: Close browser token lifecycle gaps

- Add failing browser tests proving the invitation is removed from the URL, storage keys cannot be derived from it and terminal/revoked/expired sessions are cleared.
- Exchange/bootstrap the capability, immediately replace browser history, use an opaque random request-local key and delete terminal session state.
- Use generic customer-visible failures and distinguish submitted evidence from verified evidence.
- Pass focused browser and copy-quality tests.

## Task 2: Canonical subject and creator authorization

- Add failing domain tests for creator without subject access, unsupported subject type, entity mismatch and recipient without current eligibility.
- Add a narrow exact `SubjectScopeResolver`; remove fail-open non-Matter behavior.
- Persist canonical `legal_entity_id` with migration `000037` and bounded indexes.
- Bind verified tenant/entity/actor and ignore client actor fields.

## Task 3: Transactional requester administration

- Add failing tests for requester-inclusive list, requester-authorized revoke/replace/reassign/cancel and lost subject access.
- Commit request/recipient/capability/history/event/outbox/projection changes in one transaction.
- Reassign/cancel/replace must revoke superseded invitations and sessions atomically.
- Expose only sanitized invitation metadata; token and hashes remain write-only/secret.

## Task 4: Verified, bounded HTTP contract

- Add requester and respondent keyset queues filtered by entity/subject before limit.
- Add create, eligible-recipient, invitation issue/list/replace/revoke, reassign and cancel routes with distinct capabilities.
- Return generic capability failures and update OpenAPI/route-registry coverage.
- Pass HTTP security and contract tests.

## Task 5: Evidence request administration workspace

- Add failing React tests for request creation, labelled recipient selection, invitation issue/replace/revoke, reassign, cancel and response review.
- Add a separate `EvidenceRequestAdminWorkspace`; keep the respondent workspace focused on answering.
- Refresh the exact request after every command and show current recipient, deadline, outstanding fields, invitation state and next action.
- Add copy-quality, accessibility, audience semantics, responsive fixtures and rendered evidence.

## Verification

Run evidence packages with standard and PostgreSQL tags, HTTP/OpenAPI tests, focused URL/session browser tests, web unit/type/build/copy-quality tests, then requester/respondent render and interaction suites.
