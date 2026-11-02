# ClearSight 06 · Vendor management design

Reviewed 15 October 2026. Continuation of the approved ClearSight Control Tower and 01–05 design system.

## Views
- Vendor portfolio with 4 stable measures, assessed-risk distribution, review pipeline, searchable filtered vendor register, bounded action rail and one-of-three contextual Copilot suggestion.
- Vendor relationship with onboarding stage, independent evidence/DPO gates, risk state, responsible department, due date, separate review sections.
- Add-vendor drawer with required fields and local-only draft behavior.
- Responsive mobile portfolio/detail: readable vendor cards and functional mobile navigation.

## Interaction proof
- Search/risk/status and branch/department/team/group filters update records; 30-day deadline horizon updates due counts, not current risk posture. Scope and copy use actual selected preview data.
- Row/drill, assessment/documents/onboarding tabs, DPO action, drawer validation, local draft, and 3 manual Copilot suggestions work in the standalone prototype.
- Keyboard focus/Escape, mobile overflow, reduced motion and responsive presentation checked in Chromium.

## Implementation boundaries
- Sample records and calculations in the design are illustrative, not real bank data. Do not copy their counts, ratings, deadlines, approval, or evidence status into production.
- Reuse existing vendors identity/service relationship, vendor assessment, form/evidence, source lineage, authority routes, and canonical risk/issue architecture. Do not add a second vendor schema or second assessment engine.
- Activating a vendor after due diligence or DPIA assessment is a material approval; a document upload alone does not satisfy it. Keep DPO review mandatory when configured, and distinguish response, assessor conclusion, approver authorization and actual activation.
- Production requires the reviewer provenance/freshness, tenant-group scope, authoritative document checks and exact review state before any metrics, approval or certification.
- Reuse shared filters, record browser, status badge, MetricCard, Tabs, FocusedSheet and design tokens.
- The prototype is offline and never sends vendor data to a backend; an uploaded file is not processed and new vendor creation exists only in runtime memory.

Next proposed design: **RCSA / KRI / ROPA operations**, showing distinct native register views rather than another generic Program dashboard.
