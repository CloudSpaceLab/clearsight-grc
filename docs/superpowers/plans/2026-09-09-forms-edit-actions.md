# Forms edit actions implementation plan

> Use subagent-driven-development for the independent demo policy task and staged review. Continue without redundant approval within the user-authorized correction scope.

Goal: Direct, understandable form editing for permitted users, including the CRO demo account.
Architecture: Existing server operations, immutable revisions, shared table/sheet/buttons, and scoped versioned demo ROLE policy.

- [x] Add failing UI tests for direct row edit, current-version save and unavailable/denied authority. Update TemplateLibraryTable, FormsWorkspace and TemplateDetailDrawer; consolidate edit predicates and concise version/scoring copy. Keep pending review locked and preserve keyboard/selection.
- [x] Add a narrow demo Forms author policy and role bindings for CRO + Program Owner using existing governance records. Preserve foundation checks, approval separation and production defaults. Test actual database resolution, seed idempotence, unauthorized roles and unchanged non-Forms commands.
- [x] Update affected tests, copy regression, DESIGN and canonical fixtures/harness. Render before/after affected library/detail states at desktop/mobile in both themes. Review spec then implementation.
- [ ] Run affected tests, full required CI, merge and deploy main. Verify hosted CRO opens an existing form directly; only save clearly labelled sample data if needed for verification.

Local verification: 52 affected workflow/copy tests, 33 App tests, 51 contract checks, TypeScript compilation, 54 focused rendered states, four tablet checks and all 75 Forms scenarios passed. Real PostgreSQL authority/governance approval and mixed-scope candidate tests passed; independent review cleared the corrected scope checks. Deployment remains pending the release gate.
