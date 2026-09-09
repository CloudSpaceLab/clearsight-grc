# Vendor risk register migration — local acceptance

Date: 9 September 2026. Maturity: implemented for reviewed migration, locally verified; not deployed. [Decision](../design/2026-09-09-risk-register-migration.md).

## Outcome

An extracted register can become vendor-deficiency issues with individual planned actions after confirmation of existing vendor/service relationships and eligible internal owners. Names are normalized once per import. Unique exact full-name or first-name matches are suggested, never guessed between namesakes. Vendor responsibility retains a bank owner and the mapped relationship. No external invitations are sent and no historical status or risk rating becomes a verified current conclusion.

The supplied workbook was read without modification: two assessments, five findings and the five 31 March 2026 timelines were recognized. Merged assessment values and blank continuation cells are supported. Only synthetic workbook content appears in committed fixtures. Original artifacts remain unchanged; source-row facts preserve extracted cells, including expanded merge context.

## Verification

- Domain/HTTP/composition tests cover source completeness, continuation boundaries, distinct issues/actions, normalized responsibility names, illegal source IDs, scope, missing identity, stale source/vendor versions, denied/revoked action and import routes, excluded assessments, duplicate receipts and post-save response truth.
- Disposable local PostgreSQL 18 applied the full current migration chain, including `000088`. Integration tests check canonical findings/actions/vendor links, outbox events, immutable migration history, identical receipt replay and complete rollback when a later link fails.
- UI tests cover deterministic suggestions, ambiguity, required bank ownership, assignment confirmation, save-before-import versioning, excluded unmatched assessments and retry without another save. TypeScript and affected Imports/copy-quality regressions are included.
- [Rendered evidence](../evidence/2026-09-09-risk-register-migration/results.json): six component states and the integrated Imports workspace × light/dark × 1440/390px (28 renders). The capture script checks WCAG 2 A/AA automated rules, horizontal overflow, secondary-analysis disclosure and successful interrupted-import retry. Desktop and mobile/light and dark renders were visually inspected. Automated checks are not a product-wide accessibility certification.

Reproduce with `go test ./internal/registermigration ./internal/documentimport ./internal/continuity ./internal/thirdparty ./internal/httpapi ./cmd/api`; PostgreSQL integration requires a disposable migrated `TEST_DATABASE_URL` and `go test -tags "postgres postgresintegration" ./internal/registermigration`. The optional `TestRiskRegisterLocalSample` reads `CLEARSIGHT_REGISTER_SAMPLE`; default CI uses synthetic fixtures. Web checks: `npx vitest run src/registerMigration.test.ts src/components/imports/RiskRegisterMigration.test.tsx src/components/DocumentImportWorkspace.test.tsx src/copyQuality.test.ts`, `npm run typecheck`, `npm run build`. Render with the evidence Vite server on port 5189 and `node scripts/capture-register-migration.mjs`.

## Rollout and limits

- Apply migration `000088` and verify the bank's effective import/save, issue ownership, action-creation/performer and vendor-link routes. No hard-coded approvers or new route policies are installed.
- Up to 100 findings per import; complete extracted spreadsheet rows and the recognized source headings are required. Ambiguous dates remain blank for review. Search is bounded; uncertain vendor matches require selection.
- An identical file in the same legal entity reuses its receipt. Changed files are new imports, not updates or semantic deduplication against pre-existing manually entered findings. Review such files for overlap before importing.
- Excluded rows remain excluded from that file's finalized migration receipt. Later additions require a separately reviewed source containing the outstanding findings.
- Draft choices persist on explicit save or import. No protected draft content is stored in browser storage. Unfinished edits within an application route change are not automatically saved.
- Imported issues still require present-day status review, evidence assessment and normal outcome verification/closure. Generic vendor responsibility is an association and coordination action, not an externally distributed work request.
- Production-volume load tests, bank-user task timing, live tenant authority validation and deployment remain rollout work. Local evidence does not establish those outcomes.

## Cloudspace OEM form response seed

The non-production reference installer reuses the exact Cloudspace Technologies Ltd / OEM relationship. It supersedes only the user-confirmed `Vendor security and privacy review` request on `VENDOR-DUE-DILIGENCE` revision 3, retaining the original distribution and its supersession event. The replacement contains a submitted, unreviewed sample response with the register's payment-data scope, ISO 27001/22301, VAPT, right-to-audit and expired PCI-DSS assurance gaps, including the 31 March 2026 target. Contact and subprocessor details are explicitly identified as sample assumptions; no certificate, compliance conclusion or issue closure is created.

Installer retries require both the fixture's idempotency receipt and the persisted supersession lineage when a legacy request was replaced. A pending replacement is resumed, while an existing submitted response must have one current unscored revision and the exact seeded answers. Without a confirmed legacy request, the receipt-backed standalone sample is validated. A user-owned, ambiguous, altered or out-of-scope replacement fails closed. PostgreSQL integration coverage is run in CI with the configured disposable database; local unit coverage does not establish a deployed demo result.
