# V2 navigation adaptation — source audit and decision

Date: 2026-10-21. Tracking: #462. Baseline: PR #578 head `9aee7e2ebce51d567c5868b176a641c221f18d5c`. Implementation branch: `agent/462-v2-menu-adaptation-20261021`.

**Status: partial V2 layout integration. The 0/19 whole-screen acceptance register and production cutover block remain unchanged.** This document records the menu decision and the scoped implementation review below. The review verifies shared navigation adaptation; it does not establish whole-screen or backend acceptance.

## Instruction and precedence

The user instructed: “continue, the menu should conform too not just the layout”. The shared global menu must therefore adapt to the approved V2 source, alongside each specialist page's local navigation.

This decision supersedes the earlier requirement to keep the old Portfolio hierarchy and the prohibition on adapting source global navigation. It specifically overrides that part of archived unified-specification §1, the source README's global-navigation restriction, and Reports notes' Home → Portfolio → Reports instruction. Those files remain original reference evidence: **do not edit the archived HTML, CSS, JavaScript, notes or manifest, and do not recompute their hashes to conceal a change**. Root `AGENTS.md`, `DESIGN.md`, the structural parity contract and the cutover plan carry the current implementation rule.

The existing authenticated routes, actor identity, tenant/legal-entity/organization scope, field semantics and backend authorization remain authoritative. Adapting a menu does not create a domain capability, a new store, an available search service or an approved workflow outcome.

## Source evidence

Paths below are relative to `docs/design/references/clearsight-2026-10/source/`. Line numbers refer to the immutable source, including minified lines. The [manifest](../references/clearsight-2026-10/source/MANIFEST.json) pins the complete archive.

| Source | Relevant lines | SHA-256 |
| --- | --- | --- |
| [Reports](../references/clearsight-2026-10/source/clearsight-reports-insights-designs/reports-workspace-preview.html) — `clearsight-reports-insights-designs/reports-workspace-preview.html` | Global menu 83; local menu 90–95; local selection 215–227; mobile toggle 432–433 | `97f9d04092fdad07e5081820ad913295dc17d8344a693bd229c52d521c5b8895` |
| [Program](../references/clearsight-2026-10/source/clearsight-program-designs/program-workspace-preview.html) — `clearsight-program-designs/program-workspace-preview.html` | Global menu 94–105; local menu 119–125 | `b91aa4a96bfae4fc712f49e883996eb770d74dc3ae4b588ab6e573b4ec9d846f` |
| [Assessment / Forms](../references/clearsight-2026-10/source/clearsight-designs-01-05/assessment-workspace-preview.html) — `clearsight-designs-01-05/assessment-workspace-preview.html` | Global menu 52–60; modes 65; section navigation 68 | `dc04b081d33020bb1a3b00a892d09035f506de39773bd9550b60a20c78640297` |
| [Framework Library](../references/clearsight-2026-10/source/clearsight-framework-designs/framework-library.html) — `clearsight-framework-designs/framework-library.html` | Global menu 49–57; local menu 67 | `193d13b0cf87381f20b7d84c3d4c50c80b0e064274a623fa98257506df8284bf` |
| [Framework Adoption](../references/clearsight-2026-10/source/clearsight-framework-designs/framework-adoption.html) — `clearsight-framework-designs/framework-adoption.html` | Global menu 145–153; breadcrumb 158 | `8cad235bcccc1131964f5e37c237b3034075fb8130e45d7bf667111bcfc963ed` |
| [Work](../references/clearsight-2026-10/source/clearsight-work-approvals-designs/work-approvals-preview.html) — `clearsight-work-approvals-designs/work-approvals-preview.html` | Global menu 51; local menu 57–61; compact expansion CSS 21–22 and toggle 253 | `202ed9c2b22ac4e921d9ba14aa41b61644ea54cbe55abcc81a86ec10b10ee8d2` |
| [Unified specification](../references/clearsight-2026-10/source/clearsight-unified-design-system-v1/clearsight-design-system/clearsight-design-system-spec.md) — `clearsight-unified-design-system-v1/clearsight-design-system/clearsight-design-system-spec.md` | Prior navigation decision 21; layout guidance 73–76 | `13e5531114779672c7920e939206db1e383be664af67718b617251efaf496c17` |

### Common structure and conflicting variants

Seventeen HTML files share exactly **Home, Frameworks, Programs, Forms, Vendors, Risk & Losses, Privacy, Reports**, with **Settings** separately at the bottom. They include the paired Reports, Vendor, Operational Risk and Privacy files; the Framework package's Library/Adoption; the three Program package HTML files; and the four `clearsight-designs-01-05` HTML files. These are repeated source copies, not 17 independent product decisions. The Program previews in the two packages have the same SHA-256.

The common menu is flat: icon plus label, no group headings, no Portfolio intermediary and no expanded subgroups. A tinted selected row and a leading vertical marker identify the active destination. The source uses 49px rows, 20px icons and a 6px row gap. Settings is outside the main list. Prototype Settings controls are inert or show a preview toast; production must map the destination to the existing authorized configuration route.

| Variant | Difference from the common eight-item menu | Resolution |
| --- | --- | --- |
| Work | Uses Assessments and Privacy & Data, puts Risk & Losses before Vendors, inserts Work & Approvals before Reports | Keep the common Forms/Privacy labels and order; insert Work & Approvals immediately before Reports |
| Controls / enterprise setup / integrations / search / notifications | Some add Controls & Policies and Setup; casing and Settings placement vary | Keep supported administration contextual under existing Settings; do not manufacture specialist routes |
| Audit | Adds Audit & assurance | Existing audit-related records do not establish a complete audit workspace |
| Contextual Copilot | Adds Search and Controls, omits some common entries | Global search and a control destination require their own authorized contracts |
| Unified design preview | Foundations, Components, Work patterns, States & motion, Screen library | This is documentation navigation, not a product route reference |

The prior unified specification was explicit about retaining six older product routes. It also used a 220px desktop rail, while the common product HTML uses 228px. The user's later instruction now authorizes the common product-menu adaptation; using the old specification to retain Portfolio would contradict that instruction.

Prototype links are not route authority. Reports' Home link points to a Program preview; Work's Home link points to the Reports preview; other files use placeholder fragments. Production Home must continue to use the actual Home route.

## Shared menu and real-route mapping

One shared configuration owns the desktop rail, compact icon rail and labelled compact-menu sheet. It contains no sample counts or unsupported destinations.

| Order / label | Existing destination | Active-state and capability boundary |
| --- | --- | --- |
| 1. Home | `#oversight` | Preserve the existing `#today` alias and Home subviews; retain role-specific default view |
| 2. Frameworks | `#programs/library` | Selected only for the Framework Library target; using the `programs` route family must not also select Programs |
| 3. Programs | `#programs` | Selected for the register and exact Program sections, including existing adoption handoffs |
| 4. Forms | `#forms` | Existing template library and its routes; do not relabel RCSA or one assessment as a generic assessment library |
| 5. Vendors | `#vendors` | Includes register and exact relationship targets under the existing vendor route family |
| 6. Risk & Losses | `#risks` | Existing risk register is the entry; contextual destinations cover the other backed risk surfaces |
| 7. Privacy | `#ropa` | Opens Processing activities; this does not assert that Privacy Overview, DPIAs, breaches or requests are implemented |
| 8. Work & Approvals | `#work` | Includes the existing assigned queue, matters and evidence routes; no new task/approval store |
| 9. Reports | `#reports` | Includes governed risk/loss analysis at the existing Insights deep link; preserves its exact period/entity/area contract |
| Footer. Settings | `#configure` | Shown only under the existing configuration permission; configuration subareas and Imports remain contextual |

Risk & Losses provides a bounded contextual menu in this order: **Risks** (`#risks`), **RCSA** (`#rcsa`), **Key indicators** (`#insights`), **Loss events** (`#losses`). These are existing surfaces with distinct risk, assessment, indicator and monetary-flow semantics. The primary Risk & Losses item remains selected for those targets, except that `insightsView="risk-loss"` belongs to Reports. Do not fabricate a combined Risk & Losses dashboard or select a reporting period merely to make a menu destination work.

The Reporting analysis deep link keeps the current validated `view`, `start_date`, `end_date`, `legal_entity_id` and optional `organization_scope_id` query. Indicator targets keep their exact indicator and kind. Route presentation must not change fail-closed behavior for malformed, foreign or unresolved scope. Imports retains its existing `#imports` route and document-import capability; Settings is its menu context when that footer destination is authorized. This presentation does not grant configuration permission or remove permitted contextual import entry points from other workspaces.

Remove the generic Portfolio lens above specialist pages and the old Portfolio primary entry. Remove the standalone primary Insights entry after its existing destinations remain available through Risk & Losses and Reports. Preserve exact record links, browser history, current workspace-entry semantics and authorized contextual entry points such as Imports, people and configuration subareas. A menu click must not reset the verified enterprise scope or silently open an unrelated route.

Only the actual destination receives `aria-current="page"`. Frameworks and Programs must be disjoint. A compact menu uses the same active calculation; labels and ordering must not vary by page.

## Responsive anatomy and interaction

| Width | Common Program / Framework / Assessment source | Reports source after its full CSS cascade | Shared runtime target |
| --- | --- | --- | --- |
| 1440, 1280 | 228px, icon and visible label | 228px, icon and visible label | 228px, icon and visible label |
| 1024, 768 | 72px icon rail | 72px icon rail | 72px icon rail and labelled menu trigger |
| 561–1024 | 72px icon rail | 72px icon rail | 72px icon rail and labelled menu trigger |
| 481–560 | 55px icon rail | 72px because the later ≤900 rule overrides the earlier ≤560 rule | 56px icon rail and labelled menu trigger |
| 390 and other widths ≤480 | 55px icon rail | 56px icon rail | 56px icon rail and labelled menu trigger |

Program source CSS lines 7/15/17, Framework lines 8/16/18 and Assessment lines 3/11/13 establish the common rail. Reports lines 3/11/13/54/56 establish its cascade. The shared runtime uses **228px above 1024, 72px at 561–1024, 56px at ≤560**. This restores the desktop width by 8px relative to the previous 220px runtime and normalizes the common 55px phone rail by 1px. The 481–560px Reports exception is resolved in favor of the common phone breakpoint; it is not an unnoticed CSS mismatch.

At the baseline revision, `App.tsx` lists Home / Work / Portfolio / Insights / Reports, with Configure in the footer; seven Portfolio lenses then expose governed records. Its CSS resolves to 220px at 1280/1440, 92px at 1024, 72px at 768 and 56px at 390 (`product-finish.css:16`, `enterprise-shell.css:121–222`, `approved-shell.css:44–108,172–175`). This is a code audit; actual before/after render measurements must be retained by the implementation review.

The prototypes do not provide one dependable mobile expansion. Program, Forms and Framework have no expanded navigation state. Work adds a 230px fixed expansion at ≤800px but its toggle does not implement the complete dialog keyboard contract. Reports toggles `mobile-open` in JavaScript without a matching CSS selector. Copying these behaviors would preserve their accessibility gaps.

Use the existing default-size `FocusedSheet` with `placement="start"` for a labelled compact-menu view. Its opener is **Open navigation**, its label is **Navigation**, and its close control is **Close navigation**. Keep the source icon rail and make every destination discoverable through visible text in the sheet. Reuse shared focus containment, Escape, dismiss and focus restoration. The underlying rail and page remain outside the modal interaction while it is open; this is one menu model, not a second mobile bar. Close the sheet after navigation and verify focus remains valid after route changes and resizing.

Every rail, menu and contextual action must retain a minimum 44×44px target and visible keyboard focus in both themes. At short heights, the rail/menu content must scroll without hiding Settings or the current item; do not copy the source's 840px minimum sidebar height. `ActionLink variant="navigation"` owns the full-width row and shared hover/focus treatment. Current page and current location share its selected paint; only `aria-current="page"` displays the leading marker. Its existing default presentation remains unchanged. `FocusedSheet` adds reusable `placement="start" | "end"`, with end as the default and the existing full-height mobile replacement and focus lifecycle. Shared tokens, CSS and gallery examples own these extensions; feature CSS must not override their interaction states.

`AppNavigation.tsx` owns the shared destination composition; `app-navigation.css` owns rail, sheet-content and breakpoint geometry. Both are registered with the UI contract gate. Shared navigation state styling stays in `design-system/components/navigation.css`, and sheet placement stays in `design-system/components/overlays.css`. Remove superseded sidebar geometry from the older shell styles instead of keeping competing owners. Feature geometry may set registered navigation padding, gap, font-size and minimum-height tokens; it cannot change palette, selected, hover or focus tokens.

## Local-menu audit and boundaries

| Page | Source local order and behavior | Baseline runtime / required handling |
| --- | --- | --- |
| Reports | Overview, Explore, Reports, Schedules; Build report appears only while the builder is selected (HTML 90–95, JavaScript 215–223). Single scrollable underline row, including mobile | Runtime has the same five destinations in order, with Build report always visible. This is an explicit remaining source difference, not proof of exact local parity; retain the existing real builder operation |
| Program | Overview, Requirements, Assessments, Evidence, Issues, Activity; single scrollable row, smaller gap at ≤760 | `ProgramDetailSections` already matches the six labels/order and `program-record.css` keeps the mobile row. Data collection remains contextual at its existing route |
| Framework Library | Explore, In my Programs, Edition updates; Explore selected initially | Runtime exposes the same order. In my Programs opens the existing Programs surface; Edition updates is unavailable because the current catalog has no update feed. Do not copy the sample counts |
| Assessment / Forms source | Questions, Excel import, Review; section choices Access management, Identity controls, Supplier access, Monitoring. Sections become a horizontal row at ≤850; mode tabs scroll at ≤600 | This is an individual assessment, not the Forms template library. Keep its adaptation within authorized capture/assessment work. Do not replace the existing Forms library's Templates/Sent forms/Responses/Documents/Policies/Imports/Communications with unrelated assessment modes |
| Work | My work, Approvals, Team queue, Completed; scrollable row at ≤760 | Runtime has genuine My work and Approvals. Team/Completed require their own backed population or explicit unavailable treatment; a global-menu change does not make those queues available |

Local selection must reflect the actual route/view and preserve drafts and exact record context under the existing owner. No prototype badge count may become a production total. Shared Tabs, links, buttons and sheet contracts continue to own keyboard and interaction behavior.

## Verification and acceptance

The implementation review must retain the baseline and after-state, exact revision, source hashes and the following evidence:

- Same-screen source/runtime menu pairs at 1440, 1280, 1024, 768 and 390px, with relevant light/dark states. Include 560/561 and 1024/1025 boundary checks to resolve the documented breakpoint normalization.
- Exact item order, one primary active destination, separate Frameworks/Programs selection, Risk & Losses contextual selection, Reports risk/loss deep-link selection, and absence of the old Portfolio lens or duplicate mobile bar.
- Real destination navigation and browser Back/Forward, exact-record history, Forms entry/history behavior, enterprise scope preservation, and existing Settings permission/denied behavior.
- Keyboard activation of the compact menu, initial focus, focus containment, Escape and focus restoration; visible destination labels, 44px targets, short-height access, no page-level horizontal overflow and reduced-motion behavior.
- Honest loading, unavailable, empty and denied destination states. Menu order and styling cannot conceal a missing workflow or widen an actor's access.
- Focused unit/route tests and browser assertions, customer/evidence build and applicable CI on the final implementation revision. Assertions must measure layout and routes, not merely find labels in DOM order.

The review record must name the reviewer and review type. For example, `ChatGPT /root — AI visual and interaction review` may document inspected source/runtime captures, code and browser results with exact revisions and limitations. It must not be presented as a bank-user acceptance, four-actor run, server authorization proof or production approval. Those results require their own retained backend and operational receipts.

Until those whole-screen gates are met, report the work as **partial V2 layout integration** and leave the [0/19 register](../references/clearsight-2026-10/CUTOVER-STATUS.md) unchanged.


## Implementation review — 21 October 2026

**Reviewer:** ChatGPT `/root` — AI source, visual and interaction review, with independent `/root/layout_audit` accessibility checks and `/root/ci_review` browser-evidence checks. **Result: the shared menu adaptation is verified in the recorded scope. Whole-screen acceptance remains 0/19.**

### Revisions and baseline

- Baseline: [PR #578](https://github.com/CloudSpaceLab/clearsight-grc/pull/578), head `9aee7e2ebce51d567c5868b176a641c221f18d5c`. Retained baseline UI artifact: [12351202190](https://github.com/CloudSpaceLab/clearsight-grc/actions/runs/39611243663/artifacts/12351202190); ZIP SHA-256 `69b77adb115dca760ea2a617cbef144b5f1d1ac5a6440b6af4215adbe61b02b2`.
- Menu layout and interaction checkpoint: `67e25eb2e8fb997800b4c43115af7f820259f5cc`, following `df6a75f310664fb80ddf44964a25271f330a32b0`.
- Final implementation checkpoint: [`32ac98ce80f21424904b38a57ef5e31443cac2cd`](https://github.com/CloudSpaceLab/clearsight-grc/commit/32ac98ce80f21424904b38a57ef5e31443cac2cd). This adds valid naming roles to the brand, actor context and existing noninteractive ranked-chart rows; it does not change the captured layout.
- The following evidence/guardrail commit changes scripts and documentation. Final-PR-head CI and artifact verification must be recorded in the PR before it is marked ready; this local receipt does not substitute for those checks.

The reviewer inspected both baseline Overview renders at 1440/1024, all ten updated Overview renders at the five widths in light/dark, both expanded phone-menu renders, and the five original Reports HTML renders. The original source hashes in this decision remain unchanged.

### Before and after

| Element | Baseline | Verified adaptation |
| --- | --- | --- |
| Primary destinations | Home / Work / Portfolio / Insights / Reports | Common source order with Work & Approvals inserted in its Work-source position; nine real destination links |
| Frameworks | Reached through Programs | Direct `#programs/library` link and distinct current state |
| Risk surfaces | Split between Portfolio lenses and Insights | Risk & Losses with four existing contextual destinations; governed risk/loss analysis remains under Reports |
| Settings | Configure footer action | Settings footer link, same verified capability gate |
| Desktop rail | 220px | 228px; 13px labels, 49px rows and source-spaced primary list |
| 1024px rail | 92px with small labels | 72px icon rail plus labelled navigation sheet |
| 768 / 390px rail | 72 / 56px | 72 / 56px with all nine destinations and a functioning labelled menu |
| Duplicate structure | Portfolio lens and mounted mobile navigation | Both removed from the production App shell |
| Brand subtitle | Clipped in the 1440 baseline | Fits the full desktop rail |

The labelled sheet uses 44px rows with 3px gaps so nine primary links, four expanded risk links, Settings and the signed-in actor fit at 390×844. At shorter heights its list scrolls independently. It intentionally adapts the source's incomplete mobile toggle through the shared modal contract. Original HTML sample counts, unsupported routes and product-specific palette values are not promoted into runtime facts or replacement component contracts.

### Retained browser and source evidence

The `navigation` scope in `capture-ui-evidence.mjs` reuses the ten Reports Overview states and adds the two expanded phone sheets and two Settings-denied states. **All 14 captures pass**, including current-state clipping measurements. The full suite adds only three screenshots because one denial state already existed.

| Width | Light capture | Dark capture | Rail | Source pair |
| --- | --- | --- | --- | --- |
| 1440×900 | `140a-report-overview-light-1440x900.png` | `140j-report-overview-dark-1440x900.png` | 228px | `approved-reports-insights-1440x900.png` |
| 1280×800 | `140h-report-overview-light-1280x800.png` | `140k-report-overview-dark-1280x800.png` | 228px | `approved-reports-insights-1280x800.png` |
| 1024×768 | `140f-report-overview-light-1024x768.png` | `140l-report-overview-dark-1024x768.png` | 72px | `approved-reports-insights-1024x768.png` |
| 768×900 | `140i-report-overview-light-768x900.png` | `140g-report-overview-dark-768x900.png` | 72px | `approved-reports-insights-768x900.png` |
| 390×844 | `140b-report-overview-light-mobile-390x844.png` | `140m-report-overview-dark-mobile-390x844.png` | 56px | `approved-reports-insights-390x844.png` |

Additional views are `221-navigation-menu-{light,dark}-mobile-390x844.png`, `22-no-config-access-light-1440x900.png`, and `22a-no-config-access-dark-mobile-390x844.png`. The source uses its authored light palette; dark runtime checks validate the same anatomy under the shared dark tokens.

The five source/runtime comparisons pass verified file hashes, common-destination order, active page, visible/hidden label mode, one-line desktop labels and measured rail widths. Work & Approvals is recorded as the deliberate Work-source addition. The menu review gate requires these measurements and cannot pass from screenshot names alone.

The local manifests are retained under `visual-review/menu/` in the review workspace. Their hashes are recorded here; the same named cases and menu metadata are produced by the PR's full UI workflow:

- Runtime manifest SHA-256: `a603390fab10a78bc02b21e36ccf8ea70e19dd9da13c8b66a3615b06cbbfcdfb`.
- Source comparison manifest SHA-256: `5a751d04bbb83235185aa4aa3f803a72b6830b787ddc31177dcedd145d4c8ea1`.

### Interaction, accessibility and regression results

- Real links retain Ctrl/Command/Shift/Alt/middle-click behavior. Selecting an active workspace clears its record target. Frameworks and Programs navigate distinctly; browser history restores exact risk/loss targets and the correct menu state.
- Reports remains current for its governed risk/loss Insights link, with existing period and scope validation. Runtime permissions continue to determine Settings visibility and access; denied actors have no Settings link in either rail or sheet.
- Keyboard proof covers opening, visible focus, Tab and Shift+Tab containment, direct Frameworks navigation, browser Back, risk disclosure and child selection, Escape, selection dismissal and focus restoration. Same-workspace target changes close the sheet. Expanding to desktop restores a visible current link, or a primary-link fallback for a contextual route.
- At 320×568, every primary link, expanded risk child and Settings is reachable by keyboard and scrolling, with no horizontal document overflow. In the 390×844 expanded sheet, current Reports spans y=657.14–701.14 inside the actual y=93.14–711.14 scrollport. The regression gate compares clipping bounds rather than treating a DOM rectangle as proof of visibility.
- Eight boundary checks, in both themes, confirm 560→56px, 561→72px, 1024→72px and 1025→228px. At 1024 the sheet starts at x=0 with the shared 560px width; outside-click dismissal returns focus to its opener.
- Six scoped axe WCAG 2 A/AA runs have **zero automated violations**. Both expanded-menu runs have zero incomplete findings. All 16 painted sheet controls are at least 44×44px. See the retained [accessibility receipt](2026-10-21-v2-navigation-accessibility.json), which pins the implementation files. The offscreen React Aria dismiss helper is distinguished from visible touch targets.
- Reduced-motion checks find no active animations, `animation-name:none` and `transform:none`. The pre-existing global reduced-motion rule uses 0.01ms duration values; this is not reported as literal zero duration.
- Local full web regression passed **256 files / 1,964 tests**. A focused rerun exposed a 1-second test wait for the real lazy-loaded Configuration workspace under build contention; the wait is now bounded at 5 seconds, consistent with existing workspace tests, while exact route/heading/selection assertions remain. The resulting App suite passed 65/65; the final navigation, shared-control and Reports suite passed 43/43. TypeScript, UI contracts, runtime-fixture boundary and copy-quality checks passed. Customer and evidence builds passed.

Two visual iterations corrected findings that basic DOM checks had missed: the Risk & Losses label wrapping alongside its 44px disclosure, and the floated close control narrowing the mobile menu. A third tightened the expanded sheet so the current Reports row remains visible. The added browser gates retain all three protections. Accessibility review also corrected invalid naming on generic brand/actor elements and the pre-existing static ranked-chart rows.

**Limits:** seven gradient-backed Reports eyebrow/heading/tab nodes per Overview scenario remain axe manual contrast-review entries; the navigation has no unresolved automated or incomplete findings. These deterministic fixtures are not bank-user acceptance, live authority execution, four-actor proof or source-to-report reconciliation. Reports Explore/Builder and other specialist page layouts retain their separately documented remaining work. No whole-screen acceptance count is advanced by this menu review.
