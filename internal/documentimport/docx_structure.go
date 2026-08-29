package documentimport

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type docxStructureResult struct {
	elements     []ExtractedElement
	degradations []Degradation
	truncated    bool
}

type docxStructureParser struct {
	ctx           context.Context
	policy        ExtractionPolicy
	relationships map[string]docxRelationship
	numbering     *docxNumbering
	counters      map[string][]int
	elements      []ExtractedElement
	degradations  []Degradation
	textBytes     int64
	paragraphs    int
	tables        int
	rows          int
	cells         int
	truncated     bool
}

type docxParagraphResult struct {
	text         string
	style        string
	numID        string
	level        int
	inline       []ExtractedElement
	degradations []Degradation
	truncated    bool
}

func extractDOCXStructuredElements(ctx context.Context, data []byte, policy ExtractionPolicy) (docxStructureResult, error) {
	policy = policy.normalized()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return docxStructureResult{}, fmt.Errorf("DOCX could not be parsed: %w", err)
	}
	if err := validateArchive(archive.File, policy); err != nil {
		return docxStructureResult{}, err
	}
	document := archivePart(archive.File, "word/document.xml")
	if document == nil {
		return docxStructureResult{}, fmt.Errorf("DOCX could not be parsed: word/document.xml is missing")
	}
	relationships, err := readDOCXRelationships(ctx, archive.File)
	if err != nil {
		return docxStructureResult{}, err
	}
	numbering, err := readDOCXNumbering(ctx, archive.File)
	if err != nil {
		return docxStructureResult{}, err
	}
	parser := &docxStructureParser{
		ctx: ctx, policy: policy, relationships: relationships, numbering: numbering,
		counters: make(map[string][]int), elements: make([]ExtractedElement, 0, min(policy.MaxElements, 128)),
		degradations: make([]Degradation, 0, 4),
	}
	if err := parser.parse(document); err != nil {
		return docxStructureResult{}, err
	}
	return docxStructureResult{
		elements: cloneElements(parser.elements), degradations: cloneDegradations(parser.degradations), truncated: parser.truncated,
	}, nil
}

func (p *docxStructureParser) parse(file *zip.File) error {
	stream, err := file.Open()
	if err != nil {
		return err
	}
	defer stream.Close()
	decoder := xml.NewDecoder(stream)
	for {
		if err := p.ctx.Err(); err != nil {
			return err
		}
		token, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("DOCX XML could not be parsed: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "p":
			paragraph, err := p.readParagraph(decoder, false, SourceAnchor{})
			if err != nil {
				return err
			}
			p.emitParagraph(paragraph)
		case "tbl":
			if err := p.readTable(decoder); err != nil {
				return err
			}
		case "sdt":
			control, text, truncated, err := readDOCXContentControl(p.ctx, decoder, p.policy)
			if err != nil {
				return err
			}
			p.truncated = p.truncated || truncated
			if control != nil {
				p.addElement(ExtractedElement{Kind: ElementFormControl, Text: text, Control: control})
			} else if text != "" {
				p.addElement(ExtractedElement{Kind: ElementParagraph, Text: text})
			}
		case "drawing", "pict":
			p.degrade("DOCX_IMAGES_NOT_EXTRACTED", "The DOCX contains embedded images that this parser version cannot yet preserve structurally.", nil)
		}
	}
}

func (p *docxStructureParser) readParagraph(decoder *xml.Decoder, inTable bool, anchor SourceAnchor) (docxParagraphResult, error) {
	p.paragraphs++
	if !inTable {
		anchor.Paragraph = fmt.Sprintf("paragraph-%d", p.paragraphs)
	}
	result := docxParagraphResult{level: 0}
	var text strings.Builder
	var hyperlinkText strings.Builder
	inText := false
	inInstruction := false
	hyperlinkDepth := 0
	hyperlinkTarget := ""
	hyperlinkAllowed := true
	var field docxFieldState
	depth := 1
	for depth > 0 {
		if err := p.ctx.Err(); err != nil {
			return result, err
		}
		token, err := decoder.Token()
		if err == io.EOF {
			return result, fmt.Errorf("DOCX paragraph ended unexpectedly")
		}
		if err != nil {
			return result, err
		}
		switch item := token.(type) {
		case xml.StartElement:
			depth++
			switch item.Name.Local {
			case "pStyle":
				result.style = strings.TrimSpace(xmlAttribute(item, "val"))
			case "numId":
				result.numID = strings.TrimSpace(xmlAttribute(item, "val"))
			case "ilvl":
				result.level = parseNonNegativeInt(xmlAttribute(item, "val"), 0)
			case "t":
				inText = true
			case "tab":
				appendBounded(&text, "\t", p.policy.MaxCellBytes, &result.truncated)
				if hyperlinkDepth > 0 {
					appendBounded(&hyperlinkText, "\t", p.policy.MaxCellBytes, &result.truncated)
				}
			case "br", "cr":
				appendBounded(&text, "\n", p.policy.MaxCellBytes, &result.truncated)
			case "sdt":
				control, visible, truncated, err := readDOCXContentControl(p.ctx, decoder, p.policy)
				if err != nil {
					return result, err
				}
				depth--
				result.truncated = result.truncated || truncated
				if visible != "" {
					appendBounded(&text, visible, p.policy.MaxCellBytes, &result.truncated)
				}
				if control != nil {
					result.inline = append(result.inline, ExtractedElement{Kind: ElementFormControl, Text: visible, Control: control, Anchor: anchor})
				}
			case "fldSimple":
				control, visible, truncated, err := readDOCXSimpleField(p.ctx, decoder, item, p.policy)
				if err != nil {
					return result, err
				}
				depth--
				result.truncated = result.truncated || truncated
				if visible != "" {
					appendBounded(&text, visible, p.policy.MaxCellBytes, &result.truncated)
				}
				if control != nil {
					result.inline = append(result.inline, ExtractedElement{Kind: ElementFormControl, Text: visible, Control: control, Anchor: anchor})
				}
			case "fldChar":
				typeValue := strings.ToLower(strings.TrimSpace(xmlAttribute(item, "fldCharType")))
				switch typeValue {
				case "begin":
					startComplexField(&field)
				case "separate":
					inInstruction = false
				case "end":
					control, visible, truncated := finishComplexField(&field)
					result.truncated = result.truncated || truncated
					if control != nil {
						result.inline = append(result.inline, ExtractedElement{Kind: ElementFormControl, Text: visible, Control: control, Anchor: anchor})
					}
				}
				if field.active {
					observeFieldMetadata(&field.metadata, item, p.policy)
				}
			case "instrText":
				if field.active {
					inInstruction = true
				}
			case "hyperlink":
				hyperlinkDepth++
				if hyperlinkDepth == 1 {
					hyperlinkText.Reset()
					relID := strings.TrimSpace(xmlAttribute(item, "id"))
					anchorTarget := strings.TrimSpace(xmlAttribute(item, "anchor"))
					if anchorTarget != "" {
						hyperlinkTarget = "#" + truncateUTF8(anchorTarget, p.policy.MaxCellBytes)
						hyperlinkAllowed = true
					} else if relation, ok := p.relationships[relID]; ok {
						hyperlinkTarget = relation.Target
						hyperlinkAllowed = relation.Allowed
					} else {
						hyperlinkTarget = ""
						hyperlinkAllowed = false
					}
				}
			case "drawing", "pict":
				result.degradations = append(result.degradations, Degradation{
					Code: "DOCX_IMAGES_NOT_EXTRACTED", Message: "The DOCX contains embedded images that this parser version cannot yet preserve structurally.", Recoverable: true, Anchor: cloneSourceAnchor(anchor),
				})
			default:
				if field.active {
					observeFieldMetadata(&field.metadata, item, p.policy)
				}
			}
		case xml.CharData:
			if inInstruction && field.active {
				appendBounded(&field.instruction, string(item), p.policy.MaxCellBytes, &field.truncated)
				continue
			}
			if inText {
				appendBounded(&text, string(item), p.policy.MaxCellBytes, &result.truncated)
				if field.active {
					appendBounded(&field.result, string(item), p.policy.MaxCellBytes, &field.truncated)
				}
				if hyperlinkDepth > 0 {
					appendBounded(&hyperlinkText, string(item), p.policy.MaxCellBytes, &result.truncated)
				}
			}
		case xml.EndElement:
			switch item.Name.Local {
			case "t":
				inText = false
			case "instrText":
				inInstruction = false
			case "hyperlink":
				if hyperlinkDepth == 1 {
					visible := strings.TrimSpace(hyperlinkText.String())
					if visible != "" && hyperlinkAllowed && hyperlinkTarget != "" {
						result.inline = append(result.inline, ExtractedElement{Kind: ElementLink, Text: visible, Target: hyperlinkTarget, Anchor: anchor})
					} else if visible != "" {
						result.degradations = append(result.degradations, Degradation{
							Code: "DOCX_LINK_TARGET_REJECTED", Message: "A DOCX hyperlink target was not retained because its relationship target was missing or not allowlisted.", Recoverable: true, Anchor: cloneSourceAnchor(anchor),
						})
					}
					hyperlinkTarget = ""
					hyperlinkAllowed = true
				}
				if hyperlinkDepth > 0 {
					hyperlinkDepth--
				}
			}
			depth--
		}
	}
	result.text = strings.TrimSpace(text.String())
	return result, nil
}

func (p *docxStructureParser) emitParagraph(paragraph docxParagraphResult) {
	p.truncated = p.truncated || paragraph.truncated
	for _, degradation := range paragraph.degradations {
		p.addDegradation(degradation)
	}
	text := strings.TrimSpace(paragraph.text)
	if text != "" && paragraph.numID != "" {
		if label := p.numbering.label(paragraph.numID, paragraph.level, p.counters); label != "" {
			text = strings.TrimSpace(label + " " + text)
		}
	}
	if text != "" {
		kind := ElementParagraph
		if strings.HasPrefix(strings.ToLower(paragraph.style), "heading") {
			kind = ElementHeading
		}
		anchor := SourceAnchor{Paragraph: fmt.Sprintf("paragraph-%d", p.paragraphs)}
		p.addElement(ExtractedElement{Kind: kind, Text: text, Anchor: anchor})
	}
	for _, element := range paragraph.inline {
		p.addElement(element)
	}
}

func (p *docxStructureParser) readTable(decoder *xml.Decoder) error {
	p.tables++
	tableRef := fmt.Sprintf("table-%d", p.tables)
	rows := make([][]string, 0, 8)
	inline := make([]ExtractedElement, 0, 4)
	var row []string
	var cell strings.Builder
	rowIndex := 0
	cellIndex := 0
	inRow := false
	inCell := false
	depth := 1
	for depth > 0 {
		if err := p.ctx.Err(); err != nil {
			return err
		}
		token, err := decoder.Token()
		if err == io.EOF {
			return fmt.Errorf("DOCX table ended unexpectedly")
		}
		if err != nil {
			return err
		}
		switch item := token.(type) {
		case xml.StartElement:
			depth++
			switch item.Name.Local {
			case "tr":
				p.rows++
				if p.rows > p.policy.MaxRows {
					return limitError("row count exceeds %d", p.policy.MaxRows)
				}
				rowIndex++
				row = make([]string, 0, 8)
				inRow = true
			case "tc":
				p.cells++
				if p.cells > p.policy.MaxCells {
					return limitError("cell count exceeds %d", p.policy.MaxCells)
				}
				cellIndex = len(row) + 1
				if cellIndex > p.policy.MaxColumns {
					return limitError("column count exceeds %d", p.policy.MaxColumns)
				}
				cell.Reset()
				inCell = true
			case "p":
				if !inCell {
					continue
				}
				anchor := SourceAnchor{Table: tableRef, Cell: fmt.Sprintf("r%dc%d", rowIndex, cellIndex), RowStart: rowIndex, RowEnd: rowIndex}
				paragraph, err := p.readParagraph(decoder, true, anchor)
				if err != nil {
					return err
				}
				depth--
				p.truncated = p.truncated || paragraph.truncated
				if paragraph.text != "" {
					if cell.Len() > 0 {
						appendBounded(&cell, "\n", p.policy.MaxCellBytes, &p.truncated)
					}
					appendBounded(&cell, paragraph.text, p.policy.MaxCellBytes, &p.truncated)
				}
				for _, degradation := range paragraph.degradations {
					p.addDegradation(degradation)
				}
				inline = append(inline, paragraph.inline...)
			}
		case xml.EndElement:
			switch item.Name.Local {
			case "tc":
				if inCell {
					row = append(row, strings.TrimSpace(cell.String()))
					inCell = false
				}
			case "tr":
				if inRow {
					rows = append(rows, append([]string(nil), row...))
					inRow = false
				}
			}
			depth--
		}
	}
	p.addElement(ExtractedElement{Kind: ElementTable, Values: rows, Anchor: SourceAnchor{Table: tableRef, RowStart: 1, RowEnd: len(rows)}})
	for _, element := range inline {
		p.addElement(element)
	}
	return nil
}

func (p *docxStructureParser) addElement(element ExtractedElement) {
	if len(p.elements) >= p.policy.MaxElements {
		p.truncated = true
		return
	}
	textBytes := int64(len(element.Text) + len(element.Target))
	for _, row := range element.Values {
		for _, value := range row {
			textBytes += int64(len(value))
		}
	}
	if element.Control != nil {
		textBytes += int64(len(element.Control.Kind) + len(element.Control.Label) + len(element.Control.Help))
		for _, option := range element.Control.Options {
			textBytes += int64(len(option))
		}
	}
	remaining := p.policy.MaxExtractedTextBytes - p.textBytes
	if remaining <= 0 {
		p.truncated = true
		return
	}
	if textBytes > remaining {
		p.truncated = true
		return
	}
	p.textBytes += textBytes
	if element.Ref == "" {
		element.Ref = fmt.Sprintf("docx-element-%d", len(p.elements)+1)
	}
	p.elements = append(p.elements, element)
}

func (p *docxStructureParser) degrade(code, message string, anchor *SourceAnchor) {
	p.addDegradation(Degradation{Code: code, Message: message, Recoverable: true, Anchor: anchor})
}

func (p *docxStructureParser) addDegradation(value Degradation) {
	for _, existing := range p.degradations {
		if existing.Code == value.Code && sameSourceAnchor(existing.Anchor, value.Anchor) {
			return
		}
	}
	p.degradations = append(p.degradations, value)
}

func cloneSourceAnchor(value SourceAnchor) *SourceAnchor {
	copy := value
	if value.BoundingBox != nil {
		box := *value.BoundingBox
		copy.BoundingBox = &box
	}
	return &copy
}

func sameSourceAnchor(left, right *SourceAnchor) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Page == right.Page && left.Sheet == right.Sheet && left.RowStart == right.RowStart && left.RowEnd == right.RowEnd && left.Paragraph == right.Paragraph && left.Table == right.Table && left.Cell == right.Cell
}
