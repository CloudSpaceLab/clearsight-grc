# Connected source access

## Scope

Connected source access provides one reusable, bounded contract for external operational data. It prevents connection details and field mappings from being copied into assurance rules, forms or workflows, and it does not copy source populations into ClearSight.

```text
Evidence Source                    business identity and ownership
  └─ Source Connection             technical access path
       └─ Source View              logical resource in the source's native shape
            └─ Source Binding      permitted operations, fields and limits
```

`internal/evidence.Source` remains the business-level source record. A Connection, View or Binding does not determine evidence sufficiency, compliance state, workflow state or action authority.

## Durable catalog

The catalog stores immutable revision rows in three tables:

| Record | Identity | Exact parent |
| --- | --- | --- |
| Connection revision | `connection_id` + `version` | Evidence Source |
| View revision | `view_id` + `version` | Connection ID + Connection version |
| Binding revision | `binding_id` + `version` | View ID + View version |

Each revision records tenant and Source scope, code, name, lifecycle state, current designation, effective period, creator and timestamps. A stable resource identity cannot move to another tenant, Source or parent across versions.

Only one revision of a resource may be current. Current codes are unique within their parent scope. A current View must reference the current Connection revision. A current Binding must reference the current View revision. A parent revision cannot be retired while a current child still references it.

The repository supports revision creation, exact-version reads, current-version reads and bounded revision-history lists. Permissioned `CONFIG_READ` / `CONFIG_WRITE` routes expose server-owned draft creation, immutable schema inspection, bounded preview and where-used discovery. Lifecycle activation and maker-checker transitions remain a later governance tranche rather than implicit catalog mutation.

## Evidence Source endpoint migration

`evidence_sources.endpoint` is removed by migration `000030_source_access_catalog`.

A nonblank legacy endpoint becomes an active `REFERENCE` Connection:

```text
code             PRIMARY_REFERENCE
adapter kind     REFERENCE
adapter version  reference-v1
definition       {"endpoint":"..."}
```

A `REFERENCE` Connection records a location or external reference. It has no execution capabilities, no credential and cannot own an executable View.

The existing source-creation request may still provide `endpoint`. PostgreSQL consumes it while creating the Source and its initial `REFERENCE` Connection in one transaction. Endpoint is not returned on the Source and is not stored in two places.

The down migration restores the endpoint column only when the catalog contains reference-only migrated data. It refuses rollback if any executable Connection, View, Binding or additional revision would be lost.

## Connection revision

A Connection revision records:

- adapter kind and adapter-definition version;
- opaque secret reference, never credential material;
- adapter-owned JSON configuration;
- declared and verified capabilities;
- accountable owner;
- lifecycle and effective period.

Verified capabilities must be a subset of declared capabilities. Connection configuration is bounded to 32 KiB.

PostgreSQL execution credentials must remain dedicated non-owner readers. Runtime source pools are separate from the ClearSight application database pool.

## View revision

A View is a named logical resource exposed through one exact Connection revision. It records:

- adapter-owned resource definition;
- native output kind;
- stable keys;
- native field names and types;
- schema fingerprint;
- lifecycle and effective period.

Native field names are retained. ClearSight does not require a bank-wide canonical schema before a source can be used. Current Views require an inspected schema fingerprint.

## Binding revision

A Binding defines one permitted use of a View. It records:

- purpose;
- allowed operations;
- selected fields;
- stable key fields;
- page, lookup, response-byte and timeout limits;
- parameter and output schemas;
- mapping and sensitivity-handling configuration;
- freshness and completeness requirements;
- lifecycle and effective period.

A Binding contains no query, endpoint or credential. Consumers retain its ID and version instead of copying integration configuration.

The selected fields must exist in the exact View schema. Page and lookup Bindings require selected stable keys. Resource limits are normalized and stored as concrete values when the revision is created.

## Adapter contracts

Runtime access remains capability-specific:

```go
type Adapter interface {
    Open(context.Context, Connection, SecretResolver) (Session, error)
}

type Session interface {
    Connection() Connection
    Capabilities() CapabilitySet
    Close() error
}

type SchemaReader interface {
    Inspect(context.Context, View) (SchemaResult, error)
}

type PageReader interface {
    ReadPage(context.Context, View, Binding, PageRequest) (RecordPage, error)
}

type LookupReader interface {
    Lookup(context.Context, View, Binding, LookupRequest) (LookupResult, error)
}
```

Catalog revisions compile into these bounded runtime contracts. Adapters expose only the capabilities they implement.

## Resource limits

| Resource | Hard ceiling |
| --- | ---: |
| adapter or View definition | 32 KiB |
| schema fields | 512 |
| selected fields | 512 |
| stable key fields | 4 |
| page rows | 1,000 |
| lookup values | 100 |
| returned bytes per operation | 4 MiB |
| operation timeout | 30 seconds |
| PostgreSQL connections per session | 4 |

A Binding may set lower limits. Values above a hard ceiling are rejected before source execution.

## Value representation

Bounded records use five transport value kinds:

```text
NULL | STRING | NUMBER | BOOL | TIME
```

Numbers remain canonical text in returned records. Lookup values and cursors are converted to the native PostgreSQL key type before parameter binding. Dates and timestamps use explicit PostgreSQL parameter types, and returned timestamps are normalized to UTC.

This transport representation is not a universal bank data ontology. Assurance applies its own bounded logical type mapping.

## Operation receipts

Each successful operation returns a receipt containing:

- Source, Connection, View and Binding identity and version;
- adapter kind and version;
- definition and native-schema fingerprints;
- operation and observed time;
- count, returned bytes and completeness.

Receipts exclude queries, secret references, credentials and source values. They are returned to the caller and are not persisted globally by the source-access package.

## Stateful Binding checkpoints

Bindings that support `PAGE` or `CHANGES` may own one infrastructure checkpoint for an exact Binding revision. The checkpoint records only a bounded cursor, ETag, watermark or event ID plus runtime lease/retry state; it is not business truth and it does not copy source rows.

Checkpoint workers use the same claim/lease/backoff semantics as ClearSight runtime work. Advancement requires an existing durable runtime inbox receipt for the corresponding consumer/event. If processing commits and the process dies before checkpoint advancement, the lease expires, the same source position is replayed, the inbox receipt suppresses duplicate domain processing, and the checkpoint can then advance without skipping records.

Raw source errors are not persisted in checkpoint state. Only bounded error codes are retained.

## Scoped source health

The existing `source_observations` history now accepts exact `SOURCE`, `CONNECTION`, `VIEW` and `BINDING` scopes. Child-scoped observations retain exact parent revisions. The current health of each exact scope is derived from its latest observation, and Evidence Source health is the worst current scoped state (`UNAVAILABLE` → `STALE` → `DEGRADED` → `UNKNOWN` → `CURRENT`). An unrelated healthy path therefore cannot hide a failed Binding.

Freshness maintenance re-evaluates successful observations using the Source freshness window. Observations that arrive out of order remain historical evidence but cannot replace a newer observation for the same scope. Health remains part of the existing Evidence Source / Source Observation model; no connector-health authority is introduced.

## REST/JSON adapter — T2a

`REST_JSON` is the second executable adapter and intentionally remains narrower than a generic HTTP client. A Connection owns one fixed HTTPS base origin and an optional bearer or safe-header secret reference. A View owns fixed GET paths, fixed query values, a JSON records pointer, optional bounded cursor/ETag pagination and an optional repeated-query lookup contract. Runtime callers can supply only bounded page cursors or lookup scalar values through an activated Binding; they cannot supply URLs, paths, headers or arbitrary request templates.

Successful reads return the same bounded `Record`, schema and operation-receipt contracts as PostgreSQL. Response bodies are size-bounded after decompression, redirects are rejected, authentication material is never copied into receipts, and the inspected native schema is checked on later reads. JSON optional fields may be absent, but new fields or incompatible scalar types are treated as schema drift instead of silently changing the Binding contract.

REST pagination exposes opaque cursor or ETag source positions through the existing checkpoint representation. Scheduling/retry remains owned by `internal/runtime`; the adapter does not add a pull scheduler or cache.

## Current limitations

- PostgreSQL and REST/JSON are executable adapters; tabular-file and webhook/event adapters remain T2 work.
- PostgreSQL page and lookup operations support one stable key.
- Catalog configuration has APIs but no dedicated user interface yet.
- Lifecycle transition and maker-checker services are not implemented.
- REST, file, event and non-PostgreSQL database adapters are not implemented.
- Forms, evidence contracts and workflows do not yet retain Binding references.
