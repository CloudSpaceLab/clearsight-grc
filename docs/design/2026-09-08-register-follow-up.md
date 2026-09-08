# Findings register to vendor follow-up

## Decision and before state

The supplied two-assessment XLSX produced 16 optional short-text questions from column headings. Five existing findings, recommendations and assessment dates were absent from the recipient preview; bank ratings were respondent inputs. The live before-state was inspected on 8 September 2026 in Imports as Program Owner.

Recognized XLSX registers now propose a finding follow-up, selected one numbered assessment at a time. Each finding retains read-only bank context in section help and question descriptions. Five answer fields collect a response, action or explanation, responsible person, optional proposed date and optional evidence. Mandatory response/action/owner are proposed defaults for maker and checker review. No scoring is inferred.

## Source and scope

Recognition requires the S/N, SERVICE PROVIDER, SERVICES OFFERED, FINDINGS and RECOMMENDATIONS headers. Other documents retain ordinary field conversion. Partial, truncated, orphaned, conflicting or oversized finding context fails visibly instead of silently truncating or falling back to blank questions. New XLSX extraction propagates only explicit vertical merge ranges on populated rows. Existing extracted registers use their numbered assessment boundaries with explicit author confirmation; this is not proof of a canonical vendor match. Placeholder names remain unchanged. The existing request sender selects the vendor relationship.

The proposal server requires one complete assessment and affirmative source review. Each selected group has an independently deduplicated child proposal and draft; retry reuses it, while another assessment can create its own draft from the original proposal. Generator identity separates earlier header-only receipts without rewriting their history. Migration 82 is required. Rollback refuses to discard distinct proposal history when the old unique index cannot represent it.

## UI states and acceptance

- Before selection: no recipient preview, draft creation disabled.
- Selected: only one assessment's findings and questions appear; source-confirmation checkbox enables draft creation.
- Selection change or version conflict: confirmation resets.
- Failure: retained source and recovery message remain available.
- Successful creation: exact draft link; use Imports again to select another assessment.
- Desktop preview and narrow stacked flow use existing form styles and shared SelectField.
- No form activation, delivery, bank rating change or finding closure is performed by conversion.

Required proof: five findings in 3/2 groups with original dates/recommendations; no bank rating inputs; bounded merge propagation; complete one-group acceptance; separate draft and retry behavior; preview isolation; copy regression and typecheck; light/dark desktop and 390/320px rendering. Verification results and remaining deployment limitations must be reported separately.

## Verification on 8 September 2026

The supplied workbook was extracted locally: EXTRACTED, six retained rows, five finding sections and 25 questions. Assessment 1 contains 15 questions and retains 13th February 2026; assessment 2 contains 10 and retains 6th February 2026. No source file was changed. Backend tests cover grouping, recommendations, source failures, explicit merge propagation, complete assessment selection, distinct drafts and retry reuse. Generation also recovers an already-completed competing worker result.

The five affected frontend suites passed 63 tests, including the copy-quality gate, document imports, proposal review, capture and vendor work. TypeScript, the production build and the runtime fixture boundary check passed. The Go document-import, monitoring and HTTP packages passed with the postgres build tag. PostgreSQL integration tests require the separate postgresintegration tag and a configured test database; these database tests were not run.

The isolated `web/evidence/register-followup.html` fixture uses the real proposal review component. Browser renders were inspected for selection and confirmation, isolated assessment preview, light mode at 390px and 320px, and dark mode at 320px and 1440px. Inspection led to a single-column follow-up preview, readable notice spacing and wrapping action controls. Narrow renders had no horizontal overflow; notices did not obstruct actions. Screenshots were inspected in-session, not retained as repository artifacts.

Migration 82 has not been applied to the running demo; its API and worker have not been restarted. Database migration and end-to-end draft creation in that running deployment remain unverified. No form was activated and no vendor invitation was sent. After deployment, Program Owner uses Imports to select and confirm one assessment, creates its draft and routes it for the existing Internal Auditor review before selecting the existing vendor in the request workflow.
