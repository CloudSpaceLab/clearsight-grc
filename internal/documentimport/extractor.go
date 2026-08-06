package documentimport

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type ExtractionResult struct {
	Status      ExtractionStatus
	Method      string
	Sections    []Section
	Limitations []string
}

func Extract(fileName, mediaType string, data []byte) ExtractionResult {
	extension := strings.ToLower(filepath.Ext(fileName))
	switch extension {
	case ".txt", ".md", ".markdown":
		return ExtractionResult{Status: ExtractionExtracted, Method: "PLAIN_TEXT_V1", Sections: textSections(string(data))}
	case ".csv":
		sections, err := csvSections(data)
		if err != nil {
			return failedExtraction("CSV could not be parsed: " + err.Error())
		}
		return ExtractionResult{Status: ExtractionExtracted, Method: "CSV_ROWS_V1", Sections: sections}
	case ".docx":
		sections, err := docxSections(data)
		if err != nil {
			return failedExtraction("DOCX could not be parsed: " + err.Error())
		}
		return ExtractionResult{Status: ExtractionExtracted, Method: "DOCX_XML_V1", Sections: sections}
	case ".xlsx":
		sections, err := xlsxSections(data)
		if err != nil {
			return failedExtraction("XLSX could not be parsed: " + err.Error())
		}
		return ExtractionResult{Status: ExtractionExtracted, Method: "XLSX_XML_V1", Sections: sections}
	case ".pdf":
		return ExtractionResult{Status: ExtractionUnsupported, Method: "NONE", Limitations: []string{"The original PDF was stored, but this build has no approved PDF text extractor or OCR adapter.", "No compliance proposal was generated from the PDF."}}
	default:
		if strings.HasPrefix(strings.ToLower(mediaType), "text/") {
			return ExtractionResult{Status: ExtractionExtracted, Method: "PLAIN_TEXT_V1", Sections: textSections(string(data))}
		}
		return ExtractionResult{Status: ExtractionUnsupported, Method: "NONE", Limitations: []string{"The original artifact was stored, but its document type is not supported for extraction."}}
	}
}

func failedExtraction(message string) ExtractionResult {
	return ExtractionResult{Status: ExtractionFailed, Method: "FAILED", Limitations: []string{message, "No compliance proposal was generated."}}
}

func textSections(value string) []Section {
	value = normalizeText(value)
	blocks := strings.Split(value, "\n\n")
	sections := make([]Section, 0, len(blocks))
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		lines := strings.Split(block, "\n")
		title := fmt.Sprintf("Section %d", len(sections)+1)
		if len(lines) > 1 && looksLikeHeading(lines[0]) {
			title = strings.TrimSpace(strings.TrimLeft(lines[0], "#"))
			block = strings.TrimSpace(strings.Join(lines[1:], "\n"))
		}
		if block == "" {
			continue
		}
		sections = append(sections, Section{ID: newID(), Sequence: len(sections) + 1, Title: title, Text: truncate(block, 120000)})
	}
	if len(sections) == 0 && strings.TrimSpace(value) != "" {
		sections = append(sections, Section{ID: newID(), Sequence: 1, Title: "Document text", Text: truncate(strings.TrimSpace(value), 120000)})
	}
	return sections
}

func csvSections(data []byte) ([]Section, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	headers := records[0]
	sections := make([]Section, 0, len(records)-1)
	for rowIndex, record := range records[1:] {
		parts := make([]string, 0, len(record))
		for index, value := range record {
			label := fmt.Sprintf("Column %d", index+1)
			if index < len(headers) && strings.TrimSpace(headers[index]) != "" {
				label = strings.TrimSpace(headers[index])
			}
			parts = append(parts, label+": "+strings.TrimSpace(value))
		}
		sections = append(sections, Section{ID: newID(), Sequence: len(sections) + 1, Title: fmt.Sprintf("Row %d", rowIndex+2), Text: truncate(strings.Join(parts, "\n"), 120000), RowStart: rowIndex + 2, RowEnd: rowIndex + 2})
	}
	return sections, nil
}

func docxSections(data []byte) ([]Section, error) {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	var document *zip.File
	for _, file := range archive.File {
		if file.Name == "word/document.xml" {
			document = file
			break
		}
	}
	if document == nil {
		return nil, fmt.Errorf("word/document.xml is missing")
	}
	if document.UncompressedSize64 > 20<<20 {
		return nil, fmt.Errorf("expanded document XML exceeds 20 MiB")
	}
	stream, err := document.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	decoder := xml.NewDecoder(io.LimitReader(stream, 20<<20))
	paragraphs := make([]string, 0)
	var current strings.Builder
	inParagraph := false
	for {
		token, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		}
		if tokenErr != nil {
			return nil, tokenErr
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "p" {
				inParagraph = true
				current.Reset()
			}
			if inParagraph && (value.Name.Local == "t" || value.Name.Local == "tab") {
				if value.Name.Local == "tab" {
					current.WriteByte('\t')
					continue
				}
				var text string
				if err := decoder.DecodeElement(&text, &value); err != nil {
					return nil, err
				}
				current.WriteString(text)
			}
		case xml.EndElement:
			if value.Name.Local == "p" && inParagraph {
				paragraph := strings.TrimSpace(current.String())
				if paragraph != "" {
					paragraphs = append(paragraphs, paragraph)
				}
				inParagraph = false
			}
		}
	}
	return paragraphSections(paragraphs), nil
}

func xlsxSections(data []byte) ([]Section, error) {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	shared := []string{}
	for _, file := range archive.File {
		if file.Name == "xl/sharedStrings.xml" {
			values, readErr := readSharedStrings(file)
			if readErr != nil {
				return nil, readErr
			}
			shared = values
			break
		}
	}
	worksheets := make([]*zip.File, 0)
	for _, file := range archive.File {
		if strings.HasPrefix(file.Name, "xl/worksheets/sheet") && strings.HasSuffix(file.Name, ".xml") {
			worksheets = append(worksheets, file)
		}
	}
	sort.Slice(worksheets, func(i, j int) bool { return worksheets[i].Name < worksheets[j].Name })
	sections := make([]Section, 0)
	for sheetIndex, file := range worksheets {
		rows, readErr := readWorksheet(file, shared)
		if readErr != nil {
			return nil, readErr
		}
		sheetName := fmt.Sprintf("Sheet %d", sheetIndex+1)
		for rowNumber, values := range rows {
			parts := make([]string, 0, len(values))
			for column, value := range values {
				if strings.TrimSpace(value) != "" {
					parts = append(parts, fmt.Sprintf("Column %d: %s", column+1, strings.TrimSpace(value)))
				}
			}
			if len(parts) == 0 {
				continue
			}
			sections = append(sections, Section{ID: newID(), Sequence: len(sections) + 1, Title: fmt.Sprintf("%s row %d", sheetName, rowNumber), Text: truncate(strings.Join(parts, "\n"), 120000), Sheet: sheetName, RowStart: rowNumber, RowEnd: rowNumber})
		}
	}
	return sections, nil
}

func readSharedStrings(file *zip.File) ([]string, error) {
	stream, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	decoder := xml.NewDecoder(io.LimitReader(stream, 20<<20))
	values := []string{}
	var current strings.Builder
	inItem := false
	for {
		token, tokenErr := decoder.Token()
		if tokenErr == io.EOF {
			break
		}
		if tokenErr != nil {
			return nil, tokenErr
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "si" {
				inItem = true
				current.Reset()
			}
			if inItem && value.Name.Local == "t" {
				var text string
				if err := decoder.DecodeElement(&text, &value); err != nil {
					return nil, err
				}
				current.WriteString(text)
			}
		case xml.EndElement:
			if value.Name.Local == "si" && inItem {
				values = append(values, current.String())
				inItem = false
			}
		}
	}
	return values, nil
}

func readWorksheet(file *zip.File, shared []string) (map[int][]string, error) {
	stream, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	type cell struct {
		Reference string `xml:"r,attr"`
		Type      string `xml:"t,attr"`
		Value     string `xml:"v"`
		Inline    string `xml:"is>t"`
	}
	type row struct {
		Number int    `xml:"r,attr"`
		Cells  []cell `xml:"c"`
	}
	var worksheet struct {
		Rows []row `xml:"sheetData>row"`
	}
	decoder := xml.NewDecoder(io.LimitReader(stream, 30<<20))
	if err := decoder.Decode(&worksheet); err != nil {
		return nil, err
	}
	rows := map[int][]string{}
	for _, item := range worksheet.Rows {
		values := []string{}
		for _, current := range item.Cells {
			column := cellColumn(current.Reference)
			for len(values) <= column {
				values = append(values, "")
			}
			value := current.Value
			if current.Type == "s" {
				index, parseErr := strconv.Atoi(current.Value)
				if parseErr == nil && index >= 0 && index < len(shared) {
					value = shared[index]
				}
			} else if current.Type == "inlineStr" {
				value = current.Inline
			}
			values[column] = value
		}
		rows[item.Number] = values
	}
	return rows, nil
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

func paragraphSections(paragraphs []string) []Section {
	sections := make([]Section, 0, len(paragraphs))
	currentTitle := "Document text"
	var body []string
	flush := func() {
		text := strings.TrimSpace(strings.Join(body, "\n\n"))
		if text == "" {
			return
		}
		sections = append(sections, Section{ID: newID(), Sequence: len(sections) + 1, Title: currentTitle, Text: truncate(text, 120000)})
		body = nil
	}
	for _, paragraph := range paragraphs {
		if looksLikeHeading(paragraph) {
			flush()
			currentTitle = truncate(paragraph, 240)
			continue
		}
		body = append(body, paragraph)
	}
	flush()
	if len(sections) == 0 && len(paragraphs) > 0 {
		sections = append(sections, Section{ID: newID(), Sequence: 1, Title: "Document text", Text: truncate(strings.Join(paragraphs, "\n\n"), 120000)})
	}
	return sections
}

func normalizeText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	for strings.Contains(value, "\n\n\n") {
		value = strings.ReplaceAll(value, "\n\n\n", "\n\n")
	}
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

func truncate(value string, maximum int) string {
	if len(value) <= maximum {
		return value
	}
	return value[:maximum]
}
