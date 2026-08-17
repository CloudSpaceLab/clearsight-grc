# Monitoring setup and risk scoring acceptance

Status: implemented and verified on 17 August 2026.

## Supported operating path

The Programs workspace supports this mobile-banking use case without API calls or JSON editing:

1. Create a Channel Program with owner, jurisdiction and plain-language scope.
2. Add the live face-verification and password-reset requirements.
3. Build a five-question Yes/No password-reset form, including question weights, risk values and critical answers.
4. Submit the form for approval and activate it through a different demo account.
5. Create and independently approve the form Monitoring Check.
6. Create a dated collection request on demand. The action collects form data; it does not schedule automatic weekly request creation.
7. Submit the response through the existing Evidence Request capture flow.
8. Review the automatically calculated score, risk band, coverage and failed question in the Program.
9. As a GRC administrator, connect a public HTTPS JSON status endpoint, inspect its fields, choose one field and its expected value, and save a connected-data check.
10. Independently approve the check, run it on demand and review its result in the Program.

## Executable result semantics

- Form evaluation uses the immutable Evidence submission and exact Form Template and Monitoring Check versions recorded on the request.
- Yes/No answer scores are deterministic. Missing required scored answers produce `Not assessed`; they do not become zero risk.
- A critical answer produces a Critical band even when the weighted percentage is below the Critical threshold.
- Connected data evaluation requires one complete record and the exact selected field. Partial, stale, missing or unavailable input is not assessed.
- Results are insert-only and retain input identity, evaluator version, timestamps and source or submission provenance.
- Results do not synthesize an approved Evidence Assessment or declare the Program compliant.

## Browser acceptance evidence

The local stakeholder flow was exercised with separate CRO, CCO and GRC Administrator accounts:

- Mobile banking Program created with two requirements.
- Password reset form created with five scored questions.
- Form and check approved by an account other than the submitter.
- One sample `No` response produced `20% risk`, `Critical`, `100% coverage` and identified the failed question.
- A public GitHub repository status response was inspected as a sample JSON endpoint; the selected `archived = false` condition produced `0% risk`, `Low`, `100% coverage` with a retained receipt.
- Account switching was verified after login and on a 375-pixel viewport.
- The Programs layout had no horizontal overflow at 375 pixels, and the account switcher remained in the context bar instead of covering work.

The public GitHub response proves the generic endpoint and field-comparison path. It is not evidence that a bank has deployed a face-verification SDK. A real deployment must connect the bank-owned mobile-channel status endpoint and use a monitoring statement and field that directly evidence the requirement.

## Deliberate boundaries

- New public endpoints require configuration access; the Programs UI directs other users to a GRC administrator before any source draft is created.
- Browser-managed secrets are not supported. Credentialed sources use deployment-managed secret references through the existing Source Access boundary.
- Collection requests are created on demand. Recurring schedules are not implemented.
- Adverse results are shown for review. Automatic Matter creation requires separate policy-governed automation and is not implied by this workflow.
