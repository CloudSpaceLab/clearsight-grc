package documentimport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrParserAdapterInvalid = errors.New("invalid parser adapter response")

type ParserAdapter interface {
	Name() string
	Extract(context.Context, ParserRequest) (ParserResponse, error)
}

type ParserRequest struct {
	ArtifactID string
	FileName   string
	MediaType  string
	Data       []byte
	MaxBytes   int64
	MaxPages   int
	Deadline   time.Time
}

type ParserResponse struct {
	ParserVersion string             `json:"parser_version"`
	Elements      []ExtractedElement `json:"elements"`
	Degradations  []Degradation      `json:"degradations"`
	Pages         int                `json:"pages"`
	OutputBytes   int64              `json:"output_bytes"`
}

type ParserAdapterPolicy struct {
	Enabled        bool
	Extensions     map[string]struct{}
	Timeout        time.Duration
	MaxBytes       int64
	MaxPages       int
	MaxElements    int
	MaxOutputBytes int64
}

func DefaultParserAdapterPolicy(extraction ExtractionPolicy, maxBytes int64) ParserAdapterPolicy {
	extraction = extraction.normalized()
	if maxBytes <= 0 {
		maxBytes = 20 << 20
	}
	return ParserAdapterPolicy{
		Extensions:     map[string]struct{}{“.pdf”: {}},
		Timeout:        30 * time.Second,
		MaxBytes:       maxBytes,
		MaxPages:       extraction.MaxPDFPages,
		MaxElements:    extraction.MaxElements,
		MaxOutputBytes: extraction.MaxExtractedTextBytes,
	}
}

func (p ParserAdapterPolicy) normalized(extraction ExtractionPolicy, maxBytes int64) ParserAdapterPolicy {
	defaults := DefaultParserAdapterPolicy(extraction, maxBytes)
	if p.Extensions == nil {
		p.Extensions = defaults.Extensions
	}
	if p.Timeout <= 0 {
		p.Timeout = defaults.Timeout
	}
	if p.MaxBytes <= 0 {
		p.MaxBytes = defaults.MaxBytes
	}
	if p.MaxPages <= 0 {
		p.MaxPages = defaults.MaxPages
	}
	if p.MaxElements <= 0 {
		p.MaxElements = defaults.MaxElements
	}
	if p.MaxOutputBytes <= 0 {
		p.MaxOutputBytes = defaults.MaxOutputBytes
	}
	return p
}

func ApplyParserAdapter(ctx context.Context, artifactID, fileName, mediaType string, data []byte, base ExtractionResult, extraction ExtractionPolicy, adapter ParserAdapter, policy ParserAdapterPolicy) ExtractionResult {
	policy = policy.normalized(extraction, int64(len(data)))
	if adapter == nil || !policy.Enabled || !parserAdapterEligible(base.Status) || !adapterExtensionAllowed(fileName, policy.Extensions) {
		return base
	}
	if int64(len(data)) > policy.MaxBytes {
		return parserAdapterFallback(base, "PARSER_ADAPTER_INPUT_LIMIT", "Optional parser adapter input exceeded the configured byte limit.")
	}
	deadline := time.Now().UTC().Add(policy.Timeout)
	if current, ok := ctx.Deadline(); ok && current.Before(deadline) {
		deadline = current
	}
	adapterCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	response, err := adapter.Extract(adapterCtx, ParserRequest{
		ArtifactID: strings.TrimSpace(artifactID), FileName: fileName, MediaType: mediaType,
		Data: append([]byte(nil), data...), MaxBytes: policy.MaxBytes, MaxPages: policy.MaxPages, Deadline: deadline,
	})
	if err != nil {
		if adapterCtx.Err() != nil {
			return parserAdapterFallback(base, "PARSER_ADAPTER_TIMEOUT", "Optional parser adapter did not complete within its bounded deadline.")
		}
		return parserAdapterFallback(base, "PARSER_ADAPTER_FAILED", "Optional parser adapter failed; the default extraction outcome remains authoritative.")
	}
	if err := validateParserResponse(response, policy); err != nil {
		return parserAdapterFallback(base, "PARSER_ADAPTER_INVALID", "Optional parser adapter returned an invalid or over-limit response.")
	}
	if len(response.Elements) == 0 {
		return parserAdapterFallback(base, "PARSER_ADAPTER_EMPTY", "Optional parser adapter returned no reviewable elements.")
	}

	status := ExtractionExtracted
	if len(response.Degradations) > 0 {
		status = ExtractionPartial
	}
	limitations := make([]string, 0, len(response.Degradations))
	for _, degradation := range response.Degradations {
		if message := strings.TrimSpace(degradation.Message); message != "" {
			limitations = append(limitations, message)
		}
	}
	return ExtractionResult{
		Status: status, Method: "PARSER_ADAPTER", ParserVersion: strings.TrimSpace(response.ParserVersion),
		AdapterVersion: strings.TrimSpace(adapter.Name()), Limitations: limitations,
		Sections: []Section{}, Elements: cloneElements(response.Elements), Degradations: cloneDegradations(response.Degradations),
		SectionsTotal: 0,
	}
}

func parserAdapterEligible(status ExtractionStatus) bool {
	return status == ExtractionUnsupported || status == ExtractionPartial
}

func adapterExtensionAllowed(fileName string, extensions map[string]struct{}) bool {
	extension := strings.ToLower(filepath.Ext(strings.TrimSpace(fileName)))
	_, ok := extensions[extension]
	return ok
}

func parserAdapterFallback(base ExtractionResult, code, message string) ExtractionResult {
	base.Degradations = append(cloneDegradations(base.Degradations), Degradation{Code: code, Message: message, Recoverable: true})
	base.Limitations = append(append([]string(nil), base.Limitations...), message)
	if base.Status == ExtractionExtracted {
		base.Status = ExtractionPartial
	}
	return base
}

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
		values := []float64{box.X0, box.Y0, box.X1, box.Y1}
		for _, value := range values {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
				return errors.Join(ErrParserAdapterInvalid, errors.New("bounding box is invalid"))
			}
		if box.X1 < box.X0 || box.Y1 < box.Y0 {
			return errors.Join(ErrParserAdapterInvalid, errors.New("bounding box coordinates are inverted"))
		}
	}
	return nil
}
