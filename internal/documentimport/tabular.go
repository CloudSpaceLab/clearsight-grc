package documentimport

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const TabularParserVersion = "TABULAR_V1"

type TabularFormat string

const (
	TabularCSV    TabularFormat = "CSV"
	TabularJSON   TabularFormat = "JSON"
	TabularNDJSON TabularFormat = "NDJSON"
	TabularXLSX   TabularFormat = "XLSX"
)

type TabularRowError struct {
	Resource string `json:"resource"`
	Row      int    `json:"row"`
	Message  string `json:"message"`
}

type TabularField struct {
	Name       string `json:"name"`
	NativeType string `json:"native_type"`
	Nullable   bool   `json:"nullable"`
}

type TabularResource struct {
	Name              string         `json:"name"`
	RowsTotal         int            `json:"rows_total"`
	RowsRejected      int            `json:"rows_rejected"`
	Fields            []TabularField `json:"fields"`
	SchemaFingerprint string         `json:"schema_fingerprint,omitempty"`
}

type TabularMetadata struct {
	Format        TabularFormat     `json:"format"`
	ParserVersion string            `json:"parser_version"`
	RowsTotal     int               `json:"rows_total"`
	RowsRejected  int               `json:"rows_rejected"`
	Resources     []TabularResource `json:"resources"`
	RowErrors     []TabularRowError `json:"row_errors"`
	FatalError    string            `json:"fatal_error,omitempty"`
}

type tabularCell struct {
	kind string
	text string
}

type tabularRow struct {
	Resource string
	Number   int
	Values   map[string]tabularCell
}

type tabularVisitor func(tabularRow) error

var errTabularStop = errors.New("stop tabular scan")

type tabularFieldState struct {
	native   string
	nullable bool
	present  int
}

type tabularResourceBuilder struct {
	name       string
	fixedOrder []string
	fields     map[string]*tabularFieldState
	rowsTotal  int
	rejected   int
	accepted   int
}

type tabularScanState struct {
	format    TabularFormat
	policy    ExtractionPolicy
	resources map[string]*tabularResourceBuilder
	order     []string
	errors    []TabularRowError
}

func DetectTabularFormat(fileName, mediaType string) (TabularFormat, bool) {
	ext := strings.ToLower(strings.TrimSpace(filepathExtension(fileName)))
	switch ext {
	case ".csv":
		return TabularCSV, true
	case ".json":
		return TabularJSON, true
	case ".ndjson", ".jsonl":
		return TabularNDJSON, true
	case ".xlsx":
		return TabularXLSX, true
	}
	media := strings.ToLower(strings.TrimSpace(strings.Split(mediaType, ";")[0]))
	switch media {
	case "text/csv", "application/csv":
		return TabularCSV, true
	case "application/json":
		return TabularJSON, true
	case "application/x-ndjson", "application/ndjson", "application/jsonlines":
		return TabularNDJSON, true
	case "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":
		return TabularXLSX, true
	default:
		return "", false
	}
}

func filepathExtension(value string) string {
	index := strings.LastIndex(strings.TrimSpace(value), ".")
	if index < 0 {
		return ""
	}
	return value[index:]
}

func InspectTabularArtifact(ctx context.Context, fileName, mediaType string, data []byte, policy ExtractionPolicy) (TabularMetadata, error) {
	format, ok := DetectTabularFormat(fileName, mediaType)
	if !ok {
		return TabularMetadata{}, fmt.Errorf("unsupported tabular artifact")
	}
	return scanTabularArtifact(ctx, format, data, policy, "", nil)
}

func scanTabularArtifact(ctx context.Context, format TabularFormat, data []byte, policy ExtractionPolicy, resourceFilter string, visitor tabularVisitor) (TabularMetadata, error) {
	policy = policy.normalized()
	state := &tabularScanState{format: format, policy: policy, resources: map[string]*tabularResourceBuilder{}}
	var err error
	switch format {
	case TabularCSV:
		err = scanTabularCSV(ctx, data, state, resourceFilter, visitor)
	case TabularJSON:
		err = scanTabularJSON(ctx, data, state, resourceFilter, visitor)
	case TabularNDJSON:
		err = scanTabularNDJSON(ctx, data, state, resourceFilter, visitor)
	case TabularXLSX:
		err = scanTabularXLSX(ctx, data, state, resourceFilter, visitor)
	default:
		err = fmt.Errorf("unsupported tabular format %q", format)
	}
	metadata := state.metadata()
	if err != nil && !errors.Is(err, errTabularStop) {
		metadata.FatalError = truncateUTF8(err.Error(), 1024)
	}
	return metadata, err
}

func newTabularResource(state *tabularScanState, name string, headers []string) (*tabularResourceBuilder, error) {
	if existing := state.resources[name]; existing != nil {
		return existing, nil
	}
	builder := &tabularResourceBuilder{name: name, fields: map[string]*tabularFieldState{}}
	if len(headers) > 0 {
		normalized, err := normalizeTabularHeaders(headers)
		if err != nil {
			return nil, err
		}
		builder.fixedOrder = normalized
		for _, field := range normalized {
			builder.fields[field] = &tabularFieldState{native: "tabular:string"}
		}
	}
	state.resources[name] = builder
	state.order = append(state.order, name)
	return builder, nil
}

func normalizeTabularHeaders(headers []string) ([]string, error) {
	if len(headers) == 0 {
		return nil, fmt.Errorf("tabular header row is empty")
	}
	result := make([]string, len(headers))
	seen := map[string]struct{}{}
	for index, raw := range headers {
		value := strings.TrimSpace(strings.TrimPrefix(raw, "\ufeff"))
		if value == "" {
			value = fmt.Sprintf("Column %d", index+1)
		}
		if !validTabularField(value) {
			return nil, fmt.Errorf("invalid tabular field name at column %d", index+1)
		}
		if _, exists := seen[value]; exists {
			return nil, fmt.Errorf("duplicate tabular field %q", value)
		}
		seen[value] = struct{}{}
		result[index] = value
	}
	return result, nil
}

func validTabularField(value string) bool {
	if value == "" || len(value) > 256 {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func (s *tabularScanState) seen(resource string) (*tabularResourceBuilder, error) {
	builder, err := newTabularResource(s, resource, nil)
	if err != nil {
		return nil, err
	}
	builder.rowsTotal++
	if builder.rowsTotal > s.policy.MaxRows {
		return nil, limitError("row count exceeds %d", s.policy.MaxRows)
	}
	return builder, nil
}

func (s *tabularScanState) reject(resource string, row int, message string) {
	builder := s.resources[resource]
	if builder == nil {
		builder, _ = newTabularResource(s, resource, nil)
	}
	builder.rejected++
	if len(s.errors) < s.policy.MaxRowErrors {
		s.errors = append(s.errors, TabularRowError{Resource: resource, Row: row, Message: truncateUTF8(strings.TrimSpace(message), 512)})
	}
}

func (s *tabularScanState) accept(builder *tabularResourceBuilder, row tabularRow, visitor tabularVisitor) error {
	builder.accepted++
	for name, cell := range row.Values {
		if !validTabularField(name) {
			return fmt.Errorf("invalid tabular field %q", name)
		}
		state := builder.fields[name]
		if state == nil {
			state = &tabularFieldState{}
			builder.fields[name] = state
		}
		state.present++
		if cell.kind == "null" {
			state.nullable = true
			continue
		}
		native := tabularNativeType(cell.kind)
		if state.native == "" || state.native == "tabular:null" {
			state.native = native
		} else if state.native != native {
			state.native = "tabular:mixed"
		}
	}
	if visitor != nil {
		return visitor(row)
	}
	return nil
}

func tabularNativeType(kind string) string {
	switch kind {
	case "string":
		return "tabular:string"
	case "number":
		return "tabular:number"
	case "bool":
		return "tabular:boolean"
	case "array":
		return "tabular:array"
	case "object":
		return "tabular:object"
	default:
		return "tabular:null"
	}
}

func (s *tabularScanState) metadata() TabularMetadata {
	metadata := TabularMetadata{Format: s.format, ParserVersion: TabularParserVersion, Resources: []TabularResource{}, RowErrors: append([]TabularRowError(nil), s.errors...)}
	for _, name := range s.order {
		builder := s.resources[name]
		resource := TabularResource{Name: name, RowsTotal: builder.rowsTotal, RowsRejected: builder.rejected, Fields: []TabularField{}}
		names := append([]string(nil), builder.fixedOrder...)
		if len(names) == 0 {
			for field := range builder.fields {
				names = append(names, field)
			}
			sort.Strings(names)
		}
		for _, name := range names {
			field := builder.fields[name]
			if field == nil {
				field = &tabularFieldState{native: "tabular:string", nullable: true}
			}
			native := field.native
			if native == "" || native == "tabular:null" {
				native = "tabular:string"
			}
			nullable := field.nullable || field.present < builder.accepted
			resource.Fields = append(resource.Fields, TabularField{Name: name, NativeType: native, Nullable: nullable})
		}
		if len(resource.Fields) > 0 {
			resource.SchemaFingerprint = tabularSchemaFingerprint(resource.Fields)
		}
		metadata.RowsTotal += resource.RowsTotal
		metadata.RowsRejected += resource.RowsRejected
		metadata.Resources = append(metadata.Resources, resource)
	}
	return metadata
}

func tabularSchemaFingerprint(fields []TabularField) string {
	hash := sha256.New()
	for _, field := range fields {
		_, _ = fmt.Fprintf(hash, "%s\x1f%s\x1f%t\n", field.Name, field.NativeType, field.Nullable)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func scanTabularCSV(ctx context.Context, data []byte, state *tabularScanState, resourceFilter string, visitor tabularVisitor) error {
	const resource = "records"
	if resourceFilter != "" && resourceFilter != resource {
		return fmt.Errorf("tabular resource %q not found", resourceFilter)
	}
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	headers, err := reader.Read()
	if err == io.EOF {
		return fmt.Errorf("CSV header row is required")
	}
	if err != nil {
		return fmt.Errorf("CSV header could not be parsed: %w", err)
	}
	if len(headers) > state.policy.MaxColumns {
		return limitError("column count exceeds %d", state.policy.MaxColumns)
	}
	builder, err := newTabularResource(state, resource, headers)
	if err != nil {
		return err
	}
	rowNumber := 1
	cells := len(headers)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		rowNumber++
		builder.rowsTotal++
		if builder.rowsTotal > state.policy.MaxRows {
			return limitError("row count exceeds %d", state.policy.MaxRows)
		}
		if readErr != nil {
			state.reject(resource, rowNumber, fmt.Sprintf("CSV row could not be parsed: %v", readErr))
			continue
		}
		cells += len(record)
		if cells > state.policy.MaxCells {
			return limitError("cell count exceeds %d", state.policy.MaxCells)
		}
		if len(record) > state.policy.MaxColumns {
			state.reject(resource, rowNumber, fmt.Sprintf("column count exceeds %d", state.policy.MaxColumns))
			continue
		}
		if len(record) > len(builder.fixedOrder) {
			extra := false
			for _, value := range record[len(builder.fixedOrder):] {
				if value != "" {
					extra = true
				}
			}
			if extra {
				state.reject(resource, rowNumber, "row contains data beyond the header schema")
				continue
			}
		}
		values := make(map[string]tabularCell, len(builder.fixedOrder))
		invalid := ""
		for index, name := range builder.fixedOrder {
			if index >= len(record) {
				continue
			}
			if len(record[index]) > state.policy.MaxCellBytes {
				invalid = fmt.Sprintf("cell at column %d exceeds %d bytes", index+1, state.policy.MaxCellBytes)
				break
			}
			values[name] = tabularCell{kind: "string", text: record[index]}
		}
		if invalid != "" {
			state.reject(resource, rowNumber, invalid)
			continue
		}
		if err := state.accept(builder, tabularRow{Resource: resource, Number: rowNumber, Values: values}, visitor); err != nil {
			return err
		}
	}
	return nil
}

func scanTabularJSON(ctx context.Context, data []byte, state *tabularScanState, resourceFilter string, visitor tabularVisitor) error {
	const resource = "records"
	if resourceFilter != "" && resourceFilter != resource {
		return fmt.Errorf("tabular resource %q not found", resourceFilter)
	}
	builder, _ := newTabularResource(state, resource, nil)
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("JSON could not be parsed: %w", err)
	}
	delimiter, ok := token.(json.Delim)
	if !ok || delimiter != '[' {
		return fmt.Errorf("tabular JSON must contain a top-level array of objects")
	}
	rowNumber := 0
	cells := 0
	for decoder.More() {
		if err := ctx.Err(); err != nil {
			return err
		}
		rowNumber++
		builder.rowsTotal++
		if builder.rowsTotal > state.policy.MaxRows {
			return limitError("row count exceeds %d", state.policy.MaxRows)
		}
		var raw any
		if err := decoder.Decode(&raw); err != nil {
			return fmt.Errorf("JSON row %d could not be parsed: %w", rowNumber, err)
		}
		object, ok := raw.(map[string]any)
		if !ok {
			state.reject(resource, rowNumber, "JSON row is not an object")
			continue
		}
		row, count, err := jsonObjectTabularRow(resource, rowNumber, object, state.policy)
		cells += count
		if cells > state.policy.MaxCells {
			return limitError("cell count exceeds %d", state.policy.MaxCells)
		}
		if err != nil {
			state.reject(resource, rowNumber, err.Error())
			continue
		}
		if err := state.accept(builder, row, visitor); err != nil {
			return err
		}
	}
	if _, err := decoder.Token(); err != nil {
		return fmt.Errorf("JSON array could not be closed: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("JSON contains trailing data")
	}
	return nil
}

func scanTabularNDJSON(ctx context.Context, data []byte, state *tabularScanState, resourceFilter string, visitor tabularVisitor) error {
	const resource = "records"
	if resourceFilter != "" && resourceFilter != resource {
		return fmt.Errorf("tabular resource %q not found", resourceFilter)
	}
	builder, _ := newTabularResource(state, resource, nil)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	maxToken := state.policy.MaxCellBytes * min(state.policy.MaxColumns, 16)
	if maxToken < 1<<20 {
		maxToken = 1 << 20
	}
	if maxToken > 8<<20 {
		maxToken = 8 << 20
	}
	scanner.Buffer(make([]byte, 64<<10), maxToken)
	lineNumber := 0
	cells := 0
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		lineNumber++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		builder.rowsTotal++
		if builder.rowsTotal > state.policy.MaxRows {
			return limitError("row count exceeds %d", state.policy.MaxRows)
		}
		decoder := json.NewDecoder(bytes.NewReader(line))
		decoder.UseNumber()
		var object map[string]any
		if err := decoder.Decode(&object); err != nil || object == nil {
			state.reject(resource, lineNumber, "NDJSON row is not a valid JSON object")
			continue
		}
		var trailing any
		if err := decoder.Decode(&trailing); err != io.EOF {
			state.reject(resource, lineNumber, "NDJSON row contains trailing data")
			continue
		}
		row, count, err := jsonObjectTabularRow(resource, lineNumber, object, state.policy)
		cells += count
		if cells > state.policy.MaxCells {
			return limitError("cell count exceeds %d", state.policy.MaxCells)
		}
		if err != nil {
			state.reject(resource, lineNumber, err.Error())
			continue
		}
		if err := state.accept(builder, row, visitor); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return limitError("NDJSON row exceeds the bounded line size: %v", err)
	}
	return nil
}

func jsonObjectTabularRow(resource string, rowNumber int, object map[string]any, policy ExtractionPolicy) (tabularRow, int, error) {
	if len(object) > policy.MaxColumns {
		return tabularRow{}, len(object), fmt.Errorf("column count exceeds %d", policy.MaxColumns)
	}
	values := make(map[string]tabularCell, len(object))
	for name, value := range object {
		if !validTabularField(name) {
			return tabularRow{}, len(object), fmt.Errorf("invalid JSON field name")
		}
		cell, err := jsonTabularCell(value, policy.MaxCellBytes)
		if err != nil {
			return tabularRow{}, len(object), fmt.Errorf("field %q: %w", name, err)
		}
		values[name] = cell
	}
	return tabularRow{Resource: resource, Number: rowNumber, Values: values}, len(object), nil
}

func jsonTabularCell(value any, maxBytes int) (tabularCell, error) {
	switch typed := value.(type) {
	case nil:
		return tabularCell{kind: "null"}, nil
	case string:
		if len(typed) > maxBytes {
			return tabularCell{}, fmt.Errorf("string exceeds %d bytes", maxBytes)
		}
		return tabularCell{kind: "string", text: typed}, nil
	case json.Number:
		text := typed.String()
		if len(text) > maxBytes {
			return tabularCell{}, fmt.Errorf("number exceeds %d bytes", maxBytes)
		}
		return tabularCell{kind: "number", text: text}, nil
	case bool:
		return tabularCell{kind: "bool", text: strconv.FormatBool(typed)}, nil
	case []any:
		encoded, err := json.Marshal(typed)
		if err != nil || len(encoded) > maxBytes {
			return tabularCell{}, fmt.Errorf("array exceeds %d bytes", maxBytes)
		}
		return tabularCell{kind: "array", text: string(encoded)}, nil
	case map[string]any:
		encoded, err := json.Marshal(typed)
		if err != nil || len(encoded) > maxBytes {
			return tabularCell{}, fmt.Errorf("object exceeds %d bytes", maxBytes)
		}
		return tabularCell{kind: "object", text: string(encoded)}, nil
	default:
		return tabularCell{}, fmt.Errorf("unsupported JSON value")
	}
}

func scanTabularXLSX(ctx context.Context, data []byte, state *tabularScanState, resourceFilter string, visitor tabularVisitor) error {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("XLSX could not be parsed: %w", err)
	}
	if err := validateArchive(archive.File, state.policy); err != nil {
		return err
	}
	shared := []string{}
	worksheets := make([]*zip.File, 0)
	for _, file := range archive.File {
		switch {
		case file.Name == "xl/sharedStrings.xml":
			values, readErr := readSharedStrings(ctx, file, state.policy)
			if readErr != nil {
				return readErr
			}
			shared = values
		case strings.HasPrefix(file.Name, "xl/worksheets/sheet") && strings.HasSuffix(file.Name, ".xml"):
			worksheets = append(worksheets, file)
		}
	}
	if len(worksheets) == 0 {
		return fmt.Errorf("XLSX contains no worksheets")
	}
	if len(worksheets) > state.policy.MaxSheets {
		return limitError("worksheet count exceeds %d", state.policy.MaxSheets)
	}
	sort.Slice(worksheets, func(i, j int) bool { return worksheets[i].Name < worksheets[j].Name })
	found := resourceFilter == ""
	budget := &extractionBudget{}
	for index, file := range worksheets {
		resource := fmt.Sprintf("Sheet %d", index+1)
		if resourceFilter != "" && resourceFilter != resource {
			continue
		}
		found = true
		if err := scanTabularWorksheet(ctx, file, shared, resource, state, budget, visitor); err != nil {
			return err
		}
	}
	if !found {
		return fmt.Errorf("tabular resource %q not found", resourceFilter)
	}
	return nil
}

func scanTabularWorksheet(ctx context.Context, file *zip.File, shared []string, resource string, state *tabularScanState, budget *extractionBudget, visitor tabularVisitor) error {
	stream, err := file.Open()
	if err != nil {
		return err
	}
	defer stream.Close()
	decoder := xml.NewDecoder(stream)
	rowFallback := 0
	rowNumber := 0
	inRow := false
	rowValues := map[int]string{}
	maxColumn := -1
	var builder *tabularResourceBuilder
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		token, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		}
		if tokenErr != nil {
			return fmt.Errorf("XLSX worksheet could not be parsed: %w", tokenErr)
		}
		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "row":
				budget.rows++
				if budget.rows > state.policy.MaxRows {
					return limitError("row count exceeds %d", state.policy.MaxRows)
				}
				rowFallback++
				rowNumber = rowFallback
				for _, attr := range value.Attr {
					if attr.Name.Local == "r" {
						if parsed, parseErr := strconv.Atoi(attr.Value); parseErr == nil && parsed > 0 {
							rowNumber = parsed
						}
					}
				}
				inRow = true
				rowValues = map[int]string{}
				maxColumn = -1
			case "c":
				if !inRow {
					continue
				}
				budget.cells++
				if budget.cells > state.policy.MaxCells {
					return limitError("cell count exceeds %d", state.policy.MaxCells)
				}
				reference, cellValue, decodeErr := decodeWorksheetCell(ctx, decoder, value, shared, state.policy)
				if decodeErr != nil {
					return decodeErr
				}
				column := cellColumn(reference)
				if column >= state.policy.MaxColumns {
					return limitError("column index exceeds %d", state.policy.MaxColumns)
				}
				rowValues[column] = cellValue
				if column > maxColumn {
					maxColumn = column
				}
			}
		case xml.EndElement:
			if value.Name.Local != "row" || !inRow {
				continue
			}
			inRow = false
			if maxColumn < 0 {
				continue
			}
			values := make([]string, maxColumn+1)
			for column, cell := range rowValues {
				values[column] = cell
			}
			if builder == nil {
				builder, err = newTabularResource(state, resource, values)
				if err != nil {
					return err
				}
				continue
			}
			builder.rowsTotal++
			if builder.rowsTotal > state.policy.MaxRows {
				return limitError("row count exceeds %d", state.policy.MaxRows)
			}
			if len(values) > len(builder.fixedOrder) {
				extra := false
				for _, cell := range values[len(builder.fixedOrder):] {
					if cell != "" {
						extra = true
					}
				}
				if extra {
					state.reject(resource, rowNumber, "row contains data beyond the header schema")
					continue
				}
			}
			row := tabularRow{Resource: resource, Number: rowNumber, Values: map[string]tabularCell{}}
			for index, name := range builder.fixedOrder {
				if index >= len(values) || values[index] == "" {
					continue
				}
				if len(values[index]) > state.policy.MaxCellBytes {
					state.reject(resource, rowNumber, fmt.Sprintf("cell at column %d exceeds %d bytes", index+1, state.policy.MaxCellBytes))
					row.Values = nil
					break
				}
				row.Values[name] = tabularCell{kind: "string", text: values[index]}
			}
			if row.Values == nil {
				continue
			}
			if err := state.accept(builder, row, visitor); err != nil {
				return err
			}
		}
	}
	if builder == nil {
		_, err := newTabularResource(state, resource, []string{"Column 1"})
		return err
	}
	return nil
}

func tabularSections(ctx context.Context, format TabularFormat, data []byte, collector *sectionCollector) error {
	_, err := scanTabularArtifact(ctx, format, data, collector.policy, "", func(row tabularRow) error {
		if !collector.canRetain() {
			collector.add(Section{Title: fmt.Sprintf("%s row %d", row.Resource, row.Number), Text: "omitted", Sheet: tabularSheet(row.Resource), RowStart: row.Number, RowEnd: row.Number}, true, true)
			return nil
		}
		names := make([]string, 0, len(row.Values))
		for name := range row.Values {
			names = append(names, name)
		}
		sort.Strings(names)
		parts := make([]string, 0, len(names))
		for _, name := range names {
			cell := row.Values[name]
			if cell.kind == "null" {
				continue
			}
			parts = append(parts, name+": "+cell.text)
		}
		text := strings.Join(parts, "\n")
		collector.add(Section{Title: fmt.Sprintf("%s row %d", row.Resource, row.Number), Text: text, Sheet: tabularSheet(row.Resource), RowStart: row.Number, RowEnd: row.Number}, text != "", false)
		return nil
	})
	return err
}

func tabularSheet(resource string) string {
	if strings.HasPrefix(resource, "Sheet ") {
		return resource
	}
	return ""
}
