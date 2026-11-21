> Historical design note. For the current implementation and limits, read `../DATA_AND_WORKFLOW_REVIEW.md` and `../TEST_REPORT.md`.

# Privacy record model and source notes

**Proposed product architecture, implemented as a local synthetic prototype.**

## Shared ownership, not parallel systems

`ProcessingActivity` enriches the existing processing records. It references organizational scope, people, optional program, bank controls and vendor engagement IDs. `DPIA` references an activity revision. Remediation actions use the existing task collection; privacy review/case items are read projections in Work. Evidence and requirement decisions reuse the original assessment/evidence/program identities.

Do not turn the new wizard into another general-form schema service. It is a domain-specific editing surface over structured processing records. Common field widgets can be reused without making the questionnaire the canonical privacy inventory.

## Processing activity

The prototype includes an accountable unit and owner; controller, processor or joint-controller role; applicable-regime label; description; subject and data categories; collection/source context; multiple purposes and their basis/reference; systems; recipients, roles, service engagements, destination and safeguards references; dataset-specific retention trigger, period and disposal; control links; notices and rights route; screening; next review; submitted/published/revision state.

Processor role prompts for the controller and contact; joint-controller role prompts for the responsibility arrangement. These fields illustrate role-specific requirements but are not a complete jurisdiction-by-jurisdiction statutory schema. Actual bank/controller contact directory, processors' Article 30 particulars, special-category conditions and legal basis vocabularies require reviewed locale-specific models.

A draft is mutable. A submitted proposal is not editable through the UI. A published snapshot is retained while a new proposal is reviewed. A conservative no-DPIA gate cannot be bypassed simply by leaving screening unknown. Publishing the inventory is not permission to start processing.

## DPIA

A DPIA retains processing context and activity version, necessity/alternatives, minimisation, consultation, DPO advice, harms to individuals, cause, initial/residual assessment, safeguards, implementation owner/due/evidence, external-obligation references and decision history.

The demo uses a configurable-in-production 5×5 model and an illustrative internal approval ceiling of 9/25. That ceiling is not a statutory definition of high risk. All relied-on measures must be evidenced as verified for this particular demo's approval gate. Real governance must support calibrated acceptance, escalation and jurisdiction-specific consultation/filing rather than equating a numeric threshold with legality.

Some Nigerian contexts are deliberately flagged for DPO assessment. This is a conservative design gate inspired by the reviewed GAID, not a complete codification, exemption system or legal opinion. South Africa and UK regime choices remain descriptive context, not fully implemented legal engines.

## Requests and incidents

Intake records scope even when activity linkage is unknown. The received date and incident-awareness date/time are separate. A target date needs an explicit basis. Investigation state and notification determination are separate. Response review requires a documented basis and an independent actor. No messages are sent, regulator files created or statutory deadline represented as automatically satisfied.

## Primary sources reviewed for the design

1. ICO, **Documentation**: processing records and their linked information, including purposes, sharing, retention and security.
   https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/accountability-and-governance/guide-to-accountability-and-governance/documentation/
2. ICO, **What do we need to document under Article 30 of the UK GDPR?**: distinctions between controller and processor records. The documentation guide above links the detailed guidance. UK-specific; not imported as universal banking obligations.
3. ICO, **Data protection impact assessments**: context, necessity/proportionality, assessment of risk and mitigation/consultation. Check current applicability and any guidance-under-review notice before production legal implementation.
   https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/accountability-and-governance/data-protection-impact-assessments-dpias/
4. Nigeria Data Protection Commission, **Nigeria Data Protection Act General Application and Implementation Directive 2025 (GAID)**, Article 28, including mandatory DPIA categories and the relevant filing provisions. The relevant source page was visually reviewed as well as the extracted text.
   https://ndpc.gov.ng/wp-content/uploads/2025/03/NDP-ACT-GAID-2025-MARCH-20TH.pdf

Source-informed structure does not establish compliance of the synthetic seed records. Regulatory interpretation, enforceable privacy/security controls and production acceptance remain separate work.
