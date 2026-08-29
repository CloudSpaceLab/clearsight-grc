package documentimport

import (
	"fmt"
	"strings"
)

const (
	extractionElementAdapterVersion = "SECTION_ELEMENT_ADAPTER_V1"
	docxElementAdapterVersion       = "DOCX_STRUCTURE_ADAPTER_V1"
)

type ElementKind string

const (
	ElementHeading     ElementKind = "HEADING"
	ElementParagraph   ElementKind = "PARAGRAPH"
	ElementTable       ElementKind = "TABLE"
	ElementFormControl ElementKind = "FORM_CONTROL"
	ElementImage       ElementKind = "IMAGE"
	ElementLink        ElementKind = "LINK"
)

type BoundingBox struct {
	X0 float64 `json:"x0"`
	Y0 float64 `json:"y0"`
	X1 float64 `json:"x1"`
	Y1 float64 `json:"y1"`
}

type FormControl struct {
	Kind    string   `json:"kind"`
	Label   string   `json:"label,omitempty"`
	Help    string   `json:"help,omitempty"`
	Options []string `json:"options,omitempty"`
	Checked *bool    `json:"checked,omitempty"`
}

type SourceAnchor struct {
	Page        int          `json:"page,omitempty"`
	Sheet       string       `json:"sheet,omitempty"`
	RowStart    int          `json:"row_start,omitempty"`
	RowEnd      int          `json:"row_end,omitempty"`
	Paragraph   string       `json:"paragraph,omitempty"`
	Table       string       `json:"table,omitempty"`
	Cell        string       `json:"cell,omitempty"`
	BoundingBox *BoundingBox `json:"bounding_box,omitempty"`
}

type Degradation struct {
	Code        string        `json:"code"`
	Message     string        `json:"message"`
	Recoverable bool          `json:"recoverable"`
	Anchor      *SourceAnchor `json:"anchor,omitempty"`
}

type ExtractedElement struct {
	Ref     string       `json:"ref,omitempty"`
	Kind    ElementKind  `json:"kind"`
	Text    string       `json:"text,omitempty"`
	Target  string       `json:"target,omitempty"`
	Values  [][]string   `json:"values,omitempty"`
	Control *FormControl `json:"control,omitempty"`
	Anchor  SourceAnchor `json:"anchor"`
}

func elementsFromSections(sections []Section) []ExtractedElement {
	elements := make([]ExtractedElement, 0, len(sections)*2)
	for _, section := range sections {
		anchor := sourceAnchorFromSection(section)
		if title := strings.TrimSpace(section.Title); meaningfulElementHeading(title, section) {
			elements = append(elements, ExtractedElement{
				Ref: section.ID + ":heading", Kind: ElementHeading, Text: title, Anchor: anchor,
			})
		}
		text := strings.TrimSpace(section.Text)
		if text == "" {
			continue
		}
		kind := ElementParagraph
		if section.RowStart > 0 || strings.TrimSpace(section.Sheet) != "" {
			kind = ElementTable
		}
		elements = append(elements, ExtractedElement{
			Ref: section.ID, Kind: kind, Text: text, Anchor: anchor,
		})
	}
	return elements
}

func sourceAnchorFromSection(section Section) SourceAnchor {
	return SourceAnchor{
		Page: section.Page, Sheet: section.Sheet, RowStart: section.RowStart, RowEnd: section.RowEnd,
	}
}

func meaningfulElementHeading(title string, section Section) bool {
	if title == "" || title == "Document text" {
		return false
	}
	if section.Page > 0 && title == fmt.Sprintf("Page %d", section.Page) {
		return false
	}
	if section.RowStart > 0 {
		return false
	}
	return !strings.HasPrefix(title, "Section ")
}

// withDerivedExtractionDetails keeps the existing Section projection useful
// while Task 15 introduces durable extraction_details. It derives only facts
// that are reconstructable from already-persisted fields and never invents
// controls, images, links, or parser degradations that were not retained.
func withDerivedExtractionDetails(value Document) Document {
	if value.ParserVersion == "" {
		value.ParserVersion = parserVersionFor(value)
	}
	if value.AdapterVersion == "" && value.ParserVersion != "" {
		value.AdapterVersion = extractionElementAdapterVersion
	}
	if value.Elements == nil && len(value.Sections) > 0 {
		value.Elements = elementsFromSections(value.Sections)
	}
	return value
}

func cloneElements(values []ExtractedElement) []ExtractedElement {
	if values == nil {
		return nil
	}
	result := make([]ExtractedElement, len(values))
	copy(result, values)
	for index := range result {
		result[index].Values = cloneStringMatrix(values[index].Values)
		if values[index].Control != nil {
			control := *values[index].Control
			control.Options = append([]string(nil), values[index].Control.Options...)
			if values[index].Control.Checked != nil {
				checked := *values[index].Control.Checked
				control.Checked = &checked
			}
			result[index].Control = &control
		}
		if values[index].Anchor.BoundingBox != nil {
			box := *values[index].Anchor.BoundingBox
			result[index].Anchor.BoundingBox = &box
		}
	}
	return result
}

func cloneStringMatrix(values [][]string) [][]string {
	if values == nil {
		return nil
	}
	result := make([][]string, len(values))
	for index := range values {
		result[index] = append([]string(nil), values[index]...)
	}
	return result
}

func cloneDegradations(values []Degradation) []Degradation {
	if values == nil {
		return nil
	}
	result := make([]Degradation, len(values))
	copy(result, values)
	for index := range result {
		if values[index].Anchor != nil {
			anchor := *values[index].Anchor
			if values[index].Anchor.BoundingBox != nil {
				box := *values[index].Anchor.BoundingBox
				anchor.BoundingBox = &box
			}
			result[index].Anchor = &anchor
		}
	}
	return result
}
