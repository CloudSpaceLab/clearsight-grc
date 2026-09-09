# SelectField native-event verification

Source: `3f21d774` plus the uncommitted SelectField.tsx, SelectField.test.tsx and DESIGN.md correction. Local evidence preview: http://127.0.0.1:4187. Pinned Playwright 1.55.0; no hosted verification claim.

Confirmed failures before correction:
- Three exact vendor Documents light-1280 runs failed to open the PDF option. A native unchanged document scroll reached the opening guard, but its cleanup microtask ran before the remaining scroll listeners. Observed sequence: scroll 1965.1ms; cleanup microtask 1965.5ms; document capture 1965.9ms; menu subsequently closed.
- Immediate outside-search focus during opening-position restoration remained behind an open popup. Reproduced in the browser and a focused unit test.
- Trigger re-press remained open after excluding the trigger from outside-dismissal; the installed React Aria mouse press-start opens rather than toggles. A focused regression failed before the trigger correction.

Final correction preserves the event reference until its dispatch ends, clears it on close/cleanup, permits genuine outside interaction through the existing Popover callback, and handles repeat mouse/virtual trigger presses. The existing keyboard and touch paths remain delegated to React Aria.

Final checks:
- SelectField Vitest suite: 14/14 passed. All three new regressions were observed failing before their corresponding changes.
- TypeScript project build: passed.
- Vite evidence build: passed.
- Native matrix: 24/24 passed, exit 0. Exact vendor Documents 1280 light/dark scenarios, each run twice, followed by Escape, Tab, immediate outside-search click, trigger re-press and later real document scroll. Existing exact scenario option selection and layout assertions were unchanged.
- Scoped git diff check: passed.

Native receipt: `.codex-tmp/vendor-select-native-final.log`; diagnostic runner: `.codex-tmp/diagnose-vendor-filetype.mjs`; final event traces: `.codex-tmp/vendor-filetype-{0,1,2,3}.json`. The full canonical release runner is delegated to the parent and CI.
