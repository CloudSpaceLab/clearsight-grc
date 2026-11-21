# UI/UX continuity audit — V2.15

21 November 2026 · Current source implementation review

## Gaps corrected

| Observed gap | Implemented response |
| --- | --- |
| Document folder/filter changes could be overwritten by stale URL values during render | Persist the selected filter route before the canonical render and restore omitted parameters to defaults. |
| Document title links dropped version and list context | Title links share the exact-version and return-context route used by Preview. |
| Version comparison ignored changes after its first 500 lines | Compare the complete bounded reader output, then render a bounded excerpt around the changed region. |
| Office truncation could be mistaken for complete equality | Expose shown/total Word paragraphs and disclose omitted content; matching text is described as matching preview text only. |
| Filtered-out notifications were described as up to date | Active filter summary, mailbox-specific empty states and concise matching-result status. |
| Partial select-all looked unselected and empty select-all stayed available | Native indeterminate state, accurate checked/disabled status and visible-population selection pruning. |
| Opening a work item dropped queue context | Bounded owner/lane/type/search return parameters; source lookup and the current global location remain authoritative. |
| Reports showed edited inputs alongside results from previously applied filters without distinction | Applied/pending status, Apply/Revert, guarded output actions and scope-preserving reset. |
| Response report recovery reattached invalid filters | Clear those filters after the shared context is applied, retaining organization and report clocks. |
| Rebuilt pages and dialogs lost their logical keyboard return point | Resolve the original control by stable identity and context; use the page heading when unavailable or ambiguous. |
| Native Escape left shared modal state behind | Route cancellation through the shared dirty/busy close lifecycle. |
| Back navigation during a save could rebuild the source page | Restore the original route while saving, including when Back targets the initial no-hash entry. |
| Search Arrow Up skipped the last result; IME candidate navigation triggered search behavior | Correct initial selection and ignore composing key events for both selection and announcements. |
| A correction deep link could fail when revisited | Track each actual route transition and discard stale queued focus work. |

## Design and architecture

The established V2.13/V2.14 palette, hierarchy, density and responsive surfaces remain shared across all 79 registered routes. New presentation is limited to filter summaries, truthful result/status copy and existing control states. Current modules were edited directly; no runtime module, framework, remote resource, polling loop, observer or second business store was added.

## Acceptance boundary

Current evidence is actual-source execution with controlled browser/DOM shells. Native layout, event ordering, Back history, focus visibility, screen-reader announcements, touch, download and print behavior require browser acceptance. No retry or workaround of the earlier browser restriction was attempted.
