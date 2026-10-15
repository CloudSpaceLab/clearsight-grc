# Control Tower interface and reference-data release plan

Date: 2026-10-15  
Parent: [#462](https://github.com/CloudSpaceLab/clearsight-grc/issues/462)  
Baseline: `main` at plan creation; refresh before each tranche  
Status: T0 guidance and inventory; runtime redesign and reference installs remain gated
Fidelity acceptance: [31 traceable requirement gates](2026-10-15-fidelity-acceptance-map.md).

## Decision

Use the approved light Control Tower screenshot as a **visual reference**. Its example organization, numerals, progress rings, framework count and AI copy are placeholders. Product meaning comes from current authorized API snapshots, approved Program policies and catalog publications, not screenshot pixels.

One Home route, three intents (`Oversight`, `Attention`, `My work`). The desktop composition should feel calm and refined: compact fixed navigation, contextual scope and date controls, four stable measures, one meaningful comparison, relevant library links, and a smaller Copilot rail when it has grounded suggestions. On tablets and phones, use summary-first disclosure instead of squeezing three columns.

### Actual owners to reuse

| Existing owner | Proposed work | Do not duplicate |
| --- | --- | --- |
| `App.tsx`, `EnterpriseScopeSwitcher`, `appRouting.ts` | Authorized legal-entity/organization context, route preservation, plain group naming | New ad hoc scoped state or fake branch selectors |
| `OversightWorkspace`, `DomainPostureSummary`, `OversightPeriodPicker` | Home header, temporal context, existing KPI accuracy and drilldowns | New Home metric service or dashboard builder |
| `MetricCard`, `RankedBarList`, shared tokens | Chart/card anatomy, keyboard/focus, hover/loading/quality | Feature-owned visual component families |
| `RoleAwareOnboarding`, `IntroGuide` | Role-scoped optional first-run actions, persisted progress | Permanent `Get started` billboard |
| `frameworkcatalog` (#437), continuity / Program (#433), form catalog | Read-only framework discovery, edition pinning, reviewed adoption, reusable form starters | Unapproved compliance-score calculation or duplicated obligation |
| Existing AI gateway/governance and evidence/source services | Evidence-linked suggestions and guarded command handoff | Another chatbot backend or invented facts |
| Existing reporting and imports | Source-linked spreadsheet review and reports | Spreadsheet-derived compliance verdict |
| `cmd/seed-bank-reference`, `internal/bankverticals` | Explicit non-production journey installer; harden/retire redundancies in phases | Production fixture publisher |

### Baseline code findings

- Home currently lives in `OversightWorkspace`. Role-aware intent tabs, domain posture metrics, organization risk/loss concentration, trends, attention and assigned work already exist.
- `OversightPeriodPicker` issues bounded reporting queries and offers 7/30/90 days and custom start (server contract currently fixes end date and explicitly does not offer historical end-state). The header must report the **selected effective** range, including partial/failed changes, and not suggest that current-posture metrics are historical aggregates.
- Dynamic Programs compliance percentages are under active #433 calculation work; until verified snapshots exist, show categorical status + assessed coverage, not a composed 78% number.
- `migrations/000135_framework_catalog_editions.up.sql` already constrains source, edition, license and immutable clauses; publisher/adoption work belongs to #437. Avoid concurrent schema edits.
- `cmd/seed-bank-reference/main.go` rejects `production`, but a no-mode run executes several unrelated sample workflows. Narrow, explicit modes include source loss, source records, source risks, employee, document and vendor samples.
- `source_losses_import.go` preflights eight distinct events and enforces demo tenant/legal entity, with monthly duplicates collapsed and no inferred recovery. The protected manifest must not enter Git.
- `staticDemo.ts` only executes with static UI evidence flags; fixture and runtime paths still need dependency/use classification before retirement.

## Interface contract

### Desktop at 1440px

1. **Shell**, existing verified scope/group control and search remain compact and consistent. The top filter row must not present fake independent choices when only the organization hierarchy selector is supported; add domain lenses only as real authorized filters.
2. **Header**: `Enterprise Control Tower` and operational period copy such as `Risk, compliance and action in the last 90 days`. Period selector shows the resolved period and maintains stable data until an authorized query succeeds. No motivational slogan or system-health decoration.
3. **First use**, if applicable: up to three real role-authorized actions using existing guide state. Never show invitation/create actions the actor cannot perform. After setup collapse to a quiet launcher; avoid a persistent full-width onboarding bar.
4. **Measures**: four stable current/period metrics sourced from authorized snapshot, with as-of, quality and comparable basis. If a metric is unavailable, retain its position and show `Unavailable`, not `0`.
5. **Analysis**: one primary comparison and a bounded ranked breakdown in first viewport; remaining trends, losses, investigations and matrices behind `Insights`, `Attention` and `My work`.
6. **Copilot**: a restrained illustration/orb with at least 24px internal spacing from the summary and actions, no text/glow overlap. At most three actionable evidence-linked suggestions and a grounded source timestamp; absent AI means other workspaces remain fully functional.

### Animation and interaction

- Animated transitions only on selection, popover enter/exit, scoped drill, loaded state and data change. Never animate summary counts independently of data revision.
- Typical durations 120–200ms, using existing motion tokens. No looping orb animation required; `prefers-reduced-motion: reduce` removes nonessential motion entirely.
- All charts provide textual value, source/freshness, drill equivalent and keyboard focus. Tooltip does not carry required status alone.
- Period or scope change cancels old reads, does not leak prior-tenant values, preserves last confirmed period on failed query and displays recovery at the control.
- Visual proof at 1440, 1280, 768 and 390 widths; dark/light, loading/empty/partial/stale/denied/reduced-motion; test no horizontal scrolling or inaccessible controls.

## Data program (migration sequence)

| Data class | Current surface | Desired install semantics |
| --- | --- | --- |
| Approved framework edition and clause references | `frameworkcatalog`, `000135` and #437 | Dated, licensed, vetted, immutable, idempotent release; **metadata-only must say so** |
| ROPA/KRI/RCSA/vendor/DPIA starter questionnaires | Governed form starter catalog `000060` and existing form authoring | Small default extendable, versioned forms with actual actions and reporting; never auto-answer |
| Bank-private source registers and artifacts | `source_records_*.go`, protected manifests and governed Imports | Scope-constrained validated source lineage, blank preserved, receipts, idempotent operator-run imports |
| OpsRisk loss history | `source_losses_import.go`, 2026-10-09 plan | Eight canonical events, no inferred recoveries, exact source SHA; private run with backup + two-run receipt |
| Non-production scenario journeys | `bankverticals`, default `seed-bank-reference` | Distinct opt-in mini journeys, no production execution, accurate states and authority |
| Static UI test/evidence fixtures | `staticDemo*`, `*EvidenceData`, testdata | Test-only, not advertised as catalog or tenant data |

**First seed actions:** inventory install entry points and source rights; extract an approved minimal catalog/form pack manifest; validate publisher/adoption/replay before any tenant installs; prove one complete assessment response and audit report. Complete #437 F0–F2 before claiming a standards count. Align customer-private OpsRisk import to its verified eight-event contract; do not distribute its source content.

**Quarantine protocol:** mark redundant demos for retirement, locate test/runtime imports, replace with smaller realistic fixtures behind explicit flags, run compatibility/snapshot tests, only then remove. Do not delete an operating fixture solely because it is large.

## Execution order / merge isolation

1. **T0** (this PR): issue, `AGENTS.md`, `DESIGN.md`, decision plan, source classification.
2. **T1**: Home period-aware header and first-pass copy, then authorized compact first-run actions and shell styling. Render and run scoped UI tests.
3. **T2**: trusted four-metric executive snapshot and bounded visuals; no #433 score until published.
4. **T3**: optional three-suggestion Copilot, evidence/source checks, keyboard and no-AI fallback.
5. **T4**: navigation through actual Framework, Program, Form, Vendor and report outcomes after #433/#437 merge/rebase.
6. **T5**: a separate reference-seed PR per verified content pack; preserve private OpsRisk manifest and loss reconciliation.
7. **T6**: stable responsive proof, multi-tenant/permission and stale/unknown datasets, render comparison, copy/performance/accessibility CI.

## Acceptance data matrix

For each affected release record verified actor and role, tenant/legal entity/group, selected period and actual as-of, source ID/revision, completeness, publication/approval status, expected member count, exact drill count and authoritative action receipt. Include zero, unknown, partial, stale, mismatch, denied and source unavailable cases.

No fixture count, status or source-rights claim may be copied from the screenshot.
