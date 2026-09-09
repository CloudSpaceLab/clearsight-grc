# Vendor workflow release verification

The user authorized integrating the vendor evidence and UI corrections with main and deploying through the existing demo pipeline. This receipt records local verification; the final main/deployment status is verified separately against the exact release SHA.

## User journeys

- Existing vendors: upload a checklist or historical register, review source-linked field proposals, edit scope and document questions, obtain ordinary form approval, then choose the active form for a reassessment. Match historical rows to the correct vendor/service manually; internal actions stay separate from vendor questions. Existing submitted documents can satisfy collection after explicit reconciliation, with an independent evidence decision.
- New vendors: enter vendor/service details and choose criticality and privacy role explicitly. Use an approved form to collect the remaining information. The starter accepts either an assurance document or a required missing-assurance explanation. Yes/No choices reveal the correct fields. A missing declaration does not approve the vendor.
- Organizations can revise question wording, field type, requiredness, applicability, document constraints, reviewer responsibility, scoring and policies through the existing governed Forms tools. Imported NDPA rows remain source material requiring applicability review. Conditional conclusions remain governed by the organization's policy and mandatory blockers.
- Deployment upgrades only the exact shipped legacy starter through a new draft, submission and independent approval (v3 to v6). Current principals and authority are verified. Customized/unknown families, original versions and issued-request snapshots are preserved; interrupted upgrades resume through the same guarded commands.

## Verification

- Full frontend suite before the final focused corrections: 1,209 tests passed. Subsequent vendor review/navigation tests: 85 passed; starter capture, proposal and copy regressions rerun for final edits.
- Full Go suite passed on the integrated source. Separate PostgreSQL 18.6 evidence, third-party and HTTP suites produced 1,236 passing test events including subtests; the HTTP fixtures use in-memory composition, while evidence/third-party tests exercise PostgreSQL.
- All 99 integrated migrations applied through the real deployment ledger; rerun was a no-op. Reconciliation migration 000086 reversed/reapplied on an empty schema while preserving main's 000085 history index. This is not a production data rollback test.
- Real PostgreSQL starter-upgrade tests passed for legacy successor creation, preserved issued requests, customization, interrupted/repeated upgrade, denied authority and inactive checker; the existing document-sample rerun also passed.
- TypeScript, production/evidence builds and 15 runtime/UI/contrast contract tests passed.
- [Forms render receipt](../evidence/2026-09-09-vendor-release/forms/README.md): 64 desktop/mobile and light/dark states, including vendor sections, review, policies, evidence selection and receipt review counts. The policy button contrast defect was corrected and rechecked.
- [Onboarding/proposal render manifest](../evidence/2026-09-09-vendor-release/onboarding/manifest.json): 16 states, with no horizontal overflow or page errors. Interactive Yes/No toggling reveals document/gap controls. A 47-row mobile proposal fell from 22,148px to 1,279px after containing the review region; full source fields and preview remain available.
- [Actual spreadsheet verification](2026-09-09-spreadsheet-vendor-proposal.md): 47 NDPA draft fields and five register follow-ups, exact source rows, original historical dates, no inferred vendor match or compliance conclusion. Synthetic fixtures are committed; private workbooks are not.

The Linux-only deployment verifier suite cannot run correctly under native Windows Python/WSL launch stubs; the release CI runs it on Ubuntu. No real vendor invitations, email or private sample uploads were sent. Local screenshots use explicit sample fixtures. Production object storage, scanning and legal compliance remain the documented deployment/product boundaries.
