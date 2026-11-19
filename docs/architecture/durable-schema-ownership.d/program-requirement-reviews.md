# Program requirement review

Migration `000142_program_requirement_reviews` creates tenant-scoped, independently decided review records for draft Program requirements. Existing approved records retain their status; no historical approval is inferred. A review starts pending, has one current proposal per requirement, records proposal and decision reasons, requires a different reviewer and cannot be deleted or rewritten after decision. Downgrade refuses to remove history.

The v1 service proposes a draft through the current Program owner and decides it through the current assigned authorizer. It checks the verified identity, Program version, requirement scope and draft state, independent reviewer and rationale. The scoped review list is bounded and entity-filtered. PostgreSQL updates the review and approved Requirement in the same transaction as the Program event and outbox. The test environment has demonstrated this through the UI and PostgreSQL integration test.

Important limitations: the `program.requirement.add` HTTP command now accepts only draft creation; omitted status also defaults to draft. Other paths require separate governance: superseding a requirement creates an approved replacement, and document handoff can create an approved requirement after its own review. Their approval, authority and transaction boundaries must be assessed before claiming production-wide enforcement. Review history has no dedicated user-facing history page. Additional authority/delegation, race, cross-entity HTTP and recovery tests are required before production rollout. Do not infer that older APPROVED records passed this review workflow.

<!-- schema-ownership:begin -->
| Table | Classification | Owner | Writers | Readers | Lifecycle / valid time | Retention / deletion | Executable evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `program_requirement_reviews` | active authoritative state | Program requirements | owner proposals and independent authorizer decisions | requirement review workspace | pending proposal followed by immutable decision | retain decided reviews; downgrade refuses retained history | migration 000142; continuity requirement review PostgreSQL integration |
<!-- schema-ownership:end -->
