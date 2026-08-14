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

The current repository supports revision creation, exact-version reads, current-version reads and bounded current-child lists. Lifecycle transitions, maker-checker administration and user-facing configuration are not part of this change.

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

## Current limitations

- PostgreSQL is the only executable adapter.
- PostgreSQL page and lookup operations support one stable key.
- Catalog administration has no API or user interface.
- Lifecycle transition and maker-checker services are not implemented.
- Connection-, View- and Binding-level health is not reconciled into Source health.
- Cursor, ETag and watermark checkpoints are not stored.
- REST, file, event and non-PostgreSQL database adapters are not implemented.
- Forms, evidence contracts and workflows do not yet retain Binding references.
