# Shared interaction and privacy contract — V2.11

## Core primitives

`interaction-core.js` loads after the V2.10 source/report modules and before domain interaction adapters. It does not own a domain repository or start the application.

- `ui211Route(route, patch)` retains existing route parameters and changes/removes only the requested values.
- `ui211Issues(items, options)` renders escaped correction messages with an exact route or field focus target.
- `ui211Focus(selector, root)` reveals ancestor disclosure sections, scrolls to the target and focuses an editable control or the section. URL-driven focus is confined to the main workspace and runs once for that route token.
- `ui211Next(config)` provides a compact next-action block where the current view needs it; action markup comes from registered domain actions/routes.

## Modal submission and recovery

The original generic modal submit handler delegates to `ui211SubmitModal`. Native modal validation is collected into a focused, clickable summary, including whitespace-only required input. Generated field IDs are monotonic and unique across repeated validation passes. Existing IDs are addressed by escaped attribute selectors.

The submit callback is invoked once. Pending submit buttons show Saving and are disabled. If it returns a promise, the dialog body becomes inert while pending; fields remain enabled so FormData retains their values. Inert state, busy attributes and buttons are restored on completion or rejection. Rejection preserves all entries and shows the actual error. Asynchronous action-button handlers similarly prevent repeated invocation and handle rejected promises.

A callback may return false to display a review step without committing. Its new button label is retained, and its dirty baseline is not reset. A callback that opens the next dialog is not followed by a stale close of that new dialog.

The current modal stores an in-memory field fingerprint, edited flag, pending dismissal and origin route fragment. This is transient presentation state, not a duplicate business draft. Cancel/Escape/internal links show Keep editing or Discard when entries differ from the baseline. Back-style hash navigation restores the same-origin source hash before presenting that choice; discard then follows the originally requested route. Pending/successful command navigation is not intercepted. Browser close/reload uses the native beforeunload mechanism. These native browser behaviors still need real-browser observation.

## Search, filters and focus

Global search includes stable workspace entry points and scoped programs, engagements, risks, losses, controls, documents, processing records, impact assessments, cases and KRI returns. It searches IDs, names and owners, prioritizing exact IDs/names. It does not list ID-less detail pages as workspaces or add a second Evidence destination. The catalog is prepared when the dialog opens; query filtering ranks that catalog without mutating domain state.

Arrow keys select a result and Enter opens it. Existing list filter/scroll memory remains authoritative; the shared code adds a visible clear-filter action and result announcements. Tables become keyboard-focusable only when their content overflows. Error/focus navigation expands existing disclosures without modifying source facts.

## Privacy preparation

`privacy-interactions.js` uses the existing `S.journeyDrafts`, `jWizardIssues`, save/submit and privacy review commands. The new Saved drafts tab consolidates activity, DPIA and case preparation in the selected scope and resumes each exact saved step. It creates no new active record merely by displaying or resuming a draft.

Processing/DPIA/case completeness remains derived from the existing validators. Each issue maps to its original step and field; repeated purpose, system, recipient, retention and risk rows retain their indices. Final activity review presents a section checklist; the DPIA recommendation editor remains editable beside correction links. Forward navigation/submission with unresolved items stays on the draft and shows those links.

Case handling replaces the ambiguous Next step select plus Save button with explicit Save progress and Submit for review buttons. A capture-phase submit handler sets the existing hidden status before the original domain command reads FormData. Enter/no submitter defaults to Save progress. Returned reviewer comments remain visible and escaped next to the correction form. Existing closure/reopening episodes and review gates are unchanged.

## Local monitoring and settings

The existing deterministic source fixtures can be run separately. A card shows its latest observation and quality; the history records each run. Checks neither request credentials nor create requirement conclusions. Settings copy now describes named assessment assignments and separate requirement decisions accurately.

## Tests and production boundary

`interaction_core_v211_checks.js` exercises asynchronous save/action completion/failure, duplicate submission, retained input, staged-review labels and dirty state, successive validation IDs, hash-navigation recovery, scoped search and escaped exact routes. `privacy_interactions_v211_checks.js` covers scope-preserving draft discovery, exact correction routes, blocked forward navigation, explicit case intent, returned comments and isolated fixture runs.

Tests run actual JavaScript with substituted browser elements/events. They do not establish native layout, focus, inert, screen-reader, keyboard, history, storage/download or print behavior. Server identity, transactions, durable audit and real delivery remain production responsibilities.
