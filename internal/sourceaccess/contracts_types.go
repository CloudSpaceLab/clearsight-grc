package sourceaccess

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

const (
	HardMaxDefinitionBytes   = 32 << 10
	HardMaxIdentifierBytes   = 256
	HardMaxSelectedFields    = 512
	HardMaxSchemaFields      = 512
	HardMaxStableKeyFields   = 4
	HardMaxPageRows          = 1000
	HardMaxResponseBytes     = 4 << 20
	HardMaxLookupValues      = 100
	HardMaxOperationTimeout  = 30 * time.Second
	DefaultPageRows          = 100
	DefaultResponseBytes     = 1 << 20
	DefaultLookupValues      = 25
	DefaultOperationTimeout  = 5 * time.Second
	hardMaxLookupScalarBytes = 4 << 10
	hardMaxRecordScalarBytes = 64 << 10
	hardMaxAdapterVersionLen = 128
)

var (
	ErrDefinitionInvalid     = errors.New("source access definition is invalid")
	ErrCredentials           = errors.New("source credentials are unavailable")
	ErrConnection            = errors.New("source connection is unavailable")
	ErrPrivileges            = errors.New("source credential has unsafe privileges")
	ErrExecution             = errors.New("source execution failed")
	ErrCapabilityUnavailable = errors.New("source capability is unavailable")
	ErrLimitExceeded         = errors.New("source access limit exceeded")
	ErrUnsupportedValue      = errors.New("source value is unsupported")
)

type AdapterKind string

const AdapterPostgres AdapterKind = "POSTGRES"

type Capability string

const (
	CapabilityInspect   Capability = "INSPECT"
	CapabilityPage      Capability = "PAGE"
	CapabilityLookup    Capability = "LOOKUP"
	CapabilityAggregate Capability = "AGGREGATE"
	CapabilityChanges   Capability = "CHANGES"
)

type Operation string

const (
	OperationInspect   Operation = "INSPECT"
	OperationPage      Operation = "PAGE"
	OperationLookup    Operation = "LOOKUP"
	OperationAggregate Operation = "AGGREGATE"
	OperationChanges   Operation = "CHANGES"
)

type Completeness string

const (
	CompletenessComplete Completeness = "COMPLETE"
	CompletenessPartial  Completeness = "PARTIAL"
	CompletenessUnknown  Completeness = "UNKNOWN"
)

type OutputKind string

const OutputRecords OutputKind = "RECORDS"

// SecretResolver resolves an opaque deployment-owned secret reference. The
// returned material exists only inside the adapter boundary and must never be
// copied into source definitions, receipts, logs or domain state.
type SecretResolver interface {
	Resolve(ctx context.Context, secretRef string) (string, error)
}

// Connection identifies one technical access path beneath an existing
// business-level Evidence Source. Definition is adapter-owned JSON; credentials
// remain behind SecretRef.
type Connection struct {
	ID             string          `json:"id"`
	SourceID       string          `json:"source_id"`
	Version        string          `json:"version"`
	AdapterKind    AdapterKind     `json:"adapter_kind"`
	AdapterVersion string          `json:"adapter_version"`
	SecretRef      string          `json:"secret_ref"`
	Definition     json.RawMessage `json:"definition,omitempty"`
}

// View is a reusable logical resource in the source's native shape. It keeps
// an adapter-owned definition and stable key fields; it does not materialize
// the source population in ClearSight.
type View struct {
	ID           string          `json:"id"`
	ConnectionID string          `json:"connection_id"`
	Version      string          `json:"version"`
	OutputKind   OutputKind      `json:"output_kind"`
	Definition   json.RawMessage `json:"definition"`
	StableKeys   []string        `json:"stable_keys"`
}

// Binding is a reusable, purpose-bound read contract. Consumer domains retain
// its ID and version rather than copying connection, query or mapping details.
type Binding struct {
	ID             string         `json:"id"`
	ViewID         string         `json:"view_id"`
	Version        string         `json:"version"`
	Purpose        string         `json:"purpose"`
	Operations     []Operation    `json:"operations"`
	SelectedFields []string       `json:"selected_fields"`
	KeyFields      []string       `json:"key_fields,omitempty"`
	Limits         ResourceLimits `json:"limits"`
}

type ResourceLimits struct {
	PageRows      int           `json:"page_rows"`
	ResponseBytes int64         `json:"response_bytes"`
	LookupValues  int           `json:"lookup_values"`
	Timeout       time.Duration `json:"timeout"`
}

func DefaultResourceLimits() ResourceLimits {
	return ResourceLimits{
		PageRows:      DefaultPageRows,
		ResponseBytes: DefaultResponseBytes,
		LookupValues:  DefaultLookupValues,
		Timeout:       DefaultOperationTimeout,
	}
}

func (l ResourceLimits) Normalized() (ResourceLimits, error) {
	if l.PageRows < 0 || l.ResponseBytes < 0 || l.LookupValues < 0 || l.Timeout < 0 {
		return ResourceLimits{}, ErrLimitExceeded
	}
	defaults := DefaultResourceLimits()
	if l.PageRows == 0 {
		l.PageRows = defaults.PageRows
	}
	if l.ResponseBytes == 0 {
		l.ResponseBytes = defaults.ResponseBytes
	}
	if l.LookupValues == 0 {
		l.LookupValues = defaults.LookupValues
	}
	if l.Timeout == 0 {
		l.Timeout = defaults.Timeout
	}
	if l.PageRows > HardMaxPageRows || l.ResponseBytes > HardMaxResponseBytes || l.LookupValues > HardMaxLookupValues || l.Timeout > HardMaxOperationTimeout {
		return ResourceLimits{}, ErrLimitExceeded
	}
	return l, nil
}

type CapabilitySet map[Capability]struct{}

func NewCapabilitySet(values ...Capability) CapabilitySet {
	result := make(CapabilitySet, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func (s CapabilitySet) Has(value Capability) bool {
	_, exists := s[value]
	return exists
}

type ScalarKind string

const (
	ScalarNull   ScalarKind = "NULL"
	ScalarString ScalarKind = "STRING"
	ScalarNumber ScalarKind = "NUMBER"
	ScalarBool   ScalarKind = "BOOL"
	ScalarTime   ScalarKind = "TIME"
)

// Scalar keeps exact source values as bounded canonical text. Numbers are not
// coerced through float64, so identifiers and large decimal values remain exact.
type Scalar struct {
	Kind ScalarKind `json:"kind"`
	Text string     `json:"text,omitempty"`
}

func StringValue(value string) Scalar { return Scalar{Kind: ScalarString, Text: value} }

type Record map[string]Scalar

type NativeField struct {
	Name       string `json:"name"`
	NativeType string `json:"native_type"`
	Nullable   bool   `json:"nullable"`
}

type OperationReceipt struct {
	SourceID              string       `json:"source_id"`
	ConnectionID          string       `json:"connection_id"`
	ConnectionVersion     string       `json:"connection_version"`
	AdapterKind           AdapterKind  `json:"adapter_kind"`
	AdapterVersion        string       `json:"adapter_version"`
	ViewID                string       `json:"view_id"`
	ViewVersion           string       `json:"view_version"`
	BindingID             string       `json:"binding_id,omitempty"`
	BindingVersion        string       `json:"binding_version,omitempty"`
	DefinitionFingerprint string       `json:"definition_fingerprint"`
	SchemaFingerprint     string       `json:"schema_fingerprint,omitempty"`
	Operation             Operation    `json:"operation"`
	ObservedAt            time.Time    `json:"observed_at"`
	Count                 int64        `json:"count"`
	Bytes                 int64        `json:"bytes"`
	Completeness          Completeness `json:"completeness"`
}

type SchemaResult struct {
	Fields  []NativeField    `json:"fields"`
	Receipt OperationReceipt `json:"receipt"`
}

type PageRequest struct {
	After *Scalar `json:"after,omitempty"`
	Limit int     `json:"limit,omitempty"`
}

type RecordPage struct {
	Records    []Record         `json:"records"`
	NextCursor *Scalar          `json:"next_cursor,omitempty"`
	Receipt    OperationReceipt `json:"receipt"`
}

type LookupRequest struct {
	Values []Scalar `json:"values"`
}

type LookupResult struct {
	Records []Record         `json:"records"`
	Receipt OperationReceipt `json:"receipt"`
}

type AggregateResult struct {
	Fields       []NativeField    `json:"fields"`
	TotalCount   int64            `json:"total_count"`
	MatchCount   int64            `json:"match_count"`
	UnknownCount int64            `json:"unknown_count"`
	ClearCount   int64            `json:"clear_count"`
	Receipt      OperationReceipt `json:"receipt"`
}

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
