# Enterprise scope hierarchy switcher — decision brief

Date: 2026-10-02  
Issues: #265, #267  
Baseline: `main@16eec94bbcb03479fe208191760c7b1533a28016`

## Decision

Replace the flat legal-entity selector in the persistent application context bar with a compact hierarchy browser:

```text
Organization
├── Current legal entity
├── Other server-authorized legal entity
└── …
```

The organization root is orientation only in this tranche. It is not selectable and does not create a Group/All read scope.

## Why

The runtime already returns a typed organization root and the legal entities the verified principal may enter, while the UI flattens those children into an ordinary select. Large organizations need to see the parent/child relationship before ClearSight adds deeper Group, business-unit, function or critical-service scope.

A hierarchy browser gives the shell a stable mental model without weakening the existing authority boundary or forcing a later shell redesign.

## Authority invariants

- The browser consumes only `/api/v1/context.scope_hierarchy`.
- It does not discover or infer additional entities in the browser.
- A legal entity is selectable only when the existing server capability `scope_switch` is true and the hierarchy state is `COMPLETE`.
- Selection continues through `POST /auth/scope`; application APIs still derive legal-entity scope from the rotated verified session.
- The organization root is never sent as a scope target.
- `CURRENT_ONLY`, `TRUNCATED` and `UNAVAILABLE` contexts retain the existing current-scope presentation rather than exposing an incomplete switch surface.
- Switching continues to hard reload the current route so entity-scoped state and caches rebuild from verified context.

## Interaction

- The context bar remains compact and shows the current legal entity.
- Opening the control shows the organization as the parent and authorized legal entities as children.
- Current scope is marked explicitly.
- Jurisdiction is secondary metadata.
- Search appears only when the authorized legal-entity list is large enough to benefit from it.
- Selecting the current entity is a no-op.
- Switching failure preserves the current scope and existing recovery message.

## Responsive behavior

Desktop keeps the hierarchy trigger inline with the organization name. Existing context-bar reflow remains authoritative below tablet widths; the trigger expands to the available scope column and becomes full width at the narrow breakpoint. The popover remains bounded by the shared overlay system.

## Deferred

This tranche does not add Group aggregate reads, business/function/service nodes, cross-entity reporting, aggregate drill authorization, metric projection rules, local policy inheritance, residence/transfer controls or point-in-time Group reconstruction. Those remain in #267 and can extend the same hierarchy component once the server exposes real selectable nodes and corresponding authority.
