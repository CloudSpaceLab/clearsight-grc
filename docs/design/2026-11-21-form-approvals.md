# Form approvals in Work

## Decision

The Work workspace has separate **Open work** and **Form approvals** views. The approval view asks the existing form library for pending versions in pages of 20 and shows only forms whose returned transition operation authorizes the signed-in user to approve or reject. It does not purport to include Program, document or other approval types. Missing authority is an unavailable state, not an empty list. The next-page button remains available even if the current page has no eligible forms. No counts or unimplemented approval decisions are presented.

The server filters the form-library page by pending status and returns per-form eligibility; the browser filters those bounded, generally readable form-library rows for display. That display filter is not an authorization boundary, and the paginated view is not a complete server-filtered approval inbox. The detail and command perform their own access and authority checks.

Each row opens the existing form detail carrying the listed version. The detail reloads the current form and its operations and offers the existing version-checked Approve form and Reject form actions only when the loaded version matches and remains pending. If the version changed, a notice offers the latest form; no decision is available until that form is opened explicitly. A closed or mismatched form cannot be decided from the approval link. Direct form links remain available. The command still rechecks authorization and version; this is not a historical-version reader. A unified approval source remains future work.

## Evidence and limits

Source reference `clearsight-v2.15/approval-inbox.html` SHA-256 `3b9ce135342f13c87ae529bf36266f640e43f29203a61952bf399b6d6c46714f`. Source screenshots and the previous runtime work view are retained with the updated runtime in `docs/evidence/2026-11-21-form-approvals/` at 1440, 768 and 390 pixels. Reviewer: Codex agent `/root/approval_ui`, 21 November 2026. Source composition is adapted to one real form-approval source; reference demo controls and counts are not copied.

Browser evidence uses synthetic fixtures, asserts no horizontal overflow and no axe violations with color contrast excluded, and checks that a form row leads to form detail with its available decision controls. The affected component, form-action and copy tests cover authorization metadata, pagination, unavailable authority, denied continuation, stale status and current decision API behavior. Browser evidence does not establish backend authorization, dark mode, full keyboard behavior or full approval-screen acceptance. The existing partial token set is reused; no new tokens are introduced.

The 21 November follow-up retains current and stale-form captures at 1440, 768 and 390 pixels in `docs/evidence/2026-11-21-approval-version/`. Reviewer: Codex agent `/root/approval_unify`. The browser check confirms that a changed version hides both decisions and that opening the latest form restores its independently authorized actions. The reference and previous renders above remain the before baseline. No backend authority or production acceptance is implied.

The follow-up in `docs/evidence/2026-11-21-approval-receipt/` uses a synthetic transition response to check the confirmed approval and stale-version recovery at the same widths. The confirmed result remains visible if the subsequent refresh fails; a later different version is not presented as that result. These checks do not execute a live backend decision. The component tests also cover an unconfirmed response and a synchronous refresh failure after confirmation.
