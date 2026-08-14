package sourceaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// boundedPayloadQuery makes the response-byte ceiling effective before a row
// crosses the source connection. PostgreSQL may still construct the bounded
// row inside the source transaction, but once cumulative JSON bytes exceed the
// Binding limit it returns only an empty sentinel and an overflow flag. The
// application repeats the byte check as defense in depth.
func boundedPayloadQuery(inner, keyField string, maxBytesParameter int) string {
	maximum := "$" + strconv.Itoa(maxBytesParameter)
	return "WITH clearsight_selected AS (" + inner + "), " +
		"clearsight_payload AS (" +
		"SELECT to_jsonb(clearsight_selected)::text AS payload," + qualifiedSelectedField(keyField) + " AS sort_key " +
		"FROM clearsight_selected), " +
		"clearsight_measured AS (" +
		"SELECT payload,sort_key,sum(octet_length(payload)) OVER (" +
		"ORDER BY sort_key ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS cumulative_bytes " +
		"FROM clearsight_payload) " +
		"SELECT CASE WHEN cumulative_bytes <= " + maximum + " THEN payload ELSE '' END," +
		"cumulative_bytes > " + maximum + " AS limit_exceeded " +
		"FROM clearsight_measured ORDER BY sort_key"
}

func readJSONRecords(ctx context.Context, rows pgx.Rows, selected []string, kinds map[string]ScalarKind, maxBytes int64) ([]Record, int64, error) {
	defer rows.Close()
	records := make([]Record, 0)
	var byteCount int64
	for rows.Next() {
		var payload string
		var limitExceeded bool
		if err := rows.Scan(&payload, &limitExceeded); err != nil {
			return nil, 0, postgresDatabaseError(ctx, ErrExecution)
		}
		if limitExceeded {
			return nil, 0, ErrLimitExceeded
		}
		byteCount += int64(len(payload))
		if byteCount > maxBytes {
			return nil, 0, ErrLimitExceeded
		}
		record, err := decodeJSONRecord(payload, selected, kinds)
		if err != nil {
			return nil, 0, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, postgresDatabaseError(ctx, ErrExecution)
	}
	return records, byteCount, nil
}

func decodeJSONRecord(payload string, selected []string, kinds map[string]ScalarKind) (Record, error) {
	decoder := json.NewDecoder(strings.NewReader(payload))
	decoder.UseNumber()
	var raw map[string]any
	if err := decoder.Decode(&raw); err != nil || len(raw) != len(selected) {
		return nil, ErrUnsupportedValue
	}
	record := make(Record, len(selected))
	for _, field := range selected {
		value, exists := raw[field]
		if !exists {
			return nil, ErrUnsupportedValue
		}
		scalar, err := scalarFromJSON(value, kinds[field])
		if err != nil {
			return nil, err
		}
		record[field] = scalar
	}
	return record, nil
}

func scalarFromJSON(value any, expected ScalarKind) (Scalar, error) {
	if value == nil {
		return Scalar{Kind: ScalarNull}, nil
	}
	switch typed := value.(type) {
	case string:
		if len(typed) > hardMaxRecordScalarBytes {
			return Scalar{}, ErrLimitExceeded
		}
		switch expected {
		case ScalarTime:
			canonical, ok := canonicalPostgresTime(typed)
			if !ok {
				return Scalar{}, ErrUnsupportedValue
			}
			return Scalar{Kind: ScalarTime, Text: canonical}, nil
		case ScalarString:
			return Scalar{Kind: ScalarString, Text: typed}, nil
		default:
			return Scalar{}, ErrUnsupportedValue
		}
	case json.Number:
		if expected != ScalarNumber || len(typed.String()) > hardMaxRecordScalarBytes {
			return Scalar{}, ErrUnsupportedValue
		}
		return Scalar{Kind: ScalarNumber, Text: typed.String()}, nil
	case bool:
		if expected != ScalarBool {
			return Scalar{}, ErrUnsupportedValue
		}
		return Scalar{Kind: ScalarBool, Text: strconv.FormatBool(typed)}, nil
	default:
		return Scalar{}, ErrUnsupportedValue
	}
}

func canonicalPostgresTime(value string) (string, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil && !parsed.IsZero() {
			if layout == "2006-01-02" {
				return value, true
			}
			return parsed.UTC().Format(time.RFC3339Nano), true
		}
	}
	for _, layout := range []string{"2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05.999999999"} {
		if parsed, err := time.ParseInLocation(layout, value, time.UTC); err == nil && !parsed.IsZero() {
			return parsed.UTC().Format(time.RFC3339Nano), true
		}
	}
	return "", false
}

func validateStableRecords(records []Record, keyField string, keyKind ScalarKind) error {
	seen := make(map[string]struct{}, len(records))
	for _, record := range records {
		key, exists := record[keyField]
		if !exists || key.Kind != keyKind || key.Kind == ScalarNull || key.ValidateInput() != nil {
			return ErrExecution
		}
		identity := string(key.Kind) + "\x1f" + key.Text
		if _, exists := seen[identity]; exists {
			return ErrExecution
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func postgresScalarArgument(value Scalar) (any, error) {
	if err := value.ValidateInput(); err != nil {
		return nil, err
	}
	if value.Kind == ScalarBool {
		parsed, err := strconv.ParseBool(value.Text)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid boolean value", ErrDefinitionInvalid)
		}
		return parsed, nil
	}
	return value.Text, nil
}
