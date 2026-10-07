# Indicator source acceptance

Use this command to produce a redacted acceptance receipt for a real, governed
`IT_GOVERNANCE_CHANNEL_PERFORMANCE` source Binding.

The command is read-only. It does not create or update source catalog records,
Monitoring Checks, Monitoring Results, Programs, or Indicators.

## What it proves

The command resolves sources inside one exact legal entity, selects one current
and effective channel-performance Binding, executes a bounded `PAGE` read through
the normal Source Catalog adapter, and evaluates each returned record with the
same `ParseMetricSeriesBinding` / `MapRecord` contract used by ClearSight.

A successful receipt proves:

- the selected source Connection, View and Binding revisions are current and
  effective;
- the live source returned a receipt for that exact Binding and View revision;
- the live schema fingerprint matches the governed View revision;
- at least one live source record mapped to a native metric measurement;
- any requested branch or head-office witness was present among successfully
  mapped records; and
- ambiguous or invalid historical rows were left unmapped rather than coerced
  into numeric values.

When `-monitoring-check` is supplied, the command additionally proves that the
latest persisted result for the current Check revision:

- belongs to the selected legal entity through its Program;
- uses `SOURCE` input;
- references the accepted Binding revision;
- carries a source receipt for the accepted Binding, View and schema revision;
  and
- contains a native `MonitoringResult.evaluation.measurement`.

This second proof is revision-based. It does **not** claim that a previously
persisted value must equal the current live source value.

## Run

Use the same application configuration and source credential environment used
by the deployment:

```sh
go run ./cmd/indicator-source-acceptance \
  -tenant <tenant-uuid-or-slug> \
  -legal-entity <legal-entity-uuid-or-code> \
  -binding <binding-uuid> \
  -branch-ref '<exact-branch-reference>' \
  -head-office-ref '<exact-head-office-reference>' \
  -monitoring-check <monitoring-check-uuid> \
  > indicator-source-acceptance.json
```

`-binding` is optional only when exactly one active
channel-performance Binding exists in the selected legal entity. The command
fails on ambiguity instead of choosing one by order.

`-branch-ref` and `-head-office-ref` are optional inputs. For #346
acceptance, provide each witness that the private/deployed dataset is expected
to contain. These values are matched in memory and are never emitted.

`-monitoring-check` is optional. Without it, the receipt proves the deployed
source mapping only. With it, `persisted_indicator.proved` must be `true` to
claim a persisted source-backed Indicator result.

The live read defaults to 50 rows and can only be reduced with `-limit`.
The command never reads more than the Source Catalog preview limit.

## Receipt contents

The JSON receipt contains only safe proof metadata:

- SHA-256 identities for the source, Binding, View and optional Check/Result;
- exact Binding/View revision numbers;
- governed schema fingerprint and mapping digest;
- operation-receipt digest;
- observation/completeness timestamps;
- records read, mapped and unmapped;
- distinct organization count;
- measurement-role counts;
- whether more source rows were available;
- requested witness booleans; and
- optional persisted-result revision-match booleans.

It does not serialize:

- organization or branch names;
- head-office labels;
- source record values;
- metric values;
- source, Binding, View, Check or Result UUIDs;
- credentials or secret references;
- connection definitions;
- recipient data; or
- raw source receipts.

## Interpreting the result

`records_unmapped > 0` is not automatically a failure. Historical text such
as `N/A`, ambiguous periods, or type-incompatible values must remain unmapped.
The acceptance fails only when no record maps, a required witness is absent, or
the governed source identity/schema no longer matches.

`more_available: true` means the proof used a bounded sample. `records_read`,
`records_mapped` and `distinct_organizations` are counts for that sample, not
totals for the source population.

A successful receipt is technical acceptance evidence. It does not certify the
business meaning of a bank's private source, approve a Mapping, or establish
regulatory compliance. Approval remains in the governed Source Catalog and
normal Monitoring lifecycle.

## #346 release use

For the remaining Indicator acceptance gate:

1. run the command in the deployed environment against the private/source-backed
   channel-performance Binding;
2. include the real branch and head-office witness inputs where those records
   are expected;
3. include `-monitoring-check` when a persisted source-backed Indicator exists;
4. retain the redacted JSON receipt with release evidence; and
5. leave any unavailable private population explicitly unproved rather than
   substituting fixture data.

Fixture screenshots and sample records remain useful UI evidence, but they do
not replace this deployed-source receipt.
