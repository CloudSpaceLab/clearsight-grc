package documentimport

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type parserAdapterFunc struct {
	name string
	fn   func(context.Context, ParserRequest) (ParserResponse, error)
}

func (a parserAdapterFunc) Name() string { return a.name }
func (a parserAdapterFunc) Extract(ctx context.Context, request ParserRequest) (ParserResponse, error) {
	return a.fn(ctx, request)
}

func TestParserAdapterDisabledPreservesDeterministicFallback(t *testing.T) {
	called := false
	adapter := parserAdapterFunc{name: "TEST", fn: func(context.Context, ParserRequest) (ParserResponse, error) {
		called = true
		return ParserResponse{}, nil
	}}
	base := ExtractionResult{Status: ExtractionUnsupported, Method: "PDF_TEXT_V2", Limitations: []string{"OCR required"}}
	got := ApplyParserAdapter(t.Context(), "artifact", "scan.pdf", "application/pdf", []byte("pdf"), base, DefaultExtractionPolicy(), adapter, ParserAdapterPolicy{})
	if called || got.Status != base.Status || got.Method != base.Method || len(got.Limitations) != 1 {
		t.Fatalf("disabled adapter changed deterministic fallback: %#v called=%v", got, called)
	}
}

func TestParserAdapterTimeoutKeepsDefaultOutcomeExplicit(t *testing.T) {
	adapter := parserAdapterFunc{name: "SLOW", fn: func(ctx context.Context, request ParserRequest) (ParserResponse, error) {
		_, _ = io.ReadAll(request.Data)
		<-ctx.Done()
		return ParserResponse{}, ctx.Err()
	}}
	base := ExtractionResult{Status: ExtractionUnsupported, Method: "PDF_TEXT_V2"}
	policy := DefaultParserAdapterPolicy(DefaultExtractionPolicy(), 1024)
	policy.Enabled = true
	policy.Timeout = 5 * time.Millisecond
	got := ApplyParserAdapter(t.Context(), "artifact", "scan.pdf", "application/pdf", []byte("pdf"), base, DefaultExtractionPolicy(), adapter, policy)
	if got.Status != ExtractionUnsupported || len(got.Degradations) != 1 || got.Degradations[0].Code != "PARSER_ADAPTER_TIMEOUT" {
		t.Fatalf("timeout was not explicit: %#v", got)
	}
}

func TestParserAdapterRejectsInvalidAnchorAndOutputLimit(t *testing.T) {
	cases := []struct {
		name     string
		response ParserResponse
	}{
		{name: "anchor", response: ParserResponse{ParserVersion: "adapter-v1", Pages: 1, OutputBytes: 10, Elements: []ExtractedElement{{Ref: "bad", Kind: ElementParagraph, Text: "x", Anchor: SourceAnchor{Page: -1}}}}},
		{name: "output", response: ParserResponse{ParserVersion: "adapter-v1", Pages: 1, OutputBytes: 4097, Elements: []ExtractedElement{{Ref: "ok", Kind: ElementParagraph, Text: "x", Anchor: SourceAnchor{Page: 1}}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter := parserAdapterFunc{name: "TEST", fn: func(context.Context, ParserRequest) (ParserResponse, error) { return tc.response, nil }}
			policy := DefaultParserAdapterPolicy(DefaultExtractionPolicy(), 1024)
			policy.Enabled = true
			policy.MaxOutputBytes = 4096
			got := ApplyParserAdapter(t.Context(), "artifact", "scan.pdf", "application/pdf", []byte("pdf"), ExtractionResult{Status: ExtractionUnsupported}, DefaultExtractionPolicy(), adapter, policy)
			if len(got.Degradations) != 1 || got.Degradations[0].Code != "PARSER_ADAPTER_INVALID" {
				t.Fatalf("invalid adapter response was accepted: %#v", got)
			}
		})
	}
}

func TestParserAdapterRetainsPartialResponseAndStructuredDegradation(t *testing.T) {
	adapter := parserAdapterFunc{name: "PYMUPDF:v1", fn: func(_ context.Context, request ParserRequest) (ParserResponse, error) {
		payload, err := io.ReadAll(request.Data)
		if err != nil || string(payload) != "pdf" {
			return ParserResponse{}, errors.New("unexpected request payload")
		}
		return ParserResponse{
			ParserVersion: "pymupdf-1.26", Pages: 2, OutputBytes: 256,
			Elements:     []ExtractedElement{{Ref: "p1", Kind: ElementParagraph, Text: "Recovered text", Anchor: SourceAnchor{Page: 1}}},
			Degradations: []Degradation{{Code: "IMAGE_REVIEW_REQUIRED", Message: "One image requires review.", Recoverable: true, Anchor: &SourceAnchor{Page: 2}}},
		}, nil
	}}
	policy := DefaultParserAdapterPolicy(DefaultExtractionPolicy(), 1024)
	policy.Enabled = true
	policy.MaxOutputBytes = 4096
	got := ApplyParserAdapter(t.Context(), "artifact", "scan.pdf", "application/pdf", []byte("pdf"), ExtractionResult{Status: ExtractionUnsupported}, DefaultExtractionPolicy(), adapter, policy)
	if got.Status != ExtractionPartial || got.ParserVersion != "pymupdf-1.26" || got.AdapterVersion != "PYMUPDF:v1" || len(got.Elements) != 1 || len(got.Degradations) != 1 {
		t.Fatalf("partial adapter response lost structure: %#v", got)
	}
}

func TestParserAdapterDoesNotOverrideStructuralFailure(t *testing.T) {
	called := false
	adapter := parserAdapterFunc{name: "TEST", fn: func(context.Context, ParserRequest) (ParserResponse, error) {
		called = true
		return ParserResponse{}, nil
	}}
	policy := DefaultParserAdapterPolicy(DefaultExtractionPolicy(), 1024)
	policy.Enabled = true
	base := ExtractionResult{Status: ExtractionFailed, Method: "PDF_TEXT_V2", Limitations: []string{"resource limit"}}
	got := ApplyParserAdapter(t.Context(), "artifact", "scan.pdf", "application/pdf", []byte(strings.Repeat("x", 8)), base, DefaultExtractionPolicy(), adapter, policy)
	if called || got.Status != ExtractionFailed {
		t.Fatalf("structural failure must not fall through to adapter: %#v called=%v", got, called)
	}
}
