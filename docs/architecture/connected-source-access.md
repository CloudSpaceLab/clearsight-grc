# Reusable connected-source access

## Purpose

ClearSight must connect to bank data without turning every use case into a new connector, copying source populations into ClearSight, or forcing every institution into one data representation.

The shared boundary is:

```text
Evidence Source                    business authority and ownership
  └─ Source Connection             one technical access path
       └─ Source View              reusable logical resource in native shape
            └─ Source Binding      purpose-bound read and mapping contract
                 ├─ continuous assurance
                 ├─ form prefill, options and validation
                 ├─ evidence search and capture
                 ├─ workflow context
                 └─ AI governance policy resolution
```

`internal/evidence.Source` remains the canonical business-level source registry. The source-access package does not replace it and does not determine evidence sufficiency, compliance, workflow state or action authority.

## T0 implementation decision

The first implementation tranche introduces a transient, reusable execution boundary in `internal/sourceaccess` and moves PostgreSQL connection/session ownership out of `internal/assurance`.

T0 deliberately adds:

- pure Connection, View and Binding contracts;
- capability-specific session interfaces;
- bounded schema inspection, keyset paging, lookup and aggregate execution;
- compact operation receipts;
- one PostgreSQL adapter retaining the existing isolation and timeout controls;
- an assurance compatibility consumer over a shared source session;
- executable proof that one View and Binding can drive both assurance evaluation and a non-assurance lookup.

T0 deliberately does **not** add:

- durable Connection, View or Binding tables;
- routes or user interface;
- a generic source-record table or copied source population;
- a scheduler, event bus, connector-health product or secrets platform;
- HMAC sensitivity bundles, signed lineage headers or another gateway-specific data representation;
- REST, file or event adapters before the shared contract is proven by the existing PostgreSQL implementation.

## Contract ownership

### Evidence Source

The existing Evidence Source remains the authority for:

- tenant and legal-entity scope;
- business identity and authority class;
- owner;
- expected freshness;
- source-level health and observations.

### Source Connection

A Connection identifies one technical path beneath an Evidence Source. The T0 contract records:

- connection ID and version;
- parent Source ID;
- adapter kind and adapter-definition version;
- opaque secret reference;
- bounded adapter-owned definition when an adapter needs one.

Credential material is resolved only when the adapter opens a session. It is never placed in a Connection, View, Binding, operation receipt, log or domain event.

A Source may later have several Connections, such as a database read replica, REST API, SFTP delivery and event endpoint. T0 keeps the type transient until more than one consumer has proved the lifecycle and persistence requirements.

### Source View

A View is a named, versioned logical resource exposed through one Connection. It stores an adapter-owned resource definition, output kind and declared stable keys. It does not store source rows.

Examples include:

- `active_customer_accounts`;
- `privileged_identities`;
- `critical_servers`;
- `vendor_assurance_documents`;
- `approved_model_inventory`.

Native source names and types are preserved. Field names may contain spaces, punctuation, mixed case or Unicode when the source exposes them. The PostgreSQL adapter quotes identifiers mechanically; ClearSight does not require a bank to rename its fields into a universal schema before the source can be used.

### Source Binding

A Binding is the reusable consumer contract over a View. It records:

- binding ID, version and purpose;
- allowed operations;
- selected native fields;
- stable key fields used by lookup or paging;
- hard row, value, byte and time limits.

A Binding contains no query, URL, credential or copied Connection definition. Consumers reference the Binding rather than reproducing integration configuration.

One Binding may allow more than one compatible read operation. This is intentional: a governed account binding can support source-side assurance aggregation and a form or workflow lookup without creating two copies of the same source mapping.

## Capability-specific adapter boundary

The common contract is intentionally small:

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

Adapters implement only the capabilities they support. A later REST adapter does not have to pretend it supports SQL predicates or transactions. A later file adapter does not have to pretend it is a live lookup service.

PostgreSQL source-side predicate aggregation remains an explicit adapter extension consumed by assurance. The generic Session interface is not expanded into a fat connector API merely to hide that capability difference.

## Hard resource limits

T0 enforces non-raiseable ceilings before source execution:

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

Each Binding may choose lower limits. A caller cannot raise a Binding or adapter above the hard ceiling.

Page and lookup operations currently require one declared stable key for PostgreSQL. They reject null or duplicate key results rather than silently skipping, repeating or mis-paginating records. Lookup result rows are bounded by the number of requested stable keys even when a misconfigured source view violates uniqueness.

## Source values

Read operations return bounded records whose values use a small transport representation:

```text
NULL | STRING | NUMBER | BOOL | TIME
```

Numbers remain canonical text rather than passing through `float64`; large account-related numbers and precise decimals therefore retain their exact source value. PostgreSQL timestamps are normalized to UTC for safe cursor reuse. Unsupported compound or ambiguous values fail explicitly rather than being stringified silently.

This transport representation is not a universal bank ontology. Assurance separately maps native database types into its bounded logical condition types. Other consumers may retain native mappings appropriate to their purpose.

## PostgreSQL safety and consistency

The extracted PostgreSQL adapter preserves the existing source executor controls:

- a pool separate from ClearSight's application database pool;
- explicit complete PostgreSQL URL rather than environment fallback;
- deployment-owned opaque secret resolution;
- rejection of superuser, role-creation, database-creation, replication and row-security-bypass principals;
- `default_transaction_read_only=on`;
- repeatable-read, read-only operation transactions;
- statement, lock and idle-transaction timeouts;
- UTC session time;
- removal of caller-supplied `PGOPTIONS` overrides;
- bounded pool size and connection lifetime;
- parameterized lookup and cursor values;
- sanitized connection and execution errors.

SELECT/WITH and one-statement checks remain configuration hygiene, not the security boundary. Least-privilege credentials, read-only transactions and bounded execution are the security boundary.

Schema inspection and assurance aggregate evaluation run in the same repeatable-read transaction. The assurance-owned schema guard is evaluated before the aggregate query. A condition dependency or subject-key type change therefore fails closed without evaluating against a different schema snapshot.

## Operation receipts

Every successful operation returns a compact receipt containing:

- Source ID;
- Connection ID and version;
- adapter kind and version;
- View ID and version;
- Binding ID and version when applicable;
- canonical definition fingerprint;
- native schema fingerprint;
- operation;
- observed time;
- count, returned bytes and completeness.

Receipts contain no source query, secret reference, credentials or source values. T0 returns them to the consumer but does not add a global receipt table. A consuming domain persists a receipt only when its own reconstruction contract requires it.

`PARTIAL` is explicit for a page with a continuation cursor. Source failure, schema mismatch, unsupported value and limit exhaustion are errors; they are never translated into an empty or complete result.

## Assurance compatibility and reuse proof

`PopulationDefinition` remains an assurance execution input, not connector authority. The compatibility executor compiles it into a transient PostgreSQL View and Binding and produces the same tri-state counts and schema-drift behavior as before.

For shared composition:

```text
open sourceaccess Session once
→ pass Session to assurance consumer
→ use the same Session/View/Binding through LookupReader
→ assurance evaluates source-side predicate
→ form/workflow/evidence consumer performs bounded lookup
→ both receipts identify the same Binding fingerprint
```

The assurance wrapper closes only sessions opened through its legacy compatibility constructor. A session supplied by composition remains owned by the caller and reusable by other consumers.

## Failure truth

The following remain distinct:

- invalid Connection/View/Binding definition;
- unavailable credentials;
- unavailable connection;
- unsafe source principal;
- unsupported capability;
- resource limit exhausted;
- source execution failure;
- caller cancellation or deadline;
- assurance-critical schema change.

Errors returned across the adapter boundary are sanitized. A failed source query does not expose connection strings, credentials, query text or source table names.

## Next persistence gate

Do not add `source_connections`, `source_views`, `source_bindings` or checkpoint tables merely because T0 types exist. Persistence begins only after the shared contract has been used by at least two real product consumers and the following are explicit:

- maker-checker lifecycle and effective dating;
- tenant/legal-entity ownership and repository boundaries;
- where-used references;
- adapter configuration validation and preview;
- secret-reference rotation behavior;
- source observation aggregation;
- checkpoint atomicity and replay semantics where required;
- migration ownership, retention and reconstruction tests.
