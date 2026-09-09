# Document explorer in narrow workspaces

The hosted vendor workspace at a 1280px viewport leaves roughly 630px for Documents. Viewport-only breakpoints kept a sidebar and full table inside that pane, compressing the file name to a few characters per line. The previous standalone renders did not cover this composition.

Use the existing compact File type selector and document cards when the explorer's available width is narrow. Preserve the standalone desktop sidebar and wide table, existing file filtering, selection, preview and paging. Selected-file details must not compress the table. No new tokens, data semantics or authority changes are needed.

- [x] Reproduce and retain the nested vendor before state.
- [x] Adapt navigation and table layout to available component width.
- [x] Verify nested vendor, standalone Forms and selection-dialog states in both themes, including narrow mobile widths and keyboard selection.
- [x] Update the canonical Forms journey checks to test the active layout and full-width file names.
- [x] Run copy, component and build checks; complete independent review.
- [ ] Pass the full Forms journey and release CI on the final commit.
- [ ] Merge and deploy through CI, then inspect the hosted narrow vendor pane.

The user has authorized the explorer refinement, merge and deployment. This corrects an observed failure in the requested workflow.

The preceding demo-file release is deployed as `59041af28fd52a2fc02686d924f487739ed0b4db`. Hosted sample reuse moved Security self-declaration from Missing to Awaiting review, reduced missing items from 9 to 8, persisted after refresh and displayed Unscanned · Demo. A protected download event completed successfully. The sample assessment still awaits other vendor responses, so document acceptance remains unavailable until its review stage; no vendor activation or compliance conclusion was recorded.

Both new 1280px vendor scenarios failed against the original build, confirming the regression check. The final layout has ten passing rendered cases with no document/explorer overflow or document refetch during resizing. Selection survives the layout change. The shared DataTable option passed its focus/selection test and the fifteen affected component/document tests. The copy-quality regression passed. Independent source review found no actionable P1/P2 defects; root visual inspection prompted the final card alignment and narrow-value stacking corrections.

Rendered inspection also identified a lazy-stylesheet ordering defect: the spacing token resolved to 16px but the header painted with zero padding. Declaring the approved layer order before the document feature rules restores 16px padding and the border. At table widths up to 360px, opt-in cards stack labels above values so badges and Preview controls retain readable widths. The complete Forms journey and final contract checks remain required before merge.

The final sidebar is 14rem so every visible label remains one line with the restored padding and shared button typography. Ten final cases passed again; the selected standalone table retains 706.8px and table rows. Final TypeScript and all 51 harness/contract checks passed. Follow-up review found no P1/P2 issues in the cascade and compact-card corrections.

The first full Forms run stopped at an unrelated menu test whose delayed synthetic scroll could execute outside the intended opening guard. A controlled delayed callback reproduced its exact failure; delaying the timer also demonstrated a false pass before injection. The harness now injects the opening scroll synchronously and waits for restoration, retaining full geometry and popup-bound checks. Four targeted runs passed, including three under 2x CPU slowdown. No builder behavior was changed. The full 75-scenario run and release CI remain required before merge.

A subsequent full local run passed that menu check but exceeded the existing 500ms large-form update budget. The unchanged performance check passed in isolation at 392ms (render 1143ms); its budget remains unchanged and CI still runs it. The new nested light-theme file-type journey then exposed a distinct real SelectField defect: an unchanged native scroll queued before pointer-down closed the just-opened menu. Native listener tracing reproduced this twice and showed the guard's cleanup microtask executing between native listeners during the same event dispatch. The bounded correction retains the captured event through its dispatch and uses its existing eventPhase check to avoid suppressing later user actions. Regression and native dismissal checks are required before updating the release commit.

Final menu verification passed: 14 SelectField tests, including three regressions observed failing before their corrections; 24 native checks across both themes; TypeScript and evidence build. Genuine outside interaction can dismiss during scroll restoration, and repeat mouse/virtual trigger presses close the menu while touch/keyboard retain the existing library route. Independent final review found no P1/P2 issues. The committed native receipt records the finite matrix; full release CI remains required.
