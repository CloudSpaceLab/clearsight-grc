# Program register continuation recovery

The approved [Program HTML](../../design/references/clearsight-2026-10/source/clearsight-program-designs/program-workspace-preview.html) (SHA-256 `b91aa4a96bfae4fc712f49e883996eb770d74dc3ae4b588ab6e573b4ec9d846f`) describes a Program record, not the register. The [decision](../../design/2026-11-03-program-register-retry.md) covers a recovery defect in the existing register and does not claim source parity for it. Reviewer: ChatGPT Codex `/root`, 3 November 2026, local uncommitted working tree.

Previously a failed continuation left the loaded records and button visible without an error; the new regression test reproduced that failure. The [1440px](light-1440.png) and [390px](light-390.png) captures show the corrected alert and retry alongside the retained record. The historical [register baseline](../2026-09-22-program-data-centered-ux/48-program-portfolio-list-light-1440x900.png) shows the earlier normal state, not the failed continuation.

An isolated non-production fixture supplies a continuation cursor and rejects that request. Browser checks at 1440, 1280, 1024, 768 and 390px in light and dark themes confirmed that the record remains visible, the error appears, the retry is keyboard-operable and at least 44px high, and the document does not overflow horizontally. The unit regression also confirms a successful retry uses the same cursor. Desktop and phone WCAG A/AA automated scans found no definite violations; `aria-prohibited-attr` and `color-contrast` were incomplete and require manual review.

These synthetic checks do not establish live authorization, production data quality or whole-screen acceptance. The standalone Home source remains absent; the acceptance register remains 0/19 and production cutover remains blocked.
