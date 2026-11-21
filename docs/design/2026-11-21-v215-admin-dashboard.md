# Administrative dashboard: partial v2.15 integration

## Decision

The verified session's `SYSTEM_ADMIN`, `SUPER_ADMIN`, `GRC_ADMIN`, and `GRC_ADMINISTRATOR` role codes select the Dashboard heading and navigation label. This presentation choice grants no authority. The existing scoped oversight and personal work endpoints continue to decide which information is returned. Other roles retain the Overview heading and personal-first order.

Administrative users see the existing four scoped oversight measures ahead of their personal assignments. Assignments remain available when oversight data cannot be loaded. Measures retain their original scope, freshness, and date semantics; historical dates do not change current posture. This is a first increment, not a complete role-specific dashboard or production screen acceptance.

## States and evidence

The evidence fixture supplies an explicitly labelled sample session and sample records. The normal role baseline is retained in `docs/evidence/2026-11-21-v215-home-assignments/`; the administrative state is retained in `docs/evidence/2026-11-21-v215-admin-dashboard/`. Responsive renders cover 1440, 768, and 390 pixels. They were inspected by the Codex agent on 21 November 2026; the browser check asserts no horizontal overflow and no axe violations with color contrast excluded. These renders do not establish authorization or backend acceptance.

Component tests cover administrative role codes, a non-administrative role with configuration access, ordering, and an unavailable oversight response. Additional role-specific data, contrast review, keyboard and touch review, denied/partial states, real backend reconciliation, and named screen acceptance remain necessary before cutover. No new design tokens are introduced.
