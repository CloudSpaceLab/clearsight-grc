# Connected source access

## Scope

Connected source access provides one bounded execution contract for external operational data. It prevents connector configuration from being copied into assurance rules or other product domains and does not copy source populations into ClearSight.

```text
Evidence Source                    business identity and ownership
  └─ Source Connection             technical access path
       └─ Source View              logical resource in the source's native shape
            └─ Source Binding      purpose-bound operations, fields and limits
```

`internal/evidence.Source` remains the authoritative business-level source record. Connected source access does not determine evidence sufficiency, compliance state, workflow state or action authority.

The current implementation is transient. It defines the execution contract and PostgreSQL adapter without durable Connection, View or Binding records, API routes or administration UI.

## Domain model

### Evidence Source

The existing Evidence Source owns:

- tenant and legal-entity scope;
- business identity and authority class;
- owner;
- expected freshness;
- source-level health and observations.

### Source Connection

A Connection identifies one technical access path beneath an Evidence Source. It contains:

- connection ID and version;
- parent Source ID;
- adapter kind and adapter version;
- opaque secret reference;
- bounded adapter-owned configuration where required.

Credentials are resolved only while opening an adapter session. Credential material is excluded from Connections, Views, Bindings, receipts, logs and domain events.

The PostgreSQL adapter currently accepts no connection configuration beyond the secret reference. The resolved secret must be a complete PostgreSQL URL.

### Source View

A View is a named and versioned logical resource exposed through one Connection. It contains:

- adapter-owned resource definition;
- output kind;
- declared stable keys.

A View stores no source records. Native field names and types are retained. PostgreSQL identifiers are quoted, so source fields may use mixed case, spaces, punctuation or Unicode without a ClearSight-wide rename.

Stable keys are optional for schema inspection and aggregate execution. PostgreSQL page and lookup operations require exactly one selected stable key.

### Source Binding

A Binding defines the permitted use of a View. It contains:

- binding ID, version and purpose;
- allowed operations;
- selected native fields;
- stable key fields for page and lookup operations;
- row, value, byte and time limits.

A Binding contains no query, endpoint, credential or copied Connection configuration. Consumer domains retain the Binding identity and version instead of duplicating source configuration.

## Adapter contracts

The common interfaces are capability-specific:

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

Adapters expose only supported capabilities. PostgreSQL predicate aggregation is a PostgreSQL-specific extension used by continuous assurance; it is not part of the generic Session interface.

## Resource limits

The execution boundary enforces hard ceilings:

| Resource | Hard ceiling |
|---|---:|
| adapter or view definition | 32 KiB |
| schema fields | 512 |
| selected fields | 512 |
| stable key fields | 4 |
| page rows | 1,000 |
| lookup values | 100 |
| returned bytes per operation | 4 MiB |
| operation timeout | 30 seconds |
| PostgreSQL connections per session | 4 |

A Binding may set lower limits. Values above a hard ceiling are rejected before source execution.

PostgreSQL page execution uses keyset pagination and one look-ahead key. The look-ahead record is excluded from the returned-byte budget. Null or duplicate stable keys fail the operation.

## Value representation

Bounded records use five transport value kinds:

```text
NULL | STRING | NUMBER | BOOL | TIME
```

Numbers remain canonical text in returned records to preserve integers and decimals without `float64` coercion. Lookup values and cursors are converted to the native PostgreSQL key type before parameter binding. Dates and timestamps use explicit PostgreSQL parameter types, and timestamps are normalized to UTC in returned records.

Compound or unsupported values fail explicitly. The transport representation is not a universal data ontology; assurance applies its own bounded logical type mapping.

## PostgreSQL controls

The PostgreSQL adapter enforces:

- a pool separate from ClearSight's application database pool;
- a complete PostgreSQL URL with no environment fallback;
- opaque secret resolution at session open;
- rejection of superuser, role-creation, database-creation, replication, row-security-bypass and non-system relation-owning principals;
- `default_transaction_read_only=on`;
- repeatable-read, read-only operation transactions;
- statement, lock and idle-transaction timeouts;
- UTC session time;
- removal of caller-supplied `PGOPTIONS` overrides;
- bounded pool size and connection lifetime;
- parameterized lookup and cursor values;
- sanitized connection and execution errors.

SELECT/WITH and single-statement validation provide configuration hygiene. The security boundary is the dedicated non-owner source credential, read-only transaction, bounded connection pool and operation deadline.

Schema inspection and assurance aggregation run in the same repeatable-read transaction. Assurance validates condition dependencies and the subject-key type before running the aggregate query.

## Operation receipts

Each successful operation returns a receipt containing:

- Source ID;
- Connection ID and version;
- adapter kind and version;
- View ID and version;
- Binding ID and version when applicable;
- definition fingerprint;
- native schema fingerprint;
- operation;
- observed time;
- count, returned bytes and completeness.

Receipts exclude queries, secret references, credentials and source values. The current runtime returns receipts to the caller and does not persist them globally.

`PARTIAL` identifies a page with a continuation cursor. Invalid definitions, unsupported capabilities, limit exhaustion, source failure, schema change and caller cancellation remain separate outcomes.

## Assurance integration

`PopulationDefinition` remains a compatibility input for continuous assurance. The assurance executor compiles it into a transient PostgreSQL View and Binding while retaining the previous population and full-schema fingerprints.

For reusable bindings, assurance records the selected logical schema. A condition dependency or subject-key type change fails closed; changes to unselected fields do not invalidate the condition.

A caller may open one source session and provide it to assurance and another bounded reader. The caller owns the lifecycle of an injected session. The compatibility constructor owns and closes the session it opens.

## Not implemented

The current source-access implementation does not include:

- durable, effective-dated Connection, View and Binding records;
- maker-checker administration and where-used references;
- source-access API routes and configuration UI;
- connection-, view- or binding-level health reconciliation;
- cursor or watermark checkpoints;
- REST, file, event and non-PostgreSQL database adapters;
- form, evidence and workflow binding references;
- persistent source-operation receipts.

Durable source-access records require explicit ownership, lifecycle, secret rotation, health, checkpoint, retention and reconstruction contracts.
