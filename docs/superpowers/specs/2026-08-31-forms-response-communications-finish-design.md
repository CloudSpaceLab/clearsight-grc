# Forms response and communications finish

## Decision

Finish the Responses and Communications workspaces by extending the shared design system, not by adding page-specific button colors or preserving the current raw list-button and inline-editor structures.

The work has three bounded outcomes:

1. shared primary actions retain a readable foreground, weight and focus treatment in both themes;
2. Responses becomes a clear selection-and-review workspace with explicit selected states and a stable detail hierarchy;
3. communication profile and template creation use centered, bounded dialogs with shared fields and explicit cancel/save outcomes.

## Evidence and root causes

- The dark Forms header renders the cyan primary action with an inherited light foreground. A legacy unlayered `button { color: inherit; }` rule outranks the layered design-system component rules. The same rule also overrides the shared button typography contract.
- Responses currently exposes two raw button lists and a third detail column. Browser defaults and a thin selected outline carry most of the hierarchy, so distribution, revision and selected record are difficult to scan.
- Communications uses shared action components at the workspace header, but opening either action replaces the whole tab with an unbounded inline form. The profile form still uses raw date controls, and the template editor lacks a consistent cancel path and dialog footer.

## Considered approaches

### Recommended: shared tokens, selectable records and focused dialogs

Remove the legacy reset collision, define reusable selectable-record styling in the component layer and place both communication editors in the existing `FocusedDialog`. This fixes the source of the contrast failure and provides a reusable selection pattern without introducing another navigation model.

### Rejected: patch Forms selectors only

Page-specific `color` declarations and new raw-button CSS would make the screenshots look better but would leave the cascade defect and duplicate interaction states.

### Rejected: rebuild Responses as a new route set

Separate distribution and revision routes could scale later, but they add navigation and state-restoration work that is unnecessary for the current bounded history review.

## Component design

### Primary actions

- The reset layer may set inherited font defaults, but unlayered legacy CSS must not set a foreground on all buttons.
- `Button` remains the only primary CTA component in the migrated Forms surfaces.
- Dark primary background, foreground, hover, pressed, disabled and focus states remain semantic-token driven and must meet WCAG contrast requirements.
- Communications retains one dominant primary action: create the next message-template revision. Profile revision is secondary.

### Selectable records

Add a shared `SelectableRecord` component for one record in a bounded master list. It exposes title, metadata, optional supporting text, selected state and an accessible selected announcement. Visual states use component tokens for surface, border, hover, selected rail/ring and text.

Responses uses a two-stage review hierarchy:

- distribution records remain in the left master column;
- the selected distribution owns a version-history section and a response-detail section in the right workspace;
- the current version is visibly labelled and selected; historical versions remain available without looking like editable controls;
- detail facts, sign-off and critical results stay read-only; the disabled edit action is replaced with explanatory copy because submitted records cannot be edited.

At widths below 1050px, the master list and review area stack. Selection state and headings remain visible, and no horizontal scrolling is required.

### Communications editors

- The workspace remains visible beneath a `FocusedDialog` so users retain context.
- Profile revisions use the default dialog width; template revisions use the wide dialog.
- Each dialog has a task heading, grouped shared fields, an error notice, and a footer ordered as Cancel then the dominant Save action.
- Date-time fields use the shared field shell so labels, descriptions, errors and dark surfaces match other Forms controls.
- Template subject and protected-variable actions remain explicit. Rich-text toolbar migration is limited to its visible buttons and does not alter the stored communication document contract.
- Closing or cancelling discards only the unsaved local draft and returns focus to the invoking CTA.

## Data and workflow boundaries

No API, stored record, lifecycle state or authorization route changes. Templates and profiles continue to create immutable draft revisions through the existing commands. Responses remain server-derived, read-only revision history.

## Copy

Headings name the current task or record. Buttons describe their immediate result: `Create profile revision`, `Create template revision`, `Cancel`, `Save profile revision`, and `Save template revision`. Supporting copy explains that saving creates a draft and what must happen next.

## Verification

- Test-first coverage for the CSS cascade guard, selectable-record semantics, response hierarchy and both dialog open/cancel/save paths.
- Component contract, type-check, production build and complete frontend regression suite.
- Dark and light rendered evidence at 1440×900, plus responsive evidence at the existing supported viewport.
- Computed dark primary-button foreground/background contrast check and manual pixel inspection of Responses and both Communications dialogs.
- Update `DESIGN.md`, the UI adoption record and Forms rendered-state capabilities in the same change.

## Scope left after this tranche

Imports, remaining legacy list selectors outside Responses/Communications and the rich-text editor internals remain separate migration work. This tranche does not change message content rules, delivery, response scoring or revision authority.
