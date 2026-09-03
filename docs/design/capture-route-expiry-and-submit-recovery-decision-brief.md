# Capture route expiry and submission recovery decision brief

**Date:** 2026-09-03
**Status:** Implemented for canonical form distributions

## Problem

The capture email showed a future link-expiry timestamp, but direct routes could become unavailable after one open. The browser deliberately did not retain bearer sessions, so returning to the same email produced an apparent early expiry. In the email-OTP journey, a successful autosave advanced the shared workspace version immediately before final submission, causing the same verified respondent to receive a version-only `409 Conflict`.

## Options considered

1. Persist the bearer session in browser storage. This would make reopening convenient but would leave reusable authority material on shared devices.
2. Keep one-use routes and require reissue after every open. This would contradict the displayed expiry and create unnecessary sender work.
3. Keep the route valid until its recorded expiry or explicit invalidation, mint a distinct short-lived session on every open, and retain replay protection at the OTP/session boundary. This keeps the visible contract truthful without broadening the request scope.

Option 3 is selected.

## Submission concurrency rule

Final submission accepts the exact current workspace version. It may also accept an older client version only when every intervening workspace version is a contiguous autosave created by the same verified capture session. Any other recipient's edit, a missing version step, a previous submission, revocation, expiry, lock or closed request still blocks submission.

A genuine conflict keeps the respondent's entries on screen and offers **Reload request**. Reloading fetches the current authorized workspace and re-enters the existing recovery/rebase flow.

## Required states

| State | Expected result |
| --- | --- |
| Valid direct route opened for the first time | New link-possession session |
| Same valid direct route opened again | Different new session; same route expiry |
| Valid email-OTP route opened again | New OTP challenge; prior challenge cannot be replayed |
| Expired, revoked, rotated, cancelled or superseded route | Generic unavailable state |
| Submit after this session's autosave | Submission succeeds |
| Submit after another session's edit | Conflict with **Reload request** |
| Reload after conflict | Current workspace loads; recoverable entries are rebased or presented for resolution |

## Proof

- memory and PostgreSQL access-route tests cover repeated opens, distinct sessions and revocation;
- memory and PostgreSQL workspace tests cover own-session autosaves and other-session conflicts;
- the external capture component test covers the visible conflict recovery action;
- no selector, OTP, recipient address or bearer session is written to logs, screenshots or fixtures used as release evidence.
