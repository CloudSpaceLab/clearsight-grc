# Forms and vendor release render receipt

The final [manifest](manifest.json) records **64 passing browser states** at 1440 and 390 CSS pixels in light and dark themes. Each state passed axe WCAG 2 A/AA and 2.1 AA checks, including contrast, with no horizontal overflow or application errors.

This is an isolated local evidence build at `http://127.0.0.1:4187`, built from HEAD `470ad3c2e90e84750c43985ddd835ee69be36d84` plus the integrated main changes and pending release corrections. The root task refreshed the final build after the policy contrast fix and vendor release fixture corrections. It is not hosted verification, a live vendor submission, or a compliance conclusion. The harness uses existing sample fixtures and a synthetic approved response-policy fixture; it sends no invitations.

Coverage includes the Responses portfolio and its Answers, Documents, Review and History sections; the policy editor, approved policy detail and permitted execution result; all five vendor sections; and selecting existing evidence with a reason. Reusing the sample test report changes **Missing 1 → 0** and **Awaiting review 1 → 2**, preserving the separate review requirement and unresolved applicability.

Rendered inspection found one contrast defect: a broad policy-heading `span` selector changed the Create policy button text to muted text, producing 1.06:1 light and 1.33:1 dark contrast. The selector now targets only heading support text. The before-state [light](before-policy-contrast-light-1440.png), [dark](before-policy-contrast-dark-1440.png) and [measurements](before-policy-contrast.json) are retained; the final matrix passes after correction. The eight affected policy component tests also pass.

Visual inspection covered the corrected policy button, mobile response review, evidence reuse dialog, and the checklist's separate counts. Mobile sections use the existing compact select; response review remains in a focused sheet. No new tokens or variants were introduced.

To reproduce from the repository root after building and serving the evidence entry:

```powershell
$env:PAGE_URL = 'http://127.0.0.1:4187'
node docs/evidence/2026-09-09-vendor-release/forms/capture-release-forms.mjs
```

Set `PLAYWRIGHT_MODULE` to a locally installed Playwright package if the bundled Windows runtime is unavailable. Earlier harness-only failure screenshots are excluded from the final manifest and release proof.
