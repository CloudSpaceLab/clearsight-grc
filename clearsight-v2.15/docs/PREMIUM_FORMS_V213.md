# Form and document presentation — V2.13

## Scope

The new presentation layer refines the existing assessment, KRI, loss, vendor onboarding, review/approval and Document workflows. It uses their existing sections, question controls, source facts and action groups. It adds no operational state, new approval path, alternate assessment copy or new permission rule.

Files:

- `assets/premium-forms.css` — scoped internal form and reading styles.
- `assets/premium-forms.js` — route-family styling scope, a few accessible group/scroll-region labels, and a constant recipient stylesheet.

Load the CSS after the premium foundation and existing domain styles. Load the script after the existing form, role, Documents, external-assessment and render wrappers. The module does not render at initialization; the existing application startup remains responsible for that.

## Design decisions

### Questions and named response work

Question titles, guidance, input surfaces and provenance now have separate visual levels. Titles use a readable 16 px scale and a stronger weight. Guidance uses 13 px text with comfortable line spacing, while the actual response and submitted-source reading areas use 14 px text. Stable identifiers remain present at a quieter scale.

The response canvas has more breathing room between questions. The existing question navigator and readiness checklist remain beside the form on wider screens and return to normal flow on smaller or short viewports. Named respondent, actual submitter, reviewer and due date stay visible in the existing accountability strip; its layout is more restrained than the question content.

Choice controls use a subtle selected surface. Source observations retain their existing values and statuses. Mixed grids use legible cells without losing row/column relationships. The desktop compact preference reduces padding through the shared density contract; question text remains readable.

### KRI and loss entry

The branch KRI navigator, progress area, section heading and question cards now follow the same spacing scale. Exceptions retain their red semantic marker and conditional fields. Read-only answers and explanations have a larger reading scale. No threshold, validation, population, exception ownership or submission logic changes.

Loss steps use the same neutral step treatment as vendor intake. The active step is clear without using a large coloured background. Field pairs, section summaries and final checks have consistent spacing. Financial amounts, recovery states, currency, recognition/reporting dates and review controls retain their original elements and source bindings.

The existing KRI submission, loss and vendor action bars become sticky only on screens at least 1,100 px wide and 820 px high. They keep their normal document-flow space. They return to normal flow while an input, select or textarea is focused, and remain non-sticky on smaller/shorter screens. This avoids a floating bar obscuring a field or the mobile keyboard area. No duplicate submit buttons are introduced.

### Vendor onboarding

The existing four-step form now has a clear section lead, a consistent two-column capture grid and calmer requirement/plan cards. Optional requirements, selected plans, named responsibilities and recorded facts keep their current content and controls. The active step, due-diligence selections and blockers remain distinct from completed or approved states.

The dossier reading layout gives the main source more width and keeps responsibility/readiness details in a narrower supporting column. Source history, previous submissions and next-action links retain their current destinations and controls. Existing hard-coded light palette surfaces in these components are replaced by semantic tokens so they follow the application theme.

### Review and approval

The review layout separates the source reading area from the decision area. Submitted answers use a generous reading measure, while names, timestamps, exact versions and capture references remain compact but readable. Review forms retain their actual actor, decision choices, comment fields and canonical submit command.

The source/decision arrangement collapses to normal flow on narrower screens. Revision summaries and decision timelines gain consistent spacing; no comments are truncated, no historical source is overwritten and no accepted conclusion is restyled as pending or vice versa.

### Documents

The Document workspace gives the preview a wider primary column and a quieter provenance column. The exact-version selector remains prominent above the source. Original text uses one scrollable preview region instead of a second nested vertical text scroll. The preview region can receive keyboard focus, allowing keyboard users to scroll retained source text.

The canonical expand/restore command is preserved, including the `dm212-wide` and `dm212-expanded` classes. Version selection, comparisons, checksums, owner, classification and source links keep their exact record identities. The presentation adapter does not substitute current bytes for a selected historical version.

Source worksheets and the KRI entry table retain their actual row/column layout on mobile. Their containing region scrolls rather than inheriting the register-table card treatment. Ordinary register tables continue to use the foundation’s responsive layout.

## Shared tokens and responsiveness

Internal styles use the existing semantic palette: `--surface`, `--soft`, `--ink`, `--muted`, `--line`, `--blue`, `--red`, `--amber` and `--teal`. Spacing and radii use the shared `--cs-space-*` and `--cs-radius-*` tokens with fallbacks. Sticky offsets use `--cs-sticky-top`; grid controls respect `--cs-control-height` on desktop.

The root foundation remains responsible for the shell, generic controls/buttons, tables, theme and density. Form-specific compact rules apply only at desktop width with a fine pointer, matching the foundation contract. Mobile and coarse-pointer grid controls retain a minimum 44 px height and 16 px input text, including the 651–767 px range. Card layout, sticky treatment and reading widths have explicit narrow-screen rules.

The internal layer does not add animation. The recipient stylesheet provides reduced-motion, forced-colour and print rules. Neither layout uses fixed-height question content or clips long recipient responses. Document previews intentionally remain scrollable source viewers, with their content retained.

## Isolated recipient form

The wrapper around `ex212RecipientHTML(payload)` inserts one constant `<style id="pf213-recipient-style">` immediately before the existing `</head>`. It does not use broad replacement patterns, rebuild the body or modify the recipient runtime.

The recipient styling uses the same cobalt palette as the foundation but includes its own local variables. It requires no bank stylesheet, web font, image, network service or app state. The reading canvas, title/due card, instructions, questions, file choices, saved-progress controls and declaration area receive a consistent hierarchy. All content remains available at narrow widths. The recipient form stays in normal flow; long questions and file lists are not constrained to a fixed-height panel.

The following remain byte-for-byte identical to the original recipient output:

- The minimized request JSON and manifest hash.
- The content security policy, including `connect-src 'none'`.
- Both existing script blocks and the exact recipient validator/runtime.
- All original body markup, native field identifiers and data attributes.
- Instructions, recipient identity, expiry, references and supplied-file content.

Draft save/resume, file validation, source checks, request expiry/revocation, response-file creation, coordinator receipt and mandatory independent review continue through the existing functions. The premium stylesheet adds no authentication or delivery claim.

## Adapter behavior

After each existing render, the script sets `#main-content.dataset.pf213` to one of `response`, `review`, `kri`, `loss`, `vendor`, `document`, `external` or `entry`. Unrelated routes remove the attribute, preventing form rules from leaking into other pages. The script does not change route parameters, perform a render of its own or write to `S`/local storage.

The adapter adds accessible names to the existing action groups and question navigation only when no name is already present. It marks the existing Document preview and KRI entry table wrapper as focusable reading regions. Existing roles, `aria-label`, `aria-labelledby`, IDs, names and command attributes are preserved.

`window.CLEAR_SIGHT_PREMIUM_FORMS` exposes only `version`, `family(route)` and `refresh()`. It contains no record command or persistence API.

## Verification and limits

Checks were executed against the integrated V2.13 JavaScript with the existing Node DOM shell:

| Check | Result |
| --- | --- |
| Focused presentation checks | 6/6 passed |
| Actual route variants covered by those checks | 16 |
| Existing external-assessment exchange checks, including exported runtime execution | 32/32 passed |
| Existing assurance interaction checks | 17/17 passed |
| JavaScript syntax | Passed |

The focused checks cover correct route-family assignment and reset, no change to rendered field markup or domain state, intact saved vendor/loss step controls, retention of existing accessible labels, focusable Document preview and the limited presentation API. The recipient comparison removes only the one new stylesheet and then compares the **entire** remaining HTML with the original function output; this verifies that the manifest, CSP, scripts and body are unchanged.

The external suite also continues to exercise independent runtime draft/resume, document attachment resume, source and permission races, declaration/capture attribution, review lineage, tampered responses and an exact 1 MiB file roundtrip. Results from those reruns were captured without modifying the prior version’s test result artifacts.

These checks verify source behavior and integration. They are not screenshot, browser layout, screen-reader or device acceptance tests. No browser session was attempted under the existing environment restriction. Final package-wide checks remain part of the root integration pass.
