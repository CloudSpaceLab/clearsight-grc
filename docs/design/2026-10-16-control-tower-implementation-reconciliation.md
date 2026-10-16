# Control Tower #462 — implementation and acceptance reconciliation

Date: 2026-10-16. Reviewed repository `main` at `ff846af3bb61f209637b84e646fc8638e5c4eb2d`.
Owner: [#462](https://github.com/CloudSpaceLab/clearsight-grc/issues/462). Approved contract: [design and source plan](2026-10-15-control-tower-ui-and-reference-data.md).
This file is a **checkpoint**, not a product-release certificate. Refresh branch heads and source results before accepting any tranche.

## Reconciled delivery ledger

| PR(s) | Merged behavior | Validation boundary |
| --- | --- | --- |
| #471 | Existing Home header: tokenized bordered wave and responsive controls | Header presentation only, not final page composition |
| #472, #474 | Compact first-run guide, persisted progress, action-target and authority checks | Does not prove an actor can complete a real first usable assessment |
| #473, #475–#477 | Four metric positions, unknown/no-population quality, current-period/entity Loss and historical Risk source identity | Does not create a compliance percentage or a full comparable source-backed executive composition |
| #478 | Risk/Loss register and record header presentation, page-local filter truth, shared CSS removed duplication | Existing authoritative Risk/Loss register flows retained |
| #479 | Programs loaded-page state, stable section heading and shared action controls | No Program calculation/policy or framework content activation |
| #480 | Reports loaded-run counts no longer imply enterprise totals | Does not prove download authorization, report completeness or historic all-page cardinality at scale |
| #481, #486 | ROPA activity/register headers and shared MetricCard coverage-quality display | ROPA dashboard is not proof of a complete ROPA/DPIA/consent journey |
| #484 | Vendors, Reports and Insights use shared workspace header surface | Does not implement new lifecycle controls or complete final visual composition |
| #485 | Issues/changes list header and bounded work summary | Current issue record authority and full outcome flow remain separate |
| #488 | Correct KRI/KCI empty-state wording; header/test reconciliation | Existing indicator endpoints and history are unchanged |
| #489 | Issue record title and fact summary use shared header surface | Actions, responsibilities and outcomes remain on existing machinery |
| #482, #483 | Closed without merge: superseded by #484 and #488 | Do not replay duplicate header CSS |

As of this checkpoint: **16 merged PRs** (#471–#486 excluding #482/#483, plus #488/#489); **2 active #462 drafts**, #487 and #490. Their heads and merge gates must be checked live. The non-#462 #469/#470 work remains independently owned by #437.

## T0–T6 acceptance status

| Tranche | Status | Existing proof | Remaining gate |
| --- | --- | --- | --- |
| T0 truth audit | Partial | This plan, design rules, source/fixture classifications and repo-owner reconciliation | Signed source rights, full installer inventory, production/demo/test distinction, baseline-to-approved screenshot diff with accountable owner |
| T1 shell and first run | Substantial, not closed | Home header, authorized scopes and period picker, compact guide, successful step navigation checks | Real role-specific setup: choose/adopt Program, link existing Forms, assign owner, first approved assessment; verify actions hidden for denied actors and completed journeys |
| T2 trusted executive composition | Partial | Four position-stable Risk/Loss measures, source-bound trends/period, organization concentration, exact member drills | One primary comparison plus bounded ranked breakdown in first viewport, collapse secondary work, explicit source as-of/definition, CRO usability validation |
| T3 governed contextual Copilot | Not delivered in Home | Existing AI gateway and governance infrastructure only | Exactly three or fewer current, authorized, evidence-linked suggestions with real routes, stale/no-AI fallback, interaction/keyboard validation |
| T4 Framework/Program/Form/Vendor flows | Partial; cross-issue dependent | Existing domain workflows and page navigation; #478–#489 first-pass design alignment | Single complete authorized assessment-to-review-to-action-to-report journey; #433 policy, calculation and contribution history; #437 licensed catalog/adoption/edition mapping |
| T5 reference content, seed release | Not accepted | Current protected importers, reference guidance, reference/demo gate and prior importer repairs | Written manifest inventory, rights, private OpsRisk receipt, zero-blank-preservation, two-run idempotent reconciliations, production isolation, approved publisher/form packs |
| T6 release validation | Automated foundation; not release UAT | CI, UI/UX rendered flow gate, responsive and accessibility automation | Actor/tenant and production-hosted matrix, 1440/1280/1024/768/390/320, denied/empty/partial/stale, 200% focus, time/source comparison, performance/enterprise cardinality |

The parent issue's checkboxes are **not automatically complete** because a related PR has merged. Keep the last-gate status explicit.

## Current page findings

- **Home:** Oversight/Attention/My work and real trend/concentration components exist. The Overview still lays out Risk concentration + Risk movement **and** Loss concentration + Loss movement together, with separate detailed analysis tabs. Approved one-primary-chart/one-ranked-breakdown layout is **not yet implemented**.
- **Programs:** role/scope/pagination and categorical status exist; per-Program score, critical gates, actor-safe contributors and proof of formula are not owned or complete under this issue. #433 owns them.
- **Frameworks:** provenance/licensing/adoption and clause/edition upgrades are #437; #469 and #470 are separate draft dependencies. Do not claim active catalogs or compliance grades merely from metadata.
- **Risks, Losses and ROPA:** shared UI presentation has moved forward; a visually consistent register is not a substitute for the full loss-to-recovery or privacy-review workflow release receipt.
- **Vendors:** existing relationship, activation, DPO, assessment and form flows are retained; no new end-to-end scenario has been demonstrated by the header change.
- **Forms:** current form builder/distribution/review surfaces remain; #487 owns shared compact Select keyboard dismissal, and #490 owns stale Sent-forms request/cursor response protection. Both still require exact-head gates at checkpoint.
- **Work:** list and record header now use shared surfaces; real authority, outcome verification, closure, replay and partner/vendor linkage must be validated in an end-to-end scenario.
- **Reports:** existing saved setups, generated-run list and downloads remain; #480 corrects loaded-page labeling but does not certify all authorized report downloads.

## Next independent implementation gates

1. **Close #487/#490 correctly.** Confirm latest exact head passes backend+web CI and the full deterministic screenshot/keyboard review. If the 200%-zoom Forms Tab selector still fails, fix the shared React Aria keyboard focus semantics; do not weaken the evidence check.
2. **T2 visual hierarchy.** Recompose only `OversightWorkspace` with the current `Tabs`, `MetricCard`, `RankedBarList` and trend components: four fixed metrics then one bounded Risk or Loss comparison and ranked organization context. Keep source, drill and selected period. No new metric service.
3. **T1 real setup acceptance.** One bank/telco admin installs or selects published content (only if entitled), creates a Program, links a current governed Form, assigns real work, receives and reviews one response, and records a bounded authorized decision. Verify actual API receipts rather than step-navigation counts.
4. **T4 integrate #433/#437.** Confirm versioned calculation and content-rights contracts and exact actor-safe contributions before exposing any percentage/framework grade or catalog count. No duplicate migrations.
5. **T5 seed isolation.** Collect private installer receipts and manifest/canonical loss source audit separately from UI PRs. Protect customer source material; do not copy into Git.
6. **T3 Copilot last-mile.** Only when source readers and routing are sound: reuse current AI/evidence service to present up to three grounded actionable suggestions, or nothing when unsupported.
7. **T6 signoff.** Report an actor-role × entity/group × state × viewport acceptance matrix and hosted UAT result separately from CI-green PR receipts.

For every acceptance row record: actor/permission, tenant/entity/group, selected and effective reporting period, source ID/revision, completeness/freshness, record count and exact drill result, action receipt, and failure/fallback state.

## Invariants

- No second dashboard, framework publisher, metric engine, AI gateway, work queue or scoring implementation.
- No inferred production rights, adopted standards, compliance percentage, loss currency conversion or synthetic data.
- Merge small independent PRs only after **exact-head** CI and UI/UX PASS. Preserve historical submissions and source lineage.
