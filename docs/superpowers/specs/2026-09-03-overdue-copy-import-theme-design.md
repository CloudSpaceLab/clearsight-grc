# Explicit overdue work, production copy and Imports theme parity

**Status:** Approved direction  
**Date:** 2026-09-03  
**Tracker:** GitHub issue #177

## Decision brief

ClearSight will make passed action deadlines explicit, remove implementation narration from the linked-form remediation journey, and migrate the complete Imports workspace to the existing semantic theme system. The change will use shared presentation contracts and regression gates rather than page-specific colours or phrase substitutions.

The interface must let a bank user distinguish an action's lifecycle state from its deadline state, understand what an approved form will collect and what review remains, and use the Imports workflow with equivalent hierarchy and contrast in light and dark themes.

## Baseline defects

The deployed application on 3 September 2026 shows three connected failures:

- an open action with a past deadline still displays only its ordinary lifecycle state and the original due date;
- the linked-form remediation panel uses phrases such as “mapped missing information”, “exact revision” and “closure remain separate”, which describe implementation mechanics rather than the user's work;
- `document-import.css` gives the intake form a hard-coded translucent dark background, and several review surfaces use local dark RGBA values that do not adapt to light mode.

The supplied screenshots are the before-state baseline. They remain external review evidence and must not be copied into fixtures as product data.

## Options considered

### 1. Patch the visible strings and one background

This is fast but leaves other deadline consumers, linked-form states and Imports review surfaces free to regress. It is rejected.

### 2. Add shared deadline semantics, an affected-workflow copy gate and semantic Imports tokens — selected

This option preserves the existing domain and component architecture. It introduces one deterministic deadline presentation helper, rewrites the complete linked-form workflow in working language, extends the customer-copy regression with narrowly scoped semantic patterns, and removes local theme ownership from the Imports workspace.

### 3. Redesign all operational surfaces now

A whole-application redesign would widen risk and obscure the reported defects. Broader copy findings outside the affected workflows will be recorded separately rather than mixed into this change.

## Deadline-state contract

Action lifecycle state and deadline state remain separate:

- `PLANNED`, `IN_PROGRESS` and `BLOCKED` actions are overdue only when a valid stored deadline is earlier than the current instant;
- `IMPLEMENTED` and `CANCELLED` actions are terminal for deadline presentation and are never newly labelled overdue;
- an invalid or absent deadline displays **No action deadline**;
- a future deadline displays **Due 7 Sep 2026**;
- a past open deadline displays a visible error-tone **Overdue** badge and **Due 22 Aug 2026 · 12 days overdue**;
- due-date text uses a semantic `<time>` element and remains understandable without colour.

The original deadline remains visible. The interface does not mutate the Action status to an `OVERDUE` domain value, because a passed deadline is a derived attention condition, not a lifecycle transition.

Calendar-day wording is calculated from local day boundaries for the rendered user. Tests set the system time explicitly so results do not depend on the machine clock.

## Customer-copy contract

The linked-form remediation journey will describe business work and consequences:

- section purpose: send an approved form for the outstanding information and review the response before updating the issue;
- empty state: name the issue population checked and the next actor who can send a form;
- request summary: state how many outstanding items the form covers without exposing “mappings”;
- score failure: state that the response score is below the approved threshold, that the issue remains open, and whether to review or request correction;
- review basis: ask why the response answers the outstanding items;
- setup sheet: explain that the user chooses which answer supplies each outstanding item and that those choices are fixed after sending;
- successful application: state that the information was added and that the result must still be confirmed before the issue can close.

The audit covers every customer-visible string in the affected Program, Work and Forms workflow, including headings, helper text, status, validation, empty, unavailable, conflict, success and recovery states. Internal terms may remain in TypeScript identifiers, API contracts, tests and audit-specialist detail when they are not rendered.

`copyQuality.test.ts` will add narrowly scoped patterns for implementation narration discovered in this workflow. The patterns will not ban useful business terms such as approved form revision, authoritative source or outcome check when the screen genuinely needs them.

## Imports theme contract

The Imports workspace will consume the semantic roles already defined in `DESIGN.md` and `ui-preferences.css`:

- intake form and operational cards use `--surface`, `--surface-2`, `--surface-3`, `--border`, `--text`, `--muted` and semantic status tones;
- the file-primary dropzone preserves its large interaction area, selected filename, size and explicit import action;
- inputs, selects, buttons, focus rings, disabled states and notices retain visible boundaries in both themes;
- review, limitation, score and recommendation surfaces use semantic colour mixing rather than dark RGB literals;
- the paper-like imported-document inspector may keep its deliberately scoped document tokens, because those tokens already define separate light and dark values;
- mobile replaces the parallel intake/list composition with one column without horizontal overflow.

No new theme, density or motion mode is introduced. If a new semantic component role is unavoidable, it must be added to `DESIGN.md` and the component gallery in the same change.

## Error and degraded states

- Invalid deadlines do not create an overdue claim.
- A missing deadline stays explicit and does not imply that work is current.
- Copy-audit failures name the source file and matched customer-visible phrase.
- Imports loading, empty and unavailable states retain their existing recovery actions.
- A file validation error preserves the purpose and document-type inputs while asking for a supported file.
- Theme preference changes do not reset the selected document or intake entries.

## Test and rendered-proof contract

Test-first implementation must cover:

1. overdue, due-today/future, invalid/missing, implemented and cancelled action deadlines;
2. visible and accessible overdue wording independent of colour;
3. every linked-form state: empty, loading, form setup, sent, score below threshold, ready for review, applied, conflict and unavailable;
4. copy-quality failures for the new implementation-narration patterns and passing business-language alternatives;
5. Imports intake, selected file, validation error, loading, empty, review and unavailable states;
6. light, dark, desktop, mobile/reflow, long-copy and keyboard-focus renders;
7. axe, strict TypeScript, production build and affected component suites.

Rendered review follows the repository order: correct object and action, accurate state and deadline, recovery, hierarchy, responsive replacement, accessibility and contrast. The highest-impact defect found during review is fixed and the affected render is captured again before completion.

## Out of scope

This change does not alter Action lifecycle transitions, timer/escalation policy, Matter closure rules, form scoring, response application authority, document extraction, or import durability. It does not replace the separate comprehensive product-copy programme; equivalent violations found outside the affected workflows are added to the remote tracker with source locations and acceptance criteria.
