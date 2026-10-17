# ClearSight — Audit & Assurance workspace (design 16)

**Status:** interactive visual prototype only, using synthetic data. No ClearSight APIs are called and no authoritative decision is recorded.

## Structure

- **Overview:** active audits, current evidence coverage, deadline window, unresolved findings, and three next actions. Current/stock counts are distinct from the upcoming deadline window.
- **Audit plan:** scoped audit list, owner, stage, reviewed evidence, next milestone; select a record for details.
- **Evidence requests:** request status, provenance, due dates, assessor decision with a required review note. Prototype decisions are shown locally only.
- **Findings:** severity, responsible owner, due date, remediation state, exact item inspection. Owner remediation is not independent verification.
- **Drawer:** current item and linked context; do not duplicate Programs, evidence, approval or report engines.

## Interaction constraints

- Filters apply to the actual visible sample population; `All groups` and framework selectors persist across tabs.
- Upcoming deadline window (7 / 30 / 90 days) filters review deadlines and open-finding closeouts using a fixed **15 October 2026** sample reference date; it must not imply a historical/current-posture recalculation.
- Search, tab navigation, keyboard `Ctrl/⌘ K`, `Escape`, review note requirement, and mobile record drawers are interactive.
- Sample counts reconcile: 8 audits, 147 reviewed of 252 requests, 98 outstanding plus 7 overdue = 105 not reviewed; 8 sample findings, four high priority; four upcoming closeouts in the default 30-day window.
- No new live audit storage or workflow is proposed by the prototype. Integrate with existing Program, Matter, Evidence, reporting and authority foundations after checking availability and permissions.

## Visual and accessibility

- Shared ClearSight shell, slim compact cards, restrained motion, textual status and evidence counts.
- Responsive at 390 / 768 / 1280 / 1880 CSS pixels without horizontal document overflow. Minimum interactive targets and precise focus states should be enforced in implementation using existing shared component contracts.
- Motion respects reduced-motion. SVG markers support, but do not substitute for, readable statuses.

## Files

- `audit-assurance-preview.html`: complete standalone interactive prototype.
- `audit-assurance.html`, `.css`, `.js`, and `prototype-shared.css`: editable source.
- `audit-assurance-4k.png`, `audit-assurance-plan-4k.png`, `audit-assurance-evidence-4k.png`, `audit-assurance-findings-4k.png`, `audit-assurance-record-4k.png`: high resolution screenshots.
- `audit-assurance-mobile.png`, `audit-assurance-evidence-mobile.png`, `audit-assurance-review-mobile.png`: mobile captures.
- `verify_audit_design.py`: automated interaction and responsive smoke tests.
