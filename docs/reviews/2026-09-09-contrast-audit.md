# Rendered contrast audit — 9 September 2026

The inspected UI has repeatable contrast defects in active placeholders, shared input boundaries, and light-theme status and explanation text. These affect ordinary work, including finding records, editing questions, reading whether an action is overdue, and reviewing a vendor relationship. No production source was changed by this audit.

## Scope and evidence

Source baseline: `11687656022fa6479ec9642af2bd546c875b78c9`, with the existing local working tree as served by the current evidence build at `http://127.0.0.1:4179`. This is deterministic sample-fixture evidence using production React components, not proof of a hosted production deployment or data population.

The audit uses the repository's established Playwright fixture routes and interactions, bundled Node 24.19.0, Chromium, and installed axe-core 4.12.1. All artifacts are in [`../evidence/2026-09-09-ui-audit/contrast/`](../evidence/2026-09-09-ui-audit/contrast/). [`audit.mjs`](../evidence/2026-09-09-ui-audit/contrast/audit.mjs) preserves full-page baseline PNGs, axe violations and incomplete nodes, browser errors, headings, and computed readings. [`aggregate.json`](../evidence/2026-09-09-ui-audit/contrast/aggregate.json) combines the individual results. [`focus.json`](../evidence/2026-09-09-ui-audit/contrast/focus.json) retains raw focus styles and matching CSS rules.

**104/104 requested fixture/theme/viewport combinations captured; 0 unreachable final states and 0 captured browser runtime errors.** Axe recorded **17 text-node violations across 11 captures**, all light theme, and **2,715 incomplete node occurrences across 102 captures**. These counts include repeated components and are not counts of unique defects. Incomplete means unresolved by axe; a zero-violation capture is not a contrast pass. Four initial Select-popup timing errors were resolved by waiting after scrolling before opening the menu. Their `-error.png` files are retained as harness troubleshooting artifacts, not final state evidence. The initial external-capture URL was corrected to use the harness's public sample access fragment before recording final counts.

Every row below was rendered in **light and dark**, **1440×900 and 390×844**, comfortable density: 26 named states × 4 combinations.

| Workspace | Tested states / evidence filename prefix |
| --- | --- |
| Today | `today`, `today-empty`, `today-unavailable` |
| Programs | `programs`, `program-detail` (NDPA overview) |
| Work | `issues`, `issue-detail` (overdue action), `evidence`, `imports` |
| Forms | `forms-library` (lifecycle list), `forms-builder` (scoring form canvas), `forms-sent` (delivery history), `forms-response` (revision review) |
| Vendors | `vendors`, `vendor-checklist`, `document-select`, `document-review` |
| Configuration | `configure`, `configure-authority` |
| External capture | `capture` (held-record response), `capture-received`, `capture-all-held` |
| Shared / specialist components | `gallery`, `gallery-select` (popup open), `field-review`, `field-builder` |

Example: `forms-builder-light-390.png` and `.json` identify the same state. The 104 baseline PNGs remain available for inspection. Additional desktop `gallery-fields-{light,dark}.png` and `gallery-fields-{light,dark}-focus.png` show input identification before and after keyboard focus. Visual inspection concentrated on all finding families, their theme counterparts, and the narrow-screen versions; retaining a screenshot is not itself a claim that every pixel was manually inspected.

## Criteria and measurement limits

Normal text, including placeholders, requires **4.5:1**; large text permits **3:1**. Disabled/inactive components and logos have applicable exceptions. Relevant non-text identification and state indicators require **3:1** against adjacent colors. A decorative divider or the border of a clearly labelled text button is not automatically an essential control boundary. [W3C text contrast](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html), [W3C non-text contrast](https://www.w3.org/WAI/WCAG22/Understanding/non-text-contrast.html).

The explicit check uses the browser's color parser through a canvas, converting modern `color(srgb …)`/`color-mix()` values before alpha-compositing backgrounds from ancestors and compositing the text or placeholder color. It distinguishes disabled controls and records them without claiming a text failure. It clears ancestor-gradient uncertainty when a fully opaque descendant background covers that ancestor. A gradient or image above that opaque surface, group opacity, or a filter remains marked uncertain. Supplemental colors are rounded to 8-bit sRGB and may differ slightly from axe's compositing; the supplied ratios must not be rounded up to meet a threshold.

The tool does not fully model overlapping siblings, pseudo-element backgrounds, shadow/blur painting, all internal scrolling, or every gradient's position. Axe results plus source and screenshot inspection establish the confirmed cases below. Any gradient-dependent supplemental estimate remains a follow-up, not a definitive failure. Shell gradient text remains partially unresolved. No application-wide WCAG compliance conclusion follows from this audit.

## Findings

### C-01 — P1: active placeholders are washed out in both themes

The global Tailwind preflight applies `::placeholder { color: color-mix(in oklab, currentcolor 50%, transparent); }`. Shared fields do not provide a theme-aware placeholder override. On the form canvas, this also halves an already-muted foreground. Axe did not report these as violations, so its ordinary text scan misses a recurring failure.

| Active field | Light foreground / background; ratio | Dark foreground / background; ratio |
| --- | --- | --- |
| Shared search: Forms library, document picker, gallery | `#8a929c` / `#ffffff`; **3.14:1** | `#727c88` / `#0d1826`; **4.21:1** |
| Legacy search: Programs, issues, evidence, vendor register | `#828c98` / `#eef3f7`; **3.05:1** | `#75818f` / `#132234`; **4.05:1** |
| Builder “Add guidance for the respondent…” | approximately `#a8b2bf` / `#f7f9ff`; **2.03:1** | approximately `#4f5d70` / `#0f1b2f`; **2.58:1** |

All require 4.5:1 at the rendered sizes. The builder guidance is 10px. This is active editable content, not a disabled-field exemption. The exact builder background varies by selected card; both measured cards are well below the threshold.

**Location:** `input.cs-search-field__control::placeholder`, legacy `input[placeholder]`, and `input.form-question-guidance::placeholder`. Sources: `web/node_modules/tailwindcss/preflight.css:298`; `web/src/design-system/components/fields.css:29` and `:61`; `web/src/design-system/tokens/components.css:38`; `web/src/form-builder-workspace.css:553`. **Evidence:** `forms-library-{light,dark}-390.png`, `forms-builder-{light,dark}-390.png`, `gallery-fields-{light,dark}.png`, `document-select-light-1440.png`.

**Remedy:** Give placeholders an explicit semantic foreground with sufficient rendered contrast in both themes; avoid inheriting a 50% transparent text reset. Preserve labels and distinguish guidance through layout/weight, not illegibility. Verify the shared contract and legacy fields together.

### C-02 — P1: shared text-entry boundaries disappear into same-color surfaces

The gallery TextField and SearchField use the same fill as their containing card. Their 1px boundaries are `#c8d3dd` on white (**1.52:1**) in light mode and `#26394d` on `#0d1826` (**1.51:1**) in dark mode. For entry fields, this boundary identifies the interactive input area; a label alone does not show its extent. Both are below 3:1. This finding does not classify every low-contrast table line, card edge, or button border as a violation.

**Location:** `.cs-field__control`, `.cs-search-field__control`. Source: `web/src/design-system/tokens/components.css:37` binds `--cs-field-border` to `--cs-border-default`; `web/src/design-system/components/fields.css:34` and `:66` paint it. Theme values originate in `web/src/ui-preferences.css:13` and `:63`.

**Evidence:** `gallery-fields-light.png`, `gallery-fields-dark.png`; baseline computed readings in all gallery JSONs. **Remedy:** Introduce/use an input-identification border role meeting 3:1 in the actual fill and surrounding surface. Keep decorative separators independently subtle.

### C-03 — P2: translucent light-theme badges fail on tinted parent surfaces

| Rendered text | Foreground / composited background | Ratio | Reproduced state |
| --- | --- | ---: | --- |
| In progress | `#596c80` / `#dce3e9` | **4.17:1** | Work action, desktop/mobile |
| Overdue | `#b83e53` / `#e8dde3` | **4.12:1** | Work action, desktop/mobile |
| Ready to view | `#596c80` / `#dee5e9` | **4.24:1** | Selected document row, desktop |
| Evidence needed | `#8b5b00` / `#e2e1d9` | **4.45:1** | Gallery selected row, desktop/mobile, popup closed/open |

All are 12px bold and require 4.5:1. These are actual status labels; the colored marker does not exempt their text. The selected-row background matters: the same badge can pass on another parent surface. No corresponding dark-theme axe violation was found in these states.

**Location:** `.cs-status-badge.cs-tone--neutral`, `.cs-tone--error`, `.cs-tone--warning`; source `web/src/design-system/components/feedback.css:7` paints a 12% transparent tone background and `:8` uses the same tone as text. Tone mappings are at `:21–25`; semantic theme foregrounds are in `web/src/ui-preferences.css:67` and `:79–83`. **Evidence:** `issue-detail-light-{1440,390}.png`, `document-select-light-1440.png`, `gallery-light-{1440,390}.png`.

**Remedy:** Set explicit badge foreground/background pairs for the allowed parent surfaces, including selected rows; retest neutral, warning and error together. Keep “In progress”, “Overdue”, and other statuses concise and industry-neutral.

### C-04 — P2: response review overrides the success badge's foreground

“Low concern” in the response review sheet is `#596c80` on `#d4e4e3`, **4.12:1**, at 12px bold. The green success badge is rendered with muted gray text because `.forms-response-review__summary span { color: var(--cs-text-muted); }` wins over the shared component contract. This is a separate cascade defect from choosing safe shared badge colors.

**Source:** `web/src/components/forms/responses-view.css:44`. **Selector:** `.forms-response-review__summary > .cs-status-badge.cs-tone--success`. **Evidence:** `forms-response-light-{1440,390}.png` and matching axe records. **Remedy:** Scope metadata styling to its intended content and let StatusBadge own its foreground; then verify actual badge contrast.

### C-05 — P2: vendor activation explanations fall below 4.5:1

The policy timestamp, incomplete due-diligence explanation, and next-step notice are `#596c80` on `#dce9ed`, **4.35:1**, at 11–12px. They explain whether the relationship can be activated, so they must remain legible.

**Selectors:** `.vendor-activation-panel > p`, `.vendor-activation-gates li[data-satisfied="false"] p`, `.vendor-activation-panel > .inline-notice`. **Sources:** `web/src/vendors.css:51–53` creates the cyan-tinted surface and muted paragraph; the gate paragraph uses the same muted role. **Evidence:** `vendor-checklist-light-{1440,390}.png`. **Remedy:** Use a supported paragraph foreground for that tinted surface, or reduce its tint, without weakening the activation limitation.

### C-06 — P2: text contrast coverage currently overstates what a successful runner proves

The repository accessibility runner fails after the first serious finding, so it cannot inventory the remaining routes. Axe also leaves gradient-dependent text and some short or obscured content incomplete. The explicit placeholder failures above are absent from its reported violations. A narrow button-only contrast helper in `web/scripts/forms-evidence-scenarios.mjs` parses RGB digits directly and compares the element's raw background; it is not a reliable general compositor for `color(srgb …)`, transparency, or layered surfaces.

**Remedy:** Retain separate violations, incomplete and unsupported readings; continue the audit across the whole matrix; test rendered placeholder colors and essential field boundaries; use a standards-aware color parser. Keep clear limits on the final claim. This recommendation is for the audit/test contract, not customer-facing product copy.

## Focus and unresolved areas

Keyboard Tab/Shift+Tab reaches the gallery TextField with `:focus-visible`; both themes show a substantial outline in the retained focus PNGs. The actual global override is `web/src/product-finish.css:10–12`, a 3px outline at 78% of `--focus-ring`, offset 3px. Raw computed colors, outline, border and matching rules are retained in `focus.json`. This representative check found a visible focus treatment; it does not verify every field, overlay, selected tab, browser or focus-obscuration state.

Supplemental readings also identify light-theme evidence-inspection guidance near **4.38:1** and Imports proposal/date metadata near **4.36:1**. Their painted/gradient contexts need additional verification before classifying them as confirmed failures. They remain in `aggregate.json` rather than being silently counted as passes.

Not covered: actual native browser zoom; forced colors in this new run; 320px/tablet/compact-density combinations; complete hover, pressed, invalid, loading and disabled permutations for every component; all tab-panel content, search results, error/revocation/conflict states and user role routes; all underlying document media, logos and illustrations; exhaustive status-by-parent-surface combinations; assistive-technology and human low-vision testing. State labels were read in context, but this is not a full use-of-color audit.

Recommended repair order: C-01/C-02 shared field tokens and browser reset, C-04 cascade override, C-03 shared badge pairs, C-05 tinted explanations, then C-06 broader regression coverage. Re-render and inspect affected workflows in both themes after each shared-token change; token arithmetic on a white swatch cannot substitute for the selected-row, sheet, and canvas contexts shown here.

