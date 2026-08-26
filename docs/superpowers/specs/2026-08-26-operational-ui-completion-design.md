# Operational UI Completion Design

## Outcome

ClearSight must let a bank user reach and complete the material Work and Program workflows from normal navigation without API or database access. The static stakeholder deployment must exercise the same visible workflows truthfully, including authority, maker-checker, assignment, evidence, monitoring, outcome verification, closure and history.

## Interaction design

- Program and Work list cards provide a labelled route into the complete operational record.
- Native `date`, `datetime-local`, `time`, number and file controls replace free-text approximations wherever the value has a defined browser semantic.
- Buttons remain labelled verbs; SVG glyphs are visually compact but every interactive target is at least 44px.
- Focused dialogs and drawers use a tokenized 48–60% scrim and restrained backdrop blur. Blur supports focus and never obscures errors, authority, status or required actions.
- Typography uses the existing ClearSight family with a 16px body baseline, 1.5 line height, stronger heading contrast and readable measures.
- Document import and review content uses a paper-white surface in light mode. The dark theme maps the same semantic document tokens to a high-contrast dark paper surface.

## Technical design

- The live build remains React 19. The static build may use `preact/compat` only if all tests, accessibility checks and rendered-state review pass.
- Static fixture and command runtime URLs are emitted by Vite with content hashes and the configured base path. The loader is single-flight, verifies the runtime global, and resets after failure.
- The JavaScript budget counts every delivered `.js` file recursively under `dist`; executable code cannot be hidden in `public` to evade the gate.
- Stateful static workflows preserve per-record and per-check state, optimistic versions, maker-checker identity, assignment and immutable history.
- Design changes extend existing semantic tokens and `DESIGN.md`; components do not introduce raw per-screen colors.

## Acceptance

- Matter facts/details, assignments, actions, decisions, responses, outcome checks, results and closure work from `#work`.
- Program scope, requirements, safeguards, evidence checks/results and monitoring work from `#programs`.
- Created Programs and Matters remain operable through exact routes.
- Imports are readable in light and dark modes; focused overlays remain accessible and dismissible.
- No enabled control deterministically fails because the UI offered an invalid authority or lifecycle target.
- Full Go default/PostgreSQL tests and vet, full web tests/typecheck, live/static builds, copy quality, bundle gate and rendered UI review pass before merge.
