package documentimport

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

type ExtractionResult struct {
	Status           ExtractionStatus
	Method           string
	ParserVersion    string
	AdapterVersion   string
	Sections         []Section
	Elements         []ExtractedElement
	Degradations     []Degradation
	Limitations      []string
	SectionsTotal    int
	SectionsOmitted  int
	ContentTruncated bool
}

type extractionBudget struct {
	rows  int
	cells int
}

type sectionCollector struct {
	policy       ExtractionPolicy
	sections     []Section
	degradations []Degradation
	total        int
	omitted      int
	textBytes    int64
	truncated    bool
}

func newSectionCollector(policy ExtractionPolicy) *sectionCollector {
	normalized := policy.normalized()
	return &sectionCollector{
		policy: normalized, sections: make([]Section, 0, min(normalized.MaxSections, 64)),
		degradations: make([]Degradation, 0, 4),
	}
}

func (c *sectionCollector) canRetain() bool {
	return len(c.sections) < c.policy.MaxSections && c.textBytes < c.policy.MaxExtractedTextBytes
}

func (c *sectionCollector) remainingText() int64 {
	remaining := c.policy.MaxExtractedTextBytes - c.textBytes
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (c *sectionCollector) add(section Section, contentExists, alreadyTruncated bool) {
	if !contentExists {
		return
	}
	c.total++
	if alreadyTruncated {
		c.truncated = true
	}
	if len(c.sections) >= c.policy.MaxSections || c.remainingText() <= 0 {
		c.omitted++
		c.truncated = true
		return
	}
	text := strings.TrimSpace(section.Text)
	if text == "" {
		c.omitted++
		c.truncated = true
		return
	}
	remaining := c.remainingText()
	if int64(len(text)) > remaining {
		text = truncateUTF8(text, int(remaining))
		c.truncated = true
	}
	if text == "" {
		c.omitted++
		return
	}
	section.ID = newID()
	section.Sequence = len(c.sections) + 1
	section.Text = text
	c.textBytes += int64(len(text))
	c.sections = append(c.sections, section)
}

func (c *sectionCollector) degrade(code, message string, anchor *SourceAnchor) {
	code = strings.TrimSpace(code)
	message = strings.TrimSpace(message)
	if code == "" || message == "" {
		return
	}
	for _, existing := range c.degradations {
		if existing.Code == code {
			return
		}
	}
	c.degradations = append(c.degradations, Degradation{
		Code: code, Message: message, Recoverable: true, Anchor: anchor,
	})
}

func (c *sectionCollector) result(status ExtractionStatus, method string, limitations ...string) ExtractionResult {
	status = explicitExtractionStatus(status, c.truncated, c.degradations)
	limitations = append([]string(nil), limitations...)
	for _, degradation := range c.degradations {
		limitations = append(limitations, degradation.Message)
	}
	return ExtractionResult{
		Status: status, Method: method, ParserVersion: method, AdapterVersion: extractionElementAdapterVersion,
		Sections: c.sections, Elements: elementsFromSections(c.sections), Degradations: cloneDegradations(c.degradations),
		Limitations: limitations, SectionsTotal: c.total, SectionsOmitted: c.omitted, ContentTruncated: c.truncated,
	}
}

func Extract(fileName, mediaType string, data []byte) ExtractionResult {
	return ExtractWithPolicy(context.Background(), fileName, mediaType, data, DefaultExtractionPolicy())
}

func ExtractWithPolicy(ctx context.Context, fileName, mediaType string, data []byte, policy ExtractionPolicy) ExtractionResult {
	policy = policy.normalized()
	collector := newSectionCollector(policy)
	extension := strings.ToLower(filepath.Ext(fileName))
	var err error
	var method string
	switch extension {
	case ".txt", ".md", ".markdown":
		method = "PLAIN_TEXT_V2"
		err = textSections(ctx, data, collector)
	case ".csv":
		method = "CSV_STREAM_V2"
		err = csvSections(ctx, data, collector, &extractionBudget{})
	case ".json":
		method = "JSON_TABULAR_V1"
		err = tabularSections(ctx, TabularJSON, data, collector)
	case ".ndjson", ".jsonl":
		method = "NDJSON_TABULAR_V1"
		err = tabularSections(ctx, TabularNDJSON, data, collector)
	case ".docx":
		method = docxParserVersion
		result, docxErr := extractDOCX(ctx, data, policy)
		if docxErr == nil {
			return result
		}
		err = docxErr
	case ".xlsx":
		method = "XLSX_XML_STREAM_V2"
		err = xlsxSections(ctx, data, collector, policy)
	case ".pdf":
		return extractPDF(ctx, data, collector, policy)
	default:
		media := strings.ToLower(strings.TrimSpace(strings.Split(mediaType, ";")[0]))
		if media == "application/json" {
			method = "JSON_TABULAR_V1"
			err = tabularSections(ctx, TabularJSON, data, collector)
		} else if media == "application/x-ndjson" || media == "application/ndjson" || media == "application/jsonlines" {
			method = "NDJSON_TABULAR_V1"
			err = tabularSections(ctx, TabularNDJSON, data, collector)
		} else if strings.HasPrefix(strings.ToLower(mediaType), "text/") {
			method = "PLAIN_TEXT_V2"
			err = textSections(ctx, data, collector)
		} else {
			return collector.result(ExtractionUnsupported, "NONE", "The original artifact was stored, but its document type is not supported for extraction.")
		}
	}
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return ExtractionResult{
				Status: ExtractionFailed, Method: method, ParserVersion: method, AdapterVersion: extractionElementAdapterVersion,
				Limitations: []string{err.Error()},
			}
		}
		return failedExtraction(method, err)
	}
	return collector.result(ExtractionExtracted, method)
}

func failedExtraction(method string, err error) ExtractionResult {
	message := err.Error()
	if errors.Is(err, ErrResourceLimit) {
		message = "Extraction resource limit exceeded: " + message
	}
	return ExtractionResult{
		Status: ExtractionFailed, Method: method, ParserVersion: method, AdapterVersion: extractionElementAdapterVersion,
		Limitations: []string{message, "No compliance proposal was generated."},
	}
}

func textSections(ctx context.Context, data []byte, collector *sectionCollector) error {
	value := normalizeText(string(data))
	blocks := strings.Split(value, "\n\n")
	for _, block := range blocks {
		if err := ctx.Err(); err != nil {
			return err
		}
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		lines := strings.Split(block, "\n")
		title := fmt.Sprintf("Section %d", collector.total+1)
		if len(lines) > 1 && looksLikeHeading(lines[0]) {
			title = strings.TrimSpace(strings.TrimLeft(lines[0], "#"))
			block = strings.TrimSpace(strings.Join(lines[1:], "\n"))
		}
		collector.add(Section{Title: truncateUTF8(title, 240), Text: block}, block != "", false)
	}
	if collector.total == 0 && strings.TrimSpace(value) != "" {
		collector.add(Section{Title: "Document text", Text: strings.TrimSpace(value)}, true, false)
	}
	return nil
}

func csvSections(ctx context.Context, data []byte, collector *sectionCollector, budget *extractionBudget) error {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	reader.ReuseRecord = true
	headers, err := reader.Read()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return fmt.Errorf("CSV could not be parsed: %w", err)
	}
	if err := consumeRowBudget(headers, collector.policy, budget); err != nil {
		return err
	}
	headerCopy := append([]string(nil), headers...)
	for index := range headerCopy {
		if len(headerCopy[index]) > collector.policy.MaxCellBytes {
			return limitError("CSV header cell exceeds %d bytes", collector.policy.MaxCellBytes)
		}
	}
	rowNumber := 1
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		record, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("CSV could not be parsed: %w", readErr)
		}
		rowNumber++
		if err := consumeRowBudget(record, collector.policy, budget); err != nil {
			return err
		}
		nonEmpty := false
		var parts []string
		if collector.canRetain() {
			parts = make([]string, 0, len(record))
		}
		for index, value := range record {
			if len(value) > collector.policy.MaxCellBytes {
				return limitError("CSV cell at row %d column %d exceeds %d bytes", rowNumber, index+1, collector.policy.MaxCellBytes)
			}
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			nonEmpty = true
			if parts == nil {
				continue
			}
			label := fmt.Sprintf("Column %d", index+1)
			if index < len(headerCopy) && strings.TrimSpace(headerCopy[index]) != "" {
				label = strings.TrimSpace(headerCopy[index])
			}
			parts = append(parts, label+": "+value)
		}
		collector.add(Section{Title: fmt.Sprintf("Row %d", rowNumber), Text: strings.Join(parts, "\n"), RowStart: rowNumber, RowEnd: rowNumber}, nonEmpty, parts == nil && nonEmpty)
	}
	return nil
}

func consumeRowBudget(values []string, policy ExtractionPolicy, budget *extractionBudget) error {
	budget.rows++
	if budget.rows > policy.MaxRows {
		return limitError("row count exceeds %d", policy.MaxRows)
	}
	if len(values) > policy.MaxColumns {
		return limitError("column count exceeds %d", policy.MaxColumns)
	}
	budget.cells += len(values)
	if budget.cells > policy.MaxCells {
		return limitError("cell count exceeds %d", policy.MaxCells)
	}
	return nil
}
