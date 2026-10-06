# Remaining Issue Workspace Implementation Plan

## Delivered checkpoint

- `8b8dc74e9`: issue summaries and records show applicable area, owner, deadline and impact; linked loss records use an exact bounded `matter_id` read.
- `fb19ae5a8`: employee form requests use the established distribution service, locked to the current issue; vendor requests remain relationship-scoped; issue tabs are Overview, Work, Evidence and requests, and Decisions.

The existing Activity Timeline remains the only issue comment, mention, notification and history mechanism. The existing action panel remains the only reassignment and update-request mechanism.

## Implementation status

### 1. Preserve the origin of an issue-created form — complete

Add an optional, immutable `MATTER` origin to normal form-template revisions.

- Add a migration with an all-or-none origin pair and an entity-scoped foreign key to `matters`.
- Add `FormOrigin` to monitoring models, API requests and browser types.
- Validate the referenced issue against the verified tenant and legal entity before creation. A revision must retain the exact original value; changed or newly added origins are rejected.
- Add memory and PostgreSQL tests for valid, cross-entity, missing, and altered origins.

### 2. Add one issue-linked form authoring route — complete

Add `Create linked form` beside the existing employee and vendor request paths.

- Open the existing `FormBuilder` in a focused sheet with the issue reference fixed as the form origin.
- Save through the existing form-library create/revision endpoints; do not create a separate issue-form model.
- Keep the draft in the ordinary Forms library and display its originating issue in its record details.
- After an approved form exists, use the delivered employee request or existing vendor request path to collect responses.

### 3. Finish evidence and response context — complete

Compose existing records in the Evidence and requests tab rather than duplicating them.

- Show form-response summaries, linked vendor work, and integration/source receipts as separate, bounded groups.
- State whether attention is due to a missing response, expired field/evidence, source degradation, or an internal assignment gap.
- Provide field-only renewal/reminder actions through the existing response-expiry and vendor-reminder workflows.
- Verify unavailable linked reads leave issue actions and activity available.

### 4. Add the board brief through reporting governance — complete

Use the existing report definition and run lifecycle.

- Add a `MATTER_BOARD_BRIEF` dataset with a bounded, point-in-time projection of issue position, actions, decisions, loss/recovery, forms, vendor work and connected sources.
- Render a deterministic PDF without comments or protected recipient addresses.
- Surface `Generate board brief` only when an authorised effective definition exists; retain generated files and downloads in Reports.
- Test authorization, source boundary, PDF content, and permission-checked download.

### 5. Complete responsive and usability proof — verification in progress

- [x] Keep the desktop activity rail on the right; replace it with a single Activity tab at narrow widths, without a duplicate composer.
- [x] Register rendered issue header, Work, Evidence and requests, and Activity evidence at desktop and narrow viewports.
- [ ] Require exact-head CI, Compose runtime and managed UI/UX review to pass before release.

## Completion criteria

An issue can be read, assigned, discussed, updated, linked to a loss, supplied with an employee or vendor form request, created as the immutable origin of a normal form draft, and exported as a governed board brief. Every route is tenant- and legal-entity-scoped, uses existing authority and notification services, and degrades without concealing the primary issue work.
