# Finding follow-up defect repair acceptance

Scope: the four reproduced defects in historical branch `6ebe0746`, adapted onto main `2e35415b`. The historical branch is not merged. The [plan and decision brief](../superpowers/plans/2026-09-09-finding-followup-defects.md) define the bounded user flow and intentional direct-receipt design.

## Behavioral evidence

- Structured XLSX tests preserve newline/`Column 10:` cell content, process standalone numbered assessment headers, reject ambiguous blank identifiers, and retain each finding's responsibility rather than the first row's owner.
- Ordinary V2 row proposals remain the default. Selected follow-up generation creates five fields per finding and applies limits per selected assessment. Incomplete or contradictory sources fail closed; long help has a warning and complete source anchors.
- Independent review found formatted blank rows counted differently by metadata and retained extraction. A real-XLSX red/green regression corrects that false rejection while omitted or missing findings still block specialized generation.
- Monitoring tests cover distinct/retried assessment receipts, complete selection and confirmation, source version/digest change, scope mismatch, revoked authoring route, rejection, ordinary default behavior and two separately accepted drafts. Distinct UUIDs sharing a timestamp prefix have distinct draft codes.
- Real PostgreSQL 18.6 tests cover receipt persistence, duplicate acceptance from eight concurrent calls, one draft/event/outbox outcome, missing confirmation, source changes, tenant UUID/slug scope, outbox-failure rollback and retry. The local migration runner needed ordinal filename order to match the deployment script; locale sorting was not valid.
- Real PostgreSQL response tests now exercise both the same respondent and another authorized recipient amending a submitted response. Earlier answers remain immutable, exactly one response is current, score events/outbox persist and the workspace remains open. Held-evidence source/request revalidation tests also pass. The historical branch's broken request-status mutation is excluded, not weakened or ported.
- UI tests cover explicit assessment selection, confirmation/complete-field acceptance, default selection, pending-generation return and existing conflict/source recovery. All 1,253 frontend tests in 172 files pass, including copy quality. TypeScript, production and evidence builds, and all 52 runtime-boundary/UI-contract checks pass.
- The [browser manifest](../evidence/2026-09-09-finding-followup-defects/manifest.json) records four passing light/dark desktop/mobile runs (1440px and 390px), eight rendered captures, no page errors, no horizontal overflow and no axe violations. Each run verifies explicit assessment selection, a failed generation request and retry, source isolation, pointer and keyboard confirmation, and acceptance of all five fields. The renders were inspected. Existing source-row baseline evidence remains unchanged. CI runs this interaction proof alongside the existing UI review.
- The complete Go unit suite and `go vet ./...` pass. Full real-PostgreSQL monitoring and evidence package suites pass. Migration 000087 applies and rolls back/reapplies on an empty follow-up history; production-ledger rerun and exact-release gates remain assigned to CI.
- Specification and independent code-quality reviews approved the complete change after the blank-row and pending-generation recovery corrections. No actionable findings remain from those reviews.

## Release tracking

The release PR, exact-head CI and deployment receipts are recorded on #80 after those gates pass. This local acceptance record does not claim a pending deployment. A final fetch also inspected historical-branch commit `0a6c07a08` (vendor-assessment review/related-work changes); it does not modify the four importer/amendment defects and is not included in this focused repair.

## Boundaries

Synthetic fixtures are labelled sample data. No private documents, external recipients or real email are used. This tranche does not implement workflow-access ownership metadata, automated historical vendor matching, findings reconciliation/closure, or submitted-status migration. Existing source protection, held-document validation, signatures only when specified, independent approval and distribution access controls remain unchanged. #80 retains broader lifecycle work; #200 stays closed.
