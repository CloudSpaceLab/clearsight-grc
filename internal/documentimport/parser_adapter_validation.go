package documentimport

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

func validateParserResponse(response ParserResponse, policy ParserAdapterPolicy) error {
	if strings.TrimSpace(response.ParserVersion) == "" || len(response.ParserVersion) > 128 || !utf8.ValidString(response.ParserVersion) {
		return errors.Join(ErrParserAdapterInvalid, errors.New("parser version is required"))
	}
	if response.Pages < 0 || response.Pages > policy.MaxPages || len(response.Elements) > policy.MaxElements || response.OutputBytes < 0 || response.OutputBytes > policy.MaxOutputBytes {
		return errors.Join(ErrParserAdapterInvalid, errors.New("parser response exceeds configured bounds"))
	}
	encoded, err := json.Marshal(struct {
		Elements     []ExtractedElement `json:"elements"`
		Degradations []Degradation      `json:"degradations"`
	}{response.Elements, response.Degradations})
	if err != nil || int64(len(encoded)) > policy.MaxOutputBytes {
		return errors.Join(ErrParserAdapterInvalid, errors.New("parser materialized output exceeds configured bounds"))
	}
	seen := make(map[string]struct{}, len(response.Elements))
	for _, element := range response.Elements {
		if err := validateAdapterElement(element); err != nil {
			return err
		}
		if _, exists := seen[element.Ref]; exists {
			return errors.Join(ErrParserAdapterInvalid, errors.New("element refs must be unique"))
		}
		seen[element.Ref] = struct{}{}
	}
	for _, degradation := range response.Degradations {
		if strings.TrimSpace(degradation.Code) == "" || len(degradation.Code) > 128 || len(degradation.Message) > 2000 || !utf8.ValidString(degradation.Message) {
			return errors.Join(ErrParserAdapterInvalid, errors.New("degradation is invalid"))
		}
		if degradation.Anchor != nil {
			if err := validateAdapterAnchor(*degradation.Anchor); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateAdapterElement(element ExtractedElement) error {
	if strings.TrimSpace(element.Ref) == "" || len(element.Ref) > 256 || len(element.Text) > 1<<20 || len(element.Target) > 8192 || !utf8.ValidString(element.Text) || !utf8.ValidString(element.Target) {
		return errors.Join(ErrParserAdapterInvalid, errors.New("element identity or text is invalid"))
	}
	switch element.Kind {
	case ElementHeading, ElementParagraph, ElementTable, ElementFormControl, ElementImage, ElementLink:
	default:
		return errors.Join(ErrParserAdapterInvalid, fmt.Errorf("unsupported element kind %q", element.Kind))
	}
	if element.Kind == ElementFormControl && element.Control == nil {
		return errors.Join(ErrParserAdapterInvalid, errors.New("form control element is missing control metadata"))
	}
	return validateAdapterAnchor(element.Anchor)
}

func validateAdapterAnchor(anchor SourceAnchor) error {
	if anchor.Page < 0 || anchor.RowStart < 0 || anchor.RowEnd < 0 || anchor.RowEnd > 0 && anchor.RowStart > anchor.RowEnd || len(anchor.Sheet) > 256 || len(anchor.Paragraph) > 256 || len(anchor.Table) > 256 || len(anchor.Cell) > 256 {
		return errors.Join(ErrParserAdapterInvalid, errors.New("source anchor is invalid"))
	}
	if box := anchor.BoundingBox; box != nil {
		for _, value := range []float64{box.X0, box.Y0, box.X1, box.Y1} {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
				return errors.Join(ErrParserAdapterInvalid, errors.New("bounding box is invalid"))
			}
		}
		if box.X1 < box.X0 || box.Y1 < box.Y0 {
			return errors.Join(ErrParserAdapterInvalid, errors.New("bounding box coordinates are inverted"))
		}
	}
	return nil
}
