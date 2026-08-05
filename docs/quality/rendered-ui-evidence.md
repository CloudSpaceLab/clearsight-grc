# Rendered UI evidence gate

A UI PR is not accepted from code review alone when it introduces or materially changes a screen, workflow, reusable component, responsive composition, illustration system or motion pattern.

## Required evidence

Attach or generate deterministic renders for the affected states and viewports. At minimum:

- desktop primary state;
- narrow/mobile replacement;
- loading;
- explicit empty scope;
- unavailable/error and recovery;
- long content;
- keyboard focus;
- any material warning, conflict or permission state.

Add light/dark, 200% zoom, reduced motion, translated copy and assistive-technology evidence when the changed surface supports or materially affects them.

## Review order

1. Correct object, user and primary action.
2. Accurate state, count, source, owner and deadline.
3. Safe recovery and no misleading controls.
4. Information hierarchy and scan speed.
5. Responsive replacement.
6. Keyboard, focus, semantics and contrast.
7. Typography, spacing, assets and motion.

Fix the highest-impact failure first and re-check that evidence. Do not spend the first repair round polishing decoration while the workflow, copy or state is unclear.

## State-gallery contract

Each reusable component has named fixtures for supported states. Fixtures use realistic but clearly labelled data and the production component API. A component variant that exists only in an ad hoc page is not considered part of the design system.

## Release boundary

Rendered evidence proves appearance and interaction for the tested fixtures. It does not prove authority, confidentiality, data completeness, performance or domain correctness; those remain separate release gates.
