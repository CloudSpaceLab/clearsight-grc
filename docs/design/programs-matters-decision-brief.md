# Programs and Issues/Changes UI decision brief

## Job and users

Program owners, reviewers and executives need to understand ongoing obligations and the specific items preventing a current position. Business owners need plain explanations without losing audit precision.

## Primary objects and actions

- **Program:** inspect current status and the recorded reasons behind it.
- **Issue or change:** understand the next required decision, action, response or outcome check.

The workspaces support both review and governed action. Program and issue creation, current-state changes, facts, assignment, decisions, actions, responses and outcome checks are available only when the verified actor has the required route; read-only users retain the same reconstructable record without enabled controls that cannot act.

## Structure

- Activate **Programs** as a first-class navigation destination.
- Use two tabs: **Programs** and **Issues and changes**.
- Programs use expandable rows to retain portfolio scan speed while exposing requirements and evidence checks.
- Issues/changes use expandable rows showing type, priority, owner, due date, current stage and next step.
- Portfolio queries remain bounded to 20 rows per page and are filtered by the service before rendering. Programs support text, lifecycle status, operating state, jurisdiction and verified-actor assignment. Issues and changes support text, status, type, priority, due condition and verified-actor assignment.
- Applied filters are recorded in the hash query, preserved when opening an exact record and restored on return. Query text is never parsed as part of a record identifier.
- Vendor linking opens a focused, keyboard-contained sheet with bounded search and contextual service rows instead of expanding a dense form inside the record.
- Use illustrations only in the workspace introduction; recurring objects use semantic icons.

## Working language

- ongoing compliance, not “continuous assurance operations”;
- safeguards on general screens, control implementations in specialist detail;
- evidence incomplete, not evidence insufficiency;
- confirming outcome, not verification stage;
- cannot close yet, followed by exact missing conditions.

## Required states

- loading;
- unavailable with no invented counts;
- empty within the current bank scope;
- setup in progress;
- up to date;
- needs attention;
- evidence incomplete;
- decision needed;
- overdue;
- work in progress;
- preparing response;
- confirming outcome;
- closed/cancelled in detail and history.

## Responsive replacement

Desktop rows show counts, state and next step together. Common filters remain visible in a compact wrapping toolbar. Narrow screens stack object summary, counts/state, filter controls and expansion control; details become one column. Focused material tasks use a full-width sheet on narrow screens rather than compressing the desktop form.

## Acceptance evidence

- representative Programs and Matter types;
- empty, unavailable and long-title fixtures;
- 1440px, 1024px and 390px renders;
- keyboard expansion and visible focus;
- screen-reader label check;
- no active control without behavior;
- no count without explicit scope;
- outcome confirmation remains separate from completed action.
- combined filters remain scoped and keyset-paginated;
- filter routes restore after exact-record navigation;
- native date/calendar inputs are used for dates and deadlines;
- focused sheets trap and restore focus, close with Escape, preserve recoverable input and retain an opaque fallback behind blur.
