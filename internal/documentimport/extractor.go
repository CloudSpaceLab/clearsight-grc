package documentimport

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type ExtractionResult struct {
	Status           ExtractionStatus
	Method           string
	Sections         []Section
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
	policy    ExtractionPolicy
	sections  []Section
	total     int
	omitted   int
	textBytes int64
	truncated bool
}

func newSectionCollector(policy ExtractionPolicy) *sectionCollector {
	return &sectionCollector{policy: policy.normalized(), sections: make([]Section, 0, min(policy.normalized().MaxSections, 64))}
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

func (c *sectionCollector) result(status ExtractionStatus, method string, limitations ...string) ExtractionResult {
	return ExtractionResult{
		Status: status, Method: method, Sections: c.sections, Limitations: limitations,
		SectionsTotal: c.total, SectionsOmitted: c.omitted, ContentTruncated: c.truncated,
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
	case ".docx":
		method = "DOCX_XML_STREAM_V2"
		err = docxSections(ctx, data, collector, policy)
	case ".xlsx":
		method = "XLSX_XML_STREAM_V2"
		err = xlsxSections(ctx, data, collector, policy)
	case ".pdf":
		return extractPDF(ctx, data, collector, policy)
	default:
		if strings.HasPrefix(strings.ToLower(mediaType), "text/") {
			method = "PLAIN_TEXT_V2"
			err = textSections(ctx, data, collector)
		} else {
			return collector.result(ExtractionUnsupported, "NONE", "The original artifact was stored, but its document type is not supported for extraction.")
		}
	}
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return ExtractionResult{Status: ExtractionFailed, Method: method, Limitations: []string{err.Error()}}
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
	return ExtractionResult{Status: ExtractionFailed, Method: method, Limitations: []string{message, "No compliance proposal was generated."}}
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

func docxSections(ctx context.Context, data []byte, collector *sectionCollector, policy ExtractionPolicy) error {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("DOCX could not be parsed: %w", err)
	}
	if err := validateArchive(archive.File, policy); err != nil {
		return err
	}
	var document *zip.File
	for _, file := range archive.File {
		if file.Name == "word/document.xml" {
			document = file
			break
		}
	}
	if document == nil {
		return fmt.Errorf("DOCX could not be parsed: word/document.xml is missing")
	}
	stream, err := document.Open()
	if err != nil {
		return err
	}
	defer stream.Close()
	decoder := xml.NewDecoder(stream)
	currentTitle := "Document text"
	var body strings.Builder
	hasBody := false
	bodyTruncated := false
	flush := func() {
		if !hasBody {
			return
		}
		collector.add(Section{Title: currentTitle, Text: body.String()}, true, bodyTruncated)
		body.Reset()
		hasBody = false
		bodyTruncated = false
	}
	inParagraph := false
	var paragraph strings.Builder
	paragraphTruncated := false
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		token, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		}
		if tokenErr != nil {
			return fmt.Errorf("DOCX XML could not be parsed: %w", tokenErr)
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "p" {
				inParagraph = true
				paragraph.Reset()
				paragraphTruncated = false
			} else if inParagraph && value.Name.Local == "tab" {
				appendBounded(&paragraph, "\t", policy.MaxCellBytes, &paragraphTruncated)
			}
		case xml.CharData:
			if inParagraph {
				appendBounded(&paragraph, string(value), policy.MaxCellBytes, &paragraphTruncated)
			}
		case xml.EndElement:
			if value.Name.Local != "p" || !inParagraph {
				continue
			}
			text := strings.TrimSpace(paragraph.String())
			inParagraph = false
			if text == "" {
				continue
			}
			if looksLikeHeading(text) {
				flush()
				currentTitle = truncateUTF8(text, 240)
				continue
			}
			hasBody = true
			bodyTruncated = bodyTruncated || paragraphTruncated
			if body.Len() > 0 {
				appendBounded(&body, "\n\n", int(collector.policy.MaxExtractedTextBytes), &bodyTruncated)
			}
			appendBounded(&body, text, int(collector.policy.MaxExtractedTextBytes), &bodyTruncated)
		}
	}
	flush()
	return nil
}

func xlsxSections(ctx context.Context, data []byte, collector *sectionCollector, policy ExtractionPolicy) error {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("XLSX could not be parsed: %w", err)
	}
	if err := validateArchive(archive.File, policy); err != nil {
		return err
	}
	shared := []string{}
	worksheets := make([]*zip.File, 0)
	for _, file := range archive.File {
		switch {
		case file.Name == "xl/sharedStrings.xml":
			values, readErr := readSharedStrings(ctx, file, policy)
			if readErr != nil {
				return readErr
			}
			shared = values
		case strings.HasPrefix(file.Name, "xl/worksheets/sheet") && strings.HasSuffix(file.Name, ".xml"):
			worksheets = append(worksheets, file)
		}
	}
	if len(worksheets) > policy.MaxSheets {
		return limitError("worksheet count exceeds %d", policy.MaxSheets)
	}
	sort.Slice(worksheets, func(i, j int) bool { return worksheets[i].Name < worksheets[j].Name })
	budget := &extractionBudget{}
	for sheetIndex, file := range worksheets {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := streamWorksheet(ctx, file, shared, fmt.Sprintf("Sheet %d", sheetIndex+1), collector, budget); err != nil {
			return err
		}
	}
	return nil
}

func readSharedStrings(ctx context.Context, file *zip.File, policy ExtractionPolicy) ([]string, error) {
	stream, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	decoder := xml.NewDecoder(stream)
	values := make([]string, 0, min(policy.MaxSharedStrings, 1024))
	var current strings.Builder
	inItem := false
	inText := false
	var totalBytes int64
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		token, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		}
		if tokenErr != nil {
			return nil, fmt.Errorf("XLSX shared strings could not be parsed: %w", tokenErr)
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "si" {
				if len(values) >= policy.MaxSharedStrings {
					return nil, limitError("shared-string count exceeds %d", policy.MaxSharedStrings)
				}
				inItem = true
				current.Reset()
			} else if inItem && value.Name.Local == "t" {
				inText = true
			}
		case xml.CharData:
			if inItem && inText {
				if current.Len()+len(value) > policy.MaxCellBytes {
					return nil, limitError("shared-string cell exceeds %d bytes", policy.MaxCellBytes)
				}
				current.Write(value)
			}
		case xml.EndElement:
			if value.Name.Local == "t" {
				inText = false
			}
			if value.Name.Local == "si" && inItem {
				text := current.String()
				totalBytes += int64(len(text))
				if totalBytes > policy.MaxSharedStringBytes {
					return nil, limitError("shared-string text exceeds %d bytes", policy.MaxSharedStringBytes)
				}
				values = append(values, text)
				inItem = false
			}
		}
	}
	return values, nil
}

func streamWorksheet(ctx context.Context, file *zip.File, shared []string, sheetName string, collector *sectionCollector, budget *extractionBudget) error {
	stream, err := file.Open()
	if err != nil {
		return err
	}
	defer stream.Close()
	decoder := xml.NewDecoder(stream)
	rowNumber := 0
	rowFallback := 0
	inRow := false
	rowNonEmpty := false
	var parts []string
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
				if budget.rows > collector.policy.MaxRows {
					return limitError("row count exceeds %d", collector.policy.MaxRows)
				}
				rowFallback++
				rowNumber = rowFallback
				for _, attribute := range value.Attr {
					if attribute.Name.Local == "r" {
						if parsed, parseErr := strconv.Atoi(attribute.Value); parseErr == nil && parsed > 0 {
							rowNumber = parsed
						}
					}
				}
				inRow = true
				rowNonEmpty = false
				parts = nil
				if collector.canRetain() {
					parts = make([]string, 0, 16)
				}
			case "c":
				if !inRow {
					continue
				}
				budget.cells++
				if budget.cells > collector.policy.MaxCells {
					return limitError("cell count exceeds %d", collector.policy.MaxCells)
				}
				reference, cellValue, decodeErr := decodeWorksheetCell(ctx, decoder, value, shared, collector.policy)
				if decodeErr != nil {
					return decodeErr
				}
				column := cellColumn(reference)
				if column >= collector.policy.MaxColumns {
					return limitError("column index exceeds %d", collector.policy.MaxColumns)
				}
				cellValue = strings.TrimSpace(cellValue)
				if cellValue == "" {
					continue
				}
				rowNonEmpty = true
				if parts != nil {
					parts = append(parts, fmt.Sprintf("Column %d: %s", column+1, cellValue))
				}
			}
		case xml.EndElement:
			if value.Name.Local == "row" && inRow {
				collector.add(Section{Title: fmt.Sprintf("%s row %d", sheetName, rowNumber), Text: strings.Join(parts, "\n"), Sheet: sheetName, RowStart: rowNumber, RowEnd: rowNumber}, rowNonEmpty, parts == nil && rowNonEmpty)
				inRow = false
			}
		}
	}
	return nil
}

func decodeWorksheetCell(ctx context.Context, decoder *xml.Decoder, start xml.StartElement, shared []string, policy ExtractionPolicy) (string, string, error) {
	reference := ""
	cellType := ""
	for _, attribute := range start.Attr {
		switch attribute.Name.Local {
		case "r":
			reference = attribute.Value
		case "t":
			cellType = attribute.Value
		}
	}
	var valueBuilder strings.Builder
	var inlineBuilder strings.Builder
	inValue := false
	inText := false
	for {
		if err := ctx.Err(); err != nil {
			return "", "", err
		}
		token, err := decoder.Token()
		if err != nil {
			return "", "", err
		}
		switch current := token.(type) {
		case xml.StartElement:
			if current.Name.Local == "v" {
				inValue = true
			} else if current.Name.Local == "t" {
				inText = true
			}
		case xml.CharData:
			if inValue {
				if valueBuilder.Len()+len(current) > policy.MaxCellBytes {
					return "", "", limitError("cell %s exceeds %d bytes", reference, policy.MaxCellBytes)
				}
				valueBuilder.Write(current)
			}
			if inText {
				if inlineBuilder.Len()+len(current) > policy.MaxCellBytes {
					return "", "", limitError("inline cell %s exceeds %d bytes", reference, policy.MaxCellBytes)
				}
				inlineBuilder.Write(current)
			}
		case xml.EndElement:
			switch current.Name.Local {
			case "v":
				inValue = false
			case "t":
				inText = false
			case "c":
				raw := valueBuilder.String()
				switch cellType {
				case "s":
					index, parseErr := strconv.Atoi(strings.TrimSpace(raw))
					if parseErr != nil || index < 0 || index >= len(shared) {
						return reference, "", nil
					}
					return reference, shared[index], nil
				case "inlineStr":
					return reference, inlineBuilder.String(), nil
				default:
					return reference, raw, nil
				}
			}
		}
	}
}

func validateArchive(files []*zip.File, policy ExtractionPolicy) error {
	if len(files) > policy.MaxArchiveEntries {
		return limitError("archive entry count exceeds %d", policy.MaxArchiveEntries)
	}
	var expanded uint64
	for _, file := range files {
		clean := path.Clean(strings.ReplaceAll(file.Name, "\\", "/"))
		if clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
			return limitError("archive contains an unsafe entry path")
		}
		if ^uint64(0)-expanded < file.UncompressedSize64 {
			return limitError("archive expanded size overflow")
		}
		expanded += file.UncompressedSize64
		if expanded > uint64(policy.MaxExpandedBytes) {
			return limitError("archive expanded size exceeds %d bytes", policy.MaxExpandedBytes)
		}
		if file.UncompressedSize64 < uint64(policy.CompressionRatioFloor) || file.FileInfo().IsDir() {
			continue
		}
		if file.CompressedSize64 == 0 || float64(file.UncompressedSize64)/float64(file.CompressedSize64) > policy.MaxCompressionRatio {
			return limitError("archive entry %q exceeds %.0f:1 compression ratio", file.Name, policy.MaxCompressionRatio)
		}
	}
	return nil
}

func appendBounded(builder *strings.Builder, value string, maximum int, truncated *bool) {
	if maximum <= 0 || builder.Len() >= maximum {
		*truncated = true
		return
	}
	remaining := maximum - builder.Len()
	if len(value) > remaining {
		value = truncateUTF8(value, remaining)
		*truncated = true
	}
	builder.WriteString(value)
}

func cellColumn(reference string) int {
	value := 0
	for _, character := range reference {
		if !unicode.IsLetter(character) {
			break
		}
		value = value*26 + int(unicode.ToUpper(character)-'A'+1)
	}
	if value == 0 {
		return 0
	}
	return value - 1
}

func normalizeText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	return strings.TrimSpace(value)
}

func looksLikeHeading(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 140 {
		return false
	}
	if strings.HasPrefix(value, "#") {
		return true
	}
	words := strings.Fields(value)
	if len(words) > 12 {
		return false
	}
	return !strings.ContainsAny(value, ".!?;")
}

func truncateUTF8(value string, maximum int) string {
	if maximum <= 0 {
		return ""
	}
	if len(value) <= maximum {
		return value
	}
	value = value[:maximum]
	for !utf8.ValidString(value) && len(value) > 0 {
		value = value[:len(value)-1]
	}
	return value
}
