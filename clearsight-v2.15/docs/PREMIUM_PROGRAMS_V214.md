# V2.14 — frameworks, programs and form design

## Delivered scope

`premium-programs.css` completes the presentation of eight existing routes. `premium-programs.js` adds one route marker, `data-pp214`, to the existing main content element after its normal render. It does not replace a view, change its HTML or add a command.

| Route | Covered surfaces | Refinement |
| --- | --- | --- |
| `frameworks` | Framework choices, selected seed, source edition and adoption explanation | Readable title/description hierarchy, wrapping metadata, a clear selected card and a solid preview surface. |
| `adopt` | Requirements, monitoring policies, participating teams, publication review and included forms | Consistent step controls; clearer requirement, applicability and tailoring rows; readable source disclosures; aligned scope choices and footer actions. Applicability badges remain visible on phones. |
| `programs` | Program catalogue, coverage, accepted outcomes, accountable owners and search results | Quieter existing summary figures, balanced table rows and actions that occupy their own mobile card row. |
| `program` | Overview, requirements, named assessments, Documents, monitoring and activity | Exact target selection stays prominent. Named responsibility, due date, current state and next action remain distinct. Assignment filters and action groups use the same queue rhythm as other premium workspaces. |
| `forms` | Form library, starter templates, integration instruments, named assignments and seed questions | Consistent catalogue cards, readable version/source metadata and wrapping actions. Templates retain their question/check descriptions on small screens. |
| `form-builder` | Live `cx-*` editor: question text, guidance, choices, matrix/table definitions, calculations, connected checks and question menus | Visible editing boundaries, larger contextual tools, always-visible column removal controls, readable source settings and bounded horizontal table scrolling. Existing field/source/rule dialogs receive matching spacing. |
| `check-runs` | Empty history, execution ledger and exact-run detail dialog | Clear separation of collection health and rule outcomes; readable provenance/rules; useful empty-state spacing around the existing technical-form link. |
| `setup` | Guided risk and RCSA setup, scope/population selection, baseline fields, final review and save/continue actions | Consistent step and section hierarchy, comfortable capture fields, readable review facts and a visible saved-draft indicator. Existing routed handoffs remain unchanged. |

## Concrete gaps addressed

The live form builder uses the `cx-*` editor, while the V2.13 response refinements primarily styled `fa-*` response fields. Its older controls included 23px icon targets, 24px guidance fields, 10px contextual labels and table-column removal buttons hidden until hover. The new stylesheet makes editing tools visible without hover and gives phone/coarse-pointer controls 44px targets and editable inputs at least 16px text. Question IDs, field attributes, native controls and their order in the source are untouched.

Framework and adoption previews now use solid semantic surfaces. Framework metadata wraps, and phone requirement rows have space for the requirement title, applicability badge and review action. Adoption's three step cards become readable full-width rows on phones instead of compressing their labels into narrow columns.

Program target controls and named-assignment actions wrap without squeezing the organizational target. Program and form tables continue to use the shared mobile card layout; trailing actions receive the full card width. Editable form matrices remain tables inside their existing horizontal scroll containers.

## Design and interaction boundaries

The stylesheet consumes the shared spacing, corner, surface, text and status tokens. Selected step counters use the stable `--cs-accent-solid` foreground/background pairing for both appearances. No new palette, translucent content panels, counts, charts, placeholder controls or alternate stores are introduced.

Compact density changes catalogue/queue spacing only. Form entry retains comfortable geometry. The builder's existing insertion toolbar stays available on a wide desktop and returns to normal flow on narrower or short viewports. Setup footers also return to normal flow on phones and short viewports. Column tools have reserved header space rather than overlapping their editable labels.

The marker adapter calls and returns the previous render function, sets or clears only `data-pp214`, and performs no startup render. Repeated loading is inert. It composes with the existing V2.13 markers and the separate V2.14 risk/privacy/utility markers. `CLEAR_SIGHT_PREMIUM_PROGRAMS` exposes `version`, the eight-route list and `refresh()` for integration checks.

All canonical commands, field IDs, source schemas, publication steps, assignment/reviewer rules, immutable histories, scopes and permission checks remain authoritative.

## Verification

`tests/premium_programs_v214_checks.js` passes **3/3** focused actual-source checks. Its result file is `tests/premium-programs-v214-results.json`.

1. The marker covers exactly the eight assigned routes, clears on other routes, and preserves rendered content, sibling markers, application state and saved storage.
2. Actual catalogue, adoption, program, form-builder, history and authorized risk-setup rendering retains the existing controls and registered actions.
3. Navigation preserves authored form IDs/values and adoption rationale; reloading the module preserves render identity without rerendering or changing state.

The shared-form fixture gives the acting identity only the explicit form-author permission needed for that check. Risk setup uses an existing scoped contributor. JavaScript syntax checks passed. A bounded peer source review found no blocking marker-composition, editor-overflow or phone/coarse-control issue.

Browser rendering, pointer interaction and assistive-technology behavior were not exercised. Responsive rules were checked against the actual editor markup and shared table-card structure; the full inherited regression run remains a release-level integration check.
