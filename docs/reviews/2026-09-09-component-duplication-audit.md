# Component and workflow duplication audit

Date: 9 September 2026. Scope: production frontend, with priority on Forms and Vendors. This is a read-only audit of the current working tree, including the uncommitted vendor evidence reconciliation work. No production files, tests, commits or deployed configuration were changed.

## Assessment

The most consequential bloat is repeated decisions and response detail inside an already long workspace, followed by incomplete adoption of shared controls. File size alone is not a defect. The current reconciliation work has already improved the vendor workspace by putting the checklist before response history and collapsing reference facts. Preserve those changes.

Prioritize these corrections:

1. Give the selected vendor one state-appropriate primary action; remove its duplicate generic request button.
2. Render a submitted response's automatic score, rule explanation and document action once in Forms.
3. Filter residual vendor documents individually so one uncovered document does not restore a duplicate list of all documents.
4. Migrate the remaining vendor controls and remove obsolete editor branches before adding more CSS patches.

Use concise labels such as **Pending review**, **Received**, **Accepted**, **Missing** and **Review response**, with the subject and freshness beside the label when needed. Keep receipt, evidence acceptance, field judgement, reviewer conclusion and relationship approval separate.

## Method and limits

Read the root contributor rules, README, DESIGN, docs map, application architecture, governed Forms specification, current implementation ledger, UI adoption matrix, migration manifest and vendor reconciliation acceptance record. Inspected source for the shell, Today, Programs, Forms, Vendors, Work/Evidence, document import, configuration, capture, shared UI and related review components. The source findings below have exact file/line anchors; line numbers refer to this working-tree snapshot.

The reproducible [inventory script](../evidence/2026-09-09-ui-audit/component-inventory.mjs) writes [component-inventory.json](../evidence/2026-09-09-ui-audit/component-inventory.json). Run from the repository root with `node docs/evidence/2026-09-09-ui-audit/component-inventory.mjs`.

- Reachability follows local import/export references from `web/src/main.tsx`, including lazy imports, CSS, type imports and barrel exports. It is a conservative source inventory, not the bundle's tree-shaken module graph.
- JSX tags and state calls are source-site counts. Conditional branches, repeated list items and unused branches within reachable modules are not resolved. A raw input is not necessarily wrong: file, signature and specialized capture controls can legitimately remain native.
- CSS is parsed with the installed PostCSS package. Repeated selectors are exact selector text plus at-rule context within one file. This does not measure selector coverage, cross-file cascade conflicts or unused CSS bytes.
- Unreachable files are investigated as candidates, not automatically deleted. Evidence-only components and their tests are deliberately outside production.
- This sub-audit generated no new renders or task timings. The separate visual audit generated the captures; this reviewer subsequently inspected its vendor desktop and Forms response mobile PNGs for C01/C04/C11. Other findings remain source-confirmed or explicit design proposals. No workflow tests were rerun because this audit changes no application behavior.

## Quantified inventory

| Measure | Count | Interpretation |
| --- | ---: | --- |
| Reachable TSX files | 169 | Includes shells, feature components, wrappers and shared primitives; not 169 screens |
| TSX source lines | 22,379 | Blank lines and compressed JSX affect comparison |
| Reachable CSS files / lines | 76 / 8,682 | Several minified legacy files encode many rules on one line |
| CSS rule blocks | 3,330 | Not unique selectors or rendered styles |
| `!important` declarations | 53 | Includes legitimate visually hidden and motion cases |
| Same-file repeated selector/context groups | 10 | Inspection candidates, not ten proven bugs |
| Reachable TSX entries in migration manifest | 45 | Includes 21 shared UI files and partial workspace shells; not a workspace adoption percentage |
| Native button / input / select / textarea sites | 486 / 219 / 94 / 82 | Includes specialized and legacy sites; not user-visible simultaneous controls |

These non-exhaustive, non-overlapping filename groups identify migration concentration; remaining files include shell, Today, documents and other shared features.

| Group | TSX files | Lines | Native buttons | Inputs | Selects | Textareas |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Forms workspace, builder and `forms/` | 42 | 5,903 | 61 | 45 | 0 | 7 |
| Vendor-prefixed components | 11 | 3,388 | 104 | 43 | 12 | 12 |
| Program-prefixed components | 12 | 1,749 | 81 | 43 | 28 | 25 |
| Matter/Evidence-prefixed components | 15 | 2,618 | 68 | 26 | 24 | 27 |
| Configure/Governance/Automation/Monitoring/Routing | 20 | 2,585 | 63 | 30 | 16 | 4 |
| Capture-related components | 10 | 1,513 | 29 | 11 | 1 | 2 |
| Shared `ui/` | 21 | 831 | 0 | 2 | 0 | 1 |

The [adoption matrix](../design/ui-component-adoption.md) correctly marks many workspaces as incomplete. Shared navigation does not establish consistent body fields, feedback or action behavior. Conversely, zero raw controls in a file does not mean all states or supporting CSS have been certified.

## Findings and decisions

### C01 — Duplicate generic request competes with due diligence

**P1 · Source and rendered composition confirmed · Merge/Move · Medium effort, high workflow sensitivity.**

`web/src/components/VendorsWorkspace.tsx:758` always renders a primary **Request form** for the selected relationship. `web/src/components/VendorFormsPanel.tsx:45` renders another **Request form** using the same `onRequestForm` callback. The callback is wired at `VendorsWorkspace.tsx:669`; both open the same `VendorFormRequest` at line 697. At the same time, `VendorDueDiligence.tsx:534`–542 chooses a state-specific primary action such as preparing the due-diligence request, reviewing existing evidence or recording the conclusion.

The duplicate header action gives a generic distribution prominence even when an assessment already has a next step. It can encourage a fresh request when the existing assessment needs evidence or review. This is an interaction risk, not proof that the backend creates identical records.

The [rendered desktop vendor state](../evidence/2026-09-09-ui-audit/contrast/vendor-checklist-light-1440.png) confirms both generic request buttons and the assessment conclusion action in the same relationship workspace. Its bottom Vendor requests section is a separate linked-work scope and must not be treated as the same command.

**Decision:** Keep one selected-relationship generic request entry in the Forms section or its secondary actions. Remove the duplicate primary header entry. Let the assessment's current actor/state determine the dominant action. Keep the register's batch request at `VendorsWorkspace.tsx:638`: selecting up to 50 services is a distinct scope and uses the same composer. Do not merge generic distributions into assessment commands; `docs/product/governed-forms.md` preserves the assessment origin boundary.

**Acceptance:** Empty relationship, active collection, received evidence, pending review, completed assessment, unavailable authority and multi-selection fixtures show the correct next action. One action creates or resumes the intended request once. Generic requesting remains discoverable.

### C02 — Forms response detail repeats automatic results and evidence access

**P1 · Confirmed same-response composition · Merge · Medium effort, medium risk.**

`web/src/components/forms/ResponsesView.tsx:229` renders the automatic score summary, line 237 coverage, and line 246 `ScoreExplanation`. Line 247 then mounts `ResponseAssessment` for the same response ID. `web/src/components/forms/ResponseAssessment.tsx:90` renders the automatic score again alongside the bank-assessed score, and line 121 renders the automatic rule explanation again. The parent document action at `ResponsesView.tsx:248` and child evidence action at `ResponseAssessment.tsx:120` both open `DocumentBrowser` for that response revision.

The reviewer must scan two score presentations and two paths to the same evidence. Parent and child load independently, creating an additional consistency burden even though this audit has not reproduced stale values.

**Decision:** Establish one response review composition: identity/revision/freshness once, one automatic/reviewed result comparison, one expandable automatic explanation, one document entry, then field judgements. Keep submission assurance and revision history, which the assessment alone does not replace. Reuse a shared score presentation contract for the compact list and expanded review; do not merge automatic and reviewed results into one unexplained number.

**Acceptance:** Automatic-only, required manual review, provisional, failed calculation, historical, partly replaced, no evidence and restricted evidence cases retain their distinctions. A response containing documents has one clear document action and one explanation for its automatic result.

### C03 — Mixed checklist coverage restores duplicate document rows

**P2 · Confirmed conditional behavior · Merge · Small effort, medium evidence risk.**

`web/src/components/VendorDueDiligence.tsx:518` mounts the checklist. At line 520, `showDocuments` becomes true if **any** review document's field is absent from the checklist. `ReviewSummary` at line 658 then renders **all** `review.documents`. A mixed response with one covered and one uncovered document therefore repeats the covered document and its actions below the checklist.

**Decision:** Pass the residual documents themselves, filtered by the correct requirement/field association, instead of toggling the whole list. Keep an uncovered supporting document accessible. When one artifact supports multiple fields, keep separate field decisions and provenance; do not deduplicate solely by artifact ID. Preserve the current behavior that suppresses the legacy document section when the checklist covers all fields.

**Acceptance:** All covered, none covered, mixed covered/uncovered and one artifact supporting two requirements; each requirement's review appears once, with a working source action and no lost document.

### C04 — Three vendor work views need a single default attention hierarchy

**P2 · Source and rendered composition confirmed, hierarchy proposal · Move/Keep · Medium effort, high workflow sensitivity.**

`web/src/components/VendorsWorkspace.tsx:801`–803 mounts relationship activation, Forms/responses and vendor work beneath due diligence. `VendorFormsPanel.tsx:52` renders potentially 25 detailed requests with recipient, deadline, progress, delivery and up to two score summaries. `VendorWorkPanel.tsx:227` renders every loaded current linked-work request with its own lifecycle actions. These populations can represent different lifecycles and must not be assumed to be duplicate database records.

**Decision:** Keep the active due-diligence checklist and actor's next step first. Move generic requests/response history and Program/issue-linked requests into clearly scoped sections with compact counts and drilldowns. Preserve a direct route from register attention counts to the requested section. Keep the existing collapsed reference facts at `VendorsWorkspace.tsx:760` and the existing history disclosure at `VendorWorkPanel.tsx:228`; do not undo those improvements.

**Acceptance:** Render with 0, 1 and 25 mixed requests at desktop and narrow widths. The next due-diligence action is visible before historical detail; linked work retains its Program/issue purpose and separate review result. Avoid loading protected populations into browser storage for resumption.

### C05 — Vendor controls remain split between native and shared contracts

**P2 · Confirmed implementation duplication · Merge · Medium-to-large effort, medium risk.**

`VendorDueDiligence.tsx` contains 40 native button, 17 input, 8 select and 5 textarea sites, with eight form sites; its newer checklist uses shared controls. The conclusion form at line 639 uses a native select/date/textarea and local action styling. `VendorsWorkspace.tsx:636` uses a native search field/button while line 637 uses `SelectField`; its create/edit form uses another local field convention. `VendorWorkPanel.tsx:231` uses a shared request sheet while card-level recovery and review controls remain native.

**Decision:** Migrate these complete interaction slices to existing Button, TextField, TextArea, CheckboxField, SelectField and feedback contracts. Preserve typed input restrictions, labels, required/disabled reasons, authority checks, optimistic versions and pending-command dismissal rules. Keep file and signature controls specialized where their behavior requires it. Register complete migrated files only after all controls/states are covered.

**Acceptance:** Keyboard, 200% zoom, light/dark, loading, permission, validation, conflict and narrow layouts. No generic disabled state without an explanation. No change to request preparation, sending, conclusion or approval semantics.

### C06 — Obsolete authoring branches and source-only panels are maintenance bloat

**P2 · Reachability plus reference search · Remove/Move · Small-to-medium effort, low-to-medium risk.**

The production graph does not reach `web/src/components/forms/FormPropertyPanel.tsx:30` or `FormQualityPanel.tsx:7`. Reference search finds only the former's direct test, and no consumer for the latter. Current authoring uses `forms/builder/FormInspector.tsx:30` and `FormReviewDrawer.tsx:18`.

There is also a retained `LegacyEditor` in the production-reachable `FormFieldPropertyEditor.tsx:76`. The current inspector passes `inspector` at `FormInspector.tsx:136`; the other caller is the unreachable `FormPropertyPanel`. This means raw-control totals include a legacy branch not selected by the current production caller.

`web/src/components/ReadinessPanel.tsx:4` has no source consumer; Today uses `TodayInterventions` (`AppViews.tsx:25`). `ProgramLifecycleControls.tsx:46` is consumed by tests and the evidence page, while production uses `ProgramStatusPanel` (`ProgramRecordWorkspace.tsx:203`). That is a fixture-versus-production drift risk, not permission to remove lifecycle coverage.

**Decision:** Remove obsolete Forms panels and the legacy editor branch after porting any unique reusable-section assertions to the current builder. Remove unused ReadinessPanel if no planned owner exists. Move the old Program lifecycle demonstration out of the production component namespace or migrate its evidence/tests to the actual ProgramStatusPanel. Keep evidence entrypoints and the UI gallery; they are intentionally unreachable from `main.tsx`.

**Acceptance:** Import/reference search, TypeScript/build, reusable-section behavior, production Program lifecycle tests and evidence fixture generation. Do not delete a passing test merely to make a dead component disappear.

### C07 — Competing empty-state implementations and duplicated overlay mechanics

**P2 · Confirmed shared-foundation duplication · Merge/Keep · Medium effort, medium accessibility risk.**

`web/src/components/EmptyState.tsx:13` has an illustration-first contract, optional population label and native action button. `web/src/components/ui/EmptyState.tsx:10` has a required checked population, labelled heading and shared action slot. They coexist across feature groups. `forms/FormsEmptyState.tsx:11` is only an adapter to the latter; it is not a third visual implementation.

`ui/FocusedSheet.tsx:17` and `ui/FocusedDialog.tsx:17` independently implement invoker capture, body scroll lock, scrollbar compensation and restoration. They already differ in zero-width handling and dismissal support. Centered dialogs and side sheets are justified presentation variants; duplicating focus/scroll behavior is not necessary.

**Decision:** Consolidate empty-state semantics around the shared contract, with optional illustration only for a genuine first-run/empty need. Keep context-specific recovery content. Share tested overlay mechanics or rely on one consistent underlying modal contract while retaining sheet/dialog geometry. `components/FocusedSheet.tsx:1` is a compatibility re-export, so do not classify it as a separate overlay.

**Acceptance:** Nested document sheets, Escape/backdrop, non-dismissible pending commands, focus return to existing and removed triggers, scrolling, empty data and unavailable-source recovery. Do not flatten signature editing into a modal: its specialized non-modal behavior serves capture.

### C08 — Status and response formatting are repeated across related features

**P2 · Confirmed presentation duplication · Merge/Keep · Medium effort, medium semantic risk.**

Vendor work filters are separately declared in `VendorFormsPanel.tsx:12` and `VendorsWorkspace.tsx:637`. Answer/provenance/document formatting appears in `VendorDueDiligence.tsx:702`–785, `VendorWorkPanel.tsx:413`–419 and `EvidenceWorkspace.tsx:39`–61. Automatic score presentation appears in `VendorFormsPanel.tsx:101`, `ResponseAssessment.tsx:127` and `ResponsesView.tsx:265`.

These are not necessarily interchangeable domain states. For example, file availability, evidence class, receipt, reviewer judgement and relationship status each answer a different question. However, the same filter enum and score units should not accumulate independently worded implementations.

**Decision:** Share the vendor filter registry first. Extract small pure presenters for common date/size/score/provenance facts with explicit input types and currency/unknown handling. Keep lifecycle-specific state maps and command permissions in their owning feature. Avoid a generic humanizer that converts every code to a plausible status.

**Acceptance:** Same response/revision produces the same score units, unknown coverage and freshness across list/detail/history. Neutral review labels stay consistent. Provenance remains visible without exposing protected recipient/source data to external capture.

### C09 — Broad and obsolete CSS layers make component changes harder to assess

**P2 · Confirmed layering, candidate dead rules · Merge/Remove · Medium effort, medium visual risk.**

`web/src/main.tsx:8`–25 loads the base stylesheet, feature layers, design-system contracts, preferences and two fix files. `styles.css:45`–49 defines legacy primary/secondary buttons; `ui-preferences.css:226`–254 restyles them; shared buttons use their own contract in `design-system/components/actions.css`. Form authoring also carries old foundation rules, dashboard/creation layers and the 1,170-line `form-builder-workspace.css`.

`forms-foundation.css:1` retains `.form-property-panel`, `.form-quality-panel`, `.forms-grid`, `.forms-layout-toggle`, `.forms-recent` and `.forms-library-summary`. The first two are used only by unreachable components; the latter four have no TSX source references found in this audit. These are high-confidence cleanup candidates, subject to checking generated class names and evidence consumers.

The ten same-file repeated selector groups in the inventory include `.form-builder-inspector` at `form-builder-workspace.css:143,617`, `.vdd-limitation` at `components/vendor-due-diligence.css:48,57`, and `.document-import-inspector` at `document-import.css:19,120`. Separate media queries and theme overrides are preserved in the measurement; repeated selectors alone are not a defect.

**Decision:** Remove obsolete selectors alongside retired components. Move surviving rules into their owning component/feature layer, consolidate same-context blocks and progressively retire native-button rules as adoption completes. Do not remove all `!important` or override styles indiscriminately. Do not start another universal fix stylesheet.

**Acceptance:** Before/after computed styles and renders for every affected host, including popup positioning, sticky builder geometry, focus, capture file inputs, reduced motion and both themes. Bundle-size claims need a production build comparison, not source line counts.

### C10 — Forms Imports is a redirect workspace, and builder overview repeats every field

**P3 · Confirmed composition, hierarchy proposal · Move · Small-to-medium effort, low risk.**

`web/src/components/forms/FormsTabContent.tsx:19` handles the Imports peer section by showing a full empty state whose only operation sets `window.location.hash = "#imports"`. It is an extra page before the real import workspace, while `FormsWorkspace.tsx:450` already offers import from the creation launcher. This is repeated navigation, not a second importer.

`forms/builder/FormInspector.tsx:55` defaults **Assessment and scoring** open and maps every draft field, its rubric outcomes and an edit button. The outline/canvas already exposes those fields; for a long form this makes the overview another extensive field inventory.

**Decision:** Consider a direct, clearly named import handoff while preserving old routes. In builder overview, show the assessment summary and fields with configuration problems; move full per-field/rubric detail to the existing selected-field inspector or an intentional expanded review. Keep advanced scoring and manual field assessments as distinct settings.

**Acceptance:** Existing deep links/back navigation, dirty-editor recovery and import handoff work. Long forms expose counts and configuration gaps without requiring a scan of every rubric. Measure navigation/scroll effort before declaring the redesign better.

### C11 — Narrow Responses puts the complete filter form before the results

**P2 · Source and rendered composition confirmed · Move · Medium effort, low-to-medium risk.**

`web/src/components/forms/ResponsesView.tsx:162`–177 renders all seven filtering/sorting controls: Priority, Concern, Score meaning, Score state, two completion dates and Subject type. The [390px response capture](../evidence/2026-09-09-ui-audit/contrast/forms-response-light-390.png), inspected by this reviewer and the root reviewer, shows those controls stacked into a long sequence before the list. This is excessive default control exposure, not duplicate filtering semantics. The capture has response detail open and does not establish an exact interaction-time penalty.

**Decision:** Keep the result count and concise sort control in the default narrow view. Move the secondary fields into a shared **Filters** sheet or disclosure; show active filter chips and a clear reset action outside it. Retain every filtering capability and persisted query meaning. Preserve desktop behavior where the available width supports it.

**Acceptance:** At 320px and 390px, results or a scoped empty result follow the heading/sort/active filters without traversing seven expanded controls. Opening/closing Filters preserves values, keyboard focus and the selected response; applying/resetting filters uses the existing query lifecycle and correctly reports empty/error states. Verify both themes, long labels and 200% zoom.

## App-wide keep/merge/remove/move map

| Surface | Decision | Reason and source |
| --- | --- | --- |
| Shell and exact records | Move broad introduction out of exact-record view where redundant | `AppViews.tsx:30` renders a Programs header even when `ProgramsWorkspace.tsx:63` replaces its body with the named Program record; `ProgramRecordWorkspace.tsx:228` has the record heading. Retain navigation/context, reduce repeated introductory chrome after rendered review. Work has a similar parent/record composition at `AppViews.tsx:39` and `MatterRecordWorkspace.tsx:135`. |
| Today | Keep focused intervention list; remove unused old readiness panel | Current consumer is `AppViews.tsx:25`; old ReadinessPanel has no runtime consumer. Do not reintroduce a decorative readiness hero. |
| Programs | Keep sectioned record and separate status/ownership/requirements/evidence/history | `ProgramRecordWorkspace.tsx:203`–218 already separates operational sections. Migrate native controls incrementally; do not merge operating status with calculated compliance state. |
| Forms | Merge response duplication; remove retired editor branches; move redirect-only import handoff | C02, C06, C10. Template library, sending, response review, policies and communications serve distinct tasks. |
| Vendors | Merge exact duplicate request action; move history/linked work; filter residual documents | C01, C03, C04. Keep identity editor separate from relationship editor because their versions and scope differ. |
| Work / issues | Keep distinct actions, decisions, responses and outcome checks; migrate controls | `MatterRecordWorkspace.tsx:161`–168 composes many panels. Quantity is justified by different actors/outcomes; progressive disclosure is safer than combining authority steps. |
| Evidence / Documents | Keep one protected DocumentBrowser with scoped wrappers; merge same-response duplicate launchers | Forms, vendor-wide, response-specific and checklist selection contexts correctly reuse the component. Different permission/record scopes are not duplicate viewers. |
| Imports | Keep one governed document/proposal process | `DocumentImportWorkspace.tsx` spans extraction, analysis and review; it is 551 lines, not proof of waste. Extract handlers/view sections only around real cohesion or testability needs. |
| Configure | Keep distinct policy lifecycles; merge shared fields and feedback | GovernanceAdminWorkspace has 11 native button, 9 input and 7 select sites. Configuration, authority and automation policies must retain simulation, distinct review, effective dates and rollback. |
| Explore/reference journeys | Keep evidence/reference-specific components out of production entry graph | Reference journeys provide labelled samples and real record handoffs. Evidence-only unreachable files are intentional, not cleanup targets. |
| External capture | Keep task-specific control semantics and recovery states | CaptureForm/CaptureFieldControl already centralize typed answers; upload, receipt, saving, local recovery and submitted state must not be collapsed into one generic success panel. |
| Shared primitives | Merge behavior duplication; keep thin compatibility/domain adapters | C07. A wrapper or same filename is not evidence of duplicate implementation. |

## Delivery sequence and evidence required

**First slice:** C01–C03. Prepare a compact decision brief preserving current vendor reconciliation and exact request origin. Add fixtures for competing actions, repeated same-response summaries and mixed checklist coverage. Review before/after full-host desktop and mobile renders, then run affected Forms/Vendor and copy-quality regressions. This slice should remove duplicated work without altering authority semantics.

**Second slice:** C05–C08. Migrate vendor controls in complete interaction groups; extract common presentation/overlay contracts; port useful tests and retire unreachable authoring branches. Keep state-machine adapters and command calls separate.

**Third slice:** C04, C09, C10, C11 and shell refinements. Use representative long forms and populated vendor workspaces to decide what starts collapsed or moves to a section. Remove measured unused CSS with matched visual evidence. Update DESIGN, the component adoption matrix, executable manifest and state fixtures for any changed contract.

The current evidence establishes source duplication and cleanup candidates. It does not establish runtime performance gains, bundle reduction, accessibility completion, end-user timing improvement or completion of an app-wide design-system migration.
