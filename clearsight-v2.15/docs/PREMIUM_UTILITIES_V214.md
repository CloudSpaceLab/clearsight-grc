# Shared utility refinement — V2.14

21 November 2026 · Actual implementation reference

## Scope

The utility layer completes search, mobile navigation, contextual Copilot, validation, empty states, settings and document/approval catalogs. It consumes the V2.13 semantic tokens and existing controls. It creates no new domain store, grants, notification stream or remote dependency.

The stylesheet is `assets/premium-utilities.css`. The small interaction adapter is `assets/premium-utilities.js`, loaded after the workflow-specific premium layers and before bootstrap.

## Search and unsaved work

Search keeps the existing role-aware catalog, result ranking, source routes and native link behavior. Results have a consistent icon, readable name, record type/identifier and keyboard-selected state. Long labels wrap. The result area scrolls within the existing dialog and retains the current organizational context.

The prior search shortcut could call `openDialog` while an edited review dialog was still open. It now uses the same existing unsaved-change decision as other modal navigation:

1. If a save is pending, search waits and the current operation remains intact.
2. If the current dialog is dirty, **Keep editing** retains it. **Discard changes** uses the existing discard command before opening search.
3. If no pending changes exist, search opens normally.

This guard does not rewrite the review callback or submit anything. Opening search from an ordinary page leaves that page's form, draft and filters in place.

A separate polite status region announces the existing result count and the selected result's name/context/position. It uses `textContent`. The input remains a search input, and results remain links; the adapter does not introduce an artificial listbox/combobox role. The original Arrow/Enter navigation and role filtering remain authoritative.

## Mobile navigation lifecycle

At phone widths, the existing sidebar becomes a temporary modal navigation surface with a visible close button. Desktop navigation retains its original semantics.

| Event | Behaviour |
| --- | --- |
| Open | Mark the opener expanded, expose the navigation dialog, temporarily make the main page/header inert, lock background scrolling and focus the current destination. |
| Tab / Shift+Tab | Contain keyboard focus within available navigation controls; wrapped/offscreen destinations receive normal focus scrolling. |
| Escape or close | Restore the previous inert/role/tabindex values, remove the scroll lock and return focus to the opener. |
| Destination link | Release the drawer and preserve the original destination; the canonical route handles navigation. |
| Switch to desktop | Release modal semantics and inert state without focusing a now-hidden phone opener. |
| Ctrl/⌘K | Restore the opener before search opens, giving the search dialog a visible focus-return target. |
| Existing dialog open | Leave that dialog intact; a second navigation surface does not replace it. |
| Shell replaced | Release the old drawer state before the new shell is installed. |

The only drawer state is an ephemeral reference to affected DOM nodes and their previous attributes. It never writes `S`, changes the acting person or widens available navigation. The resize listener only closes an active mobile drawer when needed; there is no polling or observer.

## Consistent secondary surfaces

Validation summaries use readable text, comfortable correction links and a clear relationship to the field/action. Unsaved-change and next-action panels share the surface, spacing and status language used by the workspaces. Empty states retain their truthful source wording and available actions; they are not populated with fictional records.

The optional Copilot panel retains its explicit preview label, exact context, source references and guarded proposal action. The update separates the answer, suggested tasks, question field and supporting sources visually. It does not replace source facts with generated content or automatically apply a proposal. The panel is omitted from printed reports; the canonical report remains the export source.

## Settings and document catalogs

Settings moves the existing nodes into a useful reading order: page title, tabs, display preferences, administrative shortcuts and the remaining controls/content. It moves nodes rather than reconstructing buttons or fields. Architecture/history tabs retain their source content. The reset action retains its existing confirmation and receives a distinct destructive treatment.

Documents combines a calmer summary, labelled responsibility/lifecycle filters, a coherent folder navigator and a responsive results toolbar. Active folder buttons expose `aria-pressed`. The selected-document action bar has a group name. File controls, folder IDs, selected records, approvals, review-history references and permission-disabled actions are unchanged.

The approval catalog receives the same identity, table and empty-state treatment. File approval and assessment review remain distinct decisions. The legacy `evidence` route retains its canonical Documents history view and backing identifiers.

## Responsive and theme decisions

The same light/dark palette, spacing, corners and control primitives are reused. Content and dialog headers are opaque. Fine-pointer desktop density remains optional; touch editing retains 16 px input text and comfortable controls. Correction links become actual inline-flex targets on coarse-pointer tablets as well as phones.

Document folders wrap into a compact group on narrow screens; record filters wrap without squeezing select values. Copilot sources move below the answer. Long search-result names remain visible. Short-view dialog rules preserve access to content without forcing a fixed desktop reading height.

## Validation

`tests/premium_utilities_v214_checks.js` contains 11 actual-source interaction checks using controlled DOM nodes. They cover drawer state/focus intent and restoration, original link destinations, desktop cleanup, existing-dialog protection, drawer-to-search focus handoff, unsaved/busy search guards, unchanged role-scoped results/state, plain-text announcements, settings node order and document selection semantics.

These checks establish the implemented state transitions. Native browser focus, `inert`, screen-reader announcements, visual layout, touch and printing still require acceptance in supported browsers. No browser workaround was attempted after the existing execution restriction.
