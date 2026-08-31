# Forms UX decision brief

Status: accepted implementation direction for #103.

## Outcome

Forms is an operational workspace, not a schema editor. The product hierarchy is:

1. find the form or operational subset;
2. create a form through one coherent entry point;
3. work on the form itself;
4. reveal configuration only when it is relevant;
5. reveal governance when review/publish decisions are being made.

Existing governed domain truth remains authoritative. This work does not create a second form, revision, approval, distribution, response, recovery, import, or AI proposal model.

## Information architecture

The Forms destinations remain:

- Templates
- Sent forms
- Responses
- Imports
- Communications

Templates is full-width by default. Selecting a record opens contextual detail without permanently consuming list width. Creation starts from one `Create form` action. The builder converges on `Outline → Canvas → Inspector`, with Review, Preview and Publish as contextual actions rather than permanent panels.

## Template record-surface migration

The primary user is a compliance form owner finding a reusable form, checking its current revision and authority-backed state, selecting approval-ready drafts or opening one form’s details. The first useful outcome is a readable bounded result with one clear create action; the repeated-use task is scanning, selecting and acting without relearning bespoke controls.

The record surface uses the shared data table, status, checkbox, button, notice, empty-state and focused-sheet contracts. This preserves the current Forms APIs, query state, exact revision identity and authority operations. Loading, populated, empty search, unavailable source, saved-filter, bulk selection and lifecycle detail states remain required. At narrow widths the table becomes labelled stacked records and the sheet becomes the shared full-height mobile surface. Copy must distinguish latest stored and currently reusable revisions and must not imply approval or availability when the authority read is missing. Icons remain labelled control aids; the shared sheet owns motion and reduced-motion behavior.

Acceptance evidence is the component gallery plus the deterministic full-host Forms matrix at desktop, mobile, 320px reflow, 200% zoom, light, dark, forced-colors and reduced-motion states. The first migrated render exposed a close action beneath sticky detail content; the shared overlay layer token was corrected. The creation launcher uses shared action-card, button, card, empty-state and centered-dialog contracts. Template filters use the shared search, filter, field, action, popover, sheet and scope contracts. All Forms selection menus use the themed select contract; Responses and Communications now use shared feedback and primary-action controls. The remaining Builder inputs, Imports, list selectors and rich-text controls are explicitly incomplete.

## Truth and safety invariants

The frontend must preserve:

- exact template and revision identifiers;
- latest stored revision separately from the currently reusable revision;
- maker/checker lifecycle transitions and backend authorization;
- immutable response and amendment history;
- access policy, OTP, recovery and file-reselection behavior;
- bounded import/conversion semantics;
- governed AI proposal/selective-acceptance boundaries;
- server-side tenant/legal-entity/actor visibility before filtering, count, limit or cursor operations.

Saved views remain presentation/query preferences only. They never become assignment, authorization, approval or workflow truth.

## Deleted or replaced composition

The target composition explicitly removes these patterns from the neutral workflow:

- permanent empty detail rails;
- permanent Quality Gate rail;
- Cards/Table duplication for the operating list;
- a second Recently updated content section;
- workspace-brand styling controls mixed with Forms work;
- separate competing Create and AI header actions;
- starter templates as a detached secondary action;
- default/no-op question settings occupying canvas space.

## Frontend module boundaries

Current and target responsibilities are bounded by feature:

```text
components/forms/
  creation/
    NewFormLauncher

  dashboard/
    TemplateLibraryTable
    TemplateDetailDrawer

  filters/
    filterModel
    filterRegistry
    FilterBar
    FilterPicker
    AdvancedFilterBuilder

  builder/
    BuilderToolbar
    FormOutline
    FormCanvas
    QuestionBlock
    Inspector

  review/
    ReviewDrawer
    PublishDialog
```

`FormsWorkspace` coordinates modes and server commands. It must not absorb creation, dashboard, filtering, builder and review rendering into one component.

Styles follow the same ownership split (`forms-dashboard-*`, `forms-creation`, later builder/review layers) instead of growing one monolithic stylesheet.

## State ownership

### URL/query state

The URL owns shareable Templates query state: search, supported filters, sort and selected target where existing routing supports it.

### Local feature state

Local React state owns transient UI only, including:

- creation launcher visibility;
- contextual drawer visibility implied by selected target;
- editor/AI proposal mode;
- temporary filter-picker or review-sheet state;
- unsaved local control state.

### Server state

The server remains authoritative for templates, revisions, approval state, saved durable views, distributions, responses, imports and proposals. Client filtering must not require fetching the entire form library.

## Creation decision

One `Create form` action opens four methods:

- Blank form → existing governed manual draft path;
- From template → existing reviewed starter instantiation path;
- Draft with AI → existing governed proposal/review path;
- Import → existing governed import/conversion workspace.

Starter previews are derived from the actual starter form contract. Catalog records are installed by database migration and read through the repository; the API does not construct a second embedded starter schema. Static HTTP fixtures are permitted only in builds marked for automated UI evidence and cannot be enabled by the deployable demo flag alone.

## Filter contract

Advanced filters will use one bounded typed expression model:

```ts
type FilterExpression =
  | {
      kind: "condition";
      field: FilterField;
      operator: FilterOperator;
      value: unknown;
    }
  | {
      kind: "group";
      operator: "and" | "or";
      children: FilterExpression[];
    };
```

A registry defines each supported field's operators, value editor, display formatter, URL/API serializer and server capability. Unsupported or unindexed dimensions are not exposed. Expression depth and node count are hard-bounded; there is no arbitrary query language.

## Realtime decision

Do not add a Forms-specific event bus. Start with bounded, abortable revalidation using existing query infrastructure. Patch individual rows only where an existing application event path supplies exact changed truth. Add streaming only after measured need and only by reusing canonical infrastructure.

## Respondent preview decision

Author preview must reuse canonical respondent rendering/components wherever possible. A visual miniature in the creation gallery is descriptive only and is derived from the exact starter contract; it is not a behavioral preview engine.

## Validation gates

Every tranche must retain:

- TypeScript typecheck;
- rendered-state and accessibility tests;
- production build;
- deterministic Forms evidence for desktop/mobile/320px reflow/200% zoom/light/dark;
- existing backend CI because frontend lifecycle actions remain coupled to governed contracts.

The deterministic evidence contract must evolve when a legacy UI surface is intentionally removed; tests must validate the new behavior rather than force obsolete composition back into the product.

## Issue #103 closure behavior

The implemented dashboard and builder retain these interaction contracts:

- clearing active library filters preserves the selected template and its URL target, then reloads that exact record in the unfiltered result;
- applying a saved view and changing newest/oldest updated order preserve the selected template and serialize the query without client-side full-list sorting;
- contextual template detail is a modal sheet with an explicit accessible name, initial close-button focus, trapped keyboard focus, locked background scrolling, Escape/backdrop dismissal and focus restoration to the invoking row;
- at 760px and below, the canonical template table becomes a labelled stacked-record presentation; the document and table wrapper do not require horizontal scrolling;
- pending approval is presented as `Awaiting approval`; internal lifecycle values remain available in API and audit surfaces;
- question drag handles perform pointer reordering within their section, while the question action menu retains Move up and Move down for keyboard users;
- a reorder that would invalidate conditional logic is rejected visibly without deleting the condition; Duplicate creates a new question key;
- Review Fix closes the review sheet and focuses the exact affected form, section or question control;
- edit and lifecycle controls render only from an exact current authority operation; an unavailable authority read exposes no material action;
- at 560px and below, Preview, Review, Save draft and Send for approval remain visible in a wrapped toolbar, with 44px targets; authoring does not hide a material action to fit the viewport.

The deterministic Forms evidence matrix includes a populated 390px library, the mobile builder at 390px, builder reflow at 320px, a real desktop mouse drag and a 120-question/10-section fixture. Scenarios check overflow, visible authoring actions, target dimensions, changed question order, sticky-chrome separation and large-form render/edit latency before capture.

Automated evidence does not replace representative-bank-user timing, actual browser zoom or hosted production acceptance. Issue closure still requires those external results, the exact deployed commit, normal-network query timing and production-shaped PostgreSQL/load evidence to be recorded rather than inferred from local fixtures.
