# Framework starter and Program assessment ownership

These entries cover recorded framework starter material and Program assessment operations. Activity is a read model over these records and existing event ledgers; it creates no separate event store.

<!-- schema-ownership:begin -->
| Table | Classification | Owner | Writers | Readers | Lifecycle / valid time | Retention / deletion | Executable evidence |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `framework_starter_requirements` | active authoritative state | framework starter catalog | dated starter catalog installation | starter review and adoption | catalog version with source reference | preserve referenced catalog versions | migration 000144; framework starter PostgreSQL integration |
| `framework_starter_requirement_links` | active authoritative state | framework starter catalog | starter installation and governed maintenance | adopted assessment setup | starter requirement to pinned form group | preserve used link versions | migrations 000144-000146; framework starter PostgreSQL integration |
| `framework_starter_requirement_events` | active authoritative state | framework starter catalog | transactional starter maintenance | starter maintenance history | append-only maintenance receipt | retain event history | migrations 000144-000146; starter maintenance integration |
| `program_framework_requirement_questions` | active authoritative state | Program framework adoption | approved adoption installation and question linkage | requirement coverage and assessment context | Program requirement with pinned form revision and group | preserve evidence and requirement reconstruction | migration 000144; adopted assessment PostgreSQL integration |
| `capture_submission_reviews` | active authoritative state | Program assessment review | independent authorized response reviewer | response history, documents and Program Activity | append-only review per immutable submission | retain previous submissions and reviews | migrations 000147/000150; adopted assessment and Activity PostgreSQL integration |
| `program_assessment_schedules` | active authoritative state | monitoring recurrence | owner proposal, independent approval, authorized stop and policy-bound worker | schedule configuration and Program Activity | versioned schedule with bounded cycles and dispatch state | retain stopped schedules and outbox receipts | migrations 000148/000149; recurrence tests and Activity PostgreSQL integration |
<!-- schema-ownership:end -->
