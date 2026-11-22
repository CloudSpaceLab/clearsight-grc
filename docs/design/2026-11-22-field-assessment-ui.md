# Field assessment review

The response detail shows recorded field assessments and, only when the scoped server permits review, accepts an outcome and reason. The request key and non-sensitive selection survive a browser reload; the reviewer must re-enter the reason. An uncertain outcome retains its key, including after a retry validation error. Decision receipts must match the response, revision, form, field, reviewer and submitted decision.

The server decision gate remains closed. The component evidence in `docs/evidence/2026-11-22-field-assessment-ui/` uses synthetic data at 1440, 768 and 390 pixels; it does not establish full-shell, production authorization or end-to-end command acceptance. Source-to-result and independent authority parity remain release work.

Submitted and assessed scores are shown separately from their recorded sources. An uncertain assessment hides the potentially stale assessed result until the outcome is resolved.
