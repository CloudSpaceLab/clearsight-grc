# ClearSight — People & Access (design 17)

## Screens
- People: scoped directory, position, sign-in state, assigned roles and current-person inspection
- Roles: explicit capabilities and scoped role membership; role membership is not final approval authority
- Approval routes: decision-specific stages, effective role resolution, independent review and explicit routing gap
- Delegations: maker, delegate, effective dates, scope, conflict boundary and draft proposal
- Modal states: invitation draft, decision route simulation, delegation proposal, compact global search

## Interaction and data constraints
- `All groups` filters the illustrative directory, route and delegation populations. Tenant switch is labelled as preview-only; no cross-tenant access is simulated.
- Ctrl/⌘ K searches people, roles and routing records; ArrowUp/ArrowDown/Enter/Escape are supported.
- Invitation draft adds an `Invited` row, **does not** send a message, create a principal or assign effective access.
- Delegation proposal creates a `Draft` record with no immediate authority. It rejects self-delegation.
- Approval route simulation distinguishes a configured route from missing authority; all outputs remain illustrative. A real route must be checked at execution under verified identity, tenant and entity scope, effective delegation, conflicts and approved policy version.
- Roles and approvals remain separate to prevent privilege escalation in the product architecture.
- No impersonation control is included without a properly governed break-glass support workflow.
- The interface is a static local prototype. The screenshots do not certify backend capability or production access semantics.

## Visual acceptance
- Single primary action `Invite person`; roles, routing and delegation each retain their own bounded task.
- One selected-record detail view, no permanent Copilot or decorative chart.
- Clear focus, selection and status markers. Reduced motion disables transitions. Keyboard shortcuts remain optional.
- Browser proof: desktop 1880 CSS pixels at device scale 2; responsive checks at 1280, 1024, 768, 390 CSS pixels with no horizontal overflow and no page JS errors.
- Prototypes use only fictional `example.invalid` email addresses. No actual accounts or permissions are read or modified.
