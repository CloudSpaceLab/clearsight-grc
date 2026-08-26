# Evidence Request recipient boundary

**Status:** #27.2b-B1 backend recipient contract — complete  
**Issue:** #27  
**Implementation:** PR #49

This document defines who an Evidence Request is actually assigned to. It separates recipient assignment from subject visibility, request creation authority, and invitation/session capability security so ClearSight does not infer work ownership from descriptive copy.

## 1. Canonical rule

`capture_requests` owns exactly one intended recipient for every newly created request.

Supported recipient modes in this tranche are deliberately narrow:

| Request audience | Recipient contract | Stored identity |
| --- | --- | --- |
| `INTERNAL` | `INTERNAL_PRINCIPAL` | exact active internal `PERSON` principal ID |
| `INVITED_EXTERNAL` | `EXTERNAL_AUDIENCE` | SHA-256 of normalized audience + masked display hint |

The raw external address/audience is not stored on `capture_requests`.

A request is not assigned merely because:

- `why_you` describes a person;
- `created_by` names the requester;
- the signed-in actor can read the subject Matter;
- an invitation was sent to an address;
- a respondent submitted a prior answer;
- an actor belongs to a similarly named function in display copy.

## 2. No unsafe historical backfill

Migration `000021_capture_recipient_truth` leaves existing rows with `recipient_type = NULL`.

There is no safe deterministic reconstruction from `created_by`, `why_you`, invitation text, prior submitter or current subject visibility. Legacy null-recipient rows therefore remain current/audit records but do **not** enter recipient actor queues and cannot be treated as newly assigned work.

Banks must recreate/replace a legacy request under the canonical recipient contract rather than silently rewriting history.

## 3. Internal recipient eligibility

A new `INTERNAL_PRINCIPAL` recipient must:

1. belong to the same tenant as the request;
2. be a current `ACTIVE` principal;
3. be a `PERSON` principal in this tranche;
4. be valid at assignment time;
5. be able to read the request subject where subject access is governed.

The database foreign key is tenant-composite so direct SQL cannot attach another tenant's principal to the request.

### Why no team/role/queue recipient yet

The repository models principal kinds such as roles/teams/queues, but it does not currently have one executable, current membership-resolution contract that can safely turn such a recipient into the actor who should see/respond to an Evidence Request.

This tranche therefore does not advertise group assignment that the product cannot resolve. Group/position routing may be added later only through a real directory/membership contract rather than display labels.

## 4. External recipient identity

`EXTERNAL_AUDIENCE` stores:

- a constant-size SHA-256 audience hash for matching;
- a masked `recipient_hint` for safe display.

The raw audience supplied when creating the request is normalized only to calculate those values. It is not retained in the request row.

Invitation issuance must match the request's existing recipient hash. Supplying a different address cannot mutate or reinterpret the request recipient.

## 5. Requester authority is separate from recipient assignment

The authenticated requester remains `created_by`; this is not the recipient unless the request explicitly says so.

Where subject access is governed, an authenticated requester must be able to read the subject before creating either an internal or invited-external request.

For the first external-recipient tranche, only the canonical requester may issue an invitation for that request. A correct external audience hash is not permission for another tenant actor to issue someone else's invitation.

Later delegated request-management authority must be explicit rather than inferred from broad subject visibility.

## 6. Actor queue boundary

Internal actor work requires **both**:

```text
request.recipient == signed-in principal
AND
signed-in principal can read the request subject
```

PostgreSQL applies both predicates before `LIMIT` for the recipient-facing request queue. Other actors' earlier-due requests and legacy null-recipient rows therefore cannot crowd the current actor's own work out of a bounded page.

Exact internal request reads enforce recipient match before subject visibility. A public/readable Matter does not make every request about that Matter visible to every actor.

## 7. Submission boundary

Authenticated `Submit` is an internal-recipient path only:

```text
submitted_by == canonical internal recipient
```

Caller-supplied `channel` text cannot convert an authenticated submission into an external capability submission.

External responses use `SubmitSession` only. That path first resolves a current capability session and the canonical external request, then executes the common validated submission logic.

Submission identity therefore does not create recipient truth; it must already match recipient truth.

## 8. Invitation and session boundary

Invitation/session rows remain capability-security state, not recipient-assignment state.

External flow:

```text
canonical EXTERNAL_AUDIENCE request
→ requester issues invitation for the same audience hash
→ token + audience proof redeem once
→ short-lived capability session
→ session resolves the same current request
→ response / artifact capability
```

Redeeming an invitation is followed by a current request-recipient check. A capability cannot be used to reinterpret an internal request as external.

Recipient redirection is intentionally outside B1. Before B2 permits external recipient replacement, session/invitation invalidation must ensure old capabilities cannot survive a recipient change; display hints alone must never be treated as sufficient identity for that transition.

## 9. Artifact boundary

An authenticated artifact upload is permitted only for the current internal request recipient.

The external upload route already validates a capability session before the service is invoked; the service additionally requires the request itself to remain `INVITED_EXTERNAL` with an external recipient.

An attachment is evidence submitted by a recipient. Upload permission therefore follows recipient/capability truth rather than general subject readability.

Form uploads are also bound to the selected request field. The upload route applies the field's format and per-file limit after recipient authorization; submission rechecks every artifact against the request, field, count and combined-size rules. Content-derived media type and bounded PDF/Open XML inspection reduce mislabeled or active-content uploads, but do not replace malware scanning. Newly stored artifacts therefore retain `STORED_UNSCANNED` until a separate security decision is recorded.

## 10. Failure behavior

Fail closed:

- missing recipient on a new request → validation failure;
- cross-tenant/inactive/non-person internal recipient → invalid recipient;
- recipient lacks subject access → invalid recipient;
- requester lacks subject access → request creation denied;
- non-recipient internal list/read/submit/upload → no recipient work / denied;
- wrong external invitation audience → denied;
- non-requester external invitation issuance → denied;
- legacy null recipient → excluded from actor work;
- expired/cancelled/submitted request → existing request lifecycle rules remain authoritative.

## 11. Acceptance evidence

B1 is complete only when exact-head CI proves:

- migration `000021` applies, rolls back and reapplies;
- tenant-composite internal-recipient integrity;
- active-person recipient validation;
- restricted Matter recipient validation;
- recipient + subject predicates occur before queue limits;
- other-recipient and legacy requests cannot leak into the actor queue;
- authenticated callers cannot spoof an external channel to submit another actor's request;
- external request rows contain hash/hint, not the raw audience;
- invitation requester and audience must match canonical request state;
- correct external invitation redemption/session flow still works;
- existing artifact, request-expiry, optimistic-version and field-contract tests remain green;
- web/type/accessibility/rendered-state gates remain unchanged.

## 12. Next slice — B2

B2 completes recipient lifecycle and actor-product integration without weakening the B1 assignment boundary:

- internal wrong-recipient declaration;
- requester correction/reassignment;
- recipient redirect/delegation only where executable authority/directory semantics exist;
- explicit external invitation replacement/revocation;
- invalidation of old external sessions on recipient replacement;
- convergence of recipient/identity changes without stale duplicate actor work;
- recipient-bound Evidence Request projection into Today;
- mobile redirect/expiry/revocation acceptance evidence.

B2 must preserve one canonical request recipient and must not introduce a generic parallel assignment framework.
