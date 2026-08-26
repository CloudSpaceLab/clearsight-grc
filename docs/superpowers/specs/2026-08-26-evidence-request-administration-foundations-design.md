# Evidence request administration foundations

**Status:** Approved direction
**Date:** 2026-08-26
**Delivery:** Secure request creation and administration before expanding the Work UI

## Decision

ClearSight will make evidence requests operable end to end from the UI: an authorized user selects an exact unresolved need, chooses an eligible recipient, sends or replaces a time-bound invitation where required, monitors the response, corrects the recipient or cancels the request, and sees reconstructable history. No step requires SQL, an API client or a raw principal ID.

The first increment closes the authorization and token-lifecycle gaps that would make those controls unsafe. It adds canonical entity/subject scope, creator and requester authorization, requester-inclusive bounded reads, and possession-bound invitation administration with sanitized metadata.

## Current defect

The response workspace exists, but there is no request-creation or invitation-administration journey. Reassignment asks for a raw principal ID. Invitation tokens can remain in the browser URL and influence storage keys. Revocation is tenant-scoped rather than requester-authorized. Request creation checks the recipient but not the creator's access to the subject, and PostgreSQL subject checks fail open for several non-Matter subject types.

## Operating contract

### Canonical request scope

- Every request stores tenant, legal entity, supported subject type and canonical subject ID.
- A narrow subject-scope resolver performs an exact indexed lookup. Unknown or unsupported subjects fail closed.
- The verified creator must be allowed to view and request evidence for the subject.
- The selected recipient must be currently eligible for the audience, entity, sensitivity and subject.
- Candidate selection is server-derived and labelled; the browser never supplies the command actor and does not require a raw recipient ID.

### Command atomicity

Create, reassign, cancel, invitation replacement and terminal submission each commit their authoritative rows, recipient/capability state, append-only history, event/outbox entry, projection job and version change in one transaction. Reassignment or cancellation revokes superseded invitation and session capabilities in that transaction.

### Invitation administration

- Invitation issue returns the raw token exactly once to the authorized requester for delivery.
- List and status reads expose only invitation ID, audience, state, issued time, expiry, replacement/revocation time and delivery status. Token, token hash and session token never appear.
- Replace/resend atomically revokes the prior invitation and sessions before issuing a new capability.
- Revoke, replace and status reads require requester/manage authorization plus current subject visibility.
- External bootstrap removes the raw token from the address bar immediately, uses an opaque random local key unrelated to the token, and clears session state after submission, revocation, expiry or unrecoverable failure.
- Generic failures do not reveal whether a token, request or recipient exists.

### Requester and respondent views

Requester queues include requests created by the verified actor that remain visible in the actor's entity and subject scope. Respondent queues include requests assigned to the actor. Both are keyset-paginated and bounded. A request detail identifies the current recipient, deadline, remaining information, invitation state, latest response and next valid action.

## UI delivery after the foundation

The Work workspace will add a separate Evidence request administration surface rather than enlarging the respondent form. It will provide `Request evidence`, `Change recipient`, `Send invitation`, `Replace invitation`, `Cancel request` and `Review response` actions only when the server returns the matching actor-visible operation.

Audience handling is semantic: internal people, invited external contributors and other supported audiences receive the correct workflow. UI logic must not rely on the literal value `EXTERNAL`. Copy distinguishes submission from independent verification.

## Failure behavior

Missing identity, unsupported subject, entity mismatch, lost subject access, ineligible recipient, stale version, expired capability or persistence failure returns a non-committed command failure. After a committed command the UI reloads the exact request; delayed projection work cannot reverse reported success.

## Acceptance proof

- Domain tests cover creator access, recipient eligibility, requester management and terminal capability revocation.
- PostgreSQL tests prove pre-limit entity/subject filtering and transactional request/history/event/outbox/projection changes.
- HTTP tests prove verified actor/entity binding and generic invitation failures.
- Browser tests prove URL scrubbing, non-derived storage keys and session cleanup.
- UI tests prove create, invite, replace, reassign, cancel and response-review journeys without raw IDs.
- Copy-quality, accessibility, responsive render and degraded-authority states pass before completion.

## Non-goals for this increment

Protected reporting, OTP identity proofing, multi-contributor packets and team/role delegation require separate identity-isolated designs. This increment does not label possession-bound submission as verified identity or verified outcome.
