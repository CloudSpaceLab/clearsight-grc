# V2.13 — responsibility, queue and governance presentation

## Scope

This refinement covers thirteen existing routes through two new assets: `premium-workspaces.js` and `premium-workspaces.css`. Load them after the V2.12 modules and the V2.13 shared foundation. The module registers presentation wrappers without rendering the application at startup.

| Area | Routes | Refinement |
| --- | --- | --- |
| Personal responsibilities | `my-workspace`, `my-access` | Separate working identity, actual scoped role assignments, responsibility lens, accountable queue and secondary workspaces. Capability cells retain their exact decision links. |
| Work | `work`, `work-item` | Group queue choices and filters; keep source, accountable owner, deadline, state and next action easy to scan. Selected queue links expose their current state to assistive technology. |
| Inbox | `inbox`, `notification`, `notification-preferences` | Give the source action its own correctly sized column, retain separate personal receipt controls, and make reading, source state and deadline distinct. Notification articles have unique accessible names. |
| Access governance | `roles`, `role-detail`, `role-change-review`, `access-review` | Quiet the existing summary figures, improve role card hierarchy, preserve readable permission matrices, and organize effective-access comparisons and assignment reviews. |
| Escalation | `escalation-matrix`, `escalation-policy` | Keep policy timing, scope, stop condition and fallback prominent; improve level and simulation scanning without adding illustrative metrics. |

## Concrete interaction fixes

The previous Inbox row reserved a 28px final track for an action block requiring at least 130px. The new track reserves 156px on a comfortable desktop and 148px in compact density. The source button uses that width and permits wrapping. Read and archive remain distinct controls underneath. At 1023px and below, the action block follows the notification text instead of competing with it for a narrow column.

Queue tables are associated with their actual panel heading. Empty action columns receive an accessible “Next action” heading. The decorator stops at the owning panel: an empty personal queue cannot accidentally label a later program table. The handwritten escalation policy table receives its existing column labels as `data-label` attributes, so the shared phone card presentation retains scope, levels and clock labels.

Notification articles use unique title references and labelled action groups. Their source-state attribute comes from the existing source resolver. A notification with a read or acknowledgement receipt continues to display an unresolved source as requiring attention. The original deadline, priority and status text remains visible.

## Layout and display behavior

The stylesheet uses the foundation's semantic colors, spacing and corner tokens. Main content surfaces remain solid in light and dark appearance. Role and policy views use their existing data; the module adds no counts, charts, filters or synthetic results.

Compact spacing applies only at desktop widths with a fine pointer. Phone and coarse-pointer controls retain 44px targets. Phone queue actions span their card; policy levels and action cells release desktop minimum widths; capability columns stop being sticky in the card layout. Permission matrices retain horizontal scrolling. Inbox navigation enters normal flow in short viewports, preventing a tall sticky rail from becoming inaccessible.

## Model and integration boundaries

There is no new state store, action handler, permission rule, identity switch, filter model or domain command. The thirteen route wrappers call their existing renderer once. The notification wrapper calls the existing row renderer once and adds labels and source-state metadata. Existing links, action identifiers, recipient checks, source resolution, independent review rules and form controls are retained.

`CLEAR_SIGHT_PREMIUM_WORKSPACES.version` is `2.13`; its read-only `routes` array reports installed route wrappers. Loading the asset again is inert. The wrapper adds a normal container, preserving the shell's single main landmark.

## Verification

Nine checks against the integrated application source passed in `tests/premium_workspaces_v213_checks.js`; results are retained in `tests/premium-workspaces-v213-results.json`:

1. All thirteen routes retain their source links, action targets and registered commands; escalation cards retain their exact existing column labels.
2. Work queue navigation preserves owner, query and work-type filters while identifying the selected queue.
3. Labelled queue tables retain their exact original rows.
4. An empty personal queue cannot label a later owned-program table.
5. Notification article names are unique and escape source titles, including quotes, markup and replacement-string characters.
6. Reading or acknowledging a notification does not present its open source as resolved.
7. A notification source action still opens its exact task and rejects another recipient.
8. Presentation leaves saved access policy, escalation policy and source records unchanged.
9. Repeated module loading is inert and preserves one main landmark.

The JavaScript syntax check passed. These checks use the actual application functions with the existing DOM shell harness. Browser layout and assistive-technology behavior were not exercised; the responsive and touch rules were inspected against the shared foundation and existing mobile table structure.
