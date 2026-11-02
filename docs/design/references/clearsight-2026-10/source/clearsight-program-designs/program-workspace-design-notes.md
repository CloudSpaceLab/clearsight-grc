# ClearSight — Program Workspace / Design 03

## Screen scope

- Single Program: Information security (`ISMS-001`), linked to ISO/IEC 27001:2022.
- `Overview`, `Requirements`, `Assessments`, `Evidence`, `Issues`, and `Activity` are distinct panels within the same page.
- The full screen uses the light ClearSight visual language approved for the Control Tower and Framework Library.
- **All data is illustrative design content.** There are no live ClearSight API requests or compliance conclusions in this prototype.

## Content and posture semantics

- 52 applicable requirements: 27 supported, 7 gaps, 4 awaiting review, 14 not assessed.
- 34 assessed = supported + gaps; 18 unconfirmed = awaiting review + not assessed.
- The horizontal status distribution is a category breakdown, not a compliance percentage or weighted score.
- Four compact first-row values remain stable in position and support authoritative drills in implementation: assessed requirements, supported, open gaps, evidence requiring review.
- Evidence is a separate population; its count is not derived from assessment coverage.
- Historical activity period changes only the assessment list, **not** current requirement posture.
- Illustrative Program states do not replace approved categorical status or verified Program-calculation policies.

## Interaction design

- Compact period selector (7/30/90 days) changes the historical activity list.
- Requirement area filter toggles All/Gaps/Unassessed without moving controls between tabs.
- Selecting a requirement area or assessment opens a focused detail drawer with source/owner/state metadata.
- Program navigation uses accessible tab semantics with left/right/home/end keyboard shortcuts.
- AI suggestions display one at a time and switch manually between **three** context-specific actions; no autoplay or animated counters.
- New assessment opens a draft preview only: no phantom successful submission.
- External global hierarchy filters in this standalone prototype explain that counts are illustrative. In the real app they must issue authorized scoped reads.
- Reduced-motion preferences disable unnecessary transitions; overlay supports focus containment, Escape close and focus return.

## Implementation handoff

- Preserve existing ClearSight Home/Portfolio routes, Program aggregate, native requirements, form assessments, evidence and issue-linked actions.
- Reuse the existing design-system contracts and Program compliance calculation from #433 only when its full verified source and denominator guarantees hold.
- Reuse the governed framework adoption work from #437; a linked framework does not establish compliance.
- Preserve snapshot identity, tenant/entity/organization scope, stale/partial/unknown states and authorized drill consistency.
- The production UI should not ship any of the illustrative totals, names, approval results or samples as defaults.

## Tested prototype

- Desktop viewport: 1880 CSS pixels at 2× device pixel ratio; full-resolution screenshot 3760 × 2922.
- Mobile viewport: 390 CSS pixels at 2× device pixel ratio; full-width screenshot 780 × 5210.
- Verified tab switching, area filter, period-dependent activity, Copilot suggestion navigation, focused review drawer and mobile horizontal overflow (0px).
