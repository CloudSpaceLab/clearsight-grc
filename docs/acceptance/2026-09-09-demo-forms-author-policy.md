# Demo Forms author policy acceptance

Scope: the persisted non-production stakeholder demo, following the [Forms editing decision](../design/2026-09-09-forms-edit-actions.md) and [implementation plan](../superpowers/plans/2026-09-09-forms-edit-actions.md). This is a managed reference configuration, not production default authority or a live bank approval.

## Configuration and impact

`CLEARSIGHT-DEMO-FORMS-AUTHOR:v1` adds three exact decision routes within the demo tenant and Nigerian legal entity. `forms.template.create` is scoped to that exact legal-entity object. `forms.template.revise` and `forms.template.transition` are scoped to Form Template objects. Each uses the existing `ACCOUNTABLE_OWNER` responsibility and a dedicated `DEMO_FORMS_AUTHOR` role, bound to the CRO and Program Owner positions with legal-entity scope and no general capabilities.

The transition command includes submission and the existing author lifecycle actions to pause, resume and retire a form. Approval and rejection still resolve the independent `REVIEWER` route; form maker-checker and optimistic-version checks remain in the existing application workflow. Pending approval cannot be edited. Program/Matter ownership, monitoring collection, response review and the original CRO role's general capabilities are unchanged.

The fixture's effective date and sample maker/checker dates are 9 September 2026 UTC. GRC Administrator and Internal Auditor are distinct sample governance identities. Installation records a SYSTEM governance decision explicitly identifying sample provenance and one matching outbox event in the same transaction as the role, bindings, policy version and effective routes. It does not record a human having performed a live approval or a bank having achieved compliance.

The new policy is independent of the original foundation policy. Both historical foundation checksums and definitions remain untouched. A repeat install preserves all new authoritative records and installation receipts. An existing changed role, binding, policy, version, checksum, approval identity or effective window fails the entire transaction; the installer does not reactivate a retired policy or erase a later governed revision. Extra bindings to the managed author role also fail installation.

## Evidence — 9 September 2026

- The initial PostgreSQL regression reproduced the defect: create, revise and transition selected only Program Owner and rejected the CRO.
- After the fixture change, `go test -tags "postgres postgresintegration" ./internal/authority -count=1` passed using PostgreSQL 18.6 with the complete current migration set. The new test executes the actual seed SQL in a rollback-only transaction and exercises both single and batch production resolvers plus simulation.
- The same test proves CRO and Program Owner eligibility, denial of the independent reviewer for author actions, independent review eligibility, unchanged unrelated ownership, tenant/entity isolation, removal of a conflicted or revoked CRO, exact repeat-install snapshots and failure on altered managed records. Existing authority tests cover delegation lineage, delegation expiry, segregation, grants and route ambiguity.
- The actual Git Bash deployment seed ran twice on the separately migrated local database `forms_author_utf8`. The resulting policy was ACTIVE v1; the original authority policy remained ACTIVE v2. Exactly one SYSTEM installation decision and one sample outbox event existed for the new policy.
- Deployment configuration tests verify the definition checksum, exact three commands, object/entity scope, the two position bindings, empty general capabilities and absence of an overwrite clause in the new fixture.
- Final clean-database authority suite passed in 1.479s; the new rollback-only fixture test also passed twice consecutively. All 18 deployment configuration tests passed. Reusing the earlier database for the entire older authority suite exposed existing tenant cleanup failures in two unrelated tests, so final suite verification used a freshly migrated database.
- The broader deployment discovery command includes email-readiness shell tests that fail on this Windows host, including exit 127 with Git Bash. Those unchanged Linux-oriented tests require the normal Linux CI gate; this receipt claims only the passing deployment configuration tests and actual local seed executions.

## Recovery and limits

Retiring this separate policy through governed configuration removes the CRO's additional author route and restores the original Program Owner route; it does not rewrite form history or publish a draft. A later deployment will reject the retired or revised managed fixture until the operator reconciles the demo fixture with that intentional change. Rollback must therefore retain its governance history and coordinate the deployment fixture; it must not delete historical policy versions or silently reactivate them.

The simulation and impact evidence above apply to the supplied demo population. Real institutions continue to use governed policy simulation, impact review, independent approval and effective dating. No production default route or identity binding changes in this slice. Browser interaction and hosted-release evidence are owned by the wider Forms editing acceptance task; these backend receipts do not claim deployment or end-user outcome completion.

## Governed ROLE approval correction

The approval checker previously required exactly one principal even for ROLE selectors, although migration 000014 and the execution resolver treat ROLE as a candidate set. The PostgreSQL checker now requires at least one distinct eligible ROLE holder, while direct PRINCIPAL and POSITION selectors still require exactly one distinct eligible person. Current active status and validity windows apply to the role, binding, position and occupant; tenant joins and the selected legal entity remain explicit. An explicit binding entity restriction must match. The existing runtime continues to re-evaluate authority, conflicts and delegation when material commands execute.

A real PostgreSQL Create → Submit → Approve regression reproduced the previous two-holder rejection, then passed with a different checker after the correction. The approved route resolves both authors and excludes the checker. Same-maker approval remains rejected. Additional PostgreSQL cases cover zero candidates, duplicate bindings, currently valid records with future expiry, future and expired records, inactive occupants, direct ambiguity, another legal entity, another tenant and deliberately mismatched tenant/entity bindings. The test is repeatable and removes its generated tenant data.

The full PostgreSQL governance package passed, as did governance and HTTP unit tests. This establishes the backend policy lifecycle for a multi-holder role; it does not relabel the demo's historical sample installation as a human approval or claim hosted Configure acceptance. The existing maker-checker, conflict, version, transaction and audit paths remain in use.

The independent security review found that runtime ROLE queries did not yet enforce the approval checker's binding scope and tenant joins. A post-approval test reproduced excluded holders being authorized. Both single and batch runtime resolution now apply the same tenant chain, explicit legal entity and binding entity restriction, including exclusion of positions without the approved entity. Five mixed-holder mutations verify that the still-eligible author can act while a differently scoped binding, foreign-tenant binding/position/occupant or entity-less position cannot. Current ROLE_ID assignment resolution uses the same predicates.
