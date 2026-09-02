# Feedback contrast audit

## Decision

Operational success receipts use the shared `Notice` component. Feature workspaces must not define a private success foreground or background. A successful state remains identifiable through its message, live-region semantics, border and surface tint; readable text uses the theme's primary text token.

## Before-state baseline

Issue ownership, actions, outcomes, vendor work, monitoring, Program setup and source connection receipts used `.inline-success`: pale green text designed for a dark canvas. In light mode the foreground nearly disappeared against the translucent green surface. The low-risk label reused the same dark-only colour.

## Scope

- migrate every `.inline-success` receipt to `Notice tone="success"`;
- remove the legacy class instead of retaining parallel feedback contracts;
- bind the low-risk label to semantic success, border and surface tokens;
- retain existing business copy and workflow behavior;
- test the resolved light and dark token pairs at WCAG AA text contrast.

No new component variant or colour token is introduced. `Notice` remains the closed feedback contract documented in `DESIGN.md`.

## Required states and evidence

| State | Light | Dark | Proof |
| --- | --- | --- | --- |
| Success receipt | Primary text on success-tinted surface | Primary text on success-tinted surface | token contrast test and affected workflow tests |
| Low-risk label | Semantic success text and tint | Semantic success text and tint | token contrast test |
| Error, warning, information | Existing `Notice` tones | Existing `Notice` tones | shared feedback tests |
| Forced colours | System border and text | System border and text | existing forced-colour component rule |

Rendered UI review must cover the issue-owner update receipt in both themes and representative desktop/mobile workspaces before visual completion is claimed.
