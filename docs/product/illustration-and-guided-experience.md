# Illustration and Guided Experience

ClearSight uses premium illustration and contextual guidance to make high-accountability work approachable without making it playful, vague, or less rigorous.

## Product purpose

Illustration and onboarding should help a user answer:

- what is this surface for;
- why am I seeing it;
- what does ClearSight already know;
- what should I do next;
- how does this state differ from missing data or system failure.

They must never replace evidence, status text, authority, or actionable content.

## Illustration language

The visual direction is **institutional editorial futurism**:

- calm geometric composition;
- layered depth, restrained glass, fine connective lines and luminous accents;
- abstract representations of evidence, routing, institutional relationships, continuous monitoring and verified outcomes;
- premium enough for executive, regulator and board environments;
- human and optimistic without mascots, cartoons, stock-office scenes or consumer-fintech styling.

Illustrations should use the same semantic cyan, violet, amber, coral and verified-green system as the interface, but decorative color never carries state.

### Required variants

Maintain production-ready light and dark variants for:

- first-run introduction;
- no assigned work;
- no material change;
- no search result;
- no active Program or Matter;
- source unavailable;
- invitation expired or revoked;
- response submitted;
- routing configuration;
- continuous readiness;
- protected reporting.

SVG is preferred for bounded interface illustration. Raster assets are used only when richer editorial detail materially improves the experience. Assets must remain responsive, optimized, theme-aware and accessible.

## Empty states

An empty state contains:

1. a relevant premium illustration;
2. a precise state title;
3. a short explanation distinguishing absence, no change, unknown, unauthorized and source failure;
4. one primary action where action is useful;
5. an optional education or source-health link.

Never use an illustration to conceal a loading error, missing permission or stale source.

## First-time guidance

Guidance is role-specific and task-led. Initial roles include executive, Program owner, reviewer, authorizer, evidence respondent and Configure administrator.

A guide must be:

- skippable and resumable;
- stored per user, role and guide version;
- short enough to complete in a few minutes;
- progressive rather than a mandatory modal tour;
- anchored to real controls and workflows;
- safe when the user has no permission for a referenced feature.

Preferred patterns:

- a premium introductory panel;
- three to five role-specific steps;
- contextual coach marks;
- a first meaningful task;
- a small setup checklist;
- inline “why this matters” explanations;
- optional demonstration data or sandbox mode.

Avoid long slide decks, forced product tours, gamification, celebratory effects and guidance that blocks urgent work.

## Adoption telemetry

Measure only privacy-minimized events:

- guide started, skipped, resumed and completed;
- time to first meaningful action;
- help opened;
- first-task completion;
- repeated confusion or abandonment by step.

Do not capture sensitive field contents or infer user competence from onboarding behavior.

## Implementation contract

The design system exposes:

- `PremiumIllustration` variants;
- `EmptyState` with state semantics;
- `IntroGuide` with versioned state;
- theme, reduced-motion and localization support;
- image-size and bundle budgets;
- visual-regression references.

A feature with an important empty or first-run state is incomplete until that state is designed and tested.
