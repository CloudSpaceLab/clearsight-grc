# Private source-record demo installation

The demo operator may install completed source captures, source-linked work and the explicitly mapped IT exception Risk profiles with `clearsight-seed-bank-reference -source-records-only -source-manifest-dir <private-directory>` and the existing explicit tenant/entity/actor/owner/reviewer flags. This operation refuses production and any scope other than the canonical Clear Bank demo.

The directory supplies `source_records_it_vendor.json` and `source_records_ops.json`. Their contents are not embedded or committed: the repository is public and source workbooks may contain confidential bank information. Keep the originals, manifests, extraction coverage, database backup and installation receipt under operator-controlled storage. Do not upload source-bearing screenshots or logs as public CI evidence.

Each manifest has `version: 1` and `groups`. Each group contains `key`, `program_code`, `title`, `source_file`, `source_sha256`, `source_sheet`, `period`, `limitations` and `records`. Each record has `key`, `title`, `source_range`, ordered `fields` (`label`, `value`, `source_cell`), `owner`, `assessor`, `status`, `rating`, ISO `due_date`, `action`, an existing Matter `kind`, and `create_matter`. Additional extraction metadata may remain in private manifests. Empty source values remain empty; unknown dates are not replaced with today's date.

The operator first verifies the source hashes and backup. The installer uses existing Program identities, canonical Risk and Matter/Action commands, governed form approval, immutable response submissions and vendor links. Only the `it-risk-exceptions` group is projected into canonical ERM Risks: its recorded Risk ID, description, category and implication define the Risk profile; its recorded risk level becomes a CURRENT source assessment; no appetite position is inferred. Findings, recommendations, deadlines and comments remain Matter/action work. It does not send emails or create compliance decisions. Exact keys and receipts support retry after interruption; changed source digests or form contracts require review instead of silent overwrite. A single-operator advisory lock prevents concurrent installers.

## Source titles and extraction quality

The source manifest must keep the original worksheet coordinates in `source_range` and `source_cell`, but a display title such as `Row 12`, `Record 4` or `Sheet1 Row 8` is not an assessment, event or service name. The installer derives **new Matter titles** from explicit descriptive source fields (branch, risk description, risk metric, transaction, process, requirement, finding), and does not mistake the source row position for the business record. Meaningful manifest-authored titles remain authoritative.

Header lookup normalizes line breaks, tabs, non-breaking spaces and repeated whitespace, while retaining original field labels and exact answers. Conflicting duplicate normalized headers are not arbitrarily resolved. A field that is missing or ambiguous must not be replaced with a guessed business fact.

If the private OpsRisk manifest still has 31 separate `ops-branch-kri-r2` … `ops-branch-kri-r32` groups, it needs a source-preserving regrouping before a V2 import: one `ops-branch-kri` group, all 31 original keyed records, the unchanged shared column schema, `response_per_record: true` and `presentation_version: 2`. Verify the same source SHA, sheet, period, Program and unique record keys first. Do not create 31 new forms or infer KRI values/units from column names. Source ranges stay in each record; they are not display titles.\n\nFor new structured form/response presentation, the **private** group may explicitly include `"presentation_version": 2`. V2 uses source-derived human-readable section/field/response titles. When all rows have the same bounded column schema (such as a 31-branch KRI sheet), the installer uses **one reusable form and one independent response per source record**. Mixed-layout historical tables retain a labelled line-per-column presentation including blanks and duplicate header cell provenance. V2 uses a different immutable form/distribution identity. The default/missing value retains the existing V1 template and response identities and content exactly: ordinary reruns will **not** rewrite or invalidate historic submissions. A V2 import alongside existing V1 data creates distinct form versions that require an operator-reviewed retirement/visibility plan for the old examples; this tool never deletes immutable responses or quietly switches governed forms.

For existing source-import Matters whose stored title still **exactly** matches an old row-number placeholder, a guarded reinstall may improve that title from the recorded source description. It verifies the original trigger key, source checksum and row coordinate; changed titles, unrelated Matters, missing descriptive values, other fields, actions and immutable submissions are not rewritten. The correction is recorded through ordinary Matter detail history. 

The public tests exercise synthetic Branch KRI, Head Office KRI, IT exception and BIA cases. The source manifests are operator-private: validate source labels/values against the original workbooks before requesting V2 presentation. Neither a row-number-only title nor a flat text screenshot is evidence of a successfully mapped risk, control or KPI measurement.

Ordinary risk registers retain a form field per source column. Large historical tables retain each complete source row as a labelled multiline answer. Captures split before the existing 200-field/20-section limits. Overlapping source views must not be interpreted as distinct incidents or added into a portfolio denominator.

Source-only calendar deadlines use West Africa Time. Bank accountability for initial review remains with the configured Program owner; named source employees remain action performers and the original source owner/assessor values remain intact. A guarded repair uses normal reassignment events only for untouched initial import records. Blank and whitespace-only source cells remain unanswered, matching the submitted response contract on retries.

Normal reference deployments also reconcile those Risk profiles from the already-persisted source Matter facts, so an existing source install does not require the private workbook again. The reconciliation is bounded to the exact source-package trigger prefix and refuses changed source lineage or user-modified baseline records.

## OpsRisk historical Loss register

Use the private `LOSS DATA BASE.xlsx` from `Ops Risk (2).zip`, not screenshots or synthetic examples. The 2025 worksheet repeats each event in 12 monthly display rows: its 96 rows represent **8 distinct historical events**, not 96 new 2025 occurrences. Source occurrence and recognition dates must be preserved independently. `CURRENCY OF LOSS=Naira` maps to NGN and source decimal amounts are converted to integer kobo without float rounding.

Place a private source manifest named `source_records_ops_loss.json` in the same `-source-manifest-dir`. It uses the existing version-1 `groups/records/fields` contract, the group `source_file: "LOSS DATA BASE.xlsx"`, the full source SHA-256, source sheet/ranges, and the recorded loss fields. The operator may provide either deduplicated eight records or all 96 monthly rows; the installer deduplicates event identity by account, category, transaction, branch, exact amount/currency, occurrence and recognition dates. It never uses month/quarter/reporting year as a new Loss identity.

After a backup and under the non-production demo scope, run the dedicated seed option with the existing explicit tenant/legal-entity/actor/owner flags:

```sh
clearsight-seed-bank-reference -source-losses-only -source-manifest-dir /secure/private/manifest-directory \
  -tenant 00000000-0000-4000-8000-000000000001 \
  -legal-entity 00000000-0000-4000-8000-000000000002 \
  -actor <existing-demo-actor-uuid> -owner <existing-demo-owner-uuid>
```

The full `-source-records-only` operation also imports canonical Losses when an eligible OpsRisk Loss group is included in the ordinary OpsRisk source manifest, or in the optional supplementary loss manifest. The dedicated command avoids rerunning unrelated Forms/Matters. The receipt reports `losses` and `losses_created`; on a repeat, the expected result is 8 and 0. It refuses conflicting source digests and edited existing entries rather than silently overwriting them.

The private workbook has a recovery narrative without a dated recovery amount. That text is retained as *unverified source narrative*, **not** posted to `operational_loss_recoveries`. Net loss remains gross less **posted, evidenced** recoveries. Branch/region/directorate values remain source provenance unless an authorized organization-scope identity is explicitly established; do not force a location onto an unrelated demo branch. Unknown event classifications remain `OTHER` rather than assigning unsupported fraud provenance.

The underlying workbook and private manifest must never be committed, uploaded in public CI evidence or logged with source rows. Public tests use synthetic values. Acceptance requires a real operator-run receipt, inspection of all eight exact Loss records and a rerun/backup comparison.

Verification is intentionally limited for this demo: build, synthetic contract check, optional source-contract check with `CLEARSIGHT_SOURCE_MANIFEST_DIR`, scoped source/count reconciliation and hosted representative reads. New source responses remain unscored and outcomes unverified. The previous generic Cloudspace questionnaire is revoked only after replacement captures exist; its history remains available. Archived generic work is recoverable through attributed archive restoration.
