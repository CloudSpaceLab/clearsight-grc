# Home insight presentation

Status: partial implementation. The 2.9 reference remains unchanged; this decision supplements it with the requested Home wording and insight treatment.

- Show a percentage alongside its numerator and named population for current, complete risk and linked-indicator posture. Suppress it when the denominator is zero, unknown, incomplete, stale or inconsistent. These percentages are not compliance scores.
- The Home date range applies to the existing period-based overview and loss queries and the risk-history query. Current risk and indicator posture remains explicitly current. The existing API supports a current reporting end date, not an arbitrary historical end date.
- Use recorded severity labels and status tones for priority issues and indicator breaches. An appetite breach is labelled as such; it does not imply a particular risk rating.
- Use direct work labels such as “Issues needing attention” and “Review work.” Prefer “Outdated documents” for a future document-age measure when authoritative document dates and the eligible population are available; the present submitted-document inventory does not provide that measure. Do not substitute expired evidence or invent a bank-wide count.
- Indicator insight currently covers indicators linked to risks. RCSA requires a separately scoped, permission-aware date-bounded aggregate or an explicitly bounded user-visible list; the existing paginated cycle list cannot be presented as a complete date-filtered insight. Program satisfaction percentages likewise require a verified eligible and satisfied population. Neither is implied by a framework catalog or a submitted document.

The implementation does not alter data contracts or the immutable reference package. Browser captures and limitations are recorded in `docs/evidence/2026-11-20-home-insights/`.
