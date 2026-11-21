# Vendor onboarding in V2.10

## What was missing

The earlier Add engagement route created a supplier/service record through a generic setup wizard. It captured an assurance-plan label but did not launch named questionnaire assignments, document requests or a reviewed intake. The V2.9 lifecycle review completed periodic review, activation, suspension and exit; it did not supply the first-time onboarding journey. The old public `newVendor`, `vendorPortal` and `activateEngagement` functions also remained callable after their visible actions had changed.

## Completed journey

**Vendors → Onboard a vendor** opens a four-step intake. The Onboarding view gives saved drafts, assigned intake decisions, active diligence dossiers and earlier service records that still need a dossier.

1. Select an existing legal supplier or record a new one. Capture a named external respondent, role and email. Duplicate legal names/registration references are flagged; existing supplier/service/location combinations cannot be recreated.
2. Define the service boundary, business purpose, exact organizational location, accountable internal owner, internal response lead, independent risk reviewer, criticality and rationale, data context, proposed start and periodic review date.
3. Set applicable baseline requirements, pinned questionnaires, named respondents, independent reviewers, due dates and required documents. Exclusions retain a reason for review. The vendor questionnaire and baseline response are required. Contract and continuity documents are required for every service; high/critical services require assurance; personal or unclassified data requires processing terms.
4. Submit the intake. The owner confirms the business scope. The assigned risk reviewer independently approves the plan or returns it to the original preparer. Corrections create another submitted intake revision and retain the earlier snapshot and review receipts.

Drafts save in the current workspace as fields are edited and on explicit navigation. Creation of a draft does not create a supplier or service record. Submission creates the onboarding record. Approval of the diligence plan creates a service-specific program, real named assessment assignments and document requests. It leaves the service in Due diligence.

## Assessment integration

The module calls `CLEAR_SIGHT_ASSESSMENTS.assign` for the canonical requirement response and every selected pinned form version. Each assignment records its program, exact engagement target, respondent, internal coordinator where applicable, reviewer, deadline and approved-intake origin.

The default external assignment records the named supplier respondent separately from the bank coordinator who captures the supplied response. The assessment workspace retains the actual submitting actor, source/capture reference, submitted answers, response cycles and independent review. The onboarding dossier links to those exact assignments and review pages.

`CLEAR_SIGHT_ASSESSMENTS.status` supplies readiness. A draft, submitted response, returned response or newly opened response cycle does not count as accepted/current. Requirement conclusions remain their own reviewed program decisions; questionnaire acceptance does not automatically set them to Satisfied.

## Documents and approval

There is one canonical Documents store. A requested document references a `documentVersionId` and an exact `approvalRequestId` from the existing file approval workflow, together with named request ownership, planned reviewer, deadline, validity and version history.

The coordinator can upload through Documents or link a current version. Linking alone leaves the item at Review required. Request document review opens the existing approval preparation screen with the planned reviewer, deadline and dossier return route. The existing process verifies source bytes and prevents requester/contributor self-approval. The created request ID is linked back to the checklist.

A previously completed exact-version approval may be explicitly reused with an applicability explanation. Its real reviewer and decision record remain visible. File approval satisfies the document checklist item only; it does not create a requirement conclusion. Missing original bytes, archived/out-of-scope files, changed versions, expired validity, mismatched snapshots or missing decision receipts prevent readiness.

## Activation and lifecycle

Activation and resumption recheck all four conditions:

- An independently approved onboarding plan.
- A current accepted response for every named assignment.
- Every required document current and independently approved by exact version.
- Current Satisfied outcomes for every applicable requirement in the linked assurance programs.

Submitting a new intake requires current vendor submission authority at the exact location. Existing service ownership is an explicit preparation entitlement, but active denies still override it. The existing V2.9 lifecycle request retains this source basis. Its reviewer must act under the assigned current identity and have current service-approval authority. A different readiness basis at decision time requires a revised request. Creating a dossier, approving a diligence plan, submitting a response or attaching a file does not activate a service.

Existing active services retain their lifecycle history. Earlier Onboarding/Due diligence services can complete the missing dossier without changing their supplier identity or organizational location. Legacy creation/setup/submit/resume, response-portal and activation entry points route through the completed flow.

## Work and Inbox

Owner confirmation, risk review, returned intake and unfulfilled document requests project into Work from the source dossier. Questionnaire and file-approval work use their existing authoritative stores. Internal coordinators receive supplier document obligations, so external names do not create orphaned internal work.

Intake notifications retain dossier, revision and stage. Returned work remains actionable during draft correction and resolves on resubmission. Document-request notifications resolve when the exact document obligation becomes ready. Reading or acknowledging a notification does not change its source.

## Module and integration

New files are `assets/vendor-onboarding.js` and `assets/vendor-onboarding.css`. Load after `workflow-completion`, `flow-polish` and `assessment-tracking`; bootstrap remains owned by the package. New routes are `vendor-intake`, `vendor-onboarding` and `vendor-requests`. The module owns `S.vendorOnboarding` and references existing engagements, programs, assignments, documents, approval requests, decision receipts and notifications.

The public `CLEAR_SIGHT_VENDOR_ONBOARDING` interface exposes `start`, `saveDraft`, `submit`, `decide`, `correct`, `dossiers`, `readiness`, `linkDocument` and `requestDocumentReview`. No real external invitation or message is sent by the local preview.

## Verification

Run `node tests/vendor_onboarding_v210_checks.js`. Results are recorded in `tests/vendor-onboarding-v210-results.json`. The suite executes actual-source commands for draft/identity/scope handling, duplication, stage review, assignment creation, return/resubmission, legacy adapters, canonical file approvals, document readiness, activation and registered route actions.

The original V2.9 operational regression fixture is archived at `tests/history/workflow_v29_checks.pre_v210.js`. Its current V2.10 copy uses the actual named preparer/reviewer identity for vendor decisions and retains the original assurance, independent decision, exit and returned-correction assertions. Run it with `node tests/workflow_v29_checks.js`.

The harness substitutes the browser shell. Browser layout, accessibility navigation and file-upload interaction have not been verified in this environment.
