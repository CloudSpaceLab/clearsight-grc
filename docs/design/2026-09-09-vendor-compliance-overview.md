# Vendor compliance overview

User request: show submitted response gaps, incomplete forms, outdated responses and overall compliance health in a compact vendor overview. Include a default Third Party Risk Compliance form aligned with the supplied sample register.

## Decision

The user's clarification makes per-requirement failures the primary outcome: any form they create from the sample and send to a vendor must expose failed configured checks as soon as the vendor submits it. This applies to ordinary custom forms, not only the default sample. Automatic findings remain visible while independent evidence review is pending. UI labels are Missing, Expired, Not met and Awaiting review.

Replace the overview's navigation-heavy opening with one Compliance section: a prominent supported status, compact counts for Incomplete, Awaiting review and Outdated, and a paginated list of forms with specific missing or expired evidence and configured assessment gaps. Open the existing response review or request from each row. Keep vendor identity and edit controls under Vendor details. Reuse existing UI tokens, buttons, badges and sheets.

Use the existing scoped vendor forms page and aggregate summary. Extend rows with labelled attention items from the exact submitted form/response and validated evidence, and explicit outdated state. Reconciled current evidence is received, not missing; a submitted negative answer can identify a gap without making the form incomplete. Request/access expiry is not response aging. Use stored document expiry, response replacement and applicable recorded renewal dates. Unknown freshness or review coverage cannot become a favourable compliance claim.

The overview is scoped to this vendor service and authorized records. A completed questionnaire is separate from an assessment conclusion. Show the latest due-diligence conclusion with its date; pending work or stale evidence remains visible alongside it. Do not average scores or invent an overall compliance percentage. If no assessed result exists, show Not assessed. If reads fail, show Unavailable with Retry rather than zero counts.

## Default form

The supplied workbook's 2026 Register contains five open findings across two anonymized service groups: ISO 27001, VAPT, contractual audit rights, ISO 22301 and expired PCI DSS. Seed a distinct reusable Third Party Risk Compliance form through the existing governed sample installer; do not alter an existing customized form or import provider identities, assessor decisions or Medium ratings as facts. Applicability and evidence questions remain configurable through Forms. Permit respondents to declare missing evidence and submit the questionnaire; review, adverse rules and required remediation remain separate. Clearly label sample policy choices and provenance.

On narrow screens, the selected vendor uses Back to vendor register in place of the repeated Vendors/Add vendor header; adding a vendor remains available from the register. The page heading stays available to assistive technology. Desktop keeps the compact header and Add vendor action.

## Alternatives and proof

Rendered review found the existing selected-vendor hero and service banner pushed failures below the initial viewport. The selected header therefore uses a single compact Vendors heading, vendor identity, a plain service summary and accountable owner. Overview work counts are inline and bold; repeated service context, observation date and assessment coverage follow the response gaps. Narrow layouts reserve intrinsic width for status badges, and names wrap in the remaining space. Unknown freshness displays Unknown rather than claiming an already reviewed form was never assessed.

Adding another dashboard or score duplicates existing assessment services; merely linking to Forms leaves the current discovery problem. The chosen summary opens the existing workflows.

Required fixtures: empty, submitted with gaps, incomplete, current reused evidence, expired evidence, partially replaced response, awaiting review, satisfactory, conditional, adverse, unavailable, paginated and restricted population. Verify desktop, 390 and 320 CSS pixels in both themes, keyboard access and contrast. Preserve the supplied screenshot as the before-state reference. Test exact scope, no favourable status from incomplete/unassessed data, expiry boundary, reuse, pagination and existing response-review handoff.
