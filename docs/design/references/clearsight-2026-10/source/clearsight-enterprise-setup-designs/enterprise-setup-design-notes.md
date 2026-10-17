# ClearSight — Enterprise Setup & Onboarding

Design 12 · 15 October 2026 · illustrative prototype

## Structure

One setup workspace with Organization, Groups, People & access, Jurisdictions, Frameworks, and Data imports. The current selection remains visible in a compact contextual rail. Six readiness areas are tracked separately from the actual compliance posture; the displayed initial state is illustrative.

## Workflow

1. Choose the verified legal entity and organization hierarchy. Branches, departments and teams are scoped records, not interchangeable display filters.
2. Create or manage reusable groups that may span organization units. Membership changes require permission checks and history in implementation.
3. Assign roles and invite users through existing identity/workflow contracts.
4. Select jurisdictions based on operating entities and business activities; selection aids discovery and does not establish legal applicability.
5. Select a framework edition for **review**; application to a Program remains governed separately with source, rights, applicability, mapping and approval checks.
6. Import structured spreadsheets through a source-anchored mapping review; blanks remain unanswered and no data silently becomes a passed compliance check.

## Interaction rules

- Real product must use existing organization hierarchy, identity, Forms and frameworkcatalog contracts, not duplicate tables or a second setup store.
- Context rail is tied to the current selection and collapses below the primary workspace on narrow screens.
- Only work-relevant state transitions animate; 120–200ms motion and reduced-motion support.
- One next action is prominent, with up to three manually selected alternatives. No ambient UI animation or AI-generated compliance claim.
- Preview buttons work **locally only**. Sample records and progress counts must never be confused with verified production data.

## QA

Desktop 1880 CSS px rendered at 2x to 3760px PNG; mobile 390 CSS px at 2x. Functional checks: organization selection, create-group draft, search/filter frameworks, review selections, source import drawer, jurisdiction toggle, suggestion change, Escape close and no horizontal overflow at 390, 768, 1024, 1280, 1880px.

Next suggested design: integration connections and organization/role setup details, or finalize cross-screen design tokens and implementation acceptance before extending the library.
