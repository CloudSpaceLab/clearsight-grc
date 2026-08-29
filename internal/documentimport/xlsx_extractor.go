package documentimport

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

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
