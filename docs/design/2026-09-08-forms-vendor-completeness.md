# Forms and vendor workflow completion

## Approved job and scope

The operator approved this bounded completion pass after live inspection of deployed `fab860ce` as Program Owner and System Administrator. A bank user must identify the subject, inspect submitted answers/documents, understand the current review state and reach the next permitted action without reconciling contradictory cards. This is not a new workflow engine, document viewer or visual redesign.

## Before-state baseline

The live audit retained eight desktop/narrow captures and extracted screen text in the local `interface-completeness-audit` evidence directory. Confirmed findings:

- Policies exposes an administrative permission denial as a retryable loading failure. An active approved revision displays “No current simulation” based on browser memory, raw form identifiers and unlinked execution outcomes.
- Responses displays raw subject IDs, a free-text subject-type filter, duplicate document launchers and repeated empty score/review warnings when review is not required.
- A vendor form has a submitted revision and timestamp but its request projection says it awaits submission. This inflates outstanding counts and hides response/document actions.
- Vendor details are stacked into approximately 5,040 CSS pixels at 390px. Request actions repeat; identity metadata precedes the operational work.
- Persisted demonstration records include engineering acceptance titles. Sample presentation must remain explicitly fictional and must not rewrite historical submissions merely to look polished.

## Selected approach

Fix authoritative read-state contradictions first, then reuse existing shared components to complete the workspaces. A cosmetic-only change leaves incorrect handoffs; rebuilding these workflows would duplicate working authority and capture foundations. No new mockups, dependencies, token families, density modes or AI claims.

Policies keeps its list/detail pattern, with named form scope, stage-appropriate simulation/approval context, purposeful denied state, and permitted links to resulting issues. Responses uses an ordinary subject selector and compact review sections: Answers, Documents, Review and History. No-score and no-review are neutral conditions, not failures. Vendor detail uses Overview, Forms, Documents, Due diligence and History with shared Tabs and narrow labelled selectors. Keep the selected relationship, owner and status visible; use one form-request entry point per context. Due diligence retains its actual incomplete gates.

## Safety and recovery

Reads remain tenant/entity/actor scoped before limits. A submitted revision means received, not evidence accepted, review complete or activation allowed. Never loosen policy administration or infer an approval route. Historical revisions remain readable only under current permission; no body-supplied actor. No request sends, signatures, approvals or deletion occur as a consequence of changing tabs. Signatures remain form-field driven. Permission denial, temporary failure, stale results and a review not required are distinct states. Never replace an unknown score with zero or expose a restricted subject through a new label/link.

## Proof and tracking

Use #143 for Policies/Responses, #139 for vendor reads/navigation, #138 for persisted sample presentation and #147 for release evidence. #200 remains closed. Required fixtures cover submitted vs draft, no-score/no-review, pending review, denied/unavailable policy, active/suspended revisions, populated/empty history, long labels and exact response documents. Verify desktop light/dark, 390/320px, keyboard tab/selector navigation, resize state retention and 720px reflow. Retain before/after captures and fix the highest-impact rendered defect before completion. Run affected tests, copy-quality, full web/Go gates, PostgreSQL integration, independent reviews, exact-head CI and hosted smoke checks. SMTP connectivity remains advisory; security and application health stay blocking.
