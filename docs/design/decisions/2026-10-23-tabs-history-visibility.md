# Selected tabs after browser history

Date: 23 October 2026  
Scope: shared `Tabs` selection visibility; follow-up to Risk & Losses [#586](https://github.com/CloudSpaceLab/clearsight-grc/pull/586).

## Observed problem

Read-only desktop and phone checks of deployed revision `02dd0b4f066ba58ec17cf0e2588437efe5e856fd` found that browser Back from RCSA to Risks restored the correct collection and selected state, but left the phone tab strip scrolled toward Loss events. Only the end of the active Risks label was visible. The 390px document itself did not overflow. Named image review by `/root/evidence_review` confirmed the defect.

The existing click/keyboard checks revealed tabs through focus, and record-history checks returned within one collection. Neither exercised history across the four peer collections. A new peer-history check reproduces the defect against the preserved baseline: the tab viewport starts at x=64, while the selected Risks tab starts at x=37 and is 44px wide. The tablist and document identities remain unchanged.

## Decision

Keep the existing shared Tabs component and horizontal-only reveal callback. Declare `selectedKey` as a dependency of React Aria's cached tab collection so the selected existing item's ref updates when a parent changes selection. Stable item keys preserve the tablist and tab nodes. The change introduces no route, permission, data, token, layout or copy change.

The whole-page authority remains the immutable [operational-risk HTML](../references/clearsight-2026-10/source/clearsight-operational-risk-designs/operational-risk-preview.html), SHA-256 `cf0a101d574904fc5f496cab0625889e640e2e30734d180d86b89171419e69dd`, under the [21 October navigation decision](2026-10-21-v2-navigation-adaptation.md). Original artifacts remain unchanged. The existing [Risk & Losses decision](2026-10-22-risk-loss-pages-v2.md) and its explicit source-layout adaptations still apply.

## Required proof

- Unit regression: reuse the same noncontextual item objects, change the controlled selection right and back left without clicking or moving focus, and assert selected semantics, persistent tablist/tab elements, horizontal reveal, unchanged vertical scroll and cleanup.
- Browser regression: visit all four collections, then use three actual Back and three Forward transitions. At every transition check the route, selected label, unchanged document/tablist, measured selected-tab containment and no document overflow. Retain this in the existing desktop-light and phone-dark navigation scenarios.
- Receipt gate: reject missing, incomplete, reordered, clipped, remounted or overflowing peer-history evidence. A boolean visibility claim without consistent geometry is insufficient.
- Recheck the existing contextual-tab, retained-panel and keyboard cases; use the normal CI, full UI/source gates, expected-head merge and exact-revision demo deployment.
- Repeat the hosted phone history check and available real-record opening/return checks after deployment. Preserve empty, denied and unexercised states without inventing demo records.

The new unit regression failed before the runtime correction (horizontal scroll remained 0 instead of 130); all 10 Tabs tests pass with the one-line dependency correction. The new browser peer-history check also fails against the unchanged baseline at the observed phone Back transition. The corrected isolated build passes both navigation profiles and all 12 history transitions; the [retained evidence and named review](../../evidence/2026-10-23-tabs-history/README.md) bind those results to the local runtime commit. Publication, CI and deployment of the follow-up remain pending.

## Hosted population and authority limits

The original smoke used the public System Administrator demo account. Its actual context reports `oversight_read: false`, and Indicators correctly returns `403 permission_required`. A separate read-only check of the public CRO account reports `oversight_read: true` and receives a successful complete empty Indicators response. Both accounts receive a successful complete empty RCSA response. Those outcomes do not establish populated RCSA or Indicator record coverage and do not authorize changing permissions or seeding data.

Raw hosted screenshots and record identities are not included in this public repository. The hosted runner preserves the exact revision check and blocks application writes after login. Its temporary Chromium certificate exception is limited to chains containing the demo leaf public key independently verified through the execution environment's CA and hostname validation; it does not establish public-client TLS trust.

This remains partial v2 layout integration. Formal whole-screen operational acceptance stays **0/19**. Four-actor execution, backend authority, source-to-report reconciliation and production acceptance require their separate retained evidence.
