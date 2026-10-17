# ClearSight — Controls & Policies (design 11)

## Design contract

The approved light Control Tower visual system is retained. This is a standalone **illustrative design prototype**, not production ClearSight data or an implementation of authorization, compliance calculation, publication or workflow approvals.

- **Overview** highlights due control tests, failed tests, policies requiring review, and certifications approaching renewal. Current findings are not retroactively recalculated by the deadline window.
- **Control library** keeps the control identity, framework relationship, tested date, status, owner and next review accessible. Filters and search operate on the selected sample population.
- **Policies** separates current published versions, review-needed items and proposed revisions awaiting approval. A draft does not replace a published version.
- **Reviews** is a sorted control/policy queue. **Certifications** is a renewal calendar, not evidence of certification validity.
- **Control test coverage** uses tested vs in-scope count, counting both passed and failed tests as tested. Untested and under-review records are not counted as passed.
- **Copilot** is visually secondary and displays one of three suggested reviews at a time. Clicking opens the relevant record. Suggestions are sample statements, not AI conclusions.
- **Responsive** stack keeps the same content reachable on mobile. Scope filters collapse behind a toggle; drawers become full-screen when space is constrained.
- **Interaction** includes active tabs, explicit status pills, keyboard focus, Escape-to-close, restored focus, source clues, reduced-motion support, record search and real in-prototype deadline filtering. No operational API calls or irreversible actions are performed.

## Implementation alignment

Use the existing Programs/Control Catalog, monitoring test/evidence, policies/authority route, workflow attention, framework publisher, reporting and established design system. Do not introduce a parallel controls table or a new scoring engine just to reproduce a dashboard. Connect measures and drills to exact authorized scope/source revision, preserve untested/unknown data, and distinguish certificate expiry from formally reviewed renewal.

## Visual acceptance

Rendered desktop screenshots at 1880 CSS px / scale 2 = **3760 px** physical resolution, including Overview, Control library, Policies, Reviews, Certifications, and detail drawers. Mobile renders at 390 CSS px / scale 2 = 780 px physical width. Automated prototype smoke checks cover filters, date window, tabs, record drawers, keyboard dismissal, suggestion selection and absence of horizontal overflow. Example records are deterministic synthetic data.
