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
	hardMaxSubjectKeyBytes       = 256
)

var (
	ErrPopulationInvalid    = errors.New("population definition is invalid")
	ErrSourceCredentials    = errors.New("source credentials are unavailable")
	ErrSourceConnection     = errors.New("source connection is unavailable")
	ErrSourceExecution      = errors.New("source execution failed")
	ErrSourceSchemaChanged  = errors.New("source schema changed")
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

// SchemaFingerprint returns the exact logical schema against which this
// condition was compiled. Executors use it to fail closed on source-schema drift
// before evaluating a predicate built for an older shape.
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
