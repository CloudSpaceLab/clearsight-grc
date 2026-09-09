# ClearSight interface contract

This is the fast, implementation-ready design contract for people and coding agents. Canonical product and safety semantics remain in `docs/`; this file defines how those semantics should appear and behave in the interface.

## Product and users

The selected vendor overview prioritizes failed requirements from submitted custom forms, using Missing, Expired, Not met and Awaiting review. A compact selected-vendor heading, plain service summary and inline work counts keep response gaps near the first viewport; identity edits remain in expandable details. The [vendor compliance overview brief](docs/design/2026-09-09-vendor-compliance-overview.md) preserves assessed conclusions, source attribution, unknown freshness and paginated scope without inventing a vendor compliance percentage.

Forms library rows expose a server-authorized Edit draft or Edit form action alongside Details. The detail sheet places its permitted actions before version and ownership facts. Editing saves a draft version through the existing approval flow; published and latest versions remain separate. Unavailable or denied actions identify the responsible editor/reviewer or the recovery step. The [Forms editing brief](docs/design/2026-09-09-forms-edit-actions.md) defines the responsive and authority state matrix.

Vendor form work follows the [8 September field assessment decision brief](docs/design/2026-09-08-vendor-field-assessment.md): compact vendor status links, separate completion and risk, side-by-side submitted answer/bank judgement with narrow-sheet stacking, and explicit per-vendor request confirmation. Existing tokens and variants apply; fixture and render requirements are in the brief.

ClearSight is repeated-use operating software for bank executives, risk and compliance leaders, reviewers, authorizers, business owners, administrators and invited respondents. It should feel calm, exact, premium and institutional—not theatrical, playful or generic.

The interface optimizes for:

- rapid scanning of real work;
- clear ownership, authority and deadlines;
- minimum-question review and capture;
- visible source, evidence and uncertainty;
- safe interruption, resume and handoff;
- explicit outcome confirmation before closure.

## Visual direction

Use restrained institutional futurism:

- deep, low-noise dark surfaces and a clear neutral light equivalent;
- fine borders and subtle depth;
- cyan for navigation and active information;
- violet for governance/configuration context;
- amber for pending review or attention;
- green for a supported current state;
- coral for blocking gaps or failed outcomes;
- premium vector illustrations only for orientation, education, onboarding and genuine empty states.

Avoid decorative gradients, glass, glow or illustration where they compete with status, evidence or decisions.

## Token, cascade and density architecture

The design-system entry point fixes the cascade order as `reset`, `tokens`, `base`, `components`, `features`, `utilities`, then `overrides`. A feature may arrange a shared component inside its workspace, but it must not win the cascade by redefining the component's fill, border, radius, focus or interaction state. `overrides` is reserved for documented compatibility corrections, not ordinary feature styling.

Tokens have three levels:

1. **Primitive tokens** (`--cs-primitive-*`) store raw colour, spacing, type, radius, border, icon, shadow, duration, easing and z-index values. Product components do not consume raw values when a semantic role exists.
2. **Semantic tokens** express a role across theme and context: canvas/surface, primary/strong/muted text, default/strong/interactive/invalid border, primary/destructive action, information/success/warning/error/unknown feedback, focus, spacing, type, motion, overlay and document roles.
3. **Component tokens** (`--cs-button-*`, `--cs-action-card-*`, `--cs-field-*`, `--cs-search-*`, `--cs-checkbox-*`, `--cs-select-*`, `--cs-filter-chip-*`, `--cs-tabs-*`, `--cs-scope-*`, `--cs-overlay-*`, `--cs-popover-*`, `--cs-table-*` and related families) bind the semantic roles to a closed component contract. A new component token must be useful to every instance of that component family; page-specific layout stays in feature CSS.

`web/src/design-system/tokens/` owns the three-level foundation. `web/src/ui-preferences.css` supplies compatible legacy theme mappings while surfaces migrate. Components must not create a private light/dark palette when a semantic role already represents the meaning.

Shared action foregrounds are owned by the component token layer (`--cs-button-primary-text`, `--cs-button-secondary-text`, `--cs-button-quiet-text` and `--cs-button-destructive-text`). Legacy feature styles must not apply an unlayered global button foreground or font shorthand because that would override the layered component contract in both themes.

Theme preference supports **System**, **Light** and **Dark**. Comfortable controls are 44px high. Compact controls are 40px high on a fine-pointer desktop; compact mode does not reduce touch targets below 44px. Spacing uses the 4px primitive scale with 8px as the normal grouping rhythm. Component radii come from the 6px, 10px and 14px primitives; large guide or illustration treatments may keep a larger documented radius outside operational controls. Shadows and blur remain subtle and never carry state.

Typography uses Inter, Segoe UI Variable, Segoe UI, then system sans-serif. Headings use tight tracking; operational copy uses normal sentence case. Uppercase is limited to compact metadata labels.

`react-aria-components` owns keyboard, focus, selection, overlay and internationalization mechanics for the complex primitives that use it. ClearSight owns the public TypeScript contract, DOM/class contract, visual states, copy and business behavior. Native controls remain appropriate where the platform interaction is the clearer contract, including date, date-time, time, number and file selection; they still use the shared field anatomy, validation, target-size and theme rules.

Focused dialogs and drawers use the theme backdrop plus `--overlay-blur` to reduce competing shell detail. The overlay must not conceal errors, authority explanations or material status inside the focused surface. Compact line icons use `--icon-size-small` inside controls that retain the `--interactive-target` hit area.

Document reading and comparison surfaces use the document token family. Light mode presents imported material on a white paper-like surface with dark text; dark mode maps the same semantic roles to a quiet dark document surface. Navigation, status and review actions remain visually distinct from the document body.

## Shared component contracts

Migrated product screens import these closed contracts from `components/ui`; feature CSS arranges them but does not restyle their internals.

| Family | Supported contract |
| --- | --- |
| `Button` | Primary, secondary, quiet and destructive actions; comfortable or compact density; disabled and loading states. |
| `ActionCard` | One consequential route with a title, concise supporting context, optional icon and disabled state. |
| `ActionLink` | Real navigation only. |
| `IconButton` | Named icon actions using the Button variants. |
| `FormField` | Shared label, required marker, guidance and validation anatomy. |
| `TextField` | Text, search, email, URL, telephone, numeric, date, time and local date-time values; numeric minimum, maximum and step constraints; disabled, read-only, invalid and loading states. |
| `SearchField` | Compact, visibly recognizable search for one named record population; labelled, loading and disabled states. |
| `SelectableRecord` | One record in a bounded master list with visible title, metadata, optional supporting detail and accessible selected state. |
| `CheckboxField` | Selected, unselected and indeterminate choices; visible or visually hidden labels; optional guidance and disabled state. |
| `TextArea` | Multi-line responses with bounded length; disabled, read-only, invalid and loading states. |
| `SelectField` | One selection from a bounded list with themed listbox keyboard behavior and a non-modal fixed overlay root outside sticky workspace layouts. The bounded listbox owns option scrolling. If focusing its initial selected option causes incidental document positioning, the field restores the pre-open position without closing. An unchanged scroll queued before opening is ignored only during that event's dispatch, including native listener microtask checkpoints; a later user page scroll still dismisses the list. Explicit keyboard and outside interaction dismiss during positioning; the trigger toggles its own menu. Opening never resizes or locks the document container, and dialogs remain the dismissal boundary for their own lists. |
| `Tabs` | Automatic peer-view navigation with one selected indicator. Optional `compactLabel` replaces the tab list with a labelled SelectField at widths up to 760px; the selected panel stays mounted and named across resize. Opt-in `retainVisitedPanels` mounts each panel on its first visit and retains local drafts, pending operations and conflicts; inactive panels are visually hidden and inert. Unvisited panels do not start reads. The owner revalidates material state when necessary. Other tab workspaces retain wrapped compact behavior. |
| `ScopeBar` | One selected bounded result scope with stored counts and horizontal-overflow replacement behavior. |
| `StatusBadge` | Neutral, information, success, warning, error and unknown labelled states. |
| `Notice` | Information, success, warning and error conditions at the point of work. |
| `Surface` | Related work containment without implying a record. |
| `Card` | One coherent object or decision. |
| `EmptyState` | A named checked population, current empty result and next valid action. |
| `FilterBar` | Responsive fields, result count and clear handling. |
| `FilterChip` | A removable applied filter or named action that reopens advanced filter logic; default and accent treatments. |
| `DataTable` | Populated, selected, loading, pagination and stacked-mobile data presentation. Optional `responsiveTo="container"` adds a named `cs-data-table` inline-size container and uses the same stacked rows when its available width is at most 700px; the default remains viewport-based and all tables retain the 700px viewport fallback. Container-responsive cards left-align their labels and keep badges/actions at intrinsic width; at 360px of table space all cells stack their labels above values. Optional column `mobileLayout="full-width"` stacks the label above a full-width value in that compact layout; ordinary columns retain the key/value layout. |
| `FocusedSheet` | Dismissable, focus-contained detail or action in default or wide composition, with full-height mobile replacement and shared heading, facts, form and action anatomy for governed work. Consequential in-flight submission may temporarily disable Escape, backdrop and close controls; the busy action and close control must explain that delivery is in progress. |
| `FocusedDialog` | Centered, dismissable desktop decision or creation surface in default or wide composition, with full-height mobile replacement. |
| `PopoverDialog` | Dismissable, focus-contained short contextual work anchored to its trigger. Long or consequential work uses `FocusedSheet`. |

The static-only UI component gallery renders every family from production exports. New variants update the closed TypeScript union, component tokens, this table and the gallery in the same change.

Migration is explicit rather than inferred from visual similarity. A file is migrated only when it appears in `web/ui-contract-migrations.json`, imports public contracts through `components/ui`, has no prohibited raw interactive controls, and is represented by a gallery or full-host state fixture. The workspace-by-family record in [`docs/design/ui-component-adoption.md`](docs/design/ui-component-adoption.md) is the human review view of that boundary. Using one shared component in an otherwise legacy workspace does not make the whole workspace migrated.

Operational success and failure receipts use `Notice` rather than feature-owned coloured text. The message remains the primary state cue; semantic tone is limited to the notice border/surface treatment while readable copy uses the theme's primary text token. Compact stored states use `StatusBadge` with a non-colour marker and an explicit semantic tone, including an explicit unknown tone in each theme. Every badge foreground must meet WCAG AA against its resolved tinted surface in both themes. Feature workspaces arrange these components but do not define private feedback or status palettes.

## Structural patterns

- **Intervention Summary:** actor-scoped read projection for one human review, decision, authorization, evidence exception, escalation or outcome check. It is not new authoritative state.
- **Today:** intervention queue first; quiet status-check context follows the work rather than preceding it with a KPI wall.
- **Oversight:** freshness and coverage first, including projection version and source high-water detail, then ranked intervention records. Risk pressure, minimum-five historical resolution ranges and owner workload/history remain supporting analysis rather than personal assignments, employee scores or rankings. Unknown event-history measures display as unknown rather than zero.
- **Programs:** ongoing responsibilities, current status and reasons. The portfolio remains a bounded searchable list; an exact Program opens a dedicated operating record. Show calculated-state freshness, reasons, named owner and one dominant actor action before details, requirements, safeguards, evidence, monitoring and linked issues.
- **Issues and changes:** bounded items needing review, decision, action, response or outcome confirmation. Show the current handoff before history.
- **Work:** review queues and focused evidence. Complete source inventories are secondary context.
- **Configure:** policy, routing, integrity and ownership.
- **Side panel:** bounded inspection or one focused action without losing list context.
- **Dedicated page:** complex or protected work requiring several sections, parallel work or a durable saved state.

The dedicated Program record keeps calculated state distinct from operating lifecycle status. Focused forms use only current Program-owned objects and authority-returned candidates. On narrow screens, the two-column record becomes a single reading order, action groups become full-width controls and fixed navigation must not cover the active form or result.

Issue and Action reassignment uses one focused sheet that shows the exact work, current responsibility, an authority-returned eligible person and the required reason. Recording the assignment and attempting the assignee email are separate durable outcomes: the assignment remains effective if no usable staff mailbox is available, and the UI states that limitation before submission.

Assigned Action cards present lifecycle and timing as separate states. An open Action whose deadline has passed keeps its lifecycle badge and adds an explicit **Overdue** badge plus the dated elapsed time. Completed or cancelled Actions retain their recorded deadline without being relabelled overdue. Missing or invalid deadlines state that no action deadline is recorded.

Response handling names the actor's exact task. Reviewer, signatory, transmitter, acknowledgement recorder and issue authorizer actions use distinct labels, required basis fields and consequence copy. Sign-off never claims transmission; transmission never claims acknowledgement; authorization never claims that assigned work or an independent outcome check has completed.

Do not default every concept to a dashboard card. Choose lists, rows, details, tables, timelines or focused forms according to the operator's task.

### Progressive disclosure for governed work

Use the same reading burden across operational surfaces:

1. **Queue:** human gate, material conclusion, scope, evidence state, deadline and next action.
2. **Current handoff:** what changed or why the current state needs this actor.
3. **Review context:** evidence, contradictions, alternatives, decisions, actions and verification.
4. **Reconstruction:** complete Program/Matter, source lineage, imported material, operator receipt and immutable history.

Complete context must remain reachable, but it must not be the default reading burden. Do not relabel ordinary status data as AI or automated work; show operator/prepared-work claims only when a governed receipt exists.

## Copy

Primary screens use familiar working language. Internal codes remain available in APIs, audit views and specialist detail.

Examples:

- `EVIDENCE_INSUFFICIENT` → **Evidence incomplete**
- `DECISION_REQUIRED` → **Decision needed**
- `VERIFICATION` → **Confirming outcome**
- `MATTER` → **Issue or change** on general screens
- `CONTROL_IMPLEMENTATION` → **Safeguard** on business-owner screens

Keep useful product nouns when they already describe the job clearly. **Today**, **Programs**, **Work**, **Evidence**, **Imports** and **Configure** are practical navigation labels; do not replace them with generic phrases such as “Your work” merely to sound conversational.

Every page answers: what is shown, current state, why now, owner, next action, source and time. Never replace an unknown population with sample or persuasive numbers.

Authority copy follows the same rule: the interface may show only roles, stages, limits and explanations returned by the authority service or other canonical records. It must not invent a familiar approval chain when the runtime has not returned one.

## Inputs and capture

Ask for the smallest fact the user actually needs to supply. If the request already knows an ATM identifier, address, vendor, legal entity, date range or other context, render it as read-only context instead of asking the respondent to type it again.

Choose controls by the fact being captured:

- **Yes/no or 2–4 choices:** large tap/radio choices, especially on mobile and external capture.
- **Short names, identifiers and one-line facts:** single-line text input.
- **Explanations and exception notes:** textarea; do not use a textarea for ordinary short text.
- **Dates:** native date control unless the domain requires a richer scheduling control.
- **Numbers:** numeric input with the appropriate input mode and server-side range validation.
- **Signature/attestation:** a bounded signature control attached to explicit attestation copy; signing does not replace the factual fields being attested.
- **Files and photos:** use the upload pattern appropriate to the importance of the artifact rather than defaulting every attachment to the same large box.

### File-upload patterns

Use one shared dropzone interaction with three presentation levels:

1. **Photo evidence:** prominent drop/tap area, camera-first on supported mobile devices, immediate thumbnail preview, filename/size, and replace action. The factual confirmation remains a separate field.
2. **Document import or another file-primary task:** large dropzone with filename/type/size shown before the explicit import/submit action. Selecting or dropping a file must not silently commit it.
3. **Incidental general attachment:** compact dropzone/file row with filename/size and replace action. Do not spend a large part of the form on an optional PDF or supporting file.

Do not use a dropzone for settings or forms where the file is incidental to a different control. Do not enable multiple files unless the request contract explicitly permits multiple artifacts and defines how each is reviewed.

When multiple files are permitted, show each accepted filename and size in a removable list, preserve files that uploaded successfully if a later file fails, and enforce minimum count, maximum count, per-file size and combined size. Client checks provide immediate guidance; the service repeats the authoritative checks against the selected field and submitted artifact IDs.

Artifact admission derives the media type from bounded file contents rather than trusting the browser. Filenames and extensions must agree with the detected type. PDF and Open XML documents receive bounded structural checks, including rejection of active PDF actions, Office macros and embedded payloads. This admission check is not a malware scan: accepted artifacts remain labelled `STORED_UNSCANNED` until the separate security workflow changes that state.

Image preview is appropriate for image evidence. For PDFs, Office files and other documents, show trustworthy metadata before upload; do not fabricate a document preview before extraction/rendering has actually succeeded.

### Submitted document browser

Forms Documents, the completed-response document action and the selected vendor document action share a Finder-style file browser. A labelled file-type rail offers All files, PDF, Images, Word documents, Spreadsheets and Other files. Filename search and version filters query the authorized server population; displayed counts identify only the bounded current page. Source form/question and submission details belong in the selected-file inspector, not repeated paragraphs in every file row.

DataTable now supports opt-in single selection and row activation: click selects, Up/Down move focus and selection, and Enter/Space or double-click opens the row. Nested buttons, links and editable controls retain their own actions. Non-selectable tables retain their existing keyboard behavior. The component gallery exercises this contract.

Quick Look reuses the centered wide FocusedDialog, with the existing restrained blur, focus containment, Escape and focus restoration. File browsers launched from a vendor or response use FocusedSheet. At widths up to 760px a labelled File type SelectField replaces the sidebar and shares its selected kind, query and pagination state. At widths up to 700px table rows become stacked data cards, with the complete filename using the full card width beneath its Name label and other metadata remaining key/value rows. The side inspector is replaced by preview details, and Quick Look fills the viewport. Resizing preserves selected files and open previews. No new theme, density mode, overlay implementation or token family is introduced.

Ordinary uploaded documents can return bytes only when AVAILABLE. PDF and supported raster images render from protected fetched content; a MIME/size mismatch or failed fetch shows recovery, not a successful preview. Office and unsupported types show metadata and a download action without pretending to provide page previews. Safety status is not authenticity, document acceptance, response acceptance or compliance. Uploaded-by and submitted-by facts remain distinct and unknown attribution is labelled Not recorded. Document decisions, replacement, disposal and AI validation are separate implementation tranches and must not appear as enabled no-op controls.

The [non-production fictional sample exception](docs/design/2026-09-08-demo-document-preview.md) permits protected preview/download only for an exact shipped-manifest match after current occurrence authorization and full stored-byte integrity verification. These samples remain STORED_UNSCANNED and display **No antivirus scan was performed** before content/actions; the exception creates no scan receipt and changes no inspection or review-acceptance gate.

Required fixtures: populated/mixed types, exact response, exact vendor, current/history, no results, unavailable inventory, pending inspection, quarantined file, unsupported preview, content failure/retry, keyboard row navigation and focus restoration; render light/dark at desktop and 390/320px, with 720px reflow. See `docs/design/2026-09-07-submitted-document-browser.md` for implementation evidence and limitations.

### Vendor due diligence

Selected relationships use Overview, Forms, Documents, Due diligence and History. Shared Tabs retain visited panels lazily; at 760px and below the labelled Vendor section selector replaces the tab strip and the selected relationship replaces the register. The persistent header shows vendor, service, relationship status and current owner; Overview owns edit controls and collapsed record metadata. Forms owns its request action, Documents reuses the relationship-scoped browser, and History rechecks current revisions on return without discarding review drafts. An unavailable currency check cannot enable a review. Guided and register-summary shortcuts select the relevant section without executing a command or stealing focus on later tab visits.

The Vendors workspace uses one dominant action for the current assessment state: start or restart onboarding, start a scheduled or event-driven reassessment, send the request, review collection status, begin review or record the conclusion. A reassessment requires the organization's schedule, change or event reference so a retry reuses the same episode. The selected vendor, service, accountable owner, exact form version and review deadline remain visible around that action.

A new vendor records legal name, service, website and registered address in the same focused setup. Website discovery is optional and never blocks creation or due diligence. The reference environment installs its standard due-diligence form through the same draft, maker submission and distinct-checker activation transitions used by configuration commands. A non-reference tenant with no active form opens a focused governed setup that selects a Program, creates the draft, submits it for approval and requires a different authorized checker before activation; the interface never silently activates a form.

External collection uses the shared request-scoped invitation and capture experience. Known vendor and service facts are prefilled or shown as context; the active form decides between Classic and Wizard presentation and renders typed controls, limits, conditional fields, uploads and attestation. Submission means the response was received, not that the evidence was accepted or the relationship was approved.

Internal review shows only the exact scoped response, answer provenance, coverage, artifact scan state, evidence classification, linked canonical findings and version-qualified provisional score. Reviewer conclusion, vendor-relationship activation and deficiency closure remain separate material outcomes. Completed assessments are read-only in the relationship workspace.

### Governed Forms workspace

Forms is a direct primary navigation surface. Its default view is a bounded searchable template library, not a creation wizard. The library distinguishes the latest stored revision from the active reusable revision, supports saved views and keeps filters available for banks with hundreds of Programs, Matters and vendor relationships. Template detail, editor, sender, response history, import handoff and communications remain tabs within the same visual system.

At widths up to 760px Forms uses the shared Tabs compact variant labelled Forms section. It replaces only the navigation controls, so viewport changes preserve the current content, editor draft and document state. Both presentations use the same selected section and route history.

The selected Forms peer section is a validated `section` hash-query value, restored on reload and browser Back/Forward. Templates remains the default for absent or unknown values; existing template IDs and library filters retain their route meaning. Tab changes add history only when the section changes. Returning to Documents re-reads the scoped inventory rather than restoring protected rows from browser storage. File filters, pagination and selected-file recovery remain separately tracked under #200. See `docs/design/2026-09-08-forms-section-resume.md`.

Advanced scoring remains inside the form inspector rather than opening a raw rules console. Contributions, typed comparisons, bounded rule groups, effects and band ranges use shared field, select, button and notice contracts. The inspector scrolls independently; narrow rule rows stack in source order. Score preview is one dominant action and is available only for an exact saved revision. Its result names raw score, adverse concern, coverage, band and rule effects without implying evidence acceptance or compliance.

Forms Policies is a peer workspace backed only by stored policy rows. Its lifecycle presents one dominant action for the selected state: simulate, submit, approve, activate, suspend or roll back. Draft creation captures the exact form revision, subject population, automation-policy revision, blast radius, effective window, Matter handling and outcome check in a focused dialog. Shared fixed-overlay selects must not resize the workspace, shift sticky columns or scroll the document. Empty, sign-in, unavailable, retained-data and command-failure states name the legal-entity policy population and the valid recovery action.

The sender has one dominant **Create and dispatch** action after an exact active revision, subject, purpose, future deadline, access expiry and at least one To recipient are valid. Internal directory search and external email entry share the same recipient list with explicit To/CC roles. Sent-form detail exposes real lifecycle actions. Amendment uses native date-time inputs, adds or revokes recipients and explains retained response history. Form replacement first shows compatible and excluded answers; it never carries answers without explicit confirmation.

Long recipient forms show saved, saving, conflict and access-ended state near the work. Browser recovery is supportive, encrypted and workspace-bound; it does not imply that unsynced answers or file bytes reached ClearSight. Document-style reading surfaces use a light paper surface in light mode and a high-contrast document surface in dark mode. Focused editors may use a restrained overlay and backdrop blur, but the active heading, status, close action and validation remain crisp, labelled and keyboard reachable.

### Vendor identity and icon

The vendor identity editor is separate from the service-relationship editor. It changes the shared legal name, trading name, registration reference, jurisdiction, optional website hostname and registered address without changing a relationship owner, service, assessment or relationship version. Website input accepts an HTTPS URL or hostname and stores a normalized hostname; a missing hostname is valid.

Vendor rows and details use a stored safe raster when available and a stable monogram otherwise. An approved logo takes precedence over a discovered website icon. Removing the approved logo restores the latest icon that matches the current website hostname; when no safe icon exists, the monogram remains available. Pending or unavailable retrieval is stated in text and never changes due-diligence or relationship status. Image sources stay on the ClearSight origin.

### Vendor requests for Programs and issues

A Program or issue or change may link one or more vendor relationships without transferring the bank owner's responsibility. `Request vendor work` uses the existing form revision, invitation, capture, artifact and submission experience. The request keeps one primary Program or Matter target, a concise purpose, a bank owner, a reviewer, a deadline and an immutable sequence of the initial request and any requested changes.

`Request vendor work` opens as a wide focused sheet on desktop and a full-height replacement on narrow screens. Request kind, linked vendor work, approved form revision, layout, recipient and deadline use shared field contracts; opening any list must not shift the page or change the sticky workspace geometry.

`Link vendor` opens a focused sheet with a blurred, opaque-fallback backdrop. Search is bounded and delayed briefly while typing; each choice shows the stored vendor icon or monogram, legal name, service, criticality and relationship status. The sheet traps keyboard focus, supports Escape, restores focus to its trigger and preserves the search, selection and purpose after a recoverable failure.

The review shows the exact current response and AVAILABLE documents before `Accept response` is enabled as the dominant conclusion. Receipt, upload, review, acceptance, implementation and verified outcome remain separate states. Acceptance never closes a Matter, completes an action, changes a Program's status or approves the vendor relationship. Ending a vendor link is unavailable while active vendor work still depends on it; ended links and prior responses remain in history.

External capture should minimize normal-path typing. A field-visit verification should ordinarily be completable through known context, tap choices, required photo evidence, an optional exception note and an attestation/signature. The interface target is under four minutes for a representative simple visit; that target is not considered proven until a timed usability run confirms it.

## States and recovery

Every significant component and screen defines:

- loading;
- live/default;
- empty for an explicitly named scope;
- stale or partial data;
- unavailable source/API;
- permission denied or wrong scope;
- validation error;
- optimistic conflict;
- success/receipt;
- long content and translated text;
- keyboard, focus, 200% zoom and reduced motion.

A disabled control explains why. A visible enabled control must perform a real action.

Deadline recovery never relies on colour alone: overdue work has a text badge, a semantic date-time value and a readable elapsed-time label in both themes.

## Responsive behavior

Vendor due diligence uses **Checklist**, **Missing**, **Awaiting review**, **Accepted** and **Received**. Do not expand these into question-like headings or explanatory status sentences. Retain supporting text only for the actor, source, condition, consequence or recovery. Unresolved applicability is labelled **Applicability pending**, independently of missing evidence. A received or linked document does not imply acceptance or vendor approval.

The due-diligence checklist precedes form history; vendor reference details are expandable. Existing documents are selected in the protected document browser, linked with a reason, and reviewed in a focused sheet with Accept/Reject. On mobile, checklist rows and summary filters stack; selection and review actions remain reachable. See the [decision brief](docs/design/2026-09-08-vendor-evidence-reconciliation.md).

Responsive work is replacement, not shrinking:

- desktop may use parallel context, dense rows and side panels;
- tablet reduces simultaneous columns and preserves the next action;
- mobile converts dense rows into stacked summaries and complex side panels into full-screen flows;
- external capture prioritizes one question group, progress, save/resume and safe receipt;
- protected work may intentionally disallow offline or shared-device use.

## Motion

Motion is functional and short: panel entry, expansion, focus, progress and state change. The initiating component owns the motion. Every animation has a reduced-motion fallback and should not delay interaction. No ambient motion around material decisions or alerts.

The Today and Vendors first-run introductions use the shared cinematic panel. Two SVG groups and the action block may enter through opacity and transform only, with each segment no longer than 400 ms. Reduced-motion preference renders the final state immediately. The guide remains non-modal, keeps navigation and work available, and exposes **Start guide** and **Skip for now** without delay. Saved progress is scoped to the actor, tenant, guide and version; a failed save does not block the workspace.

## Illustration and icons

Illustrations use an editorial, semi-abstract vector language with restrained geometry, soft depth and no mascot personality. They support first-run guidance, empty states, education and completion. Semantic line icons identify recurring object types. Neither replaces labels or status.

Illustration geometry stays shared across themes; palette comes from semantic/theme variables. Each production illustration exposes an accessible title/description rather than a generic unlabeled SVG.

Populated repeat-visit Today, Programs, Vendors, Work, Evidence and Configure states do not use decorative hero illustrations. The optional first-run cinematic panel is the bounded exception; its operational sequence and actions are also present as HTML. Primary visual hierarchy still comes from the human gate, status, evidence and next action.

## Design proof

Significant UI work requires:

1. a compact decision brief;
2. a before-state baseline when redesigning an existing screen;
3. at least the required state matrix;
4. rendered evidence at representative viewports;
5. one highest-impact repair and re-check;
6. design-token, input and copy review before merge.

Vitest/axe semantic checks are executable CI evidence, but jsdom does not prove visual contrast, 200% zoom, responsive replacement or theme parity. Those remain rendered-browser evidence gates.

See `docs/design/ui-delivery-workflow.md` and `docs/quality/rendered-ui-evidence.md`.

## UI audit corrections — 9 September 2026

- Shared input boundaries use `--cs-border-field`; placeholders use opaque `--cs-text-placeholder`. These are distinct from decorative dividers. Placeholder text uses neutral 700 in light mode and neutral 500 in dark mode; field boundaries use neutral 600. Status backgrounds are an opaque 8% tone mix over the theme surface, so selection/tint beneath a badge does not change its text pair.
- Vendor relationship creation, due-diligence editing/review and linked-work recovery use shared field/action contracts. Specialized document, radio and vendor-identity controls retain their explicit contracts. TextField supports native pattern/inputMode/focus and stable IDs; TextArea supports stable IDs and address autocomplete.
- FocusedSheet and FocusedDialog share scroll-lock and focus-return mechanics. Their geometry and in-flight dismissal behavior remain separate. Multiple overlays retain the lock until the last closes. The older domain EmptyState is a thin adapter to the shared rendering contract, with an action only when it has a handler.
- The current vendor checklist leads; **Requests** and **Linked vendor work** are secondary disclosures. Errors and reload controls remain visible outside the Requests disclosure. Attention routes expand their destination. Checklist document coverage is matched per field; uncovered documents remain available even when another field uses the same artifact.
- Response detail presents identity/history once and a single assessment/result/document entry. Automatic and reviewed results remain distinct. Response list filters are collapsed behind **Filters**; counts, active filters and reset remain visible. This replaces the default vertical filter stack at narrow widths.
- Forms Imports opens the Imports workspace directly; old section URLs hand off without another launcher. Builder overview shows review counts and configuration errors; selected questions retain detailed rubrics.
- Shared copy follows the [industry-neutral content standard](docs/product/plain-language-content-standard.md). Required state fixtures include unavailable/stale calculations, absent/invalid schedules, missing/reassigned historical reviewers and nonbank requirement authoring.

The accepted audit is the decision brief. Before-state captures remain under `docs/evidence/2026-09-09-ui-audit`; correction evidence and its limitations are recorded in the acceptance receipt. These changes do not establish product-wide component migration or WCAG conformance.

Selected values in shared SelectField show the option label only. Optional descriptions remain in the option menu; they are not concatenated into the collapsed trigger. Imports mobile actions use the existing `--cs-target-min` target height.

Spreadsheet field review uses one keyboard-focusable scrolling region capped at 720px or 65svh. On narrow screens its fields and preview stack within that region; the draft actions follow it instead of following the full spreadsheet length. New vendor classification fields start empty. The starter's Yes/No assurance declaration reveals either document collection or a required gap explanation, with receipt and review still separate.
Proposal source quotes match the retained worksheet row and location. Review-note text uses the existing primary text token on its tinted background; the response-preview heading follows the proposal heading without skipping a level. The `vendor-release-spreadsheet-source-rows` evidence fixture checks two distinct quoted rows, selection and the draft action at 1440px and 390px in both themes. This is a correction to the existing proposal screen, with no new layout, token or presentation mode.

Finding registers add an optional **Prepare finding follow-up** disclosure with a shared source-assessment selector and explicit generation action. Selected-assessment review keeps all finding fields together, displays historical source context and requires a confirmation checkbox before **Create follow-up draft**. **Review source-row proposal** remains available during pending generation and after acceptance/rejection; switching views does not cancel a durable generation request. The existing review region and responsive stacking remain unchanged. Required evidence covers choice, generation failure/retry, complete field preview, confirmation, draft receipt and return recovery at 1440px/390px in both themes. No new token, density or motion pattern.

Recognized risk registers also offer **Import vendor findings** after the source heading. Vendor/service and repeated responsibility mappings precede source-backed finding/action review. Exact unique matches are suggestions; an explicit assignment confirmation gates import. Desktop pairs search/relationship and responsibility/deadline fields; mobile replaces those pairs with one column. Excluded assessments do not require vendor matches. A saved-but-unconfirmed result retains its choices and offers an idempotent retry. Native labelled controls use existing theme tokens, 44px targets and visible focus rings. Required fixtures are review, missing vendor, ambiguous name, unavailable authority, imported receipt and interrupted-import recovery at 1440px/390px in both themes. Evidence and limitations: `docs/acceptance/2026-09-09-risk-register-migration.md`. No new token, density or motion pattern.
