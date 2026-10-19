# Assessment #549 — HTML / runtime visual inspection

Date: 19 October 2026 · Tracking: #462 · **Whole-screen verdict: REJECT / NOT ACCEPTED**.

## Captures inspected

Original hash-verified `source/clearsight-designs-01-05/assessment-workspace-preview.html` (SHA-256 `dc04b081d33020bb1a3b00a892d09035f506de39773bd9550b60a20c78640297`).

Paired original-HTML first-viewport captures and the **real browser output** of `#549` head `57a7801220c85f312ea586c43100da65e0ad69ab` were manually reviewed using [UI/UX run 39302719472, artifact 12213165964](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39302719472/artifacts/12213165964):

| Viewport | Original source capture | Runtime capture | Finding |
| --- | --- | --- | --- |
| 1440×900 light | `approved-reference/approved-assessment-workspace-1440x900.png` | `211-internal-assessment-light-1440x900.png` | Three panes exist, but header/context/actions and card proportions shift the entire working region materially downward. |
| 1280×800 original light / runtime dark | `approved-reference/approved-assessment-workspace-1280x800.png` | `212-internal-assessment-dark-1280x800.png` | Three-column work exists; reference starts section/question content much earlier. Do **not** score theme/color difference as pixel mismatch. |
| 1024×768 light | `approved-reference/approved-assessment-workspace-1024x768.png` | `215-internal-assessment-light-1024x768.png` | Source's tab/completion/question region begins notably earlier; the contextual row occupies its own band. |
| 768×900 original light / runtime dark | `approved-reference/approved-assessment-workspace-768x900.png` | `216-internal-assessment-dark-768x900.png` | Approved compact **left rail**, but runtime uses an old **bottom mobile nav**. This is a shared-shell responsive difference, not questionnaire fixture data. |
| 390×844 light | `approved-reference/approved-assessment-workspace-390x844.png` | `213-internal-assessment-light-mobile-390x844.png` | Source first viewport shows progress, sections and first question; runtime only reaches section heading near fold. |
| 390×844 original light / runtime dark | `approved-reference/approved-assessment-workspace-390x844.png` | `214-internal-assessment-dark-mobile-390x844.png` | Same structure issue independently visible in dark runtime; no new palette implied. |

The original uses illustrative access-control records and partly preanswered fields, whereas the runtime fixture uses an annual-return request with fewer fields and no submitted answers. **Those differences are legitimate**; replicate hierarchy, not prototype totals or content.

## Blocking structural differences

1. **Header/action density:** approved small breadcrumb, concise record identity, owner/status and Save draft / Review responses in a bounded header; runtime long breadcrumb + separate tall title/description + Close assessment occupies much more vertical space. Genuine save/authority and timestamp data must be sourced before exposing prototype Save draft/status controls.
2. **Working views sequence:** approved `Questions → Excel import → Review → Completion → Sections / Questions / Readiness & Evidence`. Runtime adds `Request context` before tabs, which hides substantive work. The Excel answer import remains intentionally disabled pending verified mapping; this is a functional parity blocker, not an excuse to fabricate import success.
3. **Redundant progress:** runtime has one completion strip and another broad step meter. Approved hierarchy uses one dominant completion strip plus small section metadata.
4. **390px first viewport:** first question not visible in the inspected runtime. Requires an actual browser assertion, not viewport overflow-only checks.
5. **768px shell:** bottom nav differs from approved compact left rail. Reconcile through the unified shared shell, preserving existing accessible navigation.
6. **Panels and content density:** three/two/stacked layouts exist but first-question placement, section rail density and readiness/evidence hierarchy do not yet match source; no field response or source provenance may be invented.

## Corrections implemented on #549 after inspecting source pairs

- Move the **secondary request context** disclosure below the authorized working form, without deleting recorded known facts.
- Remove the duplicate section progress meter in full-page mode, retaining its live section label and the primary completion bar. Compact/external capture remains unchanged.
- Reduce duplicated header padding/gap; keep accessible 44px navigation control.
- Assert first question is inside the 390px viewport and only one visible progressbar is rendered; preserve 1440/1280/1024/768/390 layout captures and original evidence.

**These edits require a new exact-head CI and source screenshot comparison.** The review decision stays REJECT until the changed runtime passes the full checklist. #462 remains 0/19, and four-actor authority/report-source acceptance is still separately required.
