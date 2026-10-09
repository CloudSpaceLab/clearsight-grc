# OpsRisk historical loss seed

**Status:** Approved for controlled demo installation
**Date:** 9 October 2026

## Purpose

Install the historical OpsRisk loss register into the canonical operational-loss ledger for the non-production Clear Bank demo without turning repeated monthly reporting rows into separate events.

## Source and scope

- Private source: `Ops Risk (2).zip` / `Ops Risk/LOSS DATA BASE.xlsx`.
- Source workbook hash: retained in the private manifest and in each seeded loss description.
- Scope: the canonical Clear Bank demo tenant and legal entity only.
- Population: 96 nonblank worksheet rows, reconciled to 8 distinct event identities. Each identity occurs in 12 monthly display rows.
- Recovery: the workbook has no dated recovery value. Textual recovery information remains an unverified source note and creates no recovery transaction.

## Design

Create a private `source_records_ops_loss.json` next to the existing protected OpsRisk source manifest. It follows the existing version-1 source-record contract and includes eight representative records, one for each deduplicated event. Each record retains the original field labels, values and worksheet coordinates. The group identifies the workbook, sheet and SHA-256 digest.

The established `clearsight-seed-bank-reference -source-losses-only` path reads both protected manifests, validates the entire eight-event population before a ledger write, takes the existing advisory lock, and uses the canonical operational-loss service. It rejects a partial manifest, changed source lineage, code collision or edited historical loss instead of overwriting records.

No Program, Matter, Risk, form, action, recovery, branch identity or notification is created by this operation. The existing demo owner remains the accountable loss owner.

## Execution and evidence

1. Back up the demo database before the import.
2. Copy only the protected JSON manifest to the server's protected source-record directory.
3. Run the dedicated loss-only seed command against the current non-production release.
4. Confirm the first receipt reports `losses: 8` and `losses_created: 8`.
5. Query the canonical ledger for exactly eight active losses in the demo scope, zero recovery rows, and the expected workbook digest in every loss description.
6. Run the same command again. Confirm `losses: 8` and `losses_created: 0`.

## Failure handling

Any validation, source, database or provenance failure stops the operation. The importer validates the full source population before the first ledger write; a failed preflight leaves no partial loss seed. A conflict after a previous successful run is investigated from the existing loss record and source lineage rather than repaired by deletion or overwrite.
