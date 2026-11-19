# Framework adoption questionnaires

`framework_adoption_questionnaires` stores the form revision selected for a specific Program framework adoption. The scoped foreign key binds the Program, adoption, edition, tenant and legal entity together; the form foreign key pins an exact governed revision in the same tenant and legal entity. Only one current link is allowed per adoption. Previous links are retained with their end time, actor, reason and version.

`framework_adoption_questionnaire_events` stores LINKED and UNLINKED audit events. Link replacement, the retired link and both events are committed in one transaction. The repository locks the adoption, checks that it is a draft or active, checks and locks an active current form revision, and rejects stale link versions. A submitted proposal cannot be modified while under review.

The authenticated API binds the requested Program to the actor's legal-entity scope and checks Program visibility. Selection is a Program-owner material command; the monitoring service repeats the owner authorization and takes the actor from the verified identity. A linked form does not dispatch an assessment, establish a reviewed answer or imply compliance.

Migration: `000141_framework_adoption_questionnaires`; downgrade refuses to remove recorded links. Validation uses the isolated local database copy; no original development database migration is implied by this document. Remaining acceptance includes direct unauthorized-command coverage, reviewer and routing failure cases, UI parity and a fully published edition.

<!-- schema-ownership:begin -->
| Table | Classification | Owner | Writers | Readers | Lifecycle / valid time | Retention / deletion | Executable evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `framework_adoption_questionnaires` | active authoritative state | Program framework adoption | authorized questionnaire link commands | Program setup and Activity | version-pinned current link with retained ended links | retain link history; downgrade refuses populated state | migration 000141; monitoring adopted assessment integration |
| `framework_adoption_questionnaire_events` | active authoritative state | Program framework adoption | atomic questionnaire link transactions | Program Activity | append-only link and unlink receipts | retain audit history | migration 000141; Program Activity PostgreSQL integration |
<!-- schema-ownership:end -->
