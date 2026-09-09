# Cloudspace vendor response seed

## Goal

Show a submitted, unreviewed sample response for Cloudspace Technologies Ltd's OEM service in the persistent non-production demo, using the supplied third-party risk register as the source of its declared assurance gaps.

## Problem

The hosted installer creates completed sample form responses for Paywave and ArchiveGuard, but not for Cloudspace. The existing Cloudspace OEM relationship therefore retains an empty vendor security and privacy request.

## Decision

Add one idempotent operating-form sample keyed as `cloudspace-oem-risk-register`.

- Resolve the exact Cloudspace Technologies Ltd / OEM relationship already present in the demo; create the same managed sample relationship only when it is absent.
- Use the active `VENDOR-DUE-DILIGENCE` revision and submit one external-audience response. Do not fabricate a certificate or mark assurance as current.
- Record the source-backed service scope, payment-data classification, ISO 27001 assurance gap, VAPT gap, contractual-audit-rights gap, ISO 22301 gap and expired PCI-DSS gap. Each retained gap includes the 31 March 2026 target and the supplied register's vendor or business comment where present.
- Mark non-source demographic values as sample assumptions in the submitted response. The response stays submitted and unreviewed; it does not create a compliance conclusion, assessment approval or closed issue.
- Re-running the installer validates the immutable distribution, source relationship, one current response revision and response state. It refuses to replace altered or user-superseded sample history.

## Alternatives considered

1. Seed the existing request directly. Rejected: a request has no stable fixture identity and could overwrite a user-created workflow.
2. Create two new service relationships for the spreadsheet assessments. Rejected: it would leave the visible OEM relationship empty and split a requested vendor response across records.
3. Create a separate risk-register form. Rejected: the active due-diligence form already accepts declared evidence gaps and independently reviewed follow-up.

## Validation

PostgreSQL integration coverage will prove the first installation creates the Cloudspace response, records all source findings, leaves it awaiting review, and is safe on repeat installation. The existing installer, focused web tests and hosted CI/deployment verification remain required.
