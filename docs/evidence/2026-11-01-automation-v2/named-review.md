# Automation V2 named visual and evidence review

**Reviewer:** ChatGPT, agent `/root/automation_visual`  
**Date:** 2026-11-01  
**Decision:** PASS for the bounded Automation policy review UI and its synthetic browser evidence. This review does not grant whole-screen operational acceptance or production cutover approval.

The reviewer owned the browser fixture and evidence scripts, and reviewed the runtime changes authored by the parent agent. Agent `/root/cutover_status` independently reviewed the evidence harness, identified two false-pass risks and a narrower long-content proof gap, and confirmed that all three were resolved before the final capture. The reviewer did not author the runtime layout or policy reader.

## Exact build and source identity

The final browser run used the freshly built evidence bundle from local commit `54bd6f192419c89a3713aa6c69e37205655c37e4`. The published equivalent is `697b18e6e6319a2dbf430d82cc1bdf285b792eb5`; both have Git tree `24219c2d3c7d02f2357b615f9fdc364932000ae2`. The receipt intentionally retains the local commit that was actually built and captured.

- Final asset digest, before and after capture: `78244fc114b797a60af94de172bf831ab7d1d580ccd17029adc75555f02e0dd3`.
- Baseline commit: `f9b5bfd9c27641c8acb7928b2e59d8c59b00ec64`.
- Frozen baseline asset digest: `40e12a493b9e595206607cba4729db53c7d685dc39cd5169c330fc268542649c`.
- Browser: Chromium `140.0.7339.16`, Playwright `1.55.0`.
- Final fixture SHA-256: `5f340b747124a7b20c1ed56aa1ef485200ca6fd6b420592f5ae214d62805d6fb`.
- Final capture script SHA-256: `67adaf903efbacae73735c166f77c2319fc56c8f4227a3f76a08ab29ff79c094`.
- Final reviewer script SHA-256: `8886d4b439e0c9cc149b1dd9be3929ca44e5fa013e32d25bb3af06bd36140e7f`.

The authoritative source directory is `docs/design/references/clearsight-2026-10/source/clearsight-automation-governance-designs/`. These source files were read and left unchanged:

| File | SHA-256 |
| --- | --- |
| `automation-governance.html` | `e24b9c012375175a2692be98adb2741dbea353cae0979b919fb0a18261583a98` |
| `automation-governance-preview.html` | `7dad1ffc651fd03821cc996ff13f1203cb1ec1344a8a5bdec7b00b49035a7b6f` |
| `automation-governance.css` | `0c0e19831e1a5e3def2821e5b148278a18045158cc3ce22b8804ad58413cf92d` |
| `automation-governance.js` | `a6d4aaadb3f05e2386c1ac9618fc2a2f22538f27e7b4c31605d546f944a9cdf8` |
| `automation-governance-design-notes.md` | `c377a78f24734d25dbd0d2471b465ae8596d43422b3bcfed7d9b2d5db0b93cc2` |
| `prototype-shared.css` | `5f6ffa5a8fc3c1c4fffee1848326f309cae02f65b154bd7eeee2a21aa4e2f3f9` |

The self-contained preview HTML was rendered for the source comparison. Its eight source PNGs are byte-identical between the baseline and final receipts. The source is light-only; dark screenshots review the runtime adaptation. Remote source font requests were blocked, so the source capture uses the environment's available font fallback.

## Images inspected

The retained evidence consists of **22 baseline PNGs** in `before/` and **48 final PNGs** in `after/`, each bound by filename, byte count and SHA-256 in the accompanying `receipt.json`. The exact three baseline harness inputs are retained in `baseline-inputs/`; their hashes match the baseline receipt.

Direct visual inspection covered every retained PNG byte sequence. For the final run, 43 PNGs were byte-identical to images already opened during the preceding complete review, and their identity was verified against the final receipt. The five changed final images were reopened: `layout-light-768-context.png`, `layout-dark-768-context.png`, `selection-history-dark-390.png`, `tabs-and-handoff-light-1440.png`, and `long-full.png`. The last image now captures from the document top and restores the prior scroll position afterward, so the sticky shell does not appear halfway through the full-page image.

| Image family | Exact coverage inspected |
| --- | --- |
| Approved source | `source-{1440,1280,1024,768,390}.png` and `source-{1024,768,390}-context.png`, in both evidence sets |
| Before layout | `layout-{light,dark}-{1440,1280,1024,768,390}.png` and `layout-{light,dark}-{1440,390}-full.png` |
| Final layout | `layout-{light,dark}-{1440,1280,1024,768,390}.png`, `layout-{light,dark}-{1440,390}-full.png`, and `layout-{light,dark}-{1024,768,390}-context.png` |
| Selection and history | `selection-history-light-1440.png` and `selection-history-dark-390.png` |
| Unavailable views and handoff | `tabs-and-handoff-{light-1440,dark-390}.png` and the corresponding `-schedules.png`, `-activity.png`, and `-escalations.png` images |
| State and long-content cases | `empty.png`, `truncated.png`, `denied.png`, `failure.png`, `recovery.png`, `readonly.png`, `loading-pending.png`, `loading.png`, `long.png`, and `long-full.png` |

## Source comparison and responsive findings

The runtime follows the source sequence: one page heading, the four local tabs, four compact summary cells, register heading and count, search and filters, then selectable records with policy context. On wide screens the selected context remains beside the list at the source's approximate proportion. At 1024px and below, the summary becomes two columns and the context stacks below the list. Explicit record selection reveals and focuses the context title below the sticky shell; the return control restores focus to the selected record.

The baseline already showed its first complete policy without scrolling. Its defects were the duplicated Configuration/Automation/Automation policies framing, nested configuration navigation, and absence of the source's local tab, summary, filter and selected-context anatomy. This evidence does **not** claim that the baseline first record was below the fold or that every final record starts higher.

The first complete record geometry is identical between runtime themes. Values below are viewport Y coordinates, rounded to two decimals:

| Viewport | Approved source top–bottom | Baseline top–bottom | Final top–bottom |
| --- | --- | --- | --- |
| 1440 × 900 | 557.00–636.00 | 474.75–605.80 | 570.19–649.19 |
| 1280 × 800 | 557.00–636.00 | 474.75–605.80 | 570.19–649.19 |
| 1024 × 768 | 656.00–735.00 | 499.14–630.19 | 639.19–718.19 |
| 768 × 900 | 621.00–700.00 | 595.02–726.06 | 603.19–682.19 |
| 390 × 844 | 633.83–711.83 | 571.47–770.52 | 740.38–836.38 |

The final default first policy is wholly visible at every required viewport. Phone records place the title and status together, then use the full row width for metadata, matching the source's compact anatomy while allowing long names to wrap. The phone tab strip scrolls horizontally; Activity can be selected and becomes fully visible. No document-level horizontal overflow was found.

| Source element | Runtime adaptation and reason |
| --- | --- |
| Summary counts | Counts describe the loaded latest policy revisions. Organization scope and the possibility that a newer draft replaces an active revision are explicit. They do not claim execution or schedule totals. |
| Category and status filters | Action class and policy revision status use the actual record fields. Search and counts operate on the loaded population. |
| Rule flow and selected detail | Stored eligibility, maximum-scope and outcome-check guardrails precede configuration facts. Missing values are reported as not recorded. |
| Create, edit, simulate and lifecycle commands | These capabilities are unavailable in this read surface. No synthetic operation is presented as a working business action. |
| Schedules and Activity | Each tab explains its unavailable data and offers a return to policy revisions. Effective dates are not presented as run schedules, and policy status is not presented as execution evidence. |
| Escalations | The tab hands off to the existing People & access escalation workspace and preserves its return history. |
| Shell and dark theme | The existing application shell and shared controls are retained. Dark colors adapt the light-only source through the shared theme. |
| Narrow selected context | Focus and scroll account for the sticky shell; long names, unbroken codes, guardrail keys and values wrap within their columns. |

## Behavioral and evidence checks

The final capture and standalone reviewer both passed: **10 layouts, 12 behavior scenarios, 48 images**. The asset digest was unchanged during capture. All ten layout axe scans reported zero violations; incomplete color-contrast findings were retained for the decorative, `aria-hidden` select chevrons. The controls in the measured Automation surface retained at least 44px targets.

The browser checks exercised search, a distinct non-first policy selection, Back/Forward, reload with exact policy and query parameters, status filtering, no-match state, phone focus return, all local tabs, the real escalation handoff, empty and bounded populations, simulated 403 and 503 reads, retry, read-only capability, loading completion, and long content. The 200-record scenario visibly discloses the bound and the limited meaning of search, filters and counts.

The long-content case measured the policy code, 160-character unbroken guardrail key and 240-character unbroken guardrail value. Their client widths and scroll widths were equal at 284px, 113px and 161px respectively; all had nonzero rendered height. Both the focused-context and full-page images were inspected.

Every retained observation recorded zero business commands and no pending read at completion. Selection/history retains a separate pre-reload observation so reload cannot erase earlier attempts. The fixture records the context response and binds the policy response to its observed tenant. This is synthetic context evidence, not live backend authorization proof.

The reviewer was also exercised against temporary copies of the final evidence. It accepted the unchanged control and rejected a missing pre-reload observation, a command inserted before reload, and altered screenshot bytes. The original final evidence was unchanged by these negative checks.

## Remaining acceptance boundaries

This package establishes the source-faithful read UI, supported navigation and bounded-population behavior under a synthetic transport. It does not demonstrate live actor authorization, a real source-to-report journey, execution outcomes, editing, simulation, scheduling, delivery receipts or earlier-revision history. Only the pinned Chromium environment was captured. Axe incompletes and wider accessibility requirements remain separate from this scoped review.

The broader V2 whole-screen acceptance count remains **0/19**. The full operational Automation scope and production cutover remain blocked by their existing acceptance gates. This visual PASS must not be substituted for those gates or described as full Automation operational completion.
