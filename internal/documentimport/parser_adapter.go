package documentimport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"time"
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
	Data       io.Reader
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
		Extensions:     map[string]struct{}{ ".pdf": {} },
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
		Data: bytes.NewReader(data), MaxBytes: policy.MaxBytes, MaxPages: policy.MaxPages, Deadline: deadline,
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
