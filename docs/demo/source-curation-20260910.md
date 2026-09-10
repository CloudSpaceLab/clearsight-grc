# Source-curated hosted demo

The requested population is the supplied Fidelity requirements directory and Ops Risk archive. The employee manifest records source file digests and coordinates. The current local third-party register differs byte-for-byte from the earlier hosted import; neither original is overwritten.

Manual seed operations are available through `clearsight-seed-bank-reference`:

- `-source-employees-only`: 17 named people, positions and scoped performer bindings. Logins use `firstname@demo.com` and password `password`; Victor Abejegah uses `victor@demo.com`. Existing core demo users remain. No approval authority is granted.
- `-cloudspace-relationship UUID`: source-backed submitted, unreviewed Cloudspace/OEM response for one exact existing relationship. Rejects other vendor/service identities, other demo scope and ambiguous active forms. It does not seed generic vendors or acceptance examples.

Use the existing tenant/entity/actor/owner/contributor/reviewer/signatory flags. Production execution is rejected. Employee identity conflicts abort the whole roster transaction without overwriting edits. Repeating the Cloudspace operation validates the existing immutable submission rather than creating another response.

The hosted target is Cloudspace relationship `01a0810d-9a4a-7a9e-a3c2-abf0727927eb`. Its earlier duplicate and eight out-of-scope vendor fixtures are archived from active demo lists using `deploy/curation/source-curated-20260910.sql`. The manifest also excludes fourteen associated acceptance or synthetic historical Matters. Lifecycle statuses, assessments, documents, original imports and command history remain retained. Five source-backed Program themes and operational work remain.

Set `CLEARSIGHT_DEMO_SEED_MODE=manual` before deployment. Otherwise each release installs the generic reference population again. The script must be executed only after migration 000089 and a verified nonproduction backup. It asserts exact tenant, vendor identities and target counts before writing archive receipts.

## Recovery

The pre-curation PostgreSQL custom backup is `/opt/clearsight-grc/backups/20260910-source-curation/clearsight-before.dump`, SHA-256 `a2a149b33acd486e88812862715963b66dc6e59e1fc1e65c5bd3de202ce14cf3`. A restore into a disposable database verified 10 relationships, 9 principals and 24 requests before curation. Configuration and the artifact path inventory were saved alongside it; artifacts themselves remain untouched.

Archive restoration is one attributed update of `restored_by`, `restored_at` and `restoration_reason` on the exact receipt. It does not recreate records or erase the archive reason. Restoring the whole database is an operator recovery operation requiring an outage and reconciliation of post-backup changes.

No legal compliance, accepted evidence or resolved risk is implied by sample submissions or archive exclusions.
