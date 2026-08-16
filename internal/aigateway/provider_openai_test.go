package aigateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAICompleteTranslatesUsageRefusalAndTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/chat/completions" || request.Header.Get("Authorization") != "Bearer provider-secret" {
			t.Fatalf("unexpected provider request")
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"id":"chatcmpl-provider","object":"chat.completion","created":1786800000,"model":"provider-model","choices":[{"index":0,"message":{"role":"assistant","content":"","refusal":"policy refusal","tool_calls":[{"id":"call_1","type":"function","function":{"name":"lookup","arguments":"{\"id\":1}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":12,"completion_tokens":5,"total_tokens":17,"prompt_tokens_details":{"cached_tokens":3}}}`))
	}))
	defer server.Close()
	provider := newOpenAIProvider(ResolvedProviderConfig{ProviderConfig: ProviderConfig{ID: "openai", BaseURL: server.URL}, Secret: "provider-secret", Timeout: defaultProviderTimeout, RequireUsage: true}, 1<<20, 1<<20)
	response, err := provider.Complete(context.Background(), ProviderRequest{Request: validCanonicalRequest(false), ProviderModel: "provider-model"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Refusal != "policy refusal" || response.FinishReason != "tool_calls" || len(response.ToolCalls) != 1 || response.Usage.CachedInputTokens != 3 {
		t.Fatalf("response = %#v", response)
	}
}

func TestOpenAIStreamAcceptsUsageOnlyTerminalChunk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(writer, "data: {\"id\":\"c1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"hello\"},\"finish_reason\":null}]}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"id\":\"c1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"id\":\"c1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"m\",\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":1,\"total_tokens\":5}}\n\n")
		_, _ = fmt.Fprint(writer, "data: [DONE]\n\n")
	}))
	defer server.Close()
	provider := newOpenAIProvider(ResolvedProviderConfig{ProviderConfig: ProviderConfig{ID: "openai", BaseURL: server.URL}, Secret: "secret-value", Timeout: defaultProviderTimeout, RequireUsage: true}, 1<<20, 1<<20)
	stream, err := provider.Stream(context.Background(), ProviderRequest{Request: validCanonicalRequest(true), ProviderModel: "m"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var types []StreamEventType
	for {
		event, err := stream.Recv()
		if err != nil {
			if errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			break
		}
		types = append(types, event.Type)
		if event.Type == StreamDone {
			break
		}
	}
	joined := fmt.Sprint(types)
	for _, want := range []string{string(StreamTextDelta), string(StreamFinish), string(StreamUsage), string(StreamDone)} {
		if !strings.Contains(joined, want) {
			t.Fatalf("events %v missing %s", types, want)
		}
	}
}

func TestOpenAIStreamRejectsIdentityDriftAndMissingUsage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(writer, "data: {\"id\":\"c1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"a\"},\"finish_reason\":null}]}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"id\":\"c2\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = fmt.Fprint(writer, "data: [DONE]\n\n")
	}))
	defer server.Close()
	provider := newOpenAIProvider(ResolvedProviderConfig{ProviderConfig: ProviderConfig{ID: "openai", BaseURL: server.URL}, Secret: "secret-value", Timeout: defaultProviderTimeout, RequireUsage: true}, 1<<20, 1<<20)
	stream, err := provider.Stream(context.Background(), ProviderRequest{Request: validCanonicalRequest(true), ProviderModel: "m"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if _, err := stream.Recv(); err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Recv(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("identity drift error = %v", err)
	}
}

func TestOpenAIProviderRejectsRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Location", "/elsewhere")
		writer.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	provider := newOpenAIProvider(ResolvedProviderConfig{ProviderConfig: ProviderConfig{ID: "openai", BaseURL: server.URL}, Secret: "secret-value", Timeout: defaultProviderTimeout, RequireUsage: true}, 1<<20, 1<<20)
	_, err := provider.Complete(context.Background(), ProviderRequest{Request: validCanonicalRequest(false), ProviderModel: "m"})
	if !errors.Is(err, ErrProtocol) {
		t.Fatalf("redirect error = %v", err)
	}
}

func validCanonicalRequest(stream bool) Request {
	return Request{ID: "req_test", Protocol: ProtocolChat, ModelAlias: "alias", Messages: []Message{{Role: RoleUser, Text: "hello"}}, MaxOutputTokens: 64, Stream: stream, IncludeStreamUsage: true, ToolChoice: ToolChoice{Mode: ToolChoiceAuto}}
}

func TestOpenAIUsageRejectsIntegerOverflow(t *testing.T) {
	prompt := int64(1<<63 - 1)
	completion := int64(1)
	total := int64(0)
	if _, err := openAIUsageValue(openAIUsage{PromptTokens: &prompt, CompletionTokens: &completion, TotalTokens: &total}); !errors.Is(err, ErrProtocol) {
		t.Fatalf("overflow error = %v", err)
	}
}

func TestOpenAIStreamRejectsOutputAfterFinish(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(writer, "data: {\"id\":\"c1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = fmt.Fprint(writer, "data: {\"id\":\"c1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"late\"},\"finish_reason\":null}]}\n\n")
	}))
	defer server.Close()
	provider := newOpenAIProvider(ResolvedProviderConfig{ProviderConfig: ProviderConfig{ID: "openai", BaseURL: server.URL}, Secret: "secret-value", Timeout: defaultProviderTimeout, RequireUsage: false}, 1<<20, 1<<20)
	stream, err := provider.Stream(context.Background(), ProviderRequest{Request: validCanonicalRequest(true), ProviderModel: "m"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if event, err := stream.Recv(); err != nil || event.Type != StreamFinish {
		t.Fatalf("first event=%#v err=%v", event, err)
	}
	if _, err := stream.Recv(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("post-finish error=%v", err)
	}
}

func TestUnknownFinishReasonIsNotReportedAsStop(t *testing.T) {
	if got := normalizeFinishReason("new_provider_reason"); got != "unknown" {
		t.Fatalf("finish reason = %q", got)
	}
	if got := normalizeFinishReason("refusal"); got != "refusal" {
		t.Fatalf("refusal finish reason = %q", got)
	}
}

func TestOpenAIUsageRequiresAllCanonicalCounters(t *testing.T) {
	prompt := int64(1)
	completion := int64(1)
	if _, err := openAIUsageValue(openAIUsage{PromptTokens: &prompt, CompletionTokens: &completion}); !errors.Is(err, ErrProtocol) {
		t.Fatalf("missing total-token error = %v", err)
	}
}

func TestOpenAIStreamRejectsUsageBeforeFinish(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(writer, "data: {\"id\":\"c1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"m\",\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":0,\"total_tokens\":4}}\n\n")
	}))
	defer server.Close()
	provider := newOpenAIProvider(ResolvedProviderConfig{ProviderConfig: ProviderConfig{ID: "openai", BaseURL: server.URL}, Secret: "secret-value", Timeout: defaultProviderTimeout, RequireUsage: true}, 1<<20, 1<<20)
	stream, err := provider.Stream(context.Background(), ProviderRequest{Request: validCanonicalRequest(true), ProviderModel: "m"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if _, err := stream.Recv(); !errors.Is(err, ErrProtocol) {
		t.Fatalf("early usage error = %v", err)
	}
}

func TestOpenAIStreamClassifiesRequestSpecificError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(writer, "data: {\"error\":{\"type\":\"invalid_request_error\",\"code\":\"invalid_request\",\"message\":\"raw provider detail\"}}\n\n")
	}))
	defer server.Close()
	provider := newOpenAIProvider(ResolvedProviderConfig{ProviderConfig: ProviderConfig{ID: "openai", BaseURL: server.URL}, Secret: "secret-value", Timeout: defaultProviderTimeout, RequireUsage: true}, 1<<20, 1<<20)
	stream, err := provider.Stream(context.Background(), ProviderRequest{Request: validCanonicalRequest(true), ProviderModel: "m"})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if _, err := stream.Recv(); !errors.Is(err, ErrProvider) || errors.Is(err, ErrUnavailable) || strings.Contains(err.Error(), "raw provider detail") {
		t.Fatalf("request-specific stream error = %v", err)
	}
}
