# Form scoring and response-policy workspace decision brief

## Decision

Add advanced scoring to the existing form inspector and add a peer **Policies** workspace to Forms. Both surfaces use the shared ClearSight controls and semantic tokens. Scoring rules are composed from bounded fields, operators and effects; policy commands remain server-owned and expose one dominant next action for the current lifecycle state.

## Baseline and problem

The current builder persists only per-choice weights. It cannot express cross-field conditions, floors, caps or disqualification without editing an API payload. Forms also has no customer-facing workspace for the governed response-policy lifecycle that can create a Matter after a poor completed response. Raw controls in the inspector produce inconsistent focus, theme and overlay behaviour.

## Interaction contract

- The overview inspector discloses advanced scoring only after Risk or Compliance scoring is selected.
- A contribution names one business condition and its matched/non-matched points. Optional advanced rules add a contribution, floor, cap or disqualification.
- The UI never accepts scripts or JSON. Limits and supported operators mirror the stored form contract.
- Preview answers are sent to the exact stored template ID and revision. Draft-only scoring explains that the form must first be saved; the browser never claims an authoritative score calculated locally.
- Policies list stored records only. Draft creation requires purpose, exact form revision, eligibility, blast radius, outcome check and Matter handling.
- The dominant policy action is: simulate for Draft, submit after a current simulation, approve for Pending approval, activate for Approved, suspend for Active, and roll back for Suspended. Server errors explain authority or freshness failures without weakening the route.

## Required states

| Surface | Required states |
| --- | --- |
| Advanced scoring | no scoring, first contribution, configured rules, invalid bounds, saved-revision preview, draft preview unavailable, preview loading/error/result |
| Policy list | loading, sign-in required, unavailable, empty, populated, refresh failure with retained rows |
| Policy editor | draft definition, simulation impact, stale simulation, pending independent approval, approved, active, suspended, command failure |

## Responsive and accessibility behaviour

- At desktop widths the existing outline, canvas and inspector remain independently bounded; scoring content scrolls only inside the inspector.
- At narrow widths the inspector remains a focused sheet and all rule rows stack in source order.
- Select menus use the shared fixed overlay contract and do not resize or scroll the document.
- Controls retain visible labels, at least 44px targets, keyboard operation, programmatic errors and theme contrast in light and dark modes.
- Reduced-motion users receive no non-essential transitions.

## Verification evidence

Component tests cover round-trip serialization, bounded rule editing, exact-revision preview requests and lifecycle action selection. UI-contract checks cover shared controls. Rendered light/dark desktop and narrow states are captured in the implementation evidence before completion.
