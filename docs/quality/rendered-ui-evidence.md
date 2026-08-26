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

The current suite exercises **82 deterministic rendered states/interactions** across:

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
- partial vendor-work delivery followed by a successful retry, without retaining the invitation capability in the page;
- exact submitted vendor answers with source provenance, missing and conditionally omitted fields, and available versus quarantined document handling;
- response review, change-request preparation and accepted request history with one dominant action for the current state;
- vendor-work response review at 390px and 320px, with answer, value and provenance stacked instead of compressed into desktop columns.

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
