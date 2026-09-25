# Source-grounded report packs

**Date:** 2026-09-24  
**Status:** Approved implementation direction  
**Decision:** Generate governed Excel report packs from ClearSight's current canonical records, using the supplied bank workbooks as the field and layout specification.

## Purpose

Provide operational reports that a risk, compliance or executive user can circulate without rebuilding a spreadsheet by hand. A report must identify its legal-entity scope, report period, source population, generation time and data freshness.

The supplied workbooks define the reporting language and expected columns; they are not a second runtime data source. Runtime reports query versioned ClearSight records only.

## Report catalogue

| Report pack | Primary canonical population | Workbook shape | Treatment of missing canonical facts |
| --- | --- | --- | --- |
| Third-party risk register | Vendor-linked findings, actions and source assessment facts | `Sample Third-Party Risk Register` | `Not recorded` rather than a guessed rating, framework or evidence state |
| IT risk exception register | IT findings, actions and source assessment facts | `Sample IT Risk Exception Register` | `Not recorded`; current workflow state remains separate from source status |
| IT risk workplan | Risk work programs, planned actions and vendor reviews | `Sample IT Risk Workplan` sheets | An empty source population is labelled with its checked scope and as-of time |
| NDPA compliance register | NDPA program requirements, processing activities and evidence state | `NDPA_Compliance_Checklist` | Preserve reference and evidence-required wording; do not represent a legal conclusion |
| Operations risk pack | Seeded KRI and loss-event records | Operations Risk KRI and loss workbooks | Unavailable until source data has been ingested into canonical records |

## User workflow

1. Open **Reports** from the product navigation.
2. Choose a report pack and any permitted legal entity, program, matter or date scope.
3. Run the report. ClearSight creates an immutable run receipt and generates the workbook asynchronously.
4. Download the protected report file. The workbook includes a `Report information` sheet containing scope, filters, as-of time, source count, definition revision and file checksum.
5. Open the report history to reproduce a previous run or compare its source boundary with a newer run.

Saved reports remain governed definitions. Their owner, reviewer and authorizer are distinct workflow roles; a requester cannot supply those identities in a browser request.

## Workbook rules

- Excel (`.xlsx`) is the circulation format. CSV and NDJSON remain available for controlled machine use.
- The first operational sheet uses familiar source workbook headings in a stable order. It has a frozen heading row, filters, readable column widths, wrapped long text and a report title/as-of line.
- A source fact and ClearSight's live workflow fact are distinct. Where both are useful, the workbook uses separate columns: for example `Source status` and `Current issue status`.
- Historic source comments are labelled `Source comment`; comments or activity added after migration are labelled `Current activity`.
- A blank is never silently treated as a passed control, closure, evidence item, risk rating or approval. Use `Not recorded` or a checked-population empty state.
- Every row carries a stable record reference and source provenance where present, so users can reconcile it to the record and original captured range.

## Data boundaries and availability

The report service uses bounded, scope-filtered repository queries and runs outside the interactive request. It publishes a receipt before file rendering, stores the result with a manifest and SHA-256 checksum, and retains the report according to configured retention.

The Operations Risk source files are not reportable merely because they exist on a developer workstation. Their KRI and loss data must first enter the demo environment through an immutable import with source version and row provenance. Until that occurs, Reports names the pack as unavailable and gives the precise import prerequisite.

## Navigation

Reports is a product-level destination because the catalogue spans vendor, IT, privacy and operational risk work. The legacy ROPA reporting address remains a compatible route into the same report workspace with the ROPA pack preselected.

## Acceptance criteria

- A permitted user can generate each available pack and download an `.xlsx` file with an as-of timestamp, scope and manifest details.
- Third-party and IT reports reproduce the meaningful source columns without turning people names into vendor questionnaire fields.
- A report shows source facts, current state, ownership and deadline without conflating them.
- NDPA reports identify their checklist/reference source and do not claim compliance from an incomplete source population.
- An Operations Risk report cannot generate until canonical KRI/loss source records exist.
- Scope, authorization and report-run audit events are enforced server-side.
- Report queries remain bounded and report files are reconstructable from the run manifest.
