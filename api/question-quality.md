# Question expiry and approved-source validation

A form field can set `answer_validity_days` to an integer from 1 to 3650. Omission preserves the existing no-expiry behavior. The approved form revision is copied into its request. Submission derives an immutable `answer_provenance[field_id].expires_at` from the server submission time in UTC, with PostgreSQL microsecond precision. Each answered question has its own expiry. Files attached to that answer share its expiry. A later form edit applies to future requests.

A scalar-answer field can set the following reference.

```json
{
  "validation_binding": {
    "binding_id": "approved-binding-uuid",
    "binding_version": 1,
    "value_field": "employee_id"
  }
}
```

The request resolves that exact active Source Registry revision and verifies its tenant and legal-entity source scope. Submission rechecks source availability and uses the existing bounded LOOKUP operation. The submitted scalar is the lookup key and must equal the selected returned value. File, grid and multiple-answer questions cannot use a scalar lookup.

The existing immutable `answer_provenance[field_id].validations` stores the source ID, binding ID/revision, result, observed time and operation receipt. CURRENT records a source match at submission. NOT_FOUND records a missing or mismatched value. STALE, UNAVAILABLE, PARTIAL, AMBIGUOUS, SCHEMA_DRIFT and INVALID retain distinct source-quality outcomes. A lookup failure preserves the submitted answer and its failed-check provenance for review. It does not grant an evidence review or a compliance conclusion.

`GET /api/v1/forms/validation-bindings/{binding_id}?version=1` returns the active lookup's identifier, revision, name, operations, selected fields and single key. It requires verified tenant/entity identity and an active source in that entity. It exposes no connection, credential, source rows or configuration. Source Registry administration and its lifecycle remain with the existing configuration services.

The Program assessment view shows each configured expiry and saved source result. The document inventory also returns `answer_expires_at` for the exact submission/field/artifact occurrence. Older responses retain their existing provenance without inferred expiry or validation.

## Local verification, 18 November 2026

The local sample CSV contains only the fictional identifier DEMO-EMP-017. Its connection/view/binding are explicitly prepared non-production fixtures because Source Registry activation has no exposed lifecycle command. The questionnaire went through the actual owner draft and independent Internal Auditor approval path, reaching revision 4. Evidence Respondent saved and submitted two requests. The first matched DEMO-EMP-017. The second retained an absent identifier and recorded NOT_FOUND. Seven-day and thirty-day expiry timestamps survived reload.

The focused unit suites cover matching, missing, stale and unavailable sources, source-config redaction, independent expiry and legacy no-expiry behavior. An isolated copied PostgreSQL database exercised approved form snapshots, draft recovery, immutable submission provenance, independent review, scope rejection and recurring request creation. The browser checks and screenshot evidence are recorded in the v2 acceptance checklist.
