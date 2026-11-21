# Loss register: source mapping and workflow decisions

## Reference and observations

Reference: **LOSS DATA BASE.xlsx**, Sheet1, inside **Ops Risk (2).zip** from the supplied Google Drive folder. Archive: https://drive.google.com/file/d/1tMdSTVFHPv0MjFrH5Uj6pURc2n4MYwgu/view

The workbook has 18 columns and repeats transaction examples across reporting months. Reporting month can differ from recognition month; numeric quarter can conflict with the derived reporting quarter. Recovery narrative includes claims without corresponding dated numeric receipt amounts. The UI requests resolution instead of treating each row as a new loss or each recovery statement as received cash.

Only structure and classification vocabulary are distributed. Source transaction narratives, customer names and source amounts are excluded from the synthetic demo. Structural mapping is also in `LOSS_DATABASE_MAPPING.json` and `assets/loss-schema.js`.

## Exact field mapping

| Source column | UI / model | Treatment |
| --- | --- | --- |
| ACCT_NAME | Account name / classification; `accountName` | Separate from loss group and risk category; source vocabulary plus custom bank values. |
| LOSS GROUP | Bank loss group; `lossGroup` | Source grouping, independently reported. |
| Month | `reportingPeriod` | Derived month name from explicit YYYY-MM. |
| Month Number | Reporting period | Derived number; import flags inconsistent labels. |
| Quarter | Reporting period | Derived Q1–Q4. |
| Amount | Gross booked amount; `gross` | Integer minor units; actual losses must be positive. |
| TRAN_PARTICULAR | Transaction particulars | Event explanation; separate transaction reference added. |
| Branch | Exact location; `orgScopeId` | Organizational ID, explicit import mapping. |
| Regional Bank | `region` | Captured source context. |
| Directorate | `directorate` | Captured source context. |
| YEAR | Reporting period | Derived year. |
| QTR NUMBER | Reporting period | Derived quarter; conflicts flagged. |
| DATE OF RECOVERY | Recovery receipt date | Imported narrative retained; no receipt without a numeric amount. |
| DATE OF OCCURRENCE | `date` | Event occurrence, existing dashboard cohort basis. |
| DATE OF RECOGNITION | `recognizedOn` | Financial recognition, actual losses only. |
| ROOT CAUSE ANALYSIS | `rootCause`, status and action | Confirmed cause or investigation with owner/due date. |
| RECOVERY DATA | `recoverability.note` | Narrative/claim, separate from receipt ledger and expected recovery. |
| CURRENCY OF LOSS | `currency` | Original currency; Naira maps to NGN; no FX assumptions. |

## Canonical state and inclusion

`S.losses` is the event ledger. `S.lossRegister.drafts` contains preparation; imports create only drafts. Submitted losses stay outside accepted totals. An assigned known reviewer with `risks.approve` for the location accepts or returns independently of the preparer.

Only **Accepted** and **Closed** records enter accepted totals. Corrections stay in `pendingRevision` until acceptance; prior accepted facts continue to report. Superseded accepted revisions, submission snapshots and shared review receipts remain available. Source/destination permissions, pinned evidence availability and closure/correction conflicts are rechecked before mutation. Exact transaction references cannot be reused to bypass duplicate warnings.

Risk mapping is optional. The integrity view distinguishes an explicit unmapped loss from a nonempty broken risk link. V2.8 synthetic gross/received amounts are preserved. Legacy source dates/classification remain empty with enrichment notices. Old recoveries have stable IDs and explicit legacy verification basis. Earlier setup links resume the reviewed intake instead of inserting malformed records.

## Recovery and closure

Receipts retain ID, amount, currency, received date, reference, preparer, optional exact document versions and review state. Pending receipts reserve outstanding capacity but do not reduce net. Verification records an independent decision. Unavailable evidence cannot partially update money; rejection remains possible.

Reversals have independent requests/decisions and effective dates. Original gross cash inflows remain in the ledger, allowing gross receipts, reversals and net cash movement to reconcile separately. Currency cannot change after a recovery ledger exists.

Closure requires residual-loss treatment and an explanation for any open actions. Pending financial decisions block closure. Closing does not write off amounts or complete actions. Reopening retains the earlier closure decision.

Loss actions enter Work and require substantive completion evidence plus an independent verifier. Changed accepted action plans retain prior ownership, deadline, source revision, completion and verification before reopening. A financial correction alone does not reopen an unchanged completed action.

## Import

CSV import accepts up to 1,000 rows / 2 MB with all 18 source headers. It handles quoted fields and explicit day/month/year, ISO or Excel serial dates. Currency, precision, calendar, location and period errors are visible at preview.

Duplicate comparison excludes reporting period. A distinct event with repeated details needs a unique transaction/line reference and explanation. Selected ready rows become drafts, preserving source columns, row and batch provenance. Excluded rows remain in the outcome. Zero amounts do not silently infer a near miss. Narrative recoveries do not produce receipt values.

CSV is the interoperability path for this dependency-free package. Native XLSX decoding, asynchronous ingestion, content scanning and bank master-data reconciliation require production integration.

## Read models and boundaries

`lossFinanciallyIncluded`, `lossReceivedReceiptRows`, `lossVerifiedReceiptRows`, `lossRecoveryReversalRows` and `lossVerifiedRecoveries` are authoritative financial helpers. Existing NGN financial cards use accepted NGN records; operational reports expose separate currencies. Helpers classify the currently verified ledger by effective cash date; they are not a reconstructed bitemporal query. Frozen reports retain their captured state.

All identities/data are synthetic local state. Commands demonstrate role/scope rules. Read views retain selected organizational scope and are not an authenticated confidentiality boundary. Production must supply server identity, data authorization, atomic writes, concurrency checks and secure document access.
