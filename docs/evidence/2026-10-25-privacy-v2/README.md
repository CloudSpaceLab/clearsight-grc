# Privacy v2 — retained layout and interaction evidence

Scope: [#588](https://github.com/CloudSpaceLab/clearsight-grc/pull/588), following the shared Tabs correction in [#587](https://github.com/CloudSpaceLab/clearsight-grc/pull/587).

## Result

**PASS for the supported Overview / ROPA layout adaptation and activity inspector.** The register now puts records in the first viewport at every approved width. Existing summary, bounded collection, exact activity/history and report owners remain in use. The [decision](../../design/decisions/2026-10-25-privacy-v2.md) records the full scope; the [named independent review](visual-review.md) records its inspected images and limits.

This is partial v2 integration. The other four Privacy source destinations, four-actor execution, live backend authorization, source-to-report reconciliation and production acceptance remain separate. The formal whole-screen register stays **0/19**.

## Revision and original authority

| Input | Retained identity |
| --- | --- |
| Runtime and capture scripts | `651d0692e3d4ff221389d03cf0a795be5f565b45` |
| Git tree | `69cf912b4fe37d601d585700bfab011fbfc39df3` |
| Evidence build | 173 files; SHA-256 `c4d9f917344840eaf64d4176b77a51470b86821deded7b914e8c54f6ab757533` |
| Capture inputs | SHA-256 `e62f7964e0826117058ded7b545744ce1ed892b7995a38e589d5d0f80283f3a4` |
| Released baseline | `02dd0b4f066ba58ec17cf0e2588437efe5e856fd`; 171 build files, aggregate `a20ab1aadace0972b9f35bb924c1d4361f805adca7af4a0b7131c84e1badc746` |
| Original HTML | `docs/design/references/clearsight-2026-10/source/clearsight-privacy-data-designs/privacy-workspace-preview.html` |
| Original SHA-256 | `4baf53940862d9a3af1fbc1be32da7b27595a22efb0646759cb91685a0cb5e8d` |

Original HTML/CSS/JS and source manifest are unchanged. Later evidence/documentation commits do not change application or capture code. The retained build and capture-input digests were rechecked after capture.

## Proof retained

| Evidence | Result |
| --- | --- |
| Released register | [10 baseline renders](before/manifest.json), five widths and both themes |
| Corrected runtime | [45 scoped states / 47 PNGs](after/manifest.json), including two later-page failure/empty captures |
| Original designs | [15 original renders / 30 source-runtime bindings](after/approved-reference/manifest.json) for Overview, ROPA and the selected ROPA inspector |
| Accessibility | [8 scoped states](after/accessibility.json), zero axe violations or browser errors; minimum 44px primary controls |
| Summary receipt | [Revision, hashes, measurements and limitations](after/receipt.json) |

The 45 states cover both views at 1440, 1280, 1024, 768 and 390px in light and dark; ten selected inspectors; desktop/phone history and filters; successful empty populations, denied reads, stale/partial/unknown summaries, retry, exhausted/failed later pages and a missing exact record. Every captured query and displayed population is bound to observed fixture responses. No application command is issued.

History checks use actual Back/Forward in the same document and mounted menu, preserve collection search/page context, then exercise a direct record reload separately. Counted Overview drills prove the distinction between eight total records, one ended record and seven current records; removing the scope chip or clearing filters restores the current population. Native lifecycle and recorded dates remain distinct from review/approval status.

## First useful record

Coordinates are measured from the viewport top; these are record identities, not empty placeholders. Both themes pass the viewport and overflow checks.

| Width | Baseline top | Corrected top | Corrected bottom | Original top |
| --- | ---: | ---: | ---: | ---: |
| 1440 | 944.47px | 527.73px | 545.73px | 488.50px |
| 1280 | 944.30px | 527.73px | 545.73px | 488.50px |
| 1024 | 1002.55px | 556.98px | 592.98px | 488.50px |
| 768 | 1198.33px | 510.98px | 528.98px | 473.72px |
| 390 | 2008.75px | 585.52px | 603.52px | 554.73px |

### Selected comparisons

| Screen | Original | Runtime |
| --- | --- | --- |
| ROPA, 1440 light | [Original](after/approved-reference/approved-privacy-register-1440x900.png) | [Runtime](after/270-privacy-register-layout-light-1440.png) |
| ROPA, 390 | [Original light](after/approved-reference/approved-privacy-register-390x844.png) | [Runtime light](after/270-privacy-register-layout-light-390.png) · [Runtime dark](after/270-privacy-register-layout-dark-390.png) |
| Overview, 1440 light | [Original](after/approved-reference/approved-privacy-overview-1440x900.png) | [Runtime](after/270-privacy-overview-layout-light-1440.png) |
| Activity, 390 | [Original light](after/approved-reference/approved-privacy-register-detail-390x844.png) | [Runtime light](after/270-privacy-register-detail-light-390.png) · [Runtime dark](after/270-privacy-register-detail-dark-390.png) |

Review corrected uneven metric-value alignment and the shared close wrapper squeezing the inspector body. Final facts use the full padded width: 439px in the desktop/tablet sheet and 350px on phone, with heading and unavailable-state content clear of the close control. Negative regression checks reject a full-width outer sheet with clipped inner content.

## Meaningful adaptations

- The supported local menu is **Overview / ROPA**, preserving original order. DPIAs, Breaches, Vendor reviews and Requests have no supported collection contracts and remain excluded.
- Overview uses real ROPA measures, lifecycle, completeness and coverage in the source composition. It does not invent department coverage, DPIA decisions, incidents, priority reviews or Copilot results.
- The source's review filter and department column have no matching ROPA query/field. The runtime retains actual lifecycle status, review dates and supported fields.
- The shared table uses labelled cards at 768px of viewport width because available table space is below its 700px container threshold; the original still has a table there.
- The shared shell, theme tokens, 44px controls and 20px sheet padding remain in use. Source desktop sheet padding is 25px. The original design has only a light theme; dark comparisons establish the same runtime anatomy and readable controls, not an original dark design.

## Validation and release boundary

The focused runtime suite passed **69 tests**. The receipt suite includes nine Privacy negative checks, covering missing proof, geometry, population/scope, unexpected commands, source anatomy, counted drills, cursor resets and squeezed inspectors. The full scenario contract separately checks denied/empty states and retained history identities. Four inherited ROPA browser scenarios also passed with the new heading/inspector arrangement. Typecheck, customer build, fixture separation and bundle budgets pass.

The current PR records exact-head hosted CI and release status. These synthetic reads, mocked denials and rendered records do not establish server authorization, live actor workflows or bank acceptance.
