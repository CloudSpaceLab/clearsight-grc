package sourceaccess

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// boundedPayloadQuery enforces the response-byte limit before complete record
// payloads leave PostgreSQL. The application repeats the check after scanning.
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

// boundedPagePayloadQuery excludes the single look-ahead row from the response
// byte budget and returns only its stable key. The look-ahead row determines
// whether a continuation cursor is required.
func boundedPagePayloadQuery(inner, keyField string, pageRows, maxBytesParameter int) string {
	maximum := "$" + strconv.Itoa(maxBytesParameter)
	pageLimit := strconv.Itoa(pageRows)
	return "WITH clearsight_selected AS (" + inner + "), " +
		"clearsight_payload AS (" +
		"SELECT to_jsonb(clearsight_selected)::text AS payload," + qualifiedSelectedField(keyField) + " AS sort_key," +
		"row_number() OVER (ORDER BY " + qualifiedSelectedField(keyField) + ") AS row_number " +
		"FROM clearsight_selected), " +
		"clearsight_measured AS (" +
		"SELECT payload,sort_key,row_number," +
		"sum(CASE WHEN row_number <= " + pageLimit + " THEN octet_length(payload) ELSE 0 END) OVER (" +
		"ORDER BY row_number ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS cumulative_bytes " +
		"FROM clearsight_payload) " +
		"SELECT CASE WHEN row_number <= " + pageLimit + " AND cumulative_bytes <= " + maximum + " THEN payload ELSE '' END," +
		"(row_number <= " + pageLimit + " AND cumulative_bytes > " + maximum + ") AS limit_exceeded," +
		"(row_number > " + pageLimit + ") AS lookahead," +
		"CASE WHEN row_number > " + pageLimit + " THEN to_jsonb(sort_key)::text ELSE '' END AS lookahead_key " +
		"FROM clearsight_measured ORDER BY row_number"
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

func readJSONPageRecords(ctx context.Context, rows pgx.Rows, selected []string, kinds map[string]ScalarKind, keyKind ScalarKind, maxBytes int64) ([]Record, int64, *Scalar, error) {
	defer rows.Close()
	records := make([]Record, 0)
	var byteCount int64
	var lookahead *Scalar
	for rows.Next() {
		var payload string
		var limitExceeded bool
		var isLookahead bool
		var lookaheadPayload string
		if err := rows.Scan(&payload, &limitExceeded, &isLookahead, &lookaheadPayload); err != nil {
			return nil, 0, nil, postgresDatabaseError(ctx, ErrExecution)
		}
		if limitExceeded {
			return nil, 0, nil, ErrLimitExceeded
		}
		if isLookahead {
			if lookahead != nil || lookaheadPayload == "" {
				return nil, 0, nil, ErrExecution
			}
			key, err := decodeJSONScalar(lookaheadPayload, keyKind)
			if err != nil || key.Kind == ScalarNull || key.ValidateInput() != nil {
				return nil, 0, nil, ErrExecution
			}
			keyCopy := key
			lookahead = &keyCopy
			continue
		}
		if lookahead != nil {
			return nil, 0, nil, ErrExecution
		}
		byteCount += int64(len(payload))
		if byteCount > maxBytes {
			return nil, 0, nil, ErrLimitExceeded
		}
		record, err := decodeJSONRecord(payload, selected, kinds)
		if err != nil {
			return nil, 0, nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, nil, postgresDatabaseError(ctx, ErrExecution)
	}
	return records, byteCount, lookahead, nil
}

func decodeJSONRecord(payload string, selected []string, kinds map[string]ScalarKind) (Record, error) {
	decoder := json.NewDecoder(strings.NewReader(payload))
	decoder.UseNumber()
	var raw map[string]any
	if err := decoder.Decode(&raw); err != nil || len(raw) != len(selected) {
		return nil, ErrUnsupportedValue
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
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

func decodeJSONScalar(payload string, expected ScalarKind) (Scalar, error) {
	decoder := json.NewDecoder(strings.NewReader(payload))
	decoder.UseNumber()
	var raw any
	if err := decoder.Decode(&raw); err != nil {
		return Scalar{}, ErrUnsupportedValue
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Scalar{}, ErrUnsupportedValue
	}
	return scalarFromJSON(raw, expected)
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
		identity := scalarIdentity(key)
		if _, exists := seen[identity]; exists {
			return ErrExecution
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func scalarIdentity(value Scalar) string {
	return string(value.Kind) + "\x1f" + value.Text
}

func postgresScalarArgument(value Scalar, nativeType string) (any, error) {
	if err := value.ValidateInput(); err != nil {
		return nil, err
	}
	if value.Kind != postgresScalarKind(nativeType) {
		return nil, fmt.Errorf("%w: lookup or cursor value does not match the PostgreSQL key type", ErrDefinitionInvalid)
	}
	switch strings.ToLower(strings.TrimSpace(nativeType)) {
	case "bool", "boolean":
		parsed, err := strconv.ParseBool(value.Text)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid PostgreSQL boolean key", ErrDefinitionInvalid)
		}
		return parsed, nil
	case "smallint", "int2":
		parsed, err := strconv.ParseInt(value.Text, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid PostgreSQL smallint key", ErrDefinitionInvalid)
		}
		return int16(parsed), nil
	case "integer", "int", "int4":
		parsed, err := strconv.ParseInt(value.Text, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid PostgreSQL integer key", ErrDefinitionInvalid)
		}
		return int32(parsed), nil
	case "bigint", "int8":
		parsed, err := strconv.ParseInt(value.Text, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid PostgreSQL bigint key", ErrDefinitionInvalid)
		}
		return parsed, nil
	case "numeric", "decimal":
		parsed, err := parsePostgresNumeric(value.Text)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid PostgreSQL numeric key", ErrDefinitionInvalid)
		}
		return parsed, nil
	case "real", "float4":
		parsed, err := strconv.ParseFloat(value.Text, 32)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid PostgreSQL real key", ErrDefinitionInvalid)
		}
		return float32(parsed), nil
	case "double precision", "float8", "float":
		parsed, err := strconv.ParseFloat(value.Text, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid PostgreSQL double-precision key", ErrDefinitionInvalid)
		}
		return parsed, nil
	case "date":
		parsed, err := time.Parse("2006-01-02", value.Text)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid PostgreSQL date key", ErrDefinitionInvalid)
		}
		return pgtype.Date{Time: parsed, Valid: true}, nil
	case "timestamp", "timestamp without time zone", "datetime":
		parsed, err := parseCanonicalTime(value.Text)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid PostgreSQL timestamp key", ErrDefinitionInvalid)
		}
		return pgtype.Timestamp{Time: parsed, Valid: true}, nil
	case "timestamp with time zone", "timestamptz":
		parsed, err := parseCanonicalTime(value.Text)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid PostgreSQL timestamp-with-time-zone key", ErrDefinitionInvalid)
		}
		return pgtype.Timestamptz{Time: parsed.UTC(), Valid: true}, nil
	default:
		return value.Text, nil
	}
}

func parsePostgresNumeric(value string) (pgtype.Numeric, error) {
	mantissa := value
	var exponent int64
	if index := strings.IndexAny(value, "eE"); index >= 0 {
		mantissa = value[:index]
		parsedExponent, err := strconv.ParseInt(value[index+1:], 10, 64)
		if err != nil {
			return pgtype.Numeric{}, err
		}
		exponent = parsedExponent
	}
	fractionDigits := 0
	if index := strings.IndexByte(mantissa, '.'); index >= 0 {
		fractionDigits = len(mantissa) - index - 1
		mantissa = mantissa[:index] + mantissa[index+1:]
	}
	integer := new(big.Int)
	if _, ok := integer.SetString(mantissa, 10); !ok {
		return pgtype.Numeric{}, fmt.Errorf("invalid numeric mantissa")
	}
	exponent -= int64(fractionDigits)
	if exponent < math.MinInt32 || exponent > math.MaxInt32 {
		return pgtype.Numeric{}, fmt.Errorf("numeric exponent out of range")
	}
	return pgtype.Numeric{Int: integer, Exp: int32(exponent), Valid: true}, nil
}

func parseCanonicalTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		if parsed, err := time.Parse(layout, value); err == nil && !parsed.IsZero() {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid canonical time")
}
