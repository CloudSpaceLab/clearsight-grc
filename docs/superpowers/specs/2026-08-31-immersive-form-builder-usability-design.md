# Immersive Form Builder Usability Design

**Date:** 2026-08-31

**Status:** Superseded by [`2026-08-31-clearsight-ui-foundations-and-forms-migration-design.md`](2026-08-31-clearsight-ui-foundations-and-forms-migration-design.md)

**Scope:** Governed Forms template authoring in the internal bank workspace

> This document preserves the builder-specific audit and interaction decisions. Implementation follows the broader UI-foundation and Forms-migration specification, which incorporates these requirements into shared product components.

## 1. Decision summary

Replace the current form builder embedded inside the full Forms workspace with a focused authoring shell. The shell preserves legal-entity context, authority-gated actions, revision state, Preview and Review, while removing the surrounding Forms hero, tabs and page gutters during editing.

At wide desktop sizes the authoring shell presents Outline, Canvas and Question settings together only when each pane has enough usable width. At narrower widths, the Canvas remains primary and Outline or Settings opens in an accessible sheet. The current native response-type select is replaced with a categorized, keyboard-operable picker that does not cover the working document.

This is a presentation and interaction redesign. It does not change form schemas, revisions, scoring, activation, authority, approval routing, response semantics, invitation boundaries or audit history.

## 2. Product job and users

The primary user is an authorized bank form author or reviewer preparing a governed request for approval. Their job is to create or revise the smallest useful set of questions, verify the resulting respondent experience, resolve detected form defects and submit an exact draft revision for maker-checker approval.

The first useful outcome is a valid saved draft with an understandable question structure. The repeated-use outcome is a revised form that can be reviewed and sent for approval without losing context or reopening multiple module pages.

The dominant action for a valid draft is **Send for approval**. **Save draft** is the secondary action. Preview, Review, Outline and Question settings remain readily available but do not compete visually with the approval action.

## 3. Before-state baseline

The deployed editor was inspected at 1440×900, 1280×720, 1024×768 and 390×844, together with the retained light, dark, mobile and zoom-proxy evidence.

Observed structural failures:

- At 1440×900, the editor grid has about 1,120 px for all three panes: approximately 231 px for Outline, 562 px for the Canvas column, and 326 px for Settings. The editable form document inside the Canvas is only about 454 px wide.
- At 1280×720, the form document is about 358 px wide, while the page and Settings pane both scroll.
- At 1024×768, the surrounding Forms heading and navigation consume enough height that the Canvas begins around 494 px from the top of the viewport.
- At 390×844, the Forms navigation overflows horizontally, the workspace and title field overflow their available width, the Canvas begins around 543 px from the top, and the application bottom navigation overlaps editor content.
- The 19-option native response-type select can cover most of the Canvas and parts of adjacent panes.
- Important labels and editable values clip instead of wrapping, and many desktop controls render below the 44×44 px pointer-target floor.
- The retained mobile and dark reflow images expose menu overlap, clipped question content and navigation collision even though the automated checks pass.
- The current evidence route can isolate the builder from its real host layout, so presence assertions and static fixtures do not detect the largest deployed usability failures.

The attached user screenshot is retained as part of the before-state record. Existing evidence under `web/ui-evidence/` remains the repository baseline; implementation must add full-host captures rather than overwrite the evidence of the old layout.

## 4. Alternatives considered

### 4.1 CSS-only cleanup

Reduce font compression, widen the center column and patch mobile overflow without changing the containing Forms workspace.

This is rejected because the global heading, tabs and application navigation still consume most of the usable viewport. Widening one pane also makes another pane unusable, and a styled native select still cannot provide a dependable bounded 19-option experience across browsers.

### 4.2 Step-by-step form-authoring wizard

Turn authoring into a fixed sequence for details, questions, rules, preview and approval.

This is rejected for the current change because experienced authors need rapid movement among sections and questions, and a fixed wizard would disrupt existing revision and review behavior. A respondent-facing Wizard remains separate from the authoring interaction.

### 4.3 Focused authoring shell — selected

Give the editor the available application viewport, retain a compact context/command bar, keep the document as the primary surface, and disclose structure or advanced settings according to available width.

This resolves the measured host-layout problem, supports progressive disclosure, preserves the current governed workflow and creates a stable responsive model instead of shrinking a desktop composition.

## 5. Information architecture and layout

### 5.1 Entering and leaving authoring

Selecting a form revision opens the focused authoring shell in the existing Forms route. The shell replaces the Forms hero, workspace tabs and redundant page gutters while active.

The top context bar contains:

- **Back to Forms**, which returns to the prior Forms view and safely preserves its filters;
- the editable form name and current revision/draft state;
- the current legal entity or bank scope when material;
- Preview and Review as tertiary modes;
- Save draft as a secondary action; and
- Send for approval as the single emphasized action when authorized and valid.

The context bar remains visible while the document scrolls. It must wrap into a compact two-row layout before any control clips or overlaps. Controls that are unavailable state the exact reason, such as unresolved Review findings or insufficient authority.

If a user attempts to leave after local changes have not reached the server, the editor provides a specific save/discard/cancel choice. It must not warn when the server has already confirmed the current draft. Browser-level departure uses the platform warning only while confirmed unsaved changes exist.

### 5.2 Desktop: three-pane authoring

The three-pane composition is allowed only when the builder's actual content width can provide, at minimum:

- 240 px for Outline;
- 640 px for the Canvas; and
- 320 px for Question settings;
- plus the established gaps and dividers.

The layout decision uses the builder container, not only the browser width. It must therefore remain correct with application navigation, zoom and operating-system scaling.

The Canvas is the primary scroll region. The page body does not scroll behind it. Outline and Question settings remain stable during ordinary editing; if their exceptional content exceeds the viewport, that pane may scroll within a labelled region with its own visible header. Collapsed settings groups keep ordinary question settings within the available height and avoid routine nested scrolling.

The Canvas uses the available width up to a readable form-document maximum. Empty width must expand the working document before increasing decorative gutters.

### 5.3 Compact desktop and tablet

When the three minimum pane widths do not fit, only the Canvas remains inline. Outline and Question settings are opened by labelled toolbar controls in focused sheets.

Each sheet:

- has a visible title and close control;
- traps focus only while modal;
- returns focus to the control that opened it;
- closes with Escape when no destructive choice is pending;
- owns its scroll while open; and
- leaves Preview, Review and the primary save/approval actions reachable.

No intermediate layout may show compressed versions of all three panes.

### 5.4 Mobile and 200% zoom

On narrow/reflowed layouts, authoring becomes a document-first view with a compact context bar and full-width Canvas. Outline, Question settings, Preview and Review use full-height sheets or dedicated modes. The Forms tab row and application bottom navigation are suppressed while the focused editor is open; **Back to Forms** remains the explicit route out.

Mobile requirements:

- no horizontal page or workspace overflow;
- no fixed control covers editable content, validation or the final question;
- at least 44 px is reserved for every interactive pointer target;
- question names, guidance and status text wrap without losing their full accessible name;
- the software keyboard does not hide the active field or its validation; and
- safe-area padding is applied where supported.

The same replacement behavior applies at browser 200% zoom when the effective builder width no longer supports the desktop composition.

## 6. Canvas and question editing

Each question card presents the information needed for authoring in this order:

1. drag/reorder control with an explicit accessible name and keyboard alternative;
2. editable question text;
3. optional respondent guidance;
4. a compact response-type button showing the current business label;
5. a 44 px required-response control; and
6. one 44 px actions menu for duplicate, move and delete actions.

Advanced validation, accepted-file rules, choices, conditional display and other response options remain in Question settings. The card does not duplicate the full inspector. Existing conditional-order protection, duplicate behavior, Review fixes and revision semantics remain unchanged.

Selected-question state must be conveyed by more than color. Moving focus or selection from Outline to the Canvas scrolls the exact question into view without stealing keyboard focus from an active text edit.

Section and question insertion controls use direct labels: **Add section** and **Add question**. Deletion names the affected object and explains whether dependent display rules prevent the action.

## 7. Response-type picker

Replace the native select with one accessible picker. It is a bounded popover when it fits beside the response-type button and a focused sheet on compact or mobile layouts.

The picker groups the 19 response types:

- **Text and contact:** Short text, Long text, Email address, Telephone number, Web address.
- **Numbers and dates:** Whole number, Decimal number, Percentage, Currency amount, Date.
- **Choice and confirmation:** Yes or no, Select one, Select several, Checkbox, Attestation.
- **Evidence and sign-off:** File, Photo, Signature, Vendor document.

Each option shows its business label and a short explanation of what the respondent will provide. The current type is marked visually and announced. Arrow keys move within a group, Home/End move to bounds, Enter or Space selects, Escape cancels, and focus returns to the response-type button. Typing may use typeahead but a search field is not required for nineteen categorized options.

Changing a response type must preview any destructive effect on existing type-specific settings and require confirmation before those settings are removed. Cancelling leaves the question unchanged.

## 8. Review, validation and command states

Review remains a distinct mode that lists errors and warnings against the current material draft version. Each actionable finding names the question or section and offers one direct **Fix** action that returns to the exact field. A stale Review result is labelled with the version it evaluated and cannot enable approval for a newer draft.

The authoring shell must represent these states without replacing the Canvas with a blank page:

- initial loading and recoverable load failure;
- new empty draft;
- populated draft;
- saving, saved and save failed;
- server conflict with compare/reload recovery;
- no edit authority;
- Review with no findings, warnings, and blocking errors;
- approval submission in progress, accepted and rejected;
- long content, long translated labels and the maximum supported question count;
- source or integration degradation where a controlled option cannot be resolved; and
- offline or interrupted editing where supported by the current draft contract.

Success text states the exact saved revision or approval result. A submitted draft is not described as active until checker approval has activated that revision.

## 9. Copy, visual system and motion

All visible copy names the form, section, question, finding, saved revision or required action. Internal status codes and identifiers remain out of primary labels. Helper text explains what the respondent will provide or why an action is unavailable; it does not narrate the interface.

The redesign reuses the existing semantic colors, spacing, typography and surface tokens. No new density mode, decorative illustration style or motion system is introduced. Working text is at least 12 px, with 14 px preferred for editable content and labels. Contrast, focus rings, status icons and error text meet the existing ClearSight design contract in light and dark themes.

Motion is limited to sheet/picker entry, selection indication and scroll-to-question orientation. Reduced-motion mode removes translation and nonessential smooth scrolling while retaining visible state change.

## 10. Security and governance boundaries

- The server-verified actor, tenant, legal entity and current authority remain the source of all material commands.
- Client layout state, selected question and request body identifiers do not grant edit or approval authority.
- Save draft and Send for approval continue through their canonical versioned commands and audit path.
- The redesign does not auto-activate a draft, infer an approver, weaken segregation of duties or convert Review success into approval.
- Preview contains only data the current actor may view and must not place invitation tokens or protected recipient information in URLs, logs or analytics.
- Form content remains usable without AI or a live integration.

## 11. Implementation boundaries

### Included

- focused authoring mode in the Forms workspace;
- responsive layout replacement and removal of editor-time navigation collisions;
- accessible Outline and Question settings sheets;
- categorized response-type picker;
- question-card target sizing, wrapping and hierarchy improvements;
- explicit unsaved/saving/saved/conflict presentation using the existing draft contract;
- updated production-component fixtures, tests and full-host rendered evidence; and
- `DESIGN.md` updates only if implementation introduces a new reusable component variant or token.

### Excluded

- form schema, response or scoring changes;
- new question types;
- activation, routing or authority changes;
- respondent Classic/Wizard redesign;
- a second form-builder or draft persistence stack;
- AI-generated forms;
- broad Forms library redesign outside the transition into and out of focused authoring; and
- deployment or provider configuration unrelated to presenting the editor.

## 12. Verification and acceptance evidence

Implementation is not visually complete until fresh proof on the exact final commit shows:

### 12.1 Behavioral tests

- an authorized author can edit, reorder, duplicate, save, Preview, run Review and send a valid draft for approval;
- unauthorized and stale-version commands fail without optimistic UI claiming success;
- the response-type picker supports pointer, keyboard, cancellation, focus return and destructive-change confirmation;
- Outline and Question settings sheets trap and restore focus correctly;
- unsaved-change warnings appear only for unconfirmed local changes;
- long text, translated labels and large forms retain an operable next action; and
- existing form-builder, copy-quality, accessibility and authority tests remain green.

### 12.2 Automated layout assertions

- there is no horizontal overflow at 390, 768, 1024, 1280 and 1440 CSS-pixel viewports;
- no visible enabled editor target is smaller than 44×44 px at mobile/reflow sizes;
- the three-pane layout never renders when its minimum widths are unavailable;
- a fixed or sticky control does not overlap the active field, validation, last question or sheet footer;
- only the intended scroll container moves for the tested interaction; and
- full accessible names remain available when visible text wraps.

### 12.3 Rendered review

Preserve before-state evidence and capture the production component inside the full application host for:

- light and dark 1440×900 desktop authoring;
- 1280×720 constrained-height desktop;
- 1024×768 compact layout with Outline and Settings sheets;
- 390×844 mobile with the on-screen keyboard risk represented;
- real browser 200% zoom/reflow, not device-scale emulation;
- response-type picker open at desktop and mobile;
- long names, long guidance and maximum representative settings;
- loading, save failure, conflict, read-only and blocking Review states; and
- reduced motion and visible keyboard focus.

For each failed render, fix the highest-impact defect and recapture that exact state. Evidence setup must assert the dominant action and intended state before capture. Static fixture images cannot substitute for keyboard-only, screen-reader and browser-zoom checks.

### 12.4 Usability check

With a representative authorized bank author:

- first use can add a section, create two questions with different response types, save and find the next approval action without instruction;
- repeat use can locate and correct one Review finding within two minutes;
- the user does not leave the authoring workspace for a routine edit/review/save sequence;
- returning after interruption makes saved state and the next action understandable within 30 seconds; and
- keyboard and assistive-technology paths require no materially longer sequence.

## 13. Exit criteria

The redesign is ready to merge only when the implementation and affected documentation agree, all relevant tests pass on the exact final head, the complete host-level render set has been inspected and rechecked after its highest-impact correction, and the deployed acceptance environment reports that same commit before hosted usability is treated as evidence.

Repository completion must not be described as proof that every browser, assistive technology or representative bank user has accepted the workflow. Those human and hosted checks remain explicit release evidence.
