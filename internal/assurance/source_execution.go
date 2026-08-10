package assurance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	hardMaxPopulationQueryBytes = 32 << 10
	hardMaxSubjectKeyBytes      = 256
)

var (
	ErrPopulationInvalid   = errors.New("population definition is invalid")
	ErrSourceCredentials   = errors.New("source credentials are unavailable")
	ErrSourceConnection    = errors.New("source connection is unavailable")
	ErrSourcePrivileges    = errors.New("source credential has unsafe privileges")
	ErrSourceExecution     = errors.New("source execution failed")
	ErrSourceSchemaChanged = errors.New("source schema changed")
)

// SourceSecretResolver resolves an opaque deployment-managed secret reference.
// Implementations must return credential material only to the executor boundary;
// callers must never persist the returned value in ClearSight state or events.
type SourceSecretResolver interface {
	Resolve(ctx context.Context, secretRef string) (string, error)
}

// PopulationDefinition describes one approved queryable population within an
// existing Evidence Source. It is an execution input only in this tranche; it is
// not durable configuration yet.
type PopulationDefinition struct {
	ID         string `json:"id"`
	Query      string `json:"query"`
	SubjectKey string `json:"subject_key"`
}

type SourceSchema struct {
	SourceID              string    `json:"source_id"`
	PopulationID          string    `json:"population_id"`
	PopulationFingerprint string    `json:"population_fingerprint"`
	SchemaFingerprint     string    `json:"schema_fingerprint"`
	Schema                Schema    `json:"schema"`
	InspectedAt           time.Time `json:"inspected_at"`
}

type EvaluationReceipt struct {
	SourceID              string    `json:"source_id"`
	PopulationID          string    `json:"population_id"`
	PopulationFingerprint string    `json:"population_fingerprint"`
	SchemaFingerprint     string    `json:"schema_fingerprint"`
	TotalCount            int64     `json:"total_count"`
	MatchCount            int64     `json:"match_count"`
	UnknownCount          int64     `json:"unknown_count"`
	ClearCount            int64     `json:"clear_count"`
	EvaluatedAt           time.Time `json:"evaluated_at"`
	Complete              bool      `json:"complete"`
}

func normalizePopulationDefinition(value PopulationDefinition) (PopulationDefinition, error) {
	value.ID = strings.TrimSpace(value.ID)
	value.SubjectKey = strings.TrimSpace(value.SubjectKey)
	if value.ID == "" || value.SubjectKey == "" {
		return PopulationDefinition{}, fmt.Errorf("%w: id and subject_key are required", ErrPopulationInvalid)
	}
	if len(value.SubjectKey) > hardMaxSubjectKeyBytes {
		return PopulationDefinition{}, fmt.Errorf("%w: subject_key exceeds %d bytes", ErrPopulationInvalid, hardMaxSubjectKeyBytes)
	}
	query, err := normalizePopulationQuery(value.Query)
	if err != nil {
		return PopulationDefinition{}, err
	}
	value.Query = query
	return value, nil
}

func normalizePopulationQuery(query string) (string, error) {
	if strings.IndexByte(query, 0) >= 0 {
		return "", fmt.Errorf("%w: query contains NUL", ErrPopulationInvalid)
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return "", fmt.Errorf("%w: query is required", ErrPopulationInvalid)
	}
	if len(query) > hardMaxPopulationQueryBytes {
		return "", fmt.Errorf("%w: query exceeds %d bytes", ErrPopulationInvalid, hardMaxPopulationQueryBytes)
	}
	if strings.HasSuffix(query, ";") {
		query = strings.TrimSpace(strings.TrimSuffix(query, ";"))
	}
	if query == "" || strings.Contains(query, ";") {
		return "", fmt.Errorf("%w: query must contain exactly one SELECT/WITH statement", ErrPopulationInvalid)
	}
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return "", fmt.Errorf("%w: query is required", ErrPopulationInvalid)
	}
	switch strings.ToUpper(fields[0]) {
	case "SELECT", "WITH":
		return query, nil
	default:
		return "", fmt.Errorf("%w: query must start with SELECT or WITH", ErrPopulationInvalid)
	}
}

func populationFingerprint(population PopulationDefinition) string {
	hash := sha256.Sum256([]byte(population.Query + "\x1f" + population.SubjectKey))
	return hex.EncodeToString(hash[:])
}

// SchemaFingerprint returns the complete logical schema against which this
// condition was compiled. It is useful for reconstruction and diagnostics.
func (c *CompiledCondition) SchemaFingerprint() (string, error) {
	if c == nil {
		return "", fmt.Errorf("compiled condition is required")
	}
	fields := make([]Field, 0, len(c.fields))
	for _, field := range c.fields {
		fields = append(fields, field)
	}
	return (Schema{Fields: fields}).Fingerprint()
}

// RequiredSchemaFingerprint identifies only fields that can change the rule's
// result plus the stable subject key. Unrelated projected columns may evolve
// without turning an otherwise evaluable rule into false schema drift.
func (c *CompiledCondition) RequiredSchemaFingerprint(subjectKey string) (string, error) {
	if c == nil {
		return "", fmt.Errorf("compiled condition is required")
	}
	return schemaFingerprintForFields(Schema{Fields: mapFields(c.fields)}, c.requiredSchemaFields(subjectKey))
}

func (c *CompiledCondition) requiredSchemaFields(subjectKey string) []string {
	fields := append([]string(nil), c.dependencies...)
	for _, name := range fields {
		if name == subjectKey {
			return fields
		}
	}
	return append(fields, subjectKey)
}

func schemaFingerprintForFields(schema Schema, names []string) (string, error) {
	fields := make([]Field, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if _, exists := seen[name]; exists {
			continue
		}
		field, exists := schema.Field(name)
		if !exists {
			return "", fmt.Errorf("%w: execution-critical field %q is missing", ErrSourceSchemaChanged, name)
		}
		seen[name] = struct{}{}
		fields = append(fields, field)
	}
	return (Schema{Fields: fields}).Fingerprint()
}

func mapFields(values map[string]Field) []Field {
	fields := make([]Field, 0, len(values))
	for _, field := range values {
		fields = append(fields, field)
	}
	return fields
}
