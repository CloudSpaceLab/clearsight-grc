# Shared navigation and modal continuity — V2.15

21 November 2026 · Actual-source and controlled-DOM evidence

## Focus intent

Shared helpers in `interaction-core.js` identify an existing control using its element type, stable ID/name/link/data attributes, containing form and UI area. The reference is bound to the current route, organizational scope and acting person. Resolution requires a unique, visible, enabled, connected control in the same context; otherwise closing a modal focuses the page heading.

The original connected opener still uses the existing return behavior. The fallback also handles CSS-hidden openers, including a phone menu opener hidden by desktop resize while search is open. Continuation dialogs retain the original workspace return reference rather than a removed modal child.

The final utility render adapter restores a unique inline field after the existing render replaces it. On route changes it focuses the new page without changing remembered scroll. Existing correction targets retain priority. A sequence/context guard discards stale work, and focus deliberately placed by a workflow or user is respected. No new view-memory store is added.

## Save and close lifecycle

Escape now follows the same modal close path as the existing close controls. Dirty forms keep the current Keep editing/Discard choice; busy forms stay open. Successful closure clears shared modal state and restores focus.

Hash navigation is intercepted while a modal callback is busy or its form is dirty. During a save the original URL/context is restored and no delayed destination is queued. A dirty form follows its target only after explicit discard. Back to an initial no-hash entry is supported through the entry page's default route.

## Search and correction navigation

Arrow Up starts at the last result and Arrow Down at the first when no result is selected. Selection and announcements ignore IME composition events. Existing scoped catalogs, ranks, source links and Enter behavior remain authoritative.

Correction focus tracks all route transitions, allowing the same correction URL to be revisited. A callback queued for a route already left does not focus its old target.

## Evidence and limits

`tests/navigation_continuity_v215_checks.js` contains **10 checks** for rebuilt/removed/ambiguous/hidden controls, acting-context separation, continuation and Escape cleanup, asynchronous-save navigation including no hash, dirty return to the default entry, search selection/composition, route and inline render focus, stale callback suppression and correction re-entry.

The tests execute the actual source handlers with controlled DOM objects. They do not certify native browser focus, history traversal, keyboard/IME integration or screen-reader behavior. Those remain acceptance tasks under the existing execution restriction.
