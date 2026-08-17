package aigateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnthropicCompletePreservesMultipleTextBlocksAndToolUse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("x-api-key") != "anthropic-secret" || request.Header.Get("anthropic-version") == "" {
			t.Fatalf("missing provider headers")
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-model","content":[{"type":"text","text":"first "},{"type":"thinking","thinking":"private","signature":"sig"},{"type":"text","text":"second"},{"type":"tool_use","id":"toolu_1","name":"lookup","input":{"id":2}}],"stop_reason":"tool_use","usage":{"input_tokens":10,"cache_creation_input_tokens":2,"cache_read_input_tokens":3,"output_tokens":4}}`))
	}))
	defer server.Close()
	provider := newAnthropicProvider(ResolvedProviderConfig{ProviderConfig: ProviderConfig{ID: "anthropic", BaseURL: server.URL}, Secret: "anthropic-secret", Timeout: defaultProviderTimeout, RequireUsage: true}, 1<<20, 1<<20)
	response, err := provider.Complete(context.Background(), ProviderRequest{Request: validCanonicalRequest(false), ProviderModel: "claude-model"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Text != "first second" || len(response.ToolCalls) != 1 || response.Usage.InputTokens != 15 || response.Usage.CachedInputTokens != 3 {
		t.Fatalf("response = %#v", response)
	}
}

func TestAnthropicCompletePreservesRefusal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"msg_refusal","type":"message","role":"assistant","model":"claude-model","content":[{"type":"refusal","refusal":"I cannot help with that."}],"stop_reason":"refusal","usage":{"input_tokens":6,"output_tokens":4}}`))
	}))
	defer server.Close()
	provider := newAnthropicProvider(ResolvedProviderConfig{ProviderConfig: ProviderConfig{ID: "anthropic", BaseURL: server.URL}, Secret: "anthropic-secret", Timeout: defaultProviderTimeout, RequireUsage: true}, 1<<20, 1<<20)
	response, err := provider.Complete(context.Background(), ProviderRequest{Request: validCanonicalRequest(false), ProviderModel: "claude-model"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Text != "" || response.Refusal != "I cannot help with that." || response.FinishReason != "refusal" || response.Usage.TotalTokens() != 10 {
		t.Fatalf("response = %#v", response)
	}
}

func TestAnthropicStreamValidatesBlockOrderingAndToolJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		lines := []string{
			`event: message_start\ndata: {"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"m","content":[],"usage":{"input_tokens":5,"cache_read_input_tokens":2,"output_tokens":1}}}\n\n`,
			`event: content_block_start\ndata: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}\n\n`,
			`event: content_block_delta\ndata: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}\n\n`,
			`event: content_block_stop\ndata: {"type":"content_block_stop","index":0}\n\n`,
			`event: content_block_start\ndata: {"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"lookup","input":{}}}\n\n`,
			`event: content_block_delta\ndata: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"id\":"}}\n\n`,
			`event: content_block_delta\ndata: {"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"2}"}}\n\n`,
			`event: content_block_stop\ndata: {"type":"content_block_stop","index":1}\n\n`,
			`event: message_delta\ndata: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":8}}\n\n`,
			`event: message_stop\ndata: {"type":"message_stop"}\n\n`,
		}
		for _, line := range lines {
			_, _ = fmt.Fprint(writer, strings.ReplaceAll(line, `\n`, "\n"))
		}
	}))
	defer server.Close()
	provider := newAnthropicProvider(ResolvedProviderConfig{ProviderConfig: ProviderConfig{ID: "anthropic", BaseURL: server.URL}, Secret: "secret-value", Timeout: defaultProviderTimeout, RequireUsage: true}, 1<<20, 1<<20)
	stream, err := provider.Stream(context.Background(), ProviderRequest{Request: validCanonicalRequest(true), ProviderModel: "m"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var sawTool, sawUsage, sawDone bool
	for {
		event, err := stream.Recv()
		if err != nil {
			t.Fatal(err)
		}
		switch event.Type {
		case StreamToolDelta:
			sawTool = true
		case StreamUsage:
			sawUsage = event.Usage != nil && event.Usage.InputTokens == 7 && event.Usage.OutputTokens == 8
		case StreamDone:
			sawDone = true
		}
		if sawDone {
			break
		}
	}
	if !sawTool || !sawUsage {
		t.Fatalf("tool=%v usage=%v done=%v", sawTool, sawUsage, sawDone)
	}
}

func TestAnthropicStreamClassifiesErrorEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(writer, "event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"overloaded_error\",\"message\":\"raw provider text\"}}\n\n")
	}))
	defer server.Close()
	provider := newAnthropicProvider(ResolvedProviderConfig{ProviderConfig: ProviderConfig{ID: "anthropic", BaseURL: server.URL}, Secret: "secret-value", Timeout: defaultProviderTimeout, RequireUsage: true}, 1<<20, 1<<20)
	stream, err := provider.Stream(context.Background(), ProviderRequest{Request: validCanonicalRequest(true), ProviderModel: "m"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if _, err := stream.Recv(); !errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), "raw provider text") {
		t.Fatalf("error = %v", err)
	}
}

func TestAnthropicToolChoiceNoneOmitsTools(t *testing.T) {
	provider := newAnthropicProvider(ResolvedProviderConfig{ProviderConfig: ProviderConfig{ID: "anthropic", BaseURL: "http://127.0.0.1:1"}, Secret: "secret-value", Timeout: defaultProviderTimeout, RequireUsage: true}, 1<<20, 1<<20).(*anthropicProvider)
	request := validCanonicalRequest(false)
	request.Tools = []ToolDefinition{{Name: "lookup", Parameters: json.RawMessage(`{"type":"object"}`)}}
	request.ToolChoice = ToolChoice{Mode: ToolChoiceNone}
	payload, err := provider.requestPayload(ProviderRequest{Request: request, ProviderModel: "m"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), `"tools"`) || strings.Contains(string(payload), `"tool_choice"`) {
		t.Fatalf("tool-choice none payload = %s", payload)
	}
}

func TestAnthropicStreamRejectsOverlappingContentBlocks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(writer, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"m\",\"content\":[],\"usage\":{\"input_tokens\":1,\"output_tokens\":0}}}\n\n")
		_, _ = fmt.Fprint(writer, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
		_, _ = fmt.Fprint(writer, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":1,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
	}))
	defer server.Close()
	provider := newAnthropicProvider(ResolvedProviderConfig{ProviderConfig: ProviderConfig{ID: "anthropic", BaseURL: server.URL}, Secret: "secret-value", Timeout: defaultProviderTimeout, RequireUsage: false}, 1<<20, 1<<20)
	stream, err := provider.Stream(context.Background(), ProviderRequest{Request: validCanonicalRequest(true), ProviderModel: "m"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if _, err := stream.Recv(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("overlap error=%v", err)
	}
}

func TestAnthropicUsageRequiresCanonicalCounters(t *testing.T) {
	input := int64(1)
	if _, err := anthropicUsageValue(anthropicUsage{InputTokens: &input}); !errors.Is(err, ErrProtocol) {
		t.Fatalf("missing output-token error = %v", err)
	}
}

func TestAnthropicStreamRejectsDecreasingOutputUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(writer, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"m\",\"content\":[],\"usage\":{\"input_tokens\":1,\"output_tokens\":2}}}\n\n")
		_, _ = fmt.Fprint(writer, "event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":1}}\n\n")
	}))
	defer server.Close()
	provider := newAnthropicProvider(ResolvedProviderConfig{ProviderConfig: ProviderConfig{ID: "anthropic", BaseURL: server.URL}, Secret: "secret-value", Timeout: defaultProviderTimeout, RequireUsage: true}, 1<<20, 1<<20)
	stream, err := provider.Stream(context.Background(), ProviderRequest{Request: validCanonicalRequest(true), ProviderModel: "m"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if _, err := stream.Recv(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("decreasing output usage error = %v", err)
	}
}
