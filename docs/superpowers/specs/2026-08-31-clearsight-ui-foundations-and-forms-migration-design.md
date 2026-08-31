# ClearSight UI Foundations and Forms Migration Design

**Date:** 2026-08-31

**Status:** Tranche 1 implemented and locally verified; later Forms and product migrations remain open

**Scope:** Reusable internal UI foundations with Forms navigation and Sent forms as the first migration tranche

**Supersedes:** [`2026-08-31-immersive-form-builder-usability-design.md`](2026-08-31-immersive-form-builder-usability-design.md) as the implementation authority; that document remains the builder-specific audit record

## 1. Decision summary

ClearSight will establish a reusable product UI system before applying further screen-level polish. The system defines three-layer design tokens, ClearSight-owned React component APIs, complete interaction-state contracts and deterministic component evidence. Tranche 1 implements the component foundation, component gallery, Forms peer-view navigation and the visibly broken Sent forms surface. The immersive builder and remaining Forms views retain their existing usability corrections but remain separate migration work.

ClearSight owns the visual language, copy, variants and component contracts. The exact pinned `react-aria-components` package supplies accessibility, internationalization and interaction mechanics for complex primitives such as Select, ListBox, Popover, Dialog, Menu and Tabs. It does not supply ClearSight styling, tokens, product copy or workflow semantics.

This is not a cosmetic rebrand and not a big-bang rewrite. Existing domain commands, authority, data models and routes remain unchanged. Each migrated surface replaces raw controls and local visual rules with production components, passes the shared state gallery and leaves an enforceable boundary that prevents regression.

## 2. Product problem and evidence

### 2.1 Deployed screenshot findings

The supplied Sent forms screenshot and direct browser inspection show that the failure is below page hierarchy:

- **Send form** is rendered as a transparent, borderless 64×21 px button with no radius or component state.
- The Status and Due state controls are transparent, borderless 183×20 px native selects.
- Subject type, Subject ID and Owner are transparent, borderless 183×21 px native inputs.
- Opening Status invokes a light operating-system dropdown over the dark application because no themed Select contract exists.
- The table has an 820 px minimum width while sharing its row with a 280–360 px detail pane, so an empty result still produces a horizontal scrollbar.
- The empty result and unselected detail compete even though neither contains work.
- Tabs, filters, actions and table controls do not share height, fill, border, radius, focus or selected-state rules.
- Weak surface separation and low-contrast secondary text flatten the page into one dark field.

The builder screenshot exposes the same foundation failure at greater density: a native 19-option response-type select covers the working document, controls have inconsistent hit areas and the surrounding editor uses unrelated card, toolbar and inspector treatments.

### 2.2 Source findings

The current frontend contains approximately:

- 555 raw `<button>` elements;
- 218 raw `<input>` elements;
- 120 raw `<select>` elements;
- 85 raw `<textarea>` elements;
- 43 CSS files;
- 3,858 pixel literals;
- 337 `border-radius` declarations with at least 20 distinct values; and
- 210 hard-coded color values.

The application has isolated reusable components such as `FocusedSheet`, but no common Button, Field, Select, Tabs, DataTable, Menu, Badge, Surface or status foundation. Forms uses a `forms-primary` class without a corresponding primary-button definition. Feature CSS mixes global semantic variables, feature-local variables, fallbacks and raw values.

The tests mainly prove that elements and workflow results exist. They do not currently fail when a primary action is an unstyled 21 px text control, a native popup breaks theme parity or an empty data surface scrolls horizontally.

### 2.3 Reference lessons

Browser review of Fillout, Typeform, Radix Themes and Atlassian Design establishes the useful patterns without copying their brands:

- a focused builder with one component grammar across toolbar, field library, canvas and properties;
- categorized, bounded field selection rather than a long native menu;
- component families defined across size, theme, placeholder, loading and disabled states;
- separate Select, ComboBox, popup, checkbox and radio-selection contracts for different jobs; and
- a state gallery that makes interaction consistency inspectable before components reach product screens.

Premium quality here means predictable structure, typography, surfaces, states and motion. It does not mean stronger gradients, more glow or decorative cards.

## 3. Goals and non-goals

### Goals

- Make every migrated control visibly intentional, accessible and consistent in light and dark themes.
- Give product code stable APIs for common controls rather than screen-owned markup and CSS.
- Preserve one dominant next action and clear secondary/tertiary hierarchy.
- Standardize field anatomy, errors, help, read-only, disabled, loading and conflict feedback.
- Remove operating-system theme breaks for application selection menus while preserving semantic native controls where they are the correct interaction.
- Make responsive replacement, target sizing, focus behavior and overlay layering component responsibilities.
- Add automated and rendered evidence that catches the failures visible in the supplied screenshots.
- Migrate Forms completely without requiring a simultaneous rewrite of unrelated workspaces.

### Non-goals

- Rebrand ClearSight or replace its restrained institutional visual language.
- Adopt Radix Themes, React Spectrum styling, Material styling or another third-party visual skin.
- Change Forms schemas, scoring, revisions, distributions, responses, invitations or authority routes.
- Replace useful semantic native inputs such as date, time, number or file inputs merely to make them custom.
- Build a generic page-builder framework or duplicate the existing workflow/domain architecture.
- Rewrite every application workspace in one pull request.
- Claim product-wide visual consistency before non-Forms surfaces have migrated.

## 4. Foundation architecture

### 4.1 CSS cascade and ownership

Declare one explicit cascade order:

```css
@layer reset, tokens, base, components, features, utilities, overrides;
```

- `reset` contains the minimal box-sizing and browser normalization required by ClearSight.
- `tokens` contains primitive, semantic, component and preference mappings.
- `base` contains document typography and non-component element defaults.
- `components` contains only reusable UI component styles.
- `features` contains page composition and workflow-specific layout.
- `utilities` contains a small documented set of layout/accessibility helpers.
- `overrides` is temporary migration compatibility and must not receive new feature rules.

New component CSS must not depend on feature selectors. Feature CSS may arrange a component but may not redefine its internal padding, typography, radius, focus, disabled or state treatment.

### 4.2 Three-layer tokens

Create a single import under `web/src/design-system/tokens/`:

```text
primitives.css
semantic.css
components.css
index.css
```

#### Primitive tokens

Primitive tokens name raw scales without business meaning:

- neutral and brand color ramps;
- 4 px spacing increments;
- 12, 14, 16, 18, 22, 28, 36 and 48 px typography sizes with defined line heights;
- 6, 10 and 14 px radii plus `full`;
- 1 and 2 px border widths;
- icon sizes 16, 20 and 24 px;
- restrained elevation levels 0, 1, 2 and overlay;
- 100, 150, 200 and 300 ms durations;
- named easing curves; and
- a documented z-index ladder.

Raw primitive values are never referenced directly from feature CSS.

#### Semantic tokens

Semantic tokens preserve and complete the roles already named in `DESIGN.md`:

- canvas, surface levels, document surfaces and overlays;
- primary, secondary, muted and inverse text;
- default, strong, interactive and invalid borders;
- primary action, accent information and destructive action;
- success, warning, error, information and unknown statuses;
- focus ring and focus offset;
- page, section, group and inline spacing;
- interactive target and comfortable/compact control heights;
- content-width and reading-measure roles; and
- reduced-motion mappings.

`ui-preferences.css` remains responsible for System, Light, Dark, Comfortable and Compact preference selectors, but it maps semantic roles rather than defining a parallel palette. Comfortable controls are 44 px high. Compact controls are 40 px on fine-pointer desktop only. Touch, mobile and 200%-reflow states use at least 44 px targets.

#### Component tokens

Component tokens reference semantic roles for Button, Field, Select, Checkbox, Switch, Tabs, Menu, Popover, Sheet, Dialog, Badge, Notice, Table, Toolbar, Card and EmptyState. They define internal spacing, control height, typography, border, radius, fill, elevation and state mappings.

Component tokens are allowed only when a semantic token is insufficient. They may not encode page names such as Forms, Vendors or Programs.

### 4.3 React component boundary

Reusable components live under `web/src/components/ui/` and are exported only through `web/src/components/ui/index.ts`. Product code imports ClearSight components, not `react-aria-components` directly.

Pin `react-aria-components` at exact version `1.20.0` for the first implementation. It is used only inside the UI boundary. The package is unstyled; ClearSight CSS classes and tokens own every visible result.

Use semantic HTML directly where it already provides the correct behavior. Use React Aria mechanics where rebuilding focus, keyboard, listbox, popover, dialog or selection behavior would create unnecessary accessibility risk. No React Spectrum styling packages are introduced.

## 5. Core component contracts

### 5.1 Button and IconButton

Variants:

- `primary` for the one dominant action;
- `secondary` for a safe alternative;
- `quiet` for low-emphasis handling;
- `destructive` for an immediate destructive result; and
- `link` only for real navigation.

Every button supports default, hover, pressed, focus-visible, disabled and loading states without changing its outer dimensions. Loading preserves the action label for assistive technology and prevents repeat submission. IconButton always has a business-readable accessible name and at least a 44 px target outside pointer-fine compact tables.

Buttons begin with a verb and name the immediate result. A visually enabled button performs a real action.

### 5.2 FormField, TextField and TextArea

`FormField` owns label, required indicator, optional description, control slot, validation message and read-only explanation. It binds all accessible relationships and reserves error space only when needed.

TextField and TextArea support default, filled, focus-visible, invalid, disabled, read-only, loading and server-conflict presentation. Placeholder text never substitutes for a label. Read-only is visibly and semantically distinct from disabled. Mobile text controls use at least 16 px text to avoid browser zoom.

### 5.3 SelectField, ComboBox and choice controls

`SelectField` is for a bounded controlled list. It renders a ClearSight trigger plus React Aria Select/ListBox/Popover mechanics, matches the active theme and supports typeahead, arrow keys, Home/End, Enter/Space, Escape, selected-state announcement and focus restoration.

`ComboBox` is reserved for searchable or remotely loaded options. It exposes loading, empty, partial, unavailable and retry states. It never loads a broad population and relies on server-bounded search.

Checkbox, RadioGroup and Switch remain distinct:

- Checkbox selects or confirms a value.
- RadioGroup selects one of a small visible set.
- Switch changes an immediately effective reversible setting.

A long list of response types uses a categorized picker, not SelectField or a native select. Native select remains permitted only where the platform picker is intentionally part of an external/mobile capture design and the theme difference is accepted in that context.

### 5.4 Tabs and SegmentedControl

Tabs navigate among peer content regions and implement roving keyboard focus, selected state and focus-visible state. SegmentedControl changes a view of the same content, such as list/grid or desktop/mobile Preview. The two are not visually or semantically interchangeable.

Tabs wrap into an approved replacement navigation before they overflow horizontally. A focus ring does not appear as a second selected indicator.

### 5.5 Menu, Popover, Dialog and FocusedSheet

Overlay components share the z-index ladder, scrim, elevation, radius, entry/exit motion and reduced-motion behavior. They provide labelled headings, Escape behavior, focus containment where modal and focus restoration.

- Menu presents compact actions.
- Popover presents bounded supporting controls near a trigger.
- Dialog interrupts for a decision or confirmation.
- FocusedSheet handles longer inspection or editing without losing list context.

Mobile replaces an unsuitable popover with a sheet through the component contract rather than feature-specific CSS.

### 5.6 StatusBadge, Notice and async feedback

StatusBadge uses text plus icon/shape where necessary and never color alone. API codes are translated to business language at the boundary. Notice variants are information, warning, error and success; each identifies the affected object or action and a recovery or next step where required.

Spinner is used for short local waits. Skeleton reserves final layout for longer initial loads. Inline progress identifies the running action. Toasts announce completion without stealing focus and never carry the only copy of a material failure.

### 5.7 Surface, Card and EmptyState

Surface provides only hierarchy and containment. Card is used for a coherent object or decision, not every section. Both use the shared radius/elevation scale.

EmptyState states the population or query checked, the result and the next valid action. It replaces the whole empty work region; it does not sit inside an otherwise scrollable empty table while an unrelated detail pane remains visible.

### 5.8 Toolbar, FilterBar and DataTable

Toolbar arranges actions with one primary action and an overflow contract. FilterBar composes fields, chips, clear/reset and result count; it replaces controls at compact widths instead of compressing them.

DataTable owns header, row density, selected, hover, focus, loading, empty, error and pagination states. It supports correct text/number/status/action alignment and full row accessible names. Horizontal scrolling is allowed only for populated data whose columns genuinely cannot reflow; the empty state never scrolls horizontally.

Master-detail layout is a separate composition:

- at sufficient width, table and detail may coexist when the table retains its minimum useful width;
- below that width, detail opens in a FocusedSheet; and
- when no row exists, the detail pane is absent and EmptyState receives the available width.

## 6. Component gallery and documentation

Add a development/evidence route that renders production components, not replicas. Each component displays applicable variants in:

- light and dark themes;
- comfortable and compact density;
- default, hover/pressed proxy, focus-visible, disabled, loading, invalid, read-only and selected states;
- short and long/translated content;
- narrow, 200%-reflow and forced-colors presentation; and
- reduced motion.

The gallery documents the component's job, allowed variants, content rules, keyboard behavior and prohibited substitutions. It uses clearly labelled sample data and does not imply bank state.

`DESIGN.md` is updated with the architecture, supported token roles, component inventory and migration rule in the first foundation change.

## 7. Governed Forms migration

### 7.1 Tranche 1: foundation and Sent forms

Sent forms is the proving screen because it exercises the missing Button, FormField, SelectField, TextField, Tabs, FilterBar, DataTable, StatusBadge, EmptyState and master-detail contracts together.

Required result:

- **Send form** is the one primary 44 px action and keeps existing authority behavior.
- Forms navigation uses Tabs with one selected indicator and complete keyboard behavior.
- Status and Due state use themed SelectField components; Subject type, Subject ID and Owner use TextField.
- Filters use a responsive FilterBar and preserve the current route query.
- An empty query gives the complete work width to EmptyState and hides distribution detail.
- A populated result uses table/detail only when both retain useful widths; otherwise detail uses FocusedSheet.
- The table and application page do not show an empty horizontal scrollbar.
- Loading, sign-in-required, empty, error, populated, selected, partial-page and lifecycle-action states are represented.

This tranche delivers the shared component gallery and enforcement described below. It does not migrate unrelated workspaces.

### 7.2 Tranche 2: immersive form builder

The builder adopts the same components and all retained audit requirements:

- focused authoring shell replaces the Forms hero, tabs and redundant page gutters while editing;
- three panes render only when Outline, Canvas and Question settings meet their minimum useful widths;
- Canvas remains primary at compact widths while Outline and Settings use FocusedSheet;
- response types use a categorized, keyboard-operable picker;
- question cards use shared fields, controls, menus and statuses;
- Preview, Review, Save draft and Send for approval use standard action hierarchy;
- one primary scroll model prevents simultaneous page and pane scrolling during ordinary editing;
- editable names wrap, controls meet target sizes and no application navigation covers content; and
- existing revision, conditional-order, duplicate, Review-fix and authority behavior remains unchanged.

The detailed measurements and builder state requirements remain preserved in the superseded builder audit document and are acceptance inputs to this tranche.

### 7.3 Tranche 3: remaining Forms surfaces

Migrate Templates, Responses, Imports and Communications using the established components. Remove Forms-local control styling only after each affected production route and evidence state has moved. No raw-control ban is enabled for a file until its replacement is complete and tested.

### 7.4 Later product migration

Other ClearSight workspaces adopt the foundation when changed or through separately scoped migration tranches. Product-wide claims are prohibited until the inventory confirms all customer-facing controls use an approved component or documented native exception.

## 8. Enforcement and drift control

### 8.1 Source enforcement

Add a UI-contract test or validator with an explicit migrated-file manifest. For migrated files it rejects:

- raw `<button>`, `<select>` and ordinary text `<input>`/`<textarea>` outside approved UI wrappers;
- direct imports from `react-aria-components` outside `components/ui`;
- hard-coded hex/rgb/hsl colors in component and migrated feature CSS;
- raw radius, shadow, control-height, motion-duration and z-index values where a token exists;
- new component variants absent from the gallery and `DESIGN.md`; and
- feature selectors that reach into UI component internals.

The validator uses an explicit allowlist for semantic native inputs and documented exceptions. It must report the file, line and approved alternative. A broad regex without exceptions is insufficient.

### 8.2 API enforcement

TypeScript component props use closed unions for variant, size, tone and state. Product code cannot supply raw color, radius or shadow props. Escape hatches require an explanatory code comment and design review.

### 8.3 Migration tracking

Maintain a component adoption inventory by workspace and component family. A surface is migrated only when its default, loading, empty, error, permission, conflict, long-content, responsive and theme states use the shared contract and have rendered evidence.

## 9. Accessibility, internationalization and theme behavior

- Every workflow remains keyboard-completable with visible, non-clipped focus.
- Focus order follows the rendered task order; overlays restore focus to their trigger.
- Errors are associated with fields and announced; multi-error flows provide an anchored summary.
- Pointer targets are at least 44 px on touch/mobile/reflow surfaces.
- Text and component contrast meet WCAG AA in light and dark themes; forced-colors retains boundaries and focus.
- Components support 200% browser zoom, long translated labels and operating-system text scaling without losing actions.
- Date, time, number and currency presentation remains locale-aware.
- Motion communicates state in 100–300 ms and is removed or reduced under `prefers-reduced-motion`.
- Icons use one ClearSight line-icon grammar, consistent sizes and text labels where the icon is not universally understood.

## 10. Security and governance boundaries

- UI components do not infer authority, ownership, approval or material state.
- Verified server identity, tenant, legal entity, version and current authority remain mandatory for material commands.
- Disabled presentation does not replace server authorization.
- A component may show saving or pending only for the exact command it issued; it reports success only after the server confirms the result.
- Stale Review, score or status identifies the version it reflects.
- Overlay, menu and analytics state never contains invitation tokens, protected addresses or restricted record existence.
- Shared components remain usable without AI or a live integration.

## 11. Failure and recovery behavior

- Async components preserve the current value while loading a refresh unless that value is no longer valid.
- A failed option load names the affected field and offers Retry or an approved fallback.
- A server conflict does not silently overwrite local work; the owning workflow provides compare/reload/reapply handling.
- If a material command commits but a later derived read fails, the interface reports the committed result separately from the refresh failure.
- Overlay failure leaves a keyboard-reachable close/recovery path.
- Theme or preference persistence failure keeps the current usable rendering and explains that the preference was not saved.

## 12. Verification and evidence

### 12.1 Component tests

For every shared interactive component, test:

- semantic role, label, description and error relationships;
- pointer and keyboard operation;
- focus-visible, containment and restoration;
- disabled, read-only, loading and invalid semantics;
- controlled and uncontrolled use where supported;
- light/dark token mapping;
- long-content and reflow behavior; and
- axe results for representative compositions.

SelectField additionally tests typeahead, arrow keys, Home/End, Enter/Space, Escape, selected announcement, portal layering and mobile sheet replacement. Button tests repeat-click prevention during loading. Tabs tests roving focus and manual/automatic activation behavior selected by the contract.

### 12.2 Forms workflow tests

Preserve existing workflow and authority tests. Add assertions that:

- filter state survives navigation and reload;
- empty Sent forms does not render distribution detail or horizontal table overflow;
- a populated row opens exact detail inline or in the responsive sheet;
- primary actions expose running, success and recovery states;
- the builder response picker changes, cancels and confirms destructive type changes correctly; and
- stale versions and unauthorized commands never receive optimistic success presentation.

### 12.3 Automated visual contracts

Rendered automation fails when:

- a migrated primary action is transparent, borderless or below its required target;
- a field has no visible boundary against its surface;
- a popup uses a light surface in the dark theme or escapes the supported viewport;
- fixed/sticky UI overlaps active content, validation or the last result;
- an empty data region scrolls horizontally;
- supported content clips rather than wraps; or
- a component state is absent from the gallery manifest.

### 12.4 Rendered review

Preserve the supplied and existing screenshots as before-state evidence. Capture production components inside the full host at:

- 1440×900 and 1280×720 desktop;
- 1024×768 compact desktop/tablet;
- 390×844 mobile;
- real browser 200% zoom;
- light and dark themes;
- comfortable and compact density;
- open Select, Menu, Popover and Sheet states;
- keyboard focus and forced-colors where supported; and
- loading, empty, error, read-only, conflict and long-content states.

Inspect every materially changed route, repair its highest-impact failure and recapture the failed state. Isolated component evidence supplements but never replaces full-host evidence.

### 12.5 Performance and bundle evidence

Record production bundle size before and after each tranche. Import React Aria through the UI boundary so unused components can be tree-shaken. A material initial-load increase requires route-level lazy loading or an explicit review. Interaction feedback appears within 100 ms; menu and sheet motion never delays input.

## 13. Delivery sequence and plan boundaries

This specification governs a sequence, not one oversized pull request:

1. **Foundation + gallery + Sent forms:** token layers, UI boundary, core components, enforcement and the first full production migration.
2. **Immersive builder:** builder shell, question editing and categorized response picker using the approved foundation.
3. **Remaining Forms:** Templates, Responses, Imports and Communications.
4. **Progressive product adoption:** separately reviewed workspace migrations.

Each tranche has its own test-first implementation plan, branch review and exact-head evidence. A later tranche does not delay correction of a defect discovered in an earlier shared component.

## 14. Exit criteria

The UI foundation and Forms migration are complete only when:

- the three token layers and component ownership are documented in `DESIGN.md`;
- the shared component gallery covers every supported variant and state;
- all Forms customer-facing controls use an approved component or documented native exception;
- no migrated screen reproduces the transparent 20–21 px controls, light-on-dark popup, empty horizontal scrollbar or builder obstruction from the before-state evidence;
- relevant TypeScript, copy-quality, accessibility, workflow, build and rendered-evidence gates pass on the exact final head;
- full-host light/dark, responsive and 200%-zoom evidence has been inspected and rechecked; and
- the deployed acceptance environment reports the same commit before hosted screenshots or timing results are treated as release evidence.

Repository completion is not representative-user acceptance. Timed first-use, repeat-use, keyboard and assistive-technology checks remain explicit release evidence, and any remaining non-migrated workspace is named rather than implied consistent.
