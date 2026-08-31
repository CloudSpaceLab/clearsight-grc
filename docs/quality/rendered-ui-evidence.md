# Rendered UI evidence gate

A UI PR is not accepted from code review alone when it introduces or materially changes a screen, workflow, reusable component, responsive composition, illustration system or motion pattern.

## Required evidence

Attach or generate deterministic renders for the affected states and viewports. At minimum:

- desktop primary state;
- narrow/mobile replacement;
- loading;
- explicit empty scope;
- unavailable/error and recovery;
- long content;
- keyboard focus;
- any material warning, conflict or permission state.

Add light/dark, 200% zoom, reduced motion, translated copy and assistive-technology evidence when the changed surface supports or materially affects them.

## CI capture is the default proof path

For frontend changes, `.github/workflows/ui-evidence.yml` is the default rendered-evidence path. It builds the deterministic static stakeholder application, launches it locally, and executes `web/scripts/capture-ui-evidence.mjs` with a pinned Playwright runtime.

The workflow is also triggered by backend read-contract changes that can alter what the UI may truthfully render, including Today projection, authority resolution, actor context/identity capabilities and the executable runtime API contract. This prevents a visually stable fixture from silently drifting behind server semantics.

The capture suite prefers production components and realistic deterministic fixtures over hand-built visual mocks. Its artifact contains PNG renders plus `manifest.json` with the route, fixture, viewport, theme, density, active focus and layout metrics used for each capture.

## UI foundation and capture matrix

The current suite exercises **122 deterministic rendered states/interactions** across:

- Today in light and dark themes;
- comfortable and compact desktop density;
- Program, Matter, Evidence, Imports and Configure workspaces;
- dedicated Program review in changed and acknowledged states on desktop and mobile, with calculated-state freshness, named responsibility and executable lifecycle action;
- Program safeguards, evidence results and monitoring in light/dark semantic tokens, including automated contrast validation;
- tablet, 390px mobile and 320px reflow;
- exact record-scoped authority resolution, including candidate-set semantics;
- evidence response entry → exact response review → submission receipt;
- explicit Today empty, loading and service-unavailable states;
- partial Evidence degradation where requests fail but source context remains reachable;
- partial Configure degradation where one configuration source fails but other context remains visible;
- capability-constrained navigation where Configure is unavailable to the current role;
- authority-inspection permission denial without candidate leakage;
- evidence-request not-found and expired/read-only states;
- optimistic submission conflict with an explicit reload route;
- long translated-style/mobile content expansion;
- focused mobile Capture with keyboard focus containment;
- a 200% pixel-density/reflow proxy in addition to narrow reflow evidence;
- external field-agent ATM verification on a 390px phone: known address displayed read-only, two tap confirmations, one required photo, optional note, signature, review and receipt;
- photo evidence preview before review;
- document-import dropzone selection with filename/size visible and a blank user-authored purpose before the explicit import action.
- vendor due-diligence start, ready-to-send, submitted review and mobile review states;
- vendor form-source degradation with an explicit reload action;
- vendor invitation delivery failure with recovery status and no retained raw recipient field.
- vendor work entered from exact Program and issue/change records, using the linked vendor relationship already in scope;
- vendor work preparation with Automatic, Classic and Wizard layout choices, typed email/date controls, the approved form version and its document requirement;
- governed Forms template library and filtered-empty search across desktop and mobile themes;
- sent-form recipient/status management, mobile amendment with native calendar/date-time inputs, and immutable scored response revisions;
- partial vendor-work delivery followed by a successful retry, without retaining the invitation capability in the page;
- exact submitted vendor answers with source provenance, missing and conditionally omitted fields, and available versus quarantined document handling;
- response review, change-request preparation and accepted request history with one dominant action for the current state;
- vendor-work response review at 390px and 320px, with answer, value and provenance stacked instead of compressed into desktop columns.
- populated governed Forms records at 390px, with State, Revision, Owner and Updated facts spanning labelled stacked rows rather than entering the checkbox/action column;
- governed Forms authoring at 390px and 320px, with Preview, Review, Save draft and Send for approval visible, 44px action/reorder targets and keyboard Move up/Move down alternatives.
- governed Forms authoring with a real desktop mouse drag that changes question order, plus a 120-question/10-section fixture with recorded render and edit latency.
- the production component gallery in light/comfortable and dark/compact modes, including a real themed Select popup;
- migrated Sent forms empty, populated, partial, paginated, responsive detail-sheet, 390px, 320px and effective-200%-layout states;
- forced-colour focus and reduced-motion component behavior.

## Governed Forms #103 closure inspection

The exact-head renders exposed three defects that structural unit tests did not detect: the focus-managed drawer close button shared a stacking layer with its sticky header and could not receive pointer clicks; stacked mobile form facts auto-placed into the 44px checkbox column even though the document itself did not overflow; and the sticky builder toolbar covered the signed-in account control after scrolling. The drawer close control now sits above its header, every labelled mobile fact spans the full record grid, and builder chrome stays below the active workspace context. Evidence assertions cover each regression.

The corrected matrix passed 114/114 flow records and 56/56 governed Forms capabilities at the #103 checkpoint. Direct inspection of `112-forms-library-populated-dark-mobile-390x844.png`, `113-forms-builder-actions-light-mobile-390x844.png`, `114-forms-builder-reflow-dark-320x800.png`, `115-forms-builder-pointer-reorder-light-1440x900.png` and `116-forms-builder-large-performance-light-1440x900.png` confirmed the stacked record hierarchy, visible mobile authoring actions, separated sticky chrome, changed pointer-drag order and responsive large-form editing. The 120-question fixture recorded 342ms to usable builder content and 57ms for question 100 to update on this development host. Touch scenarios claim the 44px menu/keyboard-equivalent contract, not native HTML drag. These renders remain fixture evidence; representative timed use, actual 200% browser zoom, normal-network p95, hosted exact-commit acceptance and production-scale/PostgreSQL evidence are still separate closure gates.

## UI foundations and Sent forms tranche

The retained pre-migration comparison is `99-forms-distribution-access-history-light-1440x900.png` from the exact source baseline `408306a9eae9eb750a08008df32e7bb90c697210`. It showed transparent 20–21px actions and filters, an operating-system light Select popup over the dark application, inconsistent control anatomy and an empty results region that still reserved a wide table and detail pane with horizontal overflow. The supplied builder screenshot additionally showed the native response-type menu obscuring the working document.

The corrected evidence added these exact scenarios:

| Evidence | Physical viewport | Layout viewport / state |
| --- | ---: | --- |
| `117-forms-component-gallery-light-comfortable-1440x900.png` | 1440×900 | Light, comfortable, all public component families |
| `118-forms-component-gallery-dark-compact-select-1280x720.png` | 1280×720 | Dark, compact, themed Select open |
| `119-forms-sent-empty-dark-1440x900.png` | 1440×900 | Dark Sent forms empty replacement |
| `120-forms-sent-detail-sheet-light-1024x768.png` | 1024×768 | Light responsive detail sheet and lifecycle action |
| `121-forms-sent-populated-light-mobile-390x844.png` | 390×844 | Populated stacked records and pagination |
| `122-forms-sent-populated-light-reflow-320x800.png` | 320×800 | Reflowed filters and stacked records |
| `123-forms-sent-light-effective-200pct.png` | 1440×900 | 720×450 effective CSS layout viewport at device scale factor 2, themed Select open |
| `124-forms-component-gallery-forced-colors-focus-1440x900.png` | 1440×900 | Forced colours, reduced motion and keyboard focus |

The first 320px render exposed the highest-impact remaining presentation defect: **Clear sent-form filters** wrapped to four lines and became 86px tall, consuming the result summary instead of behaving as an action. A failing scenario measurement was added. At widths up to 380px the summary now changes to a single-column replacement and gives the action full width; the replacement render keeps its label on one line and its height within the 44px contract.

The pre-migration production build measured 197,382 bytes initial JavaScript gzip, 351,211 bytes total JavaScript gzip, 32,705 bytes initial CSS gzip and 44,728 bytes total CSS gzip. After integration with the current mainline shell, the corrected review build measures 99,358 bytes initial JavaScript gzip, 449,177 bytes total JavaScript gzip, 31,737 bytes initial CSS gzip and 52,632 bytes total CSS gzip. Route-level loading reduced initial JavaScript despite the new interaction library; total JavaScript and CSS increased because the shared contracts, static evidence route and later mainline workspaces are now present. The final review budget also reports a 339,779-byte largest raw JavaScript chunk.

At implementation evidence head `1c1bbe838cb603730d0a1d370f5a619dce3206d0`, `npm run review:ui` passed 122/122 flow records, 68/68 governed Forms capabilities, 122/122 retained screenshots, all eight accessibility/touch route states and all bundle ceilings. The component and Sent forms renders were inspected at original resolution after correction.

The automated 200% case changes the effective CSS layout viewport; it is still a proxy. A manual native-browser check at the available 1039×782 window confirmed that the Forms host replaces its side rail with bottom navigation from 150% through 200% and keeps the visible template content in the accessibility document. Hardware/window constraints prevented the requested 1440×900 native-zoom viewport, and the manual run did not reach the Sent detail/Select state. Exact native 200% Sent forms and representative assistive-technology acceptance therefore remain open, alongside hosted exact-commit and timed bank-user acceptance.

This tranche migrates the UI gallery, Forms peer-view navigation and Sent forms only. Templates, Builder, Responses, Imports, Communications, external capture and all non-Forms workspaces remain in the adoption backlog in [`../design/ui-component-adoption.md`](../design/ui-component-adoption.md).

## Accepted premium first-run and vendor-brand evidence

The exact browser matrix includes and was visually inspected for:

- Today first-run introductions in dark and light themes, plus desktop 1440px, tablet 1024px and 768px, mobile 390px and 320px replacement layouts;
- reduced-motion rendering with no active guide animation and a 200% pixel-density/reflow proxy where guide actions and workspace navigation remain reachable;
- Vendors first-run introductions with one populated relationship and with an explicitly empty legal-entity register;
- stored website icon, approved logo, pending retrieval, unavailable icon and broken-image monogram fallback;
- vendor identity editing with hostname validation, staged image metadata, optimistic conflict, permission failure and retained entries/files;
- approved-logo removal restoring a matching website icon and restoring the monogram when no matching discovered icon exists;
- desktop and mobile identity editing with the first field focused, a visible **Return to relationship** action, no competing **Add vendor** action, no horizontal overflow and reachable save/brand actions.

The lossless dark-theme cover at `docs/presentation-assets/clearsight-premium-first-run-cover.png` was inspected at its original 1600×900 resolution. It retains the sidebar, guide actions and current Today work context without an open modal or focal obstruction. The exact-head static-evidence bundle measured 578,586 bytes for its largest raw JavaScript chunk and 178,064 bytes total gzip; the production build measured 473,526 bytes and 164,651 bytes. The 600 KiB/192 KiB regression ceilings include the deterministic fixture weight and remain enforced by the review script.

Mechanical checks fail CI for conditions such as unexpected horizontal overflow, browser runtime errors, loss of the first Today action from the unobstructed first viewport, focus escaping a focused-work sheet, authority-detail leakage in a forbidden state, terminal requests exposing submission actions, degraded views hiding still-available context, external capture asking for a known address again, or a required field-agent happy path depending on free-text explanation.

The suite deliberately uses production-shaped readiness and import fixtures rather than inventing stronger product truth. Static fixture schemas for authority, projection health, evidence capture and reconciliation are test-locked to the browser contracts they exercise.

## Input and upload review

Rendered review must inspect whether the chosen input is appropriate, not only whether it renders:

- short factual answers use single-line controls;
- explanations use textareas only when explanation is actually needed;
- dates and numbers use their native typed controls;
- small choice sets use touch-safe choices;
- known request context is not re-entered;
- photo evidence uses a prominent camera/drop/tap surface with immediate image preview;
- document import uses a large dropzone because the document is the primary object, but selecting a file does not itself commit the import;
- incidental attachments use a compact file surface rather than a large empty dropzone;
- document metadata is shown before upload while visual previews are limited to formats the browser can truthfully preview;
- multiple-file UI is not introduced unless the request contract explicitly permits multiple artifacts.

A successful frontend upload interaction is not evidence integrity by itself. Server tests must independently verify that required photo/file/signature answers reference artifacts belonging to the exact request and satisfy the field's media contract.

## Why screenshots are reviewed, not blindly approved by pixel diff

Before a screen has an approved visual reference, a pixel-perfect snapshot can preserve a bad hierarchy. New or materially redesigned screens are therefore reviewed from their deterministic screenshots first. Once a screen is approved and intentionally stable, a visual-regression baseline may be added for that specific stable composition.

A reviewer must inspect the actual PNGs, not only the successful workflow status. Workflow success means the requested state rendered and satisfied the encoded structural checks; it does not by itself mean the result is intuitive or visually correct.

## Review order

1. Correct object, user and primary action.
2. Accurate state, count, source, owner and deadline.
3. Appropriate input method and minimum required effort.
4. Safe recovery and no misleading controls.
5. Information hierarchy and scan speed.
6. Responsive replacement.
7. Keyboard, focus, semantics and contrast.
8. Typography, spacing, assets and motion.

Fix the highest-impact failure first and re-check that evidence. Do not spend the first repair round polishing decoration while the workflow, copy, input or state is unclear.

For qualitative review, explicitly ask:

- Can the intended user state what this screen is for within a few seconds?
- Is the dominant next action visible without reconstructing the workflow from several cards or sections?
- Is the person typing only facts that the system does not already know?
- Does each field use the easiest appropriate input method?
- Is a file upload sized according to its importance to the task rather than shown as the same large control everywhere?
- Does the screen distinguish current, unknown, stale, blocked, pending and verified states without relying on colour alone?
- Are scope, owner, source, deadline and authority visible where they affect the decision?
- Does mobile replace desktop composition rather than merely squeeze it?
- Does light mode preserve the same hierarchy and semantic emphasis rather than becoming a recoloured dark screen?
- Does compact density increase throughput without reducing comprehension or touch safety?
- Is any illustration, panel, chip, border, gradient or explanatory copy consuming more attention than the actual work?

## State-gallery contract

Each reusable component has named fixtures for supported states. Fixtures use realistic but clearly labelled data and the production component API. A component variant that exists only in an ad hoc page is not considered part of the design system.

Fixture data must not advertise stronger product truth than the production domain can currently produce. In particular, a screenshot fixture may demonstrate visual hierarchy for a future governed record only when it is explicitly labelled as such; it must not invent a complete baseline, autonomous work receipt, verified outcome, authority chain or executable permission.

## Release boundary

Rendered evidence proves appearance and interaction for the tested fixtures. It does not prove authority, confidentiality, data completeness, performance or domain correctness; those remain separate release gates.

The UI foundation gate does **not** close the broader enterprise product experience. Governed decision/execution flows, full Configure builders, production Explore, notifications, enterprise identity/security surfaces, production illustration families and representative timed bank-user usability remain governed by their own implementation and release gates.

A 200% pixel-density/reflow capture is an automated approximation, not a substitute for browser/assistive-technology zoom testing. Final accessibility acceptance still requires actual rendered contrast, 200% browser zoom/reflow and representative assistive-technology review.

The external field-visit fixture is designed around a three-minute interaction budget, but the product may not claim the under-four-minute target as proven until a representative human timed usability run confirms it.
