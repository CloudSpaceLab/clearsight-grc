# Vendor compliance overview implementation plan

Use subagent-driven-development for the independent backend/seed task and staged review; continue within the user's authorized fix, merge and deployment scope.

Goal: Make missing evidence, incomplete forms, outdated responses and supported compliance status visible in the selected vendor overview.

- [x] Extend existing vendor form rows with labelled attention items and outdated state using exact stored response, form and evidence data. Extend the authorized aggregate for outdated forms. Keep current evidence reconciliation, field visibility and restricted response access. Add unit and PostgreSQL regressions for mixed valid/expired/missing, submitted gaps, replacement and scope.
- [x] Seed the Third Party Risk Compliance sample through the governed existing installer. Cover ISO 27001/22301, VAPT, audit rights and PCI DSS with configurable applicability/evidence/review rules. Test definition validity, submitted negative answers, immutable customized form preservation and idempotence.
- [x] Add VendorComplianceOverview, integrating existing page/summary and response review. Test empty/error, status precedence, pending vs submitted gaps, existing evidence, stale data, pagination and handoffs. Move identity editing and optional identity facts into compact details.
- [x] Add required state fixtures; render and inspect desktop/mobile light/dark, keyboard and contrast. Update DESIGN and acceptance. Review backend and UI, fix findings, run affected local tests.
- [ ] Merge and deploy main under existing authorization; verify hosted CRO Forms editing and vendor overview/default form. Report exact deployment revision and remaining limitations.
