# Governed document result handoffs

## Decision and scope

Implement the first bounded slice of approved master issue #200 (IGX-00/09): document-analysis results must open their existing Program, requirement, control objective or issue. The user approved proceeding with the master on 8 September 2026. This slice changes navigation and receipt copy, not authority, conversion or legal conclusions.

Baseline `8540bda0`: approved conversion displays a raw result ID; coverage matches/related issues/applied suggestions lack links; Program routes select a tab but not a requirement/control objective. Preserve this source/state inventory as the before-state; no new mockup.

Use existing ActionLink and Program tab/record components. Requirements and objectives gain an optional typed URL target and a focusable existing record container; the authenticated Program read must contain the target before it can receive focus. Do not fetch broad populations to locate a result or invent a separate record viewer. Missing/malformed/unsupported result metadata has readable recovery rather than a broken or guessed link. Opening a record is never another review/authorization command.

Routes extend the existing Program section path with `requirement/{id}` or `control-objective/{id}` under `requirements-controls`; retain old routes. Encode each ID segment. Pass the typed optional target through App/ProgramsView/ProgramsWorkspace to ProgramRecordWorkspace. Focus only once when the requested record becomes available; background refresh must not keep stealing focus. Missing targets show a neutral notice without echoing arbitrary URL IDs. Changing tabs clears the item target. Browser Back retains the existing exact import route; complete import filter/scroll resume is outside this slice and remains IGX-00.

Approved handoff labels name the created requirement/control objective and expose Open requirement/Open control objective. Coverage links identify the existing matched requirement, related issue and supported applied result. Unknown result types, missing parent Program IDs and failed/unapplied suggestions never become guessed destinations. Supporting copy must preserve the distinction between conversion, draft creation, applicability, approval and verified outcome.

## Required proof

- Route round-trip and encoded IDs; unsupported/missing type/ID does not target an arbitrary record.
- Approved requirement/objective receipts navigate to their stored parent and result; rejected/failed/incomplete receipts never advertise successful navigation.
- Coverage match, related issue and applied Program/Requirement/Matter links use stored IDs; proposed/failed/unknown results do not.
- Loaded authorized Program focuses the requested record after section load; missing/revoked/unavailable records show safe recovery, no fallback selection. Background revalidation does not steal focus.
- Existing tab keyboard behavior, route navigation and authority-gated mutations remain unchanged.
- Render actual Imports and Program destinations in both themes at desktop and 390/320px, verify keyboard focus and no blocked actions. Use existing tokens and opaque fallback; no new density/motion/theme.

No AI, schema, API, dependency or new workflow engine is needed. This is a partial closeout of IGX-00/09, not regulatory lifecycle completion or measured bank-user usability acceptance.

## Verification and visual repair

Implementation commits `2d488fe8` and `2a76b062` add stored-result navigation and clear a previously loaded Program if its current read fails. Spec review found that preserving the old aggregate on a failed reload concealed the unavailable state; 403/404/503 regressions failed before the repair and passed afterward. All current read/target-generation guards remain in place. Failed reads use the existing unavailable/retry screen rather than adding a second stale-data mechanism.

The isolated `document-result-handoffs` and `document-result-incomplete` fixtures reuse existing sample Program items. They are evidence-only and remain unreachable from the customer entry point. Browser checks cover 30 states: approved receipt, focused requirement, focused objective, unavailable target and incomplete receipt, each in light/dark at 1440/390/320px. They assert stored destinations, browser Back, section target clearing, no guessed links, no raw missing IDs, no horizontal page overflow and receipt WCAG contrast. Narrow Program navigation exercises its actual selector replacement, not a hidden desktop tab.

Rendered inspection identified three concrete repairs, using existing tokens and breakpoints:

- The handoff used a hardcoded dark translucent background in light mode. Receipt text measured 1.96:1 and 2.06:1 contrast instead of 4.5:1. The receipt and review detail surfaces now follow existing document theme tokens; the focused contrast check passed in both themes.
- Nested coverage grids expanded to 322px on a 320px viewport. Explicit shrinkable single-column tracks contain the existing filter scroller and cards without clipping content; the same measurement became 320px.
- Control-objective cards retained two columns at 390px, leaving a narrow title column. At the existing 560px breakpoint, title/outcome and implementation details now stack in reading order. The rendered bounding-box assertion failed before this change.

Before-state artifacts are retained locally in `C:/Users/Son/AppData/Local/Temp/clearsight-result-handoffs-contrast-red-20260908` (contrast) and `clearsight-result-handoffs-final-20260908` (pre-stacking objective). CI retains the reproducible release screenshots and manifest as its normal UI/UX artifact; no new mockup or customer fixture path was introduced.

Limitations: fixture interaction checks are not timed bank-user acceptance, real provider/recipient evidence, production security certification or a document-authenticity conclusion. Existing 200% CSS/reflow proxies are not actual browser-zoom evidence. Complete filter/scroll resumption and the remaining IGX-00/09 lifecycle work stay open in #200.

Final local verification on 8 September: `go test ./...` passed; `cd web; npm test` passed 153 files / 1,018 tests, including copy quality; typecheck, runtime isolation, 11 UI contracts and customer/evidence builds passed. Independent spec and quality review passed after the recorded repairs. The serial `npm run review:ui` run passed **176/176 flows/screenshots, 76/76 Forms capabilities, 9 behavioral scenarios and 8 general accessibility routes**, plus the new receipt-specific contrast checks. Output: `C:/Users/Son/AppData/Local/Temp/clearsight-handoffs-final-serial-20260908`. Final rendered images were inspected, including the stacked focused objective and unavailable/incomplete states. An earlier concurrent full-suite run recorded a 675ms large-form edit against the unchanged 500ms budget; the isolated full rerun passed. Do not treat that earlier failed run as release proof.

Subsequent Linux PR CI on `191bc5ba` found a 328px document-result overflow at 320px despite the Windows pass. A Windows Verdana fallback-font probe reproduced 327px and identified the new related-issue link in the existing horizontal `.coverage-matter` row. That row now uses the existing 620px stacked replacement; the unchanged overflow check includes bounds diagnostics and the 320px fallback-font probe. All 30 focused captures passed after the repair (`clearsight-handoff-font-green-20260908`). PR #201 records the final exact-head CI and deployment receipts; the earlier CI failure is retained, not waived.
