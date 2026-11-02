# Governed Demo Seed Correction Design

## Goal

Make the Clear Bank demonstration distinguish source captures, governed records and derived insight projections without inventing Risk, RCSA, KRI or BIA facts that do not exist in the supplied workbooks.

## Current state

The private manifests retain the supplied IT, vendor, NDPA and OpsRisk source data. The live demo has eight canonical operational losses, but no canonical Risks, Risk assessments, RCSA cycles or Risk-to-indicator links. The two imported IT exception rows have complete lineage and can be safely reconciled into Risks. The four RCSA rows are action rows, not a complete cycle population. Historical KRI and BIA rows are source observations; they do not establish approved thresholds, current metric observations, critical services or recovery targets.

The domain posture projection is a legal-entity snapshot. The client incorrectly passes an optional organization scope to that endpoint even though the endpoint deliberately rejects it. The Loss period reader is genuinely scope-aware.

## Decision

### 1. Isolated source-Risk reconciliation

Add a `-source-risks-only` reference-seed operation. It accepts the existing non-production demo scope and reconciles only persisted `it-risk-exceptions` source Matters into canonical Risks and source-register assessments. It does not reread manifests, create forms, create Matters, import losses, change vendors or run the broad operating-demo seed.

The operation uses the existing source lineage, refuses missing or changed lineage, preserves the recorded source rating as a source assessment and leaves appetite unknown unless a source appetite statement exists. It is idempotent and emits a narrow receipt with candidate Risk and new-assessment counts.

### 2. Truthful source status for RCSA, KRI and BIA

Do not create RCSA cycles, Indicators, monitoring observations or critical-service targets from incomplete historical workbook rows. Preserve those rows as source-backed submissions and make their readiness explicit in the seed receipt and operating documentation:

- RCSA requires a defined assessment period, frozen Risk and Control population, accountable first line, independent challenge route and governed outcome.
- KRI requires an approved definition, owner, period, unit, limit, calculation and source/coverage contract.
- BIA requires approved critical-service identity, impact tolerance, dependency ownership, RTO/RPO target revision and review cadence.

This preserves the existing source data while preventing a dashboard count from being presented as an approved operating metric.

### 3. Insight-scope correction

Keep domain Risk posture at legal-entity scope until a scope-aware Risk projection exists. The client must request the legal-entity domain bundle without an organization scope, and must label its basis accordingly. Loss insights continue to use the selected authorized organization scope.

If the canonical Risk population is zero, the Risk posture UI must say that no active Risks have been loaded for the legal entity; it must not present a zero-value clear condition as evidence of healthy risk posture. Individual metric values remain visible only where their zero population and calculation basis are explicit.

### 4. Source manifest correction boundary

The next private source extraction must use the existing V2 source shape: one Branch KRI register with one response per branch, and no duplicate Head Office calculation/report population. Existing V1 captures remain attributed historical source data; deployment does not silently rewrite them. A separately reviewed source installation performs that migration after the private manifest is regenerated and validated.

### 5. Loss-to-Matter links

Do not mass-link loss events merely because nearby Matters have similar titles. Link a loss only where the source record establishes the same source identity. Unlinked losses remain a valid attention condition rather than a seed error.

## Data ownership

| Object | Owner | Meaning |
| --- | --- | --- |
| Submitted source records | Forms and source-register lineage | Immutable historical capture with source coordinates and limitations. |
| Risks and assessments | ERM Risk service | Governed current record and versioned source assessment. |
| RCSA cycles | RCSA service | Frozen assessment population and lifecycle; not a spreadsheet action row. |
| Indicators | Monitoring and Risk services | Approved definition plus bounded observed result. |
| Domain posture | `metricview` | Five-minute materialized legal-entity projection with exact membership. |
| Loss period analysis | `metricview` | Repeatable-read selected-period calculation plus a short-lived reproducibility membership set. |

## Acceptance criteria

1. A scoped source-Risk seed run creates the two eligible IT exception Risks and their source-register assessments, and a second run creates neither duplicates nor new assessments.
2. The operation rejects production, non-demo scopes and unsupported combinations with other scoped seed modes.
3. The web client never sends an organization scope to the legal-entity-only domain metric endpoint.
4. An empty canonical Risk population is explicit in the Risk posture presentation.
5. RCSA, KRI and BIA source captures remain unaltered and no cycle, indicator or critical-service facts are fabricated.
6. The source-record installation guide records the required inputs before each future canonical mapping.
