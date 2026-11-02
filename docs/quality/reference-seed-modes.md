# Reference seed installation modes

Parent: [#462](https://github.com/CloudSpaceLab/clearsight-grc/issues/462). Installer: `cmd/seed-bank-reference`.

Select **exactly one** installation mode. A command without a mode now exits before connecting to PostgreSQL. The installer also rejects `CLEARSIGHT_ENV=production`, regardless of the selected mode.

| Flag | Data scope | Additional guard |
| --- | --- | --- |
| `-demo-journeys-only` | Existing multi-workflow fictional bank sample installer (Programs, operating Matters, Vendors, Forms and monitoring) | Non-production only, explicit opt-in |
| `-source-records-only` | Private IT, vendor and operational-risk source captures and linked actions | `-source-manifest-dir` required |
| `-source-losses-only` | Canonical private OpsRisk loss events | `-source-manifest-dir` required; source fingerprint and reconciliation checks in importer |
| `-source-risks-only` | Persisted source exception-to-Risk reconciliation | Exact non-production Clear Bank demo tenant/legal entity required |
| `-source-employees-only` | Named demonstration employees | Restricted to installer-provided actor and scope requirements |
| `-document-samples-only` | Fictional document samples | Explicit shared `CLEARSIGHT_ARTIFACT_ROOT` required |
| `-cloudspace-relationship` | One specified CloudSpace demo vendor relationship | Existing relationship identifier required |

The modes above retain their original importer contracts; this change only removes the implicit all-journey selection. `-demo-journeys-only` is a compatibility option for intentional non-production demo preparation, **not** a publisher or production starter-pack installer.

## Operator preflight

1. Confirm the environment is non-production, the existing tenant/legal-entity and actor identities are correct, and the selected operation is authorized.
2. Use one mode and supply its required manifest path or relationship ID. Keep customer source files and manifests outside Git.
3. Review the operation's emitted receipt and source/record counts. Re-run only an importer whose idempotency contract is validated for that source and version.
4. Do not infer compliance scores, answered blanks, approved framework licenses, genuine bank activity or actor sign-off from sample data.

Outstanding #462 T5 work: complete the content rights/publisher inventory, installer mode audit, protected OpsRisk source two-run receipts and separate versioned approval of production reference packs. No production seed installation is authorized by this document.
