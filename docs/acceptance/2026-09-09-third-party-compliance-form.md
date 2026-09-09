# Third Party Risk Compliance sample form

The [vendor compliance overview decision](../design/2026-09-09-vendor-compliance-overview.md) adds the reusable `THIRD-PARTY-RISK-COMPLIANCE` form through the existing non-production reference installer. It is a configurable sample policy, not a bank compliance conclusion or legal advice.

## Source and contract

The user-provided workbook's `2026 Register!A1:P6`, read on 9 September 2026, contains five open findings across two anonymized service groups. It supplies the five requirement topics: ISO 27001, ISO 22301, vulnerability assessment and penetration testing, contractual audit rights, and PCI DSS. The form does not copy vendor identities, assessor decisions, open-finding records, or the source's Medium ratings.

Each area asks whether the requirement applies. An applicable requirement asks for Current/Present, Missing, Expired where relevant, or Not met. Current/Present requires a PDF vendor document. Native document answers retain document type, reference, issuer, issue date and expiry date; the reviewer must check scope, dates and validity. Missing, Expired and Not met require an explanation and planned action but no invented upload. A respondent can submit these adverse answers. Non-applicability requires a written basis and independent review.

The advanced compliance profile gives each applicable status one direct contribution: 100 for Current/Present and 0 for a declared gap. All areas have equal effective weight. The default concern bands and these weights are sample choices that bank administrators can revise through Forms. They are not extracted ratings or official certification rules. The five section weights total 100; the mutually exclusive status and non-applicability review fields each have weight 50 within their section, satisfying the existing contract validator while preserving equal weight between visible areas.

Applicable status uses required `AUTOMATIC_REVIEW` with the existing REVIEWER route. Non-applicability uses required manual review. A positive reviewer outcome cannot remove an adverse automatic answer. Pending review leaves the assessed result provisional. Form publication uses the existing draft → submission → independent activation lifecycle; the maker cannot also be its checker. No assessment finding, remediation issue, evidence acceptance or vendor approval is seeded.

## Installation and preservation

The normal reference installer adds this distinct form to its reference Program, making it available in the reusable library. Existing codes and historical forms remain unchanged. The new form's lookup is exact on tenant, legal entity, Program and code, uses the existing Program-history index, and returns at most two distinct identities to detect duplicates. It does not rely on the first 100 revisions in the Program.

An existing matching form is preserved in every state, including a customized draft, pending review, pause or retirement. The installer does not resume or approve an existing draft. If installation stops after creating a draft, its assigned author and independent reviewer complete the normal lifecycle in Forms. A duplicate identity fails installation rather than selecting or replacing a form. Existing reference-journey entrypoints retain their non-production restrictions; no production authority or request identity handling changes.

## Verification

- `go test ./internal/bankverticals ./internal/monitoring -count=1` passes.
- `TestDefaultComplianceFormAllowsSubmittedGapsAndRequiresReview` normalizes the shipped form, checks five adverse automatic contributions, proves review remains provisional and cannot erase a negative answer, and successfully submits Missing/Expired answers through the actual evidence capture validator.
- Non-applicability remains unassessed before review; current evidence uses five conditional native document fields.
- Memory tests preserve a customized draft and a paused form beyond 110 preceding records. Exact lookup tests reject missing identity, wrong tenant/entity/Program/code, and ambiguous form identities.
- `go test -tags "postgres postgresintegration" ./internal/bankverticals -run TestPostgresDefaultComplianceForm -count=1` passes on PostgreSQL 18. The test creates separate maker/checker identities, activates the form at v3, pauses it, inserts 110 earlier codes, repeats installation twice, compares the stored form exactly, and verifies one identity. It also verifies tenant isolation and duplicate-code failure.

The submission regression exposed capture validation omitting the stored advanced score profile. That evidence-layer correction is part of the accompanying overview implementation. Browser and hosted deployment evidence are recorded by the wider vendor overview task; this receipt does not claim those steps are complete.
