# 2.9 shell and Home integration

Status: partial integration; not a screen cutover or production acceptance.

The 2.9 `index.html` places a full-width context header above the rail and uses a compact drawer on phones. Move the existing brand and verified context controls into the shared header; keep the existing session, notification, scope and permission owners. Do not copy the reference's sample selectors, notification counts or user identity. Keep existing routes, risk destinations and Settings authorization.

On legal-entity Home, show at most three assigned items from the existing Today feed beside overview analysis when width permits, and stack them before analysis on narrower screens. Opening an item uses the existing action callback; loading, unavailable and empty remain distinct. The Today feed is not attributed to an organization sub-scope, so omit this summary when that filter is active. Do not infer that those items represent all enterprise work. Keep the full My work view and all metric freshness and drill semantics.

The pre-change light reference/runtime comparison is in `docs/evidence/2026-11-20-v29-shell-after/`. The current reference/runtime pairs, responsive geometry and navigation receipts are in `docs/evidence/2026-11-20-v29-shell-home/`. These fixtures are synthetic and do not prove backend permissions or source reconciliation. Further work remains to reproduce the complete 2.9 Home composition, reduce the first-run guide's displacement of first-viewport content, adapt remaining workflows, and verify production authorization, accessibility and data results.
