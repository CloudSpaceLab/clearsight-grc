# UI delivery workflow

This workflow turns ClearSight's visual principles into reviewable evidence without adding a design-tool runtime dependency.

## 1. Compact decision brief

Before significant UI code, record:

- product job and primary user;
- primary object and action;
- first useful outcome and repeated-use task;
- structural pattern and why it fits;
- required states and recovery;
- desktop, tablet, mobile or channel-specific replacement;
- copy risks, authority risks and sensitive-data constraints;
- illustration/icon role;
- motion owner and reduced-motion behavior;
- evidence required for acceptance.

Keep the brief under two pages. Use `docs/design/programs-matters-decision-brief.md` as an example.

## 2. Before-state baseline

For a redesign, preserve the current screen before proposing changes. A baseline may be:

- a screenshot set;
- a minimal static HTML replica;
- a fixture-driven review route;
- a short structure/state inventory.

The baseline contains only what exists. Proposed elements belong in alternatives, not in the baseline.

## 3. Explore only where it pays

Branch two or three alternatives for high-impact or uncertain surfaces such as:

- executive/program overview;
- Matter detail and decision context;
- routing/escalation builder;
- internal/external capture wizard;
- protected intake;
- complex empty/onboarding states.

Do not create visual variants for routine implementation where the design system already determines the answer. Record why an option was selected.

## 4. State gallery

Reusable components and major screens need representative fixtures for default, loading, empty, stale, partial, error, permission, conflict, success and long-content states. The gallery may be a development-only route or deterministic screenshot fixture; it must use the production component.

## 5. Rendered review

Inspect actual rendered output rather than inferring quality from code. Check:

- scan order and primary action;
- copy and count integrity;
- source, owner, deadline and state visibility;
- typography, spacing, contrast and overflow;
- keyboard/focus and screen-reader names;
- mobile replacement rather than compressed desktop layout;
- light/dark or supported theme parity;
- reduced motion;
- loading, empty, failure and recovery.

Repair the highest-impact failure, then re-render the failed evidence. Do not claim visual completion without rendered evidence.

## 6. Drift control

The root `DESIGN.md` is the fast agent contract. Canonical tokens and components remain in source. A UI change that introduces a new token, component variant, density mode, motion pattern or illustration style updates the contract and its state fixtures in the same PR.

## Tool neutrality

External design canvases may help explore and compare alternatives, but ClearSight does not depend on a specific design-generation tool. Product semantics, accessibility, real workflow states and repository components remain the source of truth.
