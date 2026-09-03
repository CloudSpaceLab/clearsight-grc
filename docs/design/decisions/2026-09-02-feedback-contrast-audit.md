# Feedback contrast audit

## Decision

Operational feedback uses the shared `Notice` component and compact stored states use `StatusBadge`. Feature workspaces must not define private feedback foregrounds, backgrounds or status palettes. A successful state remains identifiable through its message, live-region semantics, border and surface tint; readable text uses the theme's primary text token.

## Before-state baseline

Issue ownership, actions, outcomes, vendor work, monitoring, Program setup and source connection receipts used feature-owned green or red text designed for one canvas. In the opposite theme, text and control boundaries could disappear against translucent state surfaces. AI governance cards also retained hard-coded dark surfaces in light mode, while monitoring risk bands, organization reporting states, automation policy states and due-diligence states each maintained a separate badge palette.

## Scope

- migrate remaining continuity, vendor, due-diligence, monitoring and form-authoring feedback to `Notice`;
- migrate monitoring risk, reporting-line, automation policy, AI rollout and due-diligence states to `StatusBadge`;
- remove the legacy classes instead of retaining parallel feedback contracts;
- replace AI governance dark-only card and summary surfaces with semantic surface, border and text tokens;
- define the unknown-state foreground explicitly for both themes;
- retain existing business copy and workflow behavior;
- test every information, success, warning, error and unknown tone against its resolved light and dark surfaces at WCAG AA text contrast.

No new component variant is introduced. The existing semantic unknown token receives an explicit theme mapping; `Notice` and `StatusBadge` remain the closed feedback contracts documented in `DESIGN.md`.

## Required states and evidence

| State | Light | Dark | Proof |
| --- | --- | --- | --- |
| Success receipt | Primary text on success-tinted surface | Primary text on success-tinted surface | token contrast test and affected workflow tests |
| Stored state label | Semantic tone, marker and text | Semantic tone, marker and text | token contrast and affected workflow tests |
| Error, warning, information | Existing `Notice` tones | Existing `Notice` tones | shared feedback and workflow tests |
| Unknown state | Explicit neutral foreground and tint | Explicit neutral foreground and tint | token contrast test |
| AI governance cards | Semantic light surfaces and text | Semantic dark surfaces and text | component render and CSS contract test |
| Forced colours | System border and text | System border and text | existing forced-colour component rule |

Rendered UI review must cover the issue-owner update receipt, vendor/due-diligence feedback, AI governance cards and representative status badges in both themes and desktop/mobile workspaces before visual completion is claimed.
