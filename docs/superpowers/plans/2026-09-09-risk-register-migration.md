# Risk register migration implementation plan

Goal: import existing vendor findings with reviewed internal-person and existing-vendor mappings into canonical issues and actions.

- [x] Add bounded deterministic register parsing and provenance tests.
- [x] Add migration draft, authority/assignment validation and immutable receipt contracts.
- [x] Implement atomic canonical finding/action/vendor-link persistence and replay in memory and PostgreSQL.
- [x] Expose scoped preview/save/commit routes and runtime contract entries.
- [x] Add Imports mapping and preview controls with persisted recovery and issue receipts.
- [x] Verify the supplied workbook read-only, domain/HTTP/UI tests, PostgreSQL transaction behavior, typecheck, copy quality and responsive rendered states.
- [x] Synchronize architecture, migration ownership, acceptance evidence and current execution ledger.

Two independent-review findings were corrected and regression-tested: normal action-creation authority must hold, and excluded assessments cannot block valid imports. Deployment, live bank import and production-scale acceptance are not part of this local implementation. See the acceptance record for explicit limits.
