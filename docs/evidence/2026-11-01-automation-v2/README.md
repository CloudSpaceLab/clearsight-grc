# Automation V2 — retained review

This tranche replaces the basic Automation list with the approved four-tab page composition while retaining the existing policy reader, configuration-read permission and governed escalation owner. It is **partial V2 layout integration**; whole-screen operational acceptance remains **0/19**.

[Decision and data semantics](../../design/decisions/2026-11-01-automation-governance-v2.md) · [Named image review](named-review.md) · [Final image/behavior receipt](after/receipt.json) · [Deployed baseline receipt](before/receipt.json)

## Reviewed identity

| Checkpoint | Local commit | Published equivalent | Identical tree |
| --- | --- | --- | --- |
| Released starting main | `f9b5bfd9c27641c8acb7928b2e59d8c59b00ec64` | Same | `40cbc4f16a3c93098371520bb74ae0a8da2f3434` |
| Bounded read contract | `300ca1d46efb5907da81ce8c90dbdfa224006ff6` | `3fcd30d351679e1b7972140f6ab669f3bbfe358c` | `23479c9bbe56de423a4343d4a463a8c695ff27e9` |
| V2 runtime and regressions | `ce2b3af4a5871c7c1dd20764a66563009832f88b` | `6898145afaa593fb68528c1216012af142f03078` | `6aa79959ee4285b94ef1c5ee916befa10104ff35` |
| Final capture inputs and runtime | `54bd6f192419c89a3713aa6c69e37205655c37e4` | `697b18e6e6319a2dbf430d82cc1bdf285b792eb5` | `24219c2d3c7d02f2357b615f9fdc364932000ae2` |

The final browser receipt binds the last checkpoint to evidence build digest `78244fc114b797a60af94de172bf831ab7d1d580ccd17029adc75555f02e0dd3`, with unchanged assets before/after capture and Chromium **140.0.7339.16**. The subsequent documentation/evidence commit does not change runtime or capture inputs. Hosted checks and release status are recorded in the implementation PR.

## Source and comparison

Canonical HTML: `docs/design/references/clearsight-2026-10/source/clearsight-automation-governance-designs/automation-governance.html`, SHA-256 `e24b9c012375175a2692be98adb2741dbea353cae0979b919fb0a18261583a98`.
Rendered self-contained preview: `automation-governance-preview.html`, SHA-256 `7dad1ffc651fd03821cc996ff13f1203cb1ec1344a8a5bdec7b00b49035a7b6f`. All **96** original source files match their manifest (`validation/source-integrity.json`).

`before/` retains ten baseline theme/viewport layouts, eight source views and four continuation/full-page renders, with its original input snapshots in `baseline-inputs/`. `after/` retains **48 hashed images**, covering ten final layouts, twelve behavior cases and source/context/full-page continuations. The `inputs/` snapshots preserve the final fixture, capture and reviewer; those files must remain identical when replaying the receipt.

The comparison checks the full heading, source-order tabs, four summary cells, register heading/filter/record order, selected context, guardrails before configuration facts, responsive stacking, 44px controls and complete first record. The original is light-only; dark images evaluate the runtime adaptation. Shared shell navigation, readable production text, loaded-population disclosure and unavailable workflows are explicit adaptations, not pixel-equivalence claims.

The old first policy already fitted every baseline viewport. Final complete first-policy bounds are **570.19–649.19px** at 1440/1280, **639.19–718.19px** at 1024, **603.19–682.19px** at 768, and **740.38–836.38px** at 390. The additional navigation, filters, summaries and provenance are intentional; no claim is made that the new first row is higher than the baseline.

## Validation

| Gate | Local result |
| --- | --- |
| Full web suite | **279 files / 2,332 tests passed**; raw log retained. This precedes the final CSS-only phone refinement and deletion of an unused array-reader export. |
| Final focused UI/configuration/copy suite | **5 files / 38 tests passed**, including revocation, stale responses, identity/session changes, refresh recovery, metadata and history. |
| App integration | **78 tests passed** at the runtime implementation checkpoint. |
| TypeScript, fresh customer/evidence builds | Passed at final capture input/runtime revision. |
| Fixture isolation and three shared Node contract suites | Passed; synthetic transport is unreachable from customer runtime. |
| Autonomy and HTTP package suites | **460 top-level / 691 including subtests passed**; raw JSONL retained. |
| Focused Go race checks and PostgreSQL-tagged vet | Passed. |
| Real PostgreSQL integration | Test compiles but **skipped locally** without `TEST_DATABASE_URL`; hosted CI must execute it before merge. |
| Final browser/capture reviewer | **10 layouts / 12 behaviors / 48 image hashes passed**. |
| Accessibility | Zero axe violations in all ten layout scans. Decorative hidden-chevron checks remain incomplete; this is not full WCAG certification. |
| Customer bundle | Initial JS **161,947 B gzip**; initial CSS **32,403 B gzip**; largest JS chunk **573,982 B raw**. Existing budgets pass without adjustment. |

The browser scenarios verify actual selection changes through Back/Forward/reload, URL filters, context focus and return, source tabs and the existing escalation handoff, loading, empty, 200-row truncation, denied, unavailable/retry, read-only and long key/value/code handling. Pre-reload read/command observations are retained so navigation cannot erase an attempted material command. No material commands are issued by this view.

## Corrections and independent review

Initial renders exposed oversized inherited summary minima, phone labels splitting inside words, an incomplete first phone/tablet record and unwrapped long guardrail keys. The scoped shared metric tokens, source-sized phone title, title/status row and key wrapping correct these. Selected diagnostic receipts and screenshots remain under `diagnostics/`; these were working-tree probes, not passing release checkpoints or fully archived replay bundles.

`/root/automation_contract` independently reviewed data/authorization behavior and reproduced the long-key overflow before the fix. `/root/cutover_status` reviewed frontend regressions and the evidence harness, identifying and verifying corrections to distinct-record history, retained pre-reload observations and long-field measurements. `/root/automation_visual` inspected the original, baseline and final images and owns the named visual review. `/root` reviewed the integration, key final desktop/phone/context renders, source integrity, tests and publication tree identities.

## Replaying the retained final receipt

From `web/`, run the archived reviewer against its matching inputs. The reviewer expects `src/` and `scripts/` to contain the same fixture/capture/reviewer bytes recorded in `inputs/`; at the final revision they do.

```sh
automation_review_dir=$(mktemp -d)
ln -s "$(realpath ../docs/evidence/2026-11-01-automation-v2/after)" "$automation_review_dir/automation"
UI_EVIDENCE_DIR="$automation_review_dir" node scripts/review-automation-evidence.mjs
```

Do not rebuild or overwrite the retained PNGs merely to replay their receipt. New captures should use a separate output directory, the reviewed revision, and a fresh evidence build.

## Remaining acceptance

Latest policy revisions are not a complete inventory of currently executing rules. Schedules, execution/delivery receipts, earlier revision history and general policy editing/simulation/activation remain unavailable here. Existing AI governance and escalation owners remain separate. Synthetic browser transport establishes frontend behavior, not live backend authorization, four-actor execution or a verified outcome. Full production cutover remains gated by those distinct receipts.
