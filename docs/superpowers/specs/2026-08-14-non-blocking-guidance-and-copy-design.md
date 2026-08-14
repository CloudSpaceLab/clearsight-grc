# Non-blocking guidance and product copy design

## Outcome

First-run guidance remains available and resumable without taking control of the workspace. Visible guidance and demo-account copy use familiar banking and GRC language instead of tour-script phrases.

## Interaction

- Render the active guide as a compact inline `aside` after the workspace context bar.
- Do not use a backdrop, modal semantics, automatic focus movement or a focus trap.
- Show only the current step, its supporting text, the step count and the relevant actions.
- Keep Back, Dismiss and the current step action available with normal keyboard behavior.
- Keep the Help launcher for restarting a dismissed or completed guide.
- On small screens, stack the copy and actions in document flow; never cover the mobile navigation or workspace controls.

## Copy

- Use direct headings such as “Review priority work”, “Review a priority item” and “Check Program status”.
- Name the actual destination in buttons: “Open Today”, “Review first item”, “Open Programs” and “Open evidence requests”.
- Replace “introduction” with “guide” in visible controls and accessibility labels.
- Replace the demo-login headline with “Choose a demo account” and explain account switching directly.
- Preserve specialist terms only where they affect authority or governance meaning.

## State and accessibility

The onboarding API and saved progress model do not change. The guide is a labelled complementary region, not a dialog. It does not change focus when it appears, so the user can continue using navigation and page actions. Dismissal and completion retain their existing persisted behavior.

## Verification

- Component tests prove the guide is not modal, does not move focus, persists progress and can be restarted.
- Go tests reject the reported phrases from server-provided guides.
- Demo authentication tests use the revised sign-in heading.
- Browser checks confirm workspace controls remain usable while the guide is visible at desktop and mobile widths.
