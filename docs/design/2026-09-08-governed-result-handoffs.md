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
