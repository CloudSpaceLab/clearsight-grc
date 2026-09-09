# Forms editing and actions

User request: CRO cannot easily edit an existing form; table actions and detail copy are unclear. Continues the authorized Forms usability corrections and main/demo release work.

## Decision
Use direct Edit draft / Edit form actions on library rows, with Details secondary. The editor saves immutable draft revisions and never changes a published version in place. Details shows the same edit action near the heading; review and lifecycle actions remain separately authorized. Keep the existing table and sheet, with wrapping actions on narrow cards. Shorten latest/published version and scoring labels; no available published version is labelled Not available rather than implying it was never published. Permission failures identify the editor or recovery action instead of leaving a blank action area.

The demo currently routes all accountable-owner commands to Program Owner. Add a narrow, versioned, demo-managed Forms author policy through the existing ROLE routing mechanism so CRO and Program Owner can create and revise Forms and perform author lifecycle actions (submit, pause, resume and retire). Retain independent reviewer routing and maker/checker enforcement. No browser role bypass, production default permission expansion, global owner reassignment or modified historical form records. Organization administrators retain normal governed policy configuration.

The table uses its container width for the existing 700px card replacement. Desktop actions fit inline; the narrow editor removes redundant padding and stacks field type and Required controls. Direct entry focuses and scrolls to the editor; Back restores the initiating row action.

Governed ROLE policies accept multiple eligible holders as candidate sets, matching execution. Both execution paths enforce tenant/entity scope, binding scope, active state and validity windows. Direct selectors remain single-holder routes. Real PostgreSQL tests cover independent approval and mixed eligible/excluded role holders.

## Alternatives
A details-only edit keeps the extra click and discovery problem. A CRO bypass in component/API logic breaks configurable authority. Direct server-authorized actions plus a narrow demo author route resolves both problems.

## Proof
Before-state: user screenshot and existing deterministic library/detail fixtures. Required after states: editable draft, published version with draft revision, pending approval, denied author, unavailable authority, busy action; light/dark desktop and320/390. Verify direct edit/save uses expected version, pending approvals cannot be revised, reviewer remains distinct, and existing role permissions outside Forms do not expand.
