# ClearSight 09 — Reports & Insights design

**Date:** 15 October 2026  
**State:** Design prototype; illustrative fixtures only. No ClearSight API, report file, email, scheduled task or compliance conclusion is created.

## Visual system

- Continue the approved light Control Tower visual contract: neutral white cards, soft blue surfaces, thin borders, restrained wave header and purposeful interactive states.
- Preserve the enterprise shell, its role/organization scope and the Home → Portfolio → Reports navigation model.
- Four measures lead the Overview without an extended KPI wall. Keep the first viewport for concentration, coverage and actual losses, rather than many unrelated widgets.
- One small Copilot module, **one suggestion visible at a time; three selectable options**. The orb is decorative and must not overlap text or actions. User can complete every task without Copilot.

## Workspace anatomy

1. **Overview**: Current outside-appetite risks, current assessed-requirement coverage, period net operational loss, report availability; one ranked business-unit risk comparison, one framework assessment-coverage comparison, loss-period trend, short finding queue, recent report list.
2. **Explore**: Measure + breakdown + compatible comparison + exact-value table. Risk is a current posture stock, coverage is assessed/applicable, losses are a period flow. A framework denominator must never appear as a previous-period value.
3. **Reports**: Saved/generated output library, status filtering and preview. A ready report is not evidence that it was delivered or accepted by a recipient.
4. **Schedules**: Authorized recurring distribution, cadence, next attempt and state. The preview's Pause/Resume switch is local demonstration only.
5. **Build report**: Title, chosen approved template, selected sections, document preview, file format, delivery cadence and authorized recipient concept. Report contents reuse existing metric definitions and sources rather than local chart calculations.

## Functional prototype

- Reporting range 7/30/90 days changes period loss amounts and plot; current risk and assessment coverage remain unchanged.
- Risk concentration values (9 + 7 + 5 + 4 + 3) reconcile to 28; prior snapshot 24 reconciles to +4.
- Assessment coverage entries reconcile to 186 of 240; these are **assessment completeness** figures, not a compliance verdict.
- Illustrative 90-day loss buckets total ₦12.4m and are labelled by reporting bucket, not incorrectly as six separate months. All source data are entirely fictional.
- Saved analysis appears as a new item and reopens its measure/breakdown. Add to report selects the actual report section; preview dialog is keyboard-dismissible.
- Report search, status filter, recurring schedule toggles, period selection and three Copilot suggestions are locally interactive.
- Global organization dropdowns are illustrated, not wired to an enterprise data source; their changed state explicitly warns that numerical samples remain unchanged.

## Production mapping and guards

- Reuse existing Reporting, Metrics/Insights, exact membership drill, organization filters, Program status/compliance policy (#433), framework edition/adoption (#437), source/evidence access and verified actor/tenant controls.
- Never aggregate currencies without a reviewed FX basis; no browser-side compliance calculation; keep assessment coverage separate from verified compliance score.
- Date ranges affect period flows, not current posture unless the API explicitly supports an as-of read.
- Report creation, email delivery, approval, sign-off and recipient access each require separate durable receipts; the prototype performs none of them.
- Layouts inspected at 390, 768, 1024, 1280, 1880 CSS pixels, with no horizontal overflow. Controls have keyboard focus and reduced-motion fallback.

## Files

- `reports-workspace-preview.html`: standalone interactive prototype.
- `reports-workspace.html`, `reports-workspace.css`, `reports-workspace.js`: editable source.
- `reports-workspace-4k.png`: primary overview at 3760px width.
- `reports-explore-4k.png`, `reports-library-4k.png`, `reports-builder-4k.png`, `reports-schedules-4k.png`: other desktop views.
- `reports-workspace-mobile.png`, `reports-library-mobile.png`, `reports-builder-mobile.png`: mobile views.
