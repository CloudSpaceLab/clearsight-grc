# Framework starter requirements and native Forms

Local implementation notes, 16 November 2026.

## Deployment

Apply migrations with the repository's checksum-tracked deployment runner:

```sh
bash deploy/scripts/migrate.sh migrations
```

The runner reads DATABASE_URL. Migrations 000144–000146 extend existing framework reference templates, native Forms, Program framework adoptions and native Program requirements. They do not publish official standards or grant content approval.

For an already migrated installation, repair missing packs using:

```sql
SELECT backfill_framework_starter_forms();
-- Or limit the repair:
SELECT backfill_framework_starter_forms('legal-entity-uuid'::uuid);
```

The function inserts missing native draft questionnaires and scoped requirement links, preserving existing forms, scoped staff changes and saved adoptions. It is also called by the new legal-entity trigger. Eight starter questionnaires, 35 requirement groups and 105 fields are installed per entity. Source identities stay unchanged.

New Forms are DRAFT. Native independent review transitions produce new immutable revisions; links must be explicitly repointed to the approved revision. Existing Program snapshots are never silently upgraded. The source metadata fixes in 000145/146 preserve staff-authored wording and immutable Forms revisions. Downgrades refuse destructive removal; corrected source metadata is intentionally retained.

## API and authority

- GET `/api/v1/framework-catalog/reference-templates/{code}/requirements`: scoped source requirements, exact Forms groups, management eligibility.
- POST `/api/v1/framework-catalog/reference-templates/{code}/requirements/{requirement_code}`: update with expected requirement version; version zero creates a bank-scoped requirement. Verified actor/entity and current maintenance authority are required. Native revision/group membership is checked. A full snapshot audit event and outbox event commit with the link change.
- GET `/api/v1/programs/{id}/starter-respondent-units`: bounded search of actual active units, restricted to the visible Program and authorized owner/reviewer context.
- Existing Program `/standards/draft`, `/propose`, `/{id}/approve` and `/{id}/reject` commands accept the starter subtype with `template_code` and `configuration`; published edition behavior remains supported.

Configuration stores a canonical source snapshot, Program-specific wording/decision/reason, configured questionnaire version/group, existing Program owner and intended respondent unit IDs. The server resolves and validates source snapshots and approved native Forms. It does not trust client field definitions, identity, ownership or approval flags. Material operations retain Program optimistic concurrency and existing command authorization. Independent adoption review requires the assigned authority and a different principal from the maker.

Approval creates native Program requirement copies using MUST modality. KEEP/TAILOR materialize as approved copies; an applicability REVIEW flag remains a draft requirement pending follow-up, not an exclusion or compliance conclusion. Native question links, authoritative projection, append-only event and outbox share the same transaction. Source wording remains unchanged.

## Verification

Tagged and non-tagged tests passed for internal/continuity, internal/frameworkcatalog and internal/httpapi. Uncached real PostgreSQL tests on the isolated acceptance database passed:

```sh
go test -count=1 -tags postgres,postgresintegration ./internal/continuity -run 'TestPostgres(StarterSeedAdoptionIsolationAndVersionPins|FrameworkAdoptionEventAndCurrentProjection)' -v
```

Set TEST_DATABASE_URL to the acceptance database, never a production database. This integration creates test-scoped tenants/entities; its approval fixture is explicitly test-only. Live UI verification used separate demo Program Owner, Internal Auditor and CRO accounts against the copied familiar database, with authorization enforced.

Coverage includes all eight native form contracts; reruns preserving staff edits and adoptions; valid group/version pins; cross-tenant and invalid-group rejection; bank-defined source extensions; second-Program isolation; atomic approved native requirements with valid MUST modality; later form versions leaving existing pins unchanged; and the existing published-edition adoption workflow.

## Content and workflow limits

These are original bank starter requirements. Detailed licensed ISO/PCI/SOC content and clause-level mapping require bank content review. They are not full official framework coverage. No production approval was made. Per-question expiry, external validation policies and assessment dispatch are not configured by this adoption payload; existing whole-form/collection workflows remain separate. No assessment answers or evidence uploads are collected in View questions.

Frontend acceptance, exact live demo records and screenshots are documented in the v2 worktree at `docs/FRAMEWORK-STARTER-WORKFLOW-ACCEPTANCE.md`.
