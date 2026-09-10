# Vendor findings correction implementation plan

**Goal:** Show current source-linked findings and actions, and distinguish historical requests, collection and review.

**Architecture:** Reuse scoped vendor-link reads and exact Matter reads. Bound portfolio reads to 50 relationships, 50 links per relationship and 50 distinct Matters with four concurrent requests. Unknown, truncated or failed reads never produce complete totals. Matter/action identity deduplicates shared findings. Source dates retain their meaning; implementation is separate from verified closure.

**Tech stack:** Existing React components, typed APIs, Vitest and rendered evidence fixtures.

- [x] Add focused regressions for superseded/cancelled request labels and collecting assessments; repair compliance status selection.
- [x] Fix relationship-link client routing to the existing relationship-scoped route, with an exact URL test.
- [x] Build bounded linked-Matter loader and pure totals with tests for identity deduplication, partial reads, terminal states and deadlines.
- [x] Replace portfolio form-count and criticality widgets with stored finding/action workload and exact issue navigation; expose linked findings on vendor detail.
- [x] Run only affected tests/typecheck/copy gate and render desktop/mobile light/dark against explicit linked source fixtures.

Verification: 23 tests across five affected files pass; typecheck, production build and evidence build pass. Eight light/dark 1440/390 portfolio captures and four selected-detail/back checks pass. Dark desktop and light mobile detail renders inspected. Evidence: `docs/evidence/2026-09-10-vendor-findings/`.

Main agent owns source-record creation, source metadata, deployment and final integration. No backend schema or API contract is added by this UI correction.
