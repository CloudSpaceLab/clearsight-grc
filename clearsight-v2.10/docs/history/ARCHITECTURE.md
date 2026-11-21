> Historical design note. For the current implementation and limits, read `../DATA_AND_WORKFLOW_REVIEW.md` and `../TEST_REPORT.md`.

> Retained module reference from the preceding prototype. V2.4 workflow changes and actual delivery limits are documented in `../README.md`, `UX_REVIEW.md`, `PRIVACY_MODEL.md` and `TEST_REPORT.md`.

# ClearSight V2 — Architecture expressed by the prototype

## Ownership and traceability

A framework version owns requirement definitions and the default assessment instrument. A program adopts a pinned version and owns scope, accountability and tailoring. The source definition is never overwritten by a bank-specific edit.

```text
FrameworkVersion
  RequirementVersion[]
    QuestionTemplateVersion[] ← shared question IDs are reused
      DefaultQualityPolicy

Program
  FrameworkVersion reference
  Target[]: bank | internal unit | vendor engagement
  Requirement references
    applicability + exclusion rationale
    tailored expectation (optional)
    respondent override (optional)
  Question policy overrides
  accountable owner + default respondents + independent reviewer
  schedule configuration

AssessmentRun [program + target + form/question versions + cycle]
  Response observations → Scoped evidence → Reviewer decisions

RequirementEvaluation [program + target + requirement + run]
  conclusion + reviewer + observed/reviewed timestamps
  evidence references + rationale

VendorIdentity → VendorEngagement → ProgramTarget
BankControl ↔ FrameworkRequirement
BankControl ↔ Risk
Risk → RCSA proposal → Independent review → Effective residual rating
Risk → LossEvent → RecoveryLedger → Derived net loss
```

The demo uses one draft and one assessment run per program target for simplicity. Production needs explicit immutable run identifiers, multiple cycles, approved-version snapshots, locks and optimistic concurrency. The prototype's flat arrays and hash routes are not the proposed database or API design.

## Where mappings belong

- Program-to-requirement references define the complete obligation population, including unassessed requirements.
- Question-to-requirement links define the assessment method; shared questions are deduplicated within an assessment target.
- Implemented-control-to-requirement mappings describe how the bank intends to satisfy its obligations. A mapping is not proof of satisfaction.
- Evidence links include program, target and question. Similar document names never authorize reuse across scopes.
- Vendor identity and the specific service engagement are separate. An assurance result does not automatically become bank compliance.

## Calculation contracts

**Applicable population:** included requirements minus explicitly justified exclusions, per target. Unknown and unassessed obligations stay in the denominator.

**Current evaluation coverage:** current Satisfied, Partially satisfied or Not satisfied evaluations / applicable population. Submission completion is not evaluation coverage.

**Satisfaction:** Satisfied evaluations / applicable population. Partial findings receive no invented fractional score in this prototype. No misleading average of bank and vendor populations is shown.

**Quality:** evidence review status and freshness are separate dimensions. A historically accepted document can be stale. A failed source query yields Unknown, not a zero or pass. The demo evaluates against its fixed snapshot date.

**Risk:** current residual likelihood × impact uses an explicitly accepted assessment. Control effectiveness is not converted to an arbitrary percentage discount. A first-line proposal or challenge leaves the effective rating unchanged.

**Loss:** gross and individual recovery entries are integer minor-unit values. Net = gross − sum(received recoveries). Near misses contribute zero. The occurrence filter changes the event cohort, not the meaning of the recovery basis. Currency conversion is outside scope.

## Three-step adoption contract

1. **Requirements.** Scope first, preserve the pinned canonical source, display mapped questions, tailor the program expectation or document non-applicability. Keep an explicit population rather than deriving it solely from the questions present on a form.
2. **Data quality & monitoring.** Override maximum age, evidence expectations, verification adapter and cadence at the question configuration. Observation timestamps and verification outcomes belong to observation records. Do not call these validation rules operational bank controls.
3. **People & publishing.** Set accountable owner, one or more respondent units, reviewer, due date and recurrence configuration. Requirement-level respondent overrides inherit from the program by default. Review the exact target × requirement population, then publish as Not assessed.

## Production boundaries that must be preserved

Use server-enforced tenant-scoped foreign keys, access checks and explicit targets. Vendor portal authorization must exclude unrelated bank, supplier and service records. A browser role dropdown is never authorization.

Publish and review commands should validate version IDs and permissions, be idempotent, use concurrency tokens and create durable audit records. Approval should retain the exact response, evidence and policy revisions inspected. Exclusion acceptance and tailoring may require policy-specific approval, beyond this prototype’s simple publish review.

External verification needs tenant-scoped credentials, a server-side adapter, outbound-request policy, observation provenance, bounded retries/timeouts and failure classification. The browser must not store API credentials. A passing check can support a conclusion, but must not silently certify every mapped requirement.

Re-evaluate freshness in server-backed current-posture projections. Do not mutate historical accepted evaluations when evidence expires. Show the current reliance state separately from historical authority. Include quality/coverage denominators in read models and exports.

Reuse the current ClearSight design system and source-backed reporting architecture. This demo's CSS tokens and local render functions are a design reference, not a replacement production design system or an invitation to duplicate existing components.

## Source review

The visual and interaction reference was reconstructed from recovered October 15, 2026 HTML sources: framework adoption, program workspace, assessment workspace, vendor workspace, operational risk, and the unified interface specification. The older raw ZIP packages could not be materialized, but their individually readable HTML references were recovered as text. The new files are a rewrite of those references with the adoption workflow from this conversation, not a patch or verified build of the production repository.
