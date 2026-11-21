# 2.9 vendor relationships and assessments: existing-contract implementation plan

**Status:** Plan for existing vendor relationship, assessment, evidence and activation operations. It does not yet specify new periodic lifecycle decisions, independent suspension/resumption or verified exit required by the reference.

**Goal:** Replace current vendor list, relationship and assessment presentation with 2.9 anatomy using existing scoped identities and decisions.

**Architecture:** `internal/thirdparty` owns canonical vendor identity, legal-entity relationship, assessment and activation. `internal/evidence` retains submissions, exact files and collection receipts; work/notifications refer to the source instead of duplicating it. Backend authority and source access govern every action.

**Stack:** React/TypeScript, Go, PostgreSQL, existing authority, capture and outbox services.

## Existing exact operations

Methods and paths exist in `api/runtime.openapi.json`. `thirdPartyActor` binds tenant/entity/principal from verified identity. Struct definitions below are in `internal/thirdparty/model.go`, `assessment_model.go`, `activation_policy.go` and `vendor_work.go`; the handler remains authoritative for additional wire validation.

| Method/path | Request and result |
| --- | --- |
| `GET /api/v1/vendors?search=&cursor=&limit=` | Bounded scoped `RelationshipPage` `{items,next_cursor?}`; each aggregate has `vendor`, `relationship`, `brand`. |
| `GET /api/v1/vendors/{id}` | Exact legal-entity relationship aggregate. |
| `POST /api/v1/vendors` | `CreateRelationshipInput`: `{existing_relationship_id?,legal_name,trading_name?,registration_ref?,jurisdiction?,source_id?,external_ref?,registered_address?,website_domain?,service_name,criticality,privacy_role,effective_from?,renewal_at?}`. Existing relationship reuse is validated server-side; business owner is not chosen by a forged actor field. |
| `POST /api/v1/vendors/{id}` | `{expected_version,service_name,criticality,privacy_role,effective_from?,renewal_at?}` updates relationship, not vendor identity. |
| `PUT /api/v1/vendor-identities/{vendor_id}` | `{expected_version,legal_name,trading_name?,registration_ref?,jurisdiction?,registered_address?,website_domain?}` updates separately versioned vendor identity under current authority. |
| `POST /api/v1/vendors/{id}/assessments` | `{relationship_version,review_kind?,source_trigger?,restart_assessment_id?,form_template_id,form_template_version,scope_kind?,selected_field_ids?,review_due_at}`. Result is existing assessment setup, not an approved conclusion. |
| `GET /api/v1/vendors/{id}/assessments/current` and `GET /api/v1/vendor-assessments/{id}` | Exact scoped current assessment and review. |
| `POST /api/v1/vendor-assessments/{id}/prepare-request` | `{expected_version,audience,deadline}` → `{assessment,request,state:"PREPARED"}`; no invitation is issued. |
| `GET /api/v1/vendor-assessments/{id}/collection` and `POST /api/v1/vendor-assessments/{id}/collection/{field_id}/reconcile` | Reconcile uses `{expected_version,request_id,expected_request_version,source_submission_id,source_field_id,source_artifact_id,source_response_revision_id?,rationale}` → `{collection,receipt}`; independent document validation remains separate. |
| `POST /api/v1/vendor-assessments/{id}/complete` | `{expected_version,conclusion,uncertainty?,rationale,next_review_recommended_at?}` under the server's current review policy. |
| `GET /api/v1/vendors/{id}/activation` and `POST /api/v1/vendors/{id}/activate` | Activation input `{expected_version,intended_effective_at,rationale}`. Result contains eligible/gates, exact policy, relationship and optional receipt bound to policy, assessment and decision versions. |

The current relationship enum includes `PROPOSED`, `UNDER_REVIEW`, `ACTIVE`, `RESTRICTED`, `SUSPENDED`, `EXITING`, `TERMINATED`; enum values do not prove a reviewed command exists for each transition. Existing vendor-work prepare/send/review/change/accept/cancel APIs are recorded in the runtime inventory and `vendor_work.go`; preserve their independent capture and delivery semantics.

## Database and migration

For existing operations use `third_party_relationships`, versioned identity, `third_party_assessments`, exact capture links, document/review receipts and `third_party_activation_receipts` in migrations `000035`, `000037`, `000041`, `000042`, `000070`, `000086` and subsequent changes. No schema migration is required for the UI and existing activation/assessment flow. No additional vendor score, duplicate identity or copy of evidence is introduced.

The reference's periodic review, suspension, resumption and verified exit require a server-side transition/decision inventory before being exposed. If current services cannot enforce an independent decision and version-pinned evidence, add a narrowly owned `thirdparty` proposal/decision contract, atomic event/outbox and additive scoped migration in a separate plan. Never equate an enum value or unreviewed UI update with a lifecycle decision. Preserve legacy state and do not backfill historical reviewers or exit receipts.

## Implementation sequence

1. Add failing tests in `web/src/components/VendorsWorkspace.test.tsx`, `VendorDueDiligence.test.tsx`, `VendorEvidenceChecklist.test.tsx` for exact relationship deep link/history, scope reset, expired/returned assessment, explicit independent document review, and activation withheld when current policy or evidence changes. Backend tests in `internal/thirdparty/activation_policy_test.go` and `internal/httpapi` must assert reviewer conflict, forged actor, stale versions and no cross-entity metadata disclosure.
2. Capture the actual 2.9 `vendors.html`, `vendor-engagement.html`, `vendor-review.html` and hashed shared assets at 1440/1280/1024/768/390. Recompose `VendorsWorkspace.tsx`, `VendorRelationshipWorkspace.tsx`, `VendorDueDiligence.tsx`, `VendorWorkPanel.tsx` and scoped CSS with shared components. Preserve selected relationship, filters, exact response/version and history; clear/cancel when scope changes.
3. Use `vendorApi.ts`, `vendorAssessmentApi.ts`, `vendorCollectionApi.ts`, `vendorFormsApi.ts` and `vendorWorkApi.ts` against the existing DTOs. Use current `expected_version` and the server-returned operations/gates, never a client reviewer ID. Reuse eligible existing documents only after their exact source authorization and show freshness, unknown and separate bank review. Do not display a vendor compliance percentage from service criticality.
4. Exercise prepare → authorized send → actual submitted response → independent field/document review → current activation policy/gates → activation receipt. Do not send real email or invitations during tests; mock or isolate adapters and check persisted delivery receipts separately. Test stale artifact, revoked permission, rejected and unscanned file, setup retry and concurrent decision.
5. Run `go test ./internal/thirdparty ./internal/evidence ./internal/monitoring ./internal/httpapi`; with isolated migrated `TEST_DATABASE_URL`, run `go test -tags "postgres postgresintegration" ./internal/thirdparty ./internal/evidence ./internal/httpapi` and inspect skips. From `web`, run `npm test -- --run src/components/VendorsWorkspace.test.tsx src/components/VendorDueDiligence.test.tsx src/components/VendorEvidenceChecklist.test.tsx src/copyQuality.test.ts`, `npm run typecheck`, `npm run check:runtime-truth`, `npm run check:ui-contracts`.
6. Retain same-screen source/runtime renders and named review at all five widths and affected themes, keyboard/focus and denial/partial/error states. Reconcile exact source/form/document counts and relationship scope. Deploy a compatible frontend; rollback must preserve assessment, submission and activation history.

**Not complete:** New lifecycle proposal/decision APIs, exact transition policy and paired DDL/migration for any missing 2.9 actions are not yet specified. Do not present this as a full code-ready vendor-domain plan; no prescribed tests have been run.
