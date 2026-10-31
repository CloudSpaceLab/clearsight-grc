# Work deadline counts and source drills

Tracking: #462 / #558 / #597. Source: `docs/design/references/clearsight-2026-10/source/clearsight-work-approvals-designs/work-approvals-preview.html`, SHA-256 `202ed9c2b22ac4e921d9ba14aa41b61644ea54cbe55abcc81a86ec10b10ee8d2`.

## Problem and decision

The existing V2 Work strip in #597 displays complete actor-visible workflow deadline counts, but its metric actions filter only the bounded Today feed. Clicking a count must open the corresponding authorized assignment population, including records after the first page.

Reuse the existing metric cards, Work table, inspector, authenticated assignment reader and cursor. A source-count click clears competing title, status and due-window filters and queries the source with the count receipt's deadline boundaries. Overdue uses the exact exclusive `due_before`; upcoming work uses inclusive `due_from` and an exclusive upper bound one microsecond after the seventh day. Preserve server microsecond precision and do not re-filter those rows against a later browser/page clock. No backend, schema, queue or UI component is added.

Counts and pages are separate reads. The count timestamp stays visible; source results retain their own read timestamp. Current changes or access revocations can change membership between reads, so the UI does not promise a frozen population. Approval/evidence measures remain explicitly loaded-only. Unsupported organization-area selection retains the existing local fallback.

The baseline also put the first record below the tablet viewport: its measure strip switched to two tall rows too early. Restore the original four columns above 760px and two columns below, using existing metric tokens and automatic card height. Keep both source qualifiers visible, remove repeated queue narration, and retain 44px targets. No shared palette, density mode or component variant changes.

## Scope and recovery

- Validate every page's tenant/entity/principal, source coverage, count basis, timestamp and continuation receipt before showing it.
- Retire count/page state on every scope/query visit, including returning to a prior scope while a replacement read is pending.
- Clear source records and totals after denied access or a mismatched page receipt. Recoverable continuation failures keep prior rows with explicit retry.
- A selected source drill retains its date bounds through search, status changes and continuation; choosing another due window, another queue or Clear filters leaves that drill. Selecting a source card again clears its drill.
- Keep controls mounted during loading and failures. Shared tabs, fields, table and sheet retain keyboard, focus, touch and responsive behavior.

## Review

Preserve the pre-fix #597 runtime, reproduce the count-to-loaded-feed mismatch, and exercise the corrected source reads with more than 50 assignments and exact date boundaries. Compare the source-shaped strip, menu, filters, list and context at 1440/1280/1024/768/390 in both themes. Include denied/mismatched receipts, count-unavailable fallback, continuation recovery and scope re-entry tests. Record concrete results in the PR and retained evidence index.

This is partial V2 integration. Team/Completed, organization-area source filters, complete approval/evidence totals and whole-screen operational acceptance remain outside this tranche.
