# Form preview entry

## Decision brief

Hosted verification of `726544ff` completed spreadsheet import, draft creation, editing, saving and reload. The resulting draft retained its title, short-answer question and required flag. Opening Preview then showed an empty drawer until the user chose Preview Classic or Preview Wizard; a second set of layout controls appeared afterward. The before-state screenshot is preserved in the release task, using the synthetic draft `01a084aa-6131-7a0f-af34-9c50ffb80313`.

Open the questions immediately in the form's configured presentation mode. Keep the existing respondent layout control and respect fixed-layout settings. The spreadsheet proposal retains its explicit all-questions preview. This change introduces no design tokens, layout variants or governance changes.

## Verification

- All four new preview regressions failed against the previous implementation. They cover automatic, all-questions and section layouts, a single switch, and fixed-layout forms.
- All 29 affected preview, proposal, builder and copy tests passed. TypeScript and the evidence build passed. The existing builder test also verifies immediate configured presentation and saved presentation settings.
- [Eight rendered states](../evidence/2026-09-09-form-preview/manifest.json) cover immediate section preview and switching to all questions at 1440 and 390 pixels in light and dark themes. No page errors or horizontal overflow occurred; close actions remained within the viewport. Desktop light and mobile light/dark screenshots were inspected.
- Independent read-only review found no actionable defects. Full CI and hosted verification remain release gates.

Reproduce the render check with `web/scripts/capture-form-preview.mjs` against the evidence build on port 4187. The fixture uses sample data only.
