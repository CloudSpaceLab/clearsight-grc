# T2 binding reuse acceptance

## Scope

T2d/T2e reuse the existing versioned Source Binding catalog in capture, evidence and workflow flows. They do not add connector configuration, copied queries, source-row storage, a consumer-link authority table, another queue or another workflow model.

## Owning records

| Concern | Owning record | Durable content |
|---|---|---|
| Field use | `capture_requests.fields` | exact binding ID/version, use mode, selected scalar or internal evidence result, and operation receipt |
| Evidence search | `capture_requests.source_bindings` | exact binding ID/version, explicit subject/known-fact lookup selector, bounded result and receipt |
| Respondent truth | `capture_submissions.answers` | exact submitted answer, never overwritten by a source |
| Answer origin/validation | `capture_submissions.answer_provenance` | `SOURCE_PREFILLED`, `RESPONDENT_ENTERED` or `RESPONDENT_CORRECTED`, original source value/receipt and non-destructive validation result |
| Workflow context | `workflow_tasks.source_bindings` | deduplicated exact binding IDs/versions projected from the authoritative request; no source values, assignment or lifecycle authority |

## Behavioural guarantees

- Every configured consumer reference must resolve to the exact active Binding ID/version before request creation.
- `PREFILL` performs one bounded `LOOKUP`; only one complete, fresh selected scalar becomes an initial value.
- `OPTIONS` performs one bounded `PAGE`; only a complete, fresh 2–50 value set replaces the configured fallback options.
- `VALIDATE` performs a bounded lookup at submission and records `CURRENT`, `NOT_FOUND`, `PARTIAL`, `STALE`, `SCHEMA_DRIFT` or `UNAVAILABLE`. It does not overwrite or discard the respondent answer.
- Field/request `EVIDENCE` performs a bounded lookup before human capture and retains selected records plus the canonical operation receipt.
- Missing, stale, partial, ambiguous, drifted and unavailable states remain explicit. None is represented as current or sufficient evidence.
- Respondent request views expose only current `PREFILL`/`OPTIONS` provenance. Validation rules, lookup selectors, schema field names, request/field `EVIDENCE` rows and connector failures remain internal.
- Consumer records retain binding identity/version and operation provenance, not connection definitions, SQL, URL templates, credentials or mappings.
- Workflow projection carries only exact references and the existing evidence-request action target. The evidence request remains authoritative for source context and the workflow domain remains authoritative only for coordination state.

## Persistence and rollback

Migration `000033_t2_binding_reuse` is ALTER-only:

- `capture_requests.source_bindings` as a JSONB array;
- `capture_submissions.answer_provenance` as a JSONB object;
- `workflow_tasks.source_bindings` as a JSONB array.

Rollback removes only those consumer-owned columns. No shared source catalog or runtime state is destroyed.
