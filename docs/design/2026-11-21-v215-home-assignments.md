# Home: personal assignments

Home places the current actor's open workflow assignments ahead of the shared overview. It uses the existing assignment-count endpoint and checks the returned tenant, legal entity, principal, coverage, count basis and timestamp before displaying counts. The overdue and next-seven-day counts retain their source meanings; they are not a complete count of every kind of work. The card remains usable when the shared overview is unavailable, and its timestamp is independent of the selected reporting range. The Work link opens the existing actor-scoped workspace. No role selection grants authority and no sample values enter the customer runtime.

Loading, unavailable, malformed and mismatched-scope responses do not show zero counts. Identity changes discard stale requests and suppress results from the previous person. An unavailable count offers retry.

The previous Home sample is `docs/evidence/2026-11-20-v210-home-period/`; synthetic after-renders are `docs/evidence/2026-11-21-v215-home-assignments/runtime-{1440,768,390}.png`. The after-state had no horizontal overflow and axe reported no violations with color contrast excluded. Codex inspected desktop and mobile. These checks do not establish contrast, dark-theme, real authorization, full role-based dashboards or production screen acceptance.
