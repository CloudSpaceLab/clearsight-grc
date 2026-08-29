package documentimport

import (
	"context"
	"fmt"
	"strings"
)

const docxParserVersion = "DOCX_XML_STREAM_V3"

func extractDOCX(ctx context.Context, data []byte, policy ExtractionPolicy) (ExtractionResult, error) {
	structured, err := extractDOCXStructuredElements(ctx, data, policy)
	if err != nil {
		return ExtractionResult{}, err
	}

	collector := newSectionCollector(policy)
	projectDOCXSections(structured.elements, collector)
	collector.truncated = collector.truncated || structured.truncated
	for _, degradation := range structured.degradations {
		collector.degrade(degradation.Code, degradation.Message, degradation.Anchor)
	}

	result := collector.result(ExtractionExtracted, docxParserVersion)
	result.AdapterVersion = docxElementAdapterVersion
	result.Elements = cloneElements(structured.elements)
	result.Degradations = cloneDegradations(structured.degradations)
	return result, nil
}

func projectDOCXSections(elements []ExtractedElement, collector *sectionCollector) {
	if collector == nil {
		return
	}
	currentTitle := "Document text"
	var body strings.Builder
	bodyTruncated := false
	flush := func() {
		text := strings.TrimSpace(body.String())
		if text != "" {
			collector.add(Section{Title: currentTitle, Text: text}, true, bodyTruncated)
		}
		body.Reset()
		bodyTruncated = false
	}
	appendBody := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if body.Len() > 0 {
			appendBounded(&body, "\n\n", int(collector.policy.MaxExtractedTextBytes), &bodyTruncated)
		}
		appendBounded(&body, value, int(collector.policy.MaxExtractedTextBytes), &bodyTruncated)
	}

	for _, element := range elements {
		switch element.Kind {
		case ElementHeading:
			if title := strings.TrimSpace(element.Text); title != "" {
				flush()
				currentTitle = truncateUTF8(title, 240)
			}
		case ElementParagraph:
			text := strings.TrimSpace(element.Text)
			if looksLikeHeading(text) {
				flush()
				currentTitle = truncateUTF8(text, 240)
				continue
			}
			appendBody(text)
		case ElementTable:
			appendBody(flattenDOCXTable(element.Values))
		case ElementFormControl:
			// Paragraph- and table-anchored controls already contribute their
			// visible text through the containing paragraph/table projection.
			// A top-level content control has no containing legacy projection,
			// so retain only its visible source text; labels/options remain rich
			// element metadata for proposal authoring rather than obligations.
			if element.Anchor.Paragraph == "" && element.Anchor.Table == "" {
				appendBody(element.Text)
			}
		}
	}
	flush()
}

func flattenDOCXTable(rows [][]string) string {
	if len(rows) == 0 {
		return ""
	}
	var builder strings.Builder
	for rowIndex, row := range rows {
		nonEmpty := false
		for _, value := range row {
			if strings.TrimSpace(value) != "" {
				nonEmpty = true
				break
			}
		}
		if !nonEmpty {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteByte('\n')
		}
		builder.WriteString(fmt.Sprintf("Row %d: ", rowIndex+1))
		for columnIndex, value := range row {
			if columnIndex > 0 {
				builder.WriteString(" | ")
			}
			builder.WriteString(strings.TrimSpace(value))
		}
	}
	return strings.TrimSpace(builder.String())
}
