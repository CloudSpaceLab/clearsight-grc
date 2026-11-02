# Framework catalog content verification

Owner: #437 / F1 reference-content pipeline
Related: #433 calculation, #270 shared controls, plan `docs/superpowers/plans/2026-10-15-governed-framework-library-and-program-adoption.md`

## Source verification

`cmd/framework-catalog` is an **offline verifier only**. It does not write PostgreSQL, create Programs, call an AI service, retrieve external URLs or approve content. An operator supplies a reviewed edition manifest and the exact local authoritative artifact whose SHA-256 is declared by that manifest.

```bash
go run ./cmd/framework-catalog \
  -manifest internal/frameworkcatalog/testdata/synthetic_manifest.json \
  -source internal/frameworkcatalog/testdata/synthetic_source.txt
```

The command emits a single JSON verification receipt with publisher/framework/edition codes, clause count, exact source length and SHA-256, deterministic release SHA-256, and `"published":false`. A nonzero exit means no valid receipt. The checked-in example is **synthetic test data**, not an installed or applicable compliance standard.

Verification rejects unsupported source/manifest sizes, malformed JSON, nested duplicate keys, unknown fields, unexpected extra JSON, invalid edition/anchor/parent lineage, cycles, non-HTTPS references, unsupported rights declarations and mismatched source bytes. It does **not** prove that a URL belongs to an issuer or that a referenced license is genuine.

## Content publication responsibilities

1. Curator records a real official source (issuer, publication/edition, source date, original bytes and checksum), confirmed citation anchors, permission to redistribute any summaries, and declared scope/completeness. Do not treat locally interpreted obligations as normative source text.
2. An independent reviewer compares extracted clauses and anchors against the original artifact and checks applicable license/rights. Maintain named approval evidence and approved manifest version in the repository's change review history. Do not assume the presence of a string `approval_reference` is maker-checker approval.
3. Run source verifier on the exact release candidate. Preserve the receipt and source artifact in a controlled document/retention process; do not commit licensed source full text to the repository without rights.
4. Only the already implemented trusted internal `frameworkcatalog.PostgresRepository.Publish` method can atomically write the vetted immutable edition/clauses. **It is not connected to a CLI, HTTP handler, generic tenant actor or automatic deployment.** Production publication remains withheld until the distinct authorization/audit/entitlement boundary has its own accepted implementation.
5. A published edition neither activates a tenant Program nor creates an approved Program Requirement. #437 F2 owns adoption and mapping; #433 solely owns calculation and score history.

## Rollout and limitations

- Synthetic fixture manifests are for tests only. No official standard is preloaded by this tranche.
- Canonical edition key is (issuer, framework code, edition), not spreadsheet row or display label. Conflicting same-edition content is rejected; the database trigger rejects mutation/deletion of published facts.
- Published catalog metadata must stay independent of tenant-owned applicability, control implementations, forms, vendor responses, evidence and local Program history.
- A new standard edition is a new immutable release; operator-only source verification is not a legal conclusion or an approval to replace an activated edition.
- When publication/withdrawal controls are implemented, use existing governed versioning and audit conventions; a global reference-content curator must not gain authority to edit tenant Programs.
