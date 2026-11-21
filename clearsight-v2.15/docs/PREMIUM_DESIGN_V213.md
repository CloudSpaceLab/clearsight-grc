# ClearSight V2.13 — Premium interface system

21 November 2026 · Interactive implementation reference

## Design decisions

The interface gives priority to the current task, its accountable person and the next available action. It uses a restrained navy and cobalt palette, consistent type and spacing, and solid reading surfaces. Translucency belongs to the fixed navigation bar and modal backdrop. Records, decision forms, document previews and charts retain an opaque background.

The original ClearSight mark, canonical terminology and role-aware navigation are retained. The update adds no invented metrics, trend lines, personas or permissions. A changed visual emphasis does not change a source result or its meaning.

### Shared visual language

| Element | Decision |
| --- | --- |
| Text | Local system/Inter stack; no font download. Page titles 28–35 px, main section headings 15–17 px, record text 12–13 px, long-form source text 13–14 px. |
| Spacing | Shared 4, 8, 12, 16, 20, 24, 32 and 40 px tokens. Related fields sit closer than separate task sections. |
| Corners | 8 px small elements; 12 px grouped controls; 16 px main panels; 20 px dialogs. |
| Surfaces | Light canvas `#f3f5f9`, white content, navy `#17263d`; dark canvas `#0f1724`, content `#172234`, text `#ecf1fa`. |
| Action accent | Cobalt `#245dd8` for solid primary controls. Links and semantic status colours adapt to theme. |
| Elevation | A subtle shared card shadow; greater elevation only for dialogs. No animated lift on data rows. |
| Numbers | Tabular numerals for comparable measures, amounts, dates and record facts. Units and source context remain explicit. |
| Status | Existing words and icons remain visible. Colour supplements the text rather than replacing it. |
| Focus | A visible 3 px outline; selected radio-card surfaces also retain a boundary. Forced-colour and reduced-motion rules are included. |

A calculation of the new shared foreground/background token pairs covered body, secondary and subdued text; five semantic statuses across light/dark themes; and the primary button label. All 17 checked pairs exceed 4.5:1. The lowest is light subdued text on a soft surface at 4.82:1. This is a token-level calculation, not whole-application accessibility certification: existing domain styles, native controls, real layout and assistive technology need browser acceptance.

## Display preferences

The topbar appearance control opens **Display preferences**. The same command is available in Workspace settings. It contains explicit Light/Dark and Comfortable/Compact choices, selected with native radios and applied through the existing guarded save dialog.

- Theme remains `S.theme`.
- Density is `S.displayPreferences.density`, defaulting to `comfortable`.
- Compact spacing is effective on desktop at 1024 px or wider with a fine pointer. It reduces table padding, toolbar spacing and compact queue rhythm while retaining readable type.
- Phone and coarse-pointer controls keep comfortable target sizes. Long-form inputs retain their normal spacing.
- Applying preferences updates root attributes and explanatory labels in place. It does not call `render`, replace the current form, change URL filters, reset scroll, clear a draft or modify access rights.
- Invalid values are rejected before any state mutation. Missing or malformed stored choices use safe display defaults. A different open dialog is protected from replacement.
- Persistence uses the existing browser workspace state. If persistence fails, the notification describes session-only application.

The existing quick theme action also preserves the current page. The V2.13 state key reads V2.12 and older supported state before falling back to the synthetic seed.

## Navigation and responsive layout

The full-width desktop header is 80 px high. Scope selectors have visible labels, readable values and consistent control geometry. Search, appearance and acting identity have distinct click targets. The desktop rail is 236 px, with a compact 80 px icon rail on tablet. Icon-rail navigation retains explicit accessible names and titles.

At widths below 1024 px, scope moves onto a second header row; the header becomes 140 px. It keeps Branch, Department and Unit available without squeezing their text beside the brand and utilities. On phones, each selector remains a native select with a 16 px value and 44 px height. Choosing its All option clears that level; the tiny individual clear icons are omitted at this width. The rail uses the existing mobile drawer.

Content grids collapse before narrow reading columns become impractical. Ordinary register rows keep the existing phone card layout, while KRI grids and document worksheets preserve row/column relationships in scrollable tables. Detail sidebars and sticky submit areas return to normal flow on smaller or short viewports where they would compete with editable content.

## Workflow refinement

| Workflow | Presentation improvement | Preserved behaviour |
| --- | --- | --- |
| My workspace and Work | Clearer named identity, responsibility lens, pending action, source, owner and due date; readable queue headers. | Actual actor, exact scope, responsibility lens and source command guards. |
| Inbox | Dedicated action track, responsive action placement, distinct read/acknowledgement versus source-resolution cues, labelled notification groups. | Recipient isolation, original source routes, read/archive actions and actual source state. |
| Roles and escalation | More readable capability comparisons, assignment facts, policy levels and simulation timeline. | Explicit restrictions, expiry, proposed changes and actual effective-access evaluation. |
| Assessments and reviews | Clear question headings, readable help, response identity, source facts, review decisions and history. | Field names, IDs, response versions, independent review and document lineage. |
| Branch KRI | Stronger section orientation, exception detail grouping and actionable return controls. | Source workbook schema, explicit unknown/unavailable/N/A states and review history. |
| Losses and vendors | Calmer step progression, grouped capture fields, clearer summaries and review actions. | Capture rules, financial ledgers, maker/reviewer responsibilities and vendor activation requirements. |
| Documents | More generous reading area, clearer version controls, preview framing and compare layout. | Exact file versions, canonical metadata, preview safety, visibility and management authority. |
| External recipient form | Standalone readable typography, question cards, supplied document controls and mobile layout. | Entire original recipient HTML remains identical after removing the one constant added stylesheet. |
| Reporting | A clear narrative lead, factual measures, accountable decisions, chart labels, confidence and source context. | Original values, units, filters, links, captured populations and frozen report output. |

The workflow-specific details are in `PREMIUM_WORKSPACES_V213.md`, `PREMIUM_FORMS_V213.md` and `PREMIUM_REPORTING_V213.md`.

## Print behaviour

Printing uses light, opaque surfaces regardless of the screen theme. Application chrome and interactive controls are suppressed. Reporting styles preserve scope, reporting clocks, units and captured context; long registers may continue across pages with repeated headers.

Only analytical disclosures inside reporting views are temporarily opened before print: chart equivalents, methodology, KRI group details and response history. Their exact previous state is restored afterward, including when a browser emits repeated print-entry events. This does not rewrite `S`, the frozen renderer or captured report objects. Native print pagination remains a browser acceptance task.

## Integration and performance

The four premium adapters load after the established V2.12 modules and before bootstrap:

1. `premium-foundation` — semantic tokens, shell, shared controls, tables, preferences and display attributes.
2. `premium-workspaces` — route-specific queue/governance presentation and narrow accessibility decoration.
3. `premium-forms` — workflow family styling and the isolated recipient stylesheet.
4. `premium-reporting` — report wrappers and temporary print disclosure state.

There are no new remote dependencies, runtime frameworks, font requests, polling loops, mutation observers or image assets. Most work is CSS. The JavaScript adapters make small changes during the existing render sequence; appearance changes do not trigger a page render. Glass blur is confined to a limited navigation/backdrop area and has opaque fallbacks for reduced transparency.

## Acceptance boundary

The source suite checks functional preferences, preserved response/permission/report contracts, recipient isolation, route/action wiring and transient print state. A browser was not retried after the previously observed execution restriction. The final acceptance pass must inspect actual light/dark layouts at desktop, tablet and phone widths; keyboard focus and native dialogs; text zoom; document scrolling; and print-to-PDF. The historical V2.7 screenshots do not represent V2.13.
