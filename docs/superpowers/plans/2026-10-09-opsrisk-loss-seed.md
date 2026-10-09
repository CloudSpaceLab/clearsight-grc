# OpsRisk Historical Loss Seed Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (\`- [ ]\`) syntax for tracking.

**Goal:** Seed the eight deduplicated historical OpsRisk losses into the non-production Clear Bank ledger with source lineage and no inferred recoveries.

**Architecture:** The protected workbook is transformed into an operator-private version-1 source manifest. The existing loss-only importer validates all eight distinct events before the first write and delegates ledger creation to the canonical operational-loss service. A first-run receipt and idempotent rerun provide the installation evidence.

**Tech Stack:** Protected XLSX source, JSON source-record manifest, Go \`clearsight-seed-bank-reference\` image, PostgreSQL 18, Docker Compose.

---

## File structure

| Path | Responsibility |
| --- | --- |
| \`Ops Risk (2).zip\` (private source) | Original \`LOSS DATA BASE.xlsx\` workbook; never committed or copied to public CI. |
| \`source_records_ops_loss.json\` (protected host state) | Eight source-anchored loss records and workbook digest. |
| \`cmd/seed-bank-reference/source_losses_import.go\` | Existing whole-manifest preflight and canonical loss install. |
| \`cmd/seed-bank-reference/source_losses.go\` | Existing identity, source-provenance and no-recovery mapping. |

### Task 1: Produce and validate the protected manifest

**Files:**
- Read: \`C:\\Users\\Son\\Downloads\\fidelitybankgrcusecasesandrequirements\\Ops Risk (2).zip\`
- Create outside the repository: protected \`source_records_ops_loss.json\`

- [ ] **Step 1: Read the workbook without modifying it**

Read \`Ops Risk/LOSS DATA BASE.xlsx\` directly from the ZIP archive. Retain the SHA-256 digest, worksheet name, source headers and coordinates. Do not extract or commit a workbook copy.

- [ ] **Step 2: Select one record for each canonical event**

Group the 96 nonblank source rows by the importer identity:

\`\`\`text
ACCT_NAME + LOSS GROUP + TRAN_PARTICULAR + Branch
+ exact Amount + CURRENCY OF LOSS
+ DATE OF OCCURRENCE + DATE OF RECOGNITION
\`\`\`

Require exactly eight groups of twelve. Choose a source row with \`RECOVERY DATA\` when present; otherwise use the first row. Preserve all original source columns as labelled fields, including blank \`DATE OF RECOVERY\` values.

- [ ] **Step 3: Write a private version-1 manifest**

Use one group with these source facts:

\`\`\`json
{
  "key": "opsrisk-historical-losses",
  "source_file": "LOSS DATA BASE.xlsx",
  "source_sheet": "Sheet1",
  "source_sha256": "765d0681c3472aa492c1f8193566e949ea680ee36881b0039325c4262b045970",
  "period": "Historical 2025 monthly reporting views",
  "records": ["eight source-anchored records"]
}
\`\`\`

Each record has a stable key, \`source_range\`, and source field cells. No owner, Matter, Program or recovery command is required by the loss-only importer.

- [ ] **Step 4: Verify the manifest before transfer**

Confirm the manifest has one group, eight records, the workbook SHA-256, all required event identity values, positive two-decimal-or-less amounts, valid occurrence and recognition dates, and no recovery amount field. Confirm it contains no workbook bytes or credentials.

### Task 2: Back up and install the loss seed

**Files:**
- Create on protected host: \`/opt/clearsight-grc/state/source-records-20260910/manifests/source_records_ops_loss.json\`
- Read: \`/opt/clearsight-grc/releases/22f97dae9d5c3f813e125757685fae8c781ff8c6/compose.demo.yaml\`

- [ ] **Step 1: Create a timestamped database backup**

Use the release’s configured PostgreSQL connection. Store a compressed database dump under \`/opt/clearsight-grc/backups/\` with restricted permissions. Do not print the connection URL or dump contents.

- [ ] **Step 2: Transfer only the manifest**

Copy the locally validated JSON to the existing protected source-record manifest directory with mode \`0600\`. Confirm the file’s SHA-256 after transfer. Do not upload the workbook.

- [ ] **Step 3: Run the exact release image in loss-only mode**

Use the current release’s seed image and environment, with the required canonical demo tenant/legal entity plus existing installed actor and owner principal identifiers:

\`\`\`text
clearsight-seed-bank-reference -source-losses-only \
  -source-manifest-dir /protected/manifest-directory \
  -tenant 00000000-0000-4000-8000-000000000001 \
  -legal-entity 00000000-0000-4000-8000-000000000002 \
  -actor 00000000-0000-4000-8000-000000000101 \
  -owner 00000000-0000-4000-8000-000000000107
\`\`\`

Expected receipt: \`losses: 8\`, \`losses_created: 8\`.

### Task 3: Reconcile and prove safe rerun

**Files:**
- Read: canonical \`operational_losses\` and \`operational_loss_recoveries\` tables in the demo scope.

- [ ] **Step 1: Reconcile the first install**

Confirm exactly eight active ledger rows for the Clear Bank demo, zero recovery rows, one complete source digest per loss description, and no loss rows created from the 96 monthly source rows beyond the eight canonical identities.

- [ ] **Step 2: Run the importer once more**

Run the identical loss-only command with the same protected manifest. Expected receipt: \`losses: 8\`, \`losses_created: 0\`.

- [ ] **Step 3: Preserve the receipt and backup reference**

Keep the backup path, source manifest digest, first-run receipt and rerun receipt in protected operator storage. Do not place source rows, financial values, credentials or manifests in Git or CI output.

### Task 4: Commit the operational design and plan

**Files:**
- Create: \`docs/superpowers/specs/2026-10-09-opsrisk-loss-seed-design.md\`
- Create: \`docs/superpowers/plans/2026-10-09-opsrisk-loss-seed.md\`

- [ ] **Step 1: Check documentation scope**

Run:

\`\`\`powershell
git diff --check
git status --short
\`\`\`

Expected: only the two operational documents are staged; no source workbook, source manifest, backup or secret is tracked.

- [ ] **Step 2: Commit the documentation boundary**

Run:

\`\`\`powershell
git add docs/superpowers/specs/2026-10-09-opsrisk-loss-seed-design.md docs/superpowers/plans/2026-10-09-opsrisk-loss-seed.md
git commit -m "docs: record OpsRisk loss seed procedure"
\`\`\`

Expected: one documentation-only commit on the isolated branch.
