package aigateway

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testWorkloadKey = "workload-secret-key"

func testRuntimeConfig() RuntimeConfig {
	keyDigest := sha256.Sum256([]byte(testWorkloadKey))
	metricsDigest := sha256.Sum256([]byte("metrics-secret-key"))
	return RuntimeConfig{
		Environment:          "test",
		ListenAddr:           ":0",
		RequestTimeout:       5 * time.Second,
		ShutdownTimeout:      time.Second,
		MaxRequestBytes:      1 << 20,
		MaxProviderBodyBytes: 1 << 20,
		MaxSSEEventBytes:     1 << 20,
		MetricsDigest:        &metricsDigest,
		CircuitBreaker:       CircuitBreakerConfig{FailureThreshold: 2, OpenDurationMS: 1000},
		Workloads: []ConfiguredWorkload{{
			Workload: Workload{
				ID: "workload-a", TenantID: "tenant-a", AllowedModels: map[string]struct{}{"governed-chat": {}},
				RequestsPerMinute: 1000, TokensPerMinute: 10_000_000, CostMicroUSDPerMinute: 10_000_000, MaxConcurrent: 32,
			},
			KeyDigest: keyDigest,
		}},
		Models: []ModelConfig{{Alias: "governed-chat", Routes: []RouteConfig{
			{ID: "route-primary", ProviderID: "primary", Model: "provider-primary", Weight: 1, InputMicroUSDPerMillionTokens: 1000, OutputMicroUSDPerMillionTokens: 2000},
			{ID: "route-secondary", ProviderID: "secondary", Model: "provider-secondary", Weight: 1, InputMicroUSDPerMillionTokens: 1200, OutputMicroUSDPerMillionTokens: 2200},
		}}},
	}
}

func newTestHTTPHandler(t *testing.T, primary, secondary *fakeProvider, logger *slog.Logger) (*Gateway, http.Handler) {
	t.Helper()
	config := testRuntimeConfig()
	gateway, err := newGatewayWithProviders(config, map[string]*providerRuntime{
		"primary":   {provider: primary},
		"secondary": {provider: secondary},
	}, logger)
	if err != nil {
		t.Fatal(err)
	}
	return gateway, NewHTTPHandler(gateway, logger)
}

func authorizedRequest(method, path, payload string) *http.Request {
	request := httptest.NewRequest(method, path, strings.NewReader(payload))
	request.Header.Set("Authorization", "Bearer "+testWorkloadKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Request-ID", "req_00112233445566778899aabbccddeeff")
	return request
}

func TestHTTPCompleteFallsBackBeforeReturningOutput(t *testing.T) {
	primary := &fakeProvider{id: "primary", completeErr: ErrUnavailable}
	secondary := &fakeProvider{id: "secondary", response: Response{
		ID: "provider-id", CreatedAt: time.Unix(1, 0).UTC(), Text: "secondary answer", FinishReason: "stop",
		Usage: Usage{InputTokens: 10, OutputTokens: 3},
	}}
	_, handler := newTestHTTPHandler(t, primary, secondary, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authorizedRequest(http.MethodPost, "/v1/chat/completions", `{"model":"governed-chat","messages":[{"role":"user","content":"hello"}],"max_completion_tokens":64}`))
	if recorder.Code != http.StatusOK || recorder.Header().Get("X-ClearSight-Route") != "route-secondary" || !strings.Contains(recorder.Body.String(), "secondary answer") {
		t.Fatalf("status=%d route=%q body=%s", recorder.Code, recorder.Header().Get("X-ClearSight-Route"), recorder.Body.String())
	}
	primaryComplete, _ := primary.counts()
	secondaryComplete, _ := secondary.counts()
	if primaryComplete != 1 || secondaryComplete != 1 {
		t.Fatalf("complete calls primary=%d secondary=%d", primaryComplete, secondaryComplete)
	}
}

func TestHTTPCompleteDoesNotFallbackOnProviderRequestRejection(t *testing.T) {
	primary := &fakeProvider{id: "primary", completeErr: ErrProvider}
	secondary := &fakeProvider{id: "secondary", response: Response{Text: "must not run", FinishReason: "stop", Usage: Usage{InputTokens: 1, OutputTokens: 1}}}
	_, handler := newTestHTTPHandler(t, primary, secondary, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authorizedRequest(http.MethodPost, "/v1/chat/completions", `{"model":"governed-chat","messages":[{"role":"user","content":"hello"}],"max_completion_tokens":64}`))
	if recorder.Code != http.StatusBadGateway || !strings.Contains(recorder.Body.String(), `"code":"provider_error"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	_, secondaryStreams := secondary.counts()
	secondaryComplete, _ := secondary.counts()
	if secondaryComplete != 0 || secondaryStreams != 0 {
		t.Fatalf("secondary provider unexpectedly called: complete=%d stream=%d", secondaryComplete, secondaryStreams)
	}
}

func TestHTTPStreamFallsBackOnlyBeforeDownstreamCommit(t *testing.T) {
	t.Run("before first event", func(t *testing.T) {
		primary := &fakeProvider{id: "primary", streamFactory: func() ProviderStream {
			return &fakeStream{errors: map[int]error{0: ErrUnavailable}}
		}}
		secondary := &fakeProvider{id: "secondary", streamFactory: func() ProviderStream {
			usage := Usage{InputTokens: 4, OutputTokens: 1}
			return &fakeStream{events: []StreamEvent{
				{Type: StreamTextDelta, Text: "fallback text"},
				{Type: StreamFinish, FinishReason: "stop"},
				{Type: StreamUsage, Usage: &usage},
				{Type: StreamDone},
			}}
		}}
		_, handler := newTestHTTPHandler(t, primary, secondary, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, authorizedRequest(http.MethodPost, "/v1/chat/completions", `{"model":"governed-chat","messages":[{"role":"user","content":"hello"}],"max_completion_tokens":64,"stream":true,"stream_options":{"include_usage":true}}`))
		if recorder.Code != http.StatusOK || recorder.Header().Get("X-ClearSight-Route") != "route-secondary" || !strings.Contains(recorder.Body.String(), "fallback text") || !strings.Contains(recorder.Body.String(), "[DONE]") {
			t.Fatalf("status=%d route=%q body=%s", recorder.Code, recorder.Header().Get("X-ClearSight-Route"), recorder.Body.String())
		}
		_, primaryStreams := primary.counts()
		_, secondaryStreams := secondary.counts()
		if primaryStreams != 1 || secondaryStreams != 1 {
			t.Fatalf("stream calls primary=%d secondary=%d", primaryStreams, secondaryStreams)
		}
	})

	t.Run("after first event", func(t *testing.T) {
		primary := &fakeProvider{id: "primary", streamFactory: func() ProviderStream {
			return &fakeStream{events: []StreamEvent{{Type: StreamTextDelta, Text: "committed text"}}, errors: map[int]error{1: ErrUnavailable}}
		}}
		secondary := &fakeProvider{id: "secondary", streamFactory: func() ProviderStream {
			return &fakeStream{events: []StreamEvent{{Type: StreamTextDelta, Text: "must not run"}, {Type: StreamDone}}}
		}}
		_, handler := newTestHTTPHandler(t, primary, secondary, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, authorizedRequest(http.MethodPost, "/v1/chat/completions", `{"model":"governed-chat","messages":[{"role":"user","content":"hello"}],"max_completion_tokens":64,"stream":true}`))
		body := recorder.Body.String()
		if recorder.Code != http.StatusOK || recorder.Header().Get("X-ClearSight-Route") != "route-primary" || !strings.Contains(body, "committed text") || !strings.Contains(body, `"code":"provider_unavailable"`) || strings.Contains(body, "must not run") {
			t.Fatalf("status=%d route=%q body=%s", recorder.Code, recorder.Header().Get("X-ClearSight-Route"), body)
		}
		_, secondaryStreams := secondary.counts()
		if secondaryStreams != 0 {
			t.Fatalf("secondary stream calls=%d", secondaryStreams)
		}
	})
}

func TestResponsesStreamingEmitsTruthfulLifecycle(t *testing.T) {
	usage := Usage{InputTokens: 7, OutputTokens: 2}
	primary := &fakeProvider{id: "primary", streamFactory: func() ProviderStream {
		return &fakeStream{events: []StreamEvent{
			{Type: StreamTextDelta, Text: "hello"},
			{Type: StreamFinish, FinishReason: "stop"},
			{Type: StreamUsage, Usage: &usage},
			{Type: StreamDone},
		}}
	}}
	secondary := &fakeProvider{id: "secondary"}
	_, handler := newTestHTTPHandler(t, primary, secondary, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authorizedRequest(http.MethodPost, "/v1/responses", `{"model":"governed-chat","input":"hello","max_output_tokens":64,"stream":true}`))
	body := recorder.Body.String()
	for _, event := range []string{"event: response.created", "event: response.in_progress", "event: response.output_item.added", "event: response.output_text.delta", "event: response.output_text.done", "event: response.completed"} {
		if !strings.Contains(body, event) {
			t.Fatalf("stream missing %q: %s", event, body)
		}
	}
	if strings.Contains(body, "response.failed") {
		t.Fatalf("successful stream reported failure: %s", body)
	}
}

func TestGatewayLogsAndMetricsExcludeRawContent(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	primary := &fakeProvider{id: "primary", response: Response{
		Text: "completion-secret-value", FinishReason: "stop", Usage: Usage{InputTokens: 10, OutputTokens: 2},
	}}
	secondary := &fakeProvider{id: "secondary"}
	_, handler := newTestHTTPHandler(t, primary, secondary, logger)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authorizedRequest(http.MethodPost, "/v1/chat/completions", `{"model":"governed-chat","messages":[{"role":"user","content":"prompt-secret-value"}],"max_completion_tokens":64}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	for _, secret := range []string{"prompt-secret-value", "completion-secret-value", testWorkloadKey} {
		if strings.Contains(logs.String(), secret) {
			t.Fatalf("logs contain raw or credential material %q: %s", secret, logs.String())
		}
	}

	unknown := httptest.NewRecorder()
	handler.ServeHTTP(unknown, authorizedRequest(http.MethodPost, "/v1/chat/completions", `{"model":"attacker-controlled-alias","messages":[{"role":"user","content":"hello"}],"max_completion_tokens":64}`))
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown alias status=%d body=%s", unknown.Code, unknown.Body.String())
	}
	metricsRequest := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRequest.Header.Set("Authorization", "Bearer metrics-secret-key")
	metrics := httptest.NewRecorder()
	handler.ServeHTTP(metrics, metricsRequest)
	if metrics.Code != http.StatusOK || !strings.Contains(metrics.Body.String(), `model_alias="unknown"`) || strings.Contains(metrics.Body.String(), "attacker-controlled-alias") {
		t.Fatalf("metrics status=%d body=%s", metrics.Code, metrics.Body.String())
	}
}

func TestHTTPAuthenticationAndStrictDecoding(t *testing.T) {
	primary := &fakeProvider{id: "primary", response: Response{FinishReason: "stop", Usage: Usage{InputTokens: 1, OutputTokens: 1}}}
	secondary := &fakeProvider{id: "secondary"}
	_, handler := newTestHTTPHandler(t, primary, secondary, nil)

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"governed-chat"}`)))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d", unauthorized.Code)
	}

	unknownField := httptest.NewRecorder()
	handler.ServeHTTP(unknownField, authorizedRequest(http.MethodPost, "/v1/chat/completions", `{"model":"governed-chat","messages":[{"role":"user","content":"hello"}],"max_completion_tokens":64,"prompt":"must fail"}`))
	if unknownField.Code != http.StatusBadRequest {
		t.Fatalf("unknown-field status=%d body=%s", unknownField.Code, unknownField.Body.String())
	}

	metadata := httptest.NewRecorder()
	handler.ServeHTTP(metadata, authorizedRequest(http.MethodPost, "/v1/responses", `{"model":"governed-chat","input":"hello","max_output_tokens":64,"metadata":{"case":"discarded"}}`))
	if metadata.Code != http.StatusBadRequest || !strings.Contains(metadata.Body.String(), `"param":"metadata"`) {
		t.Fatalf("metadata status=%d body=%s", metadata.Code, metadata.Body.String())
	}
}

func TestGatewayCompleteHonorsCancellation(t *testing.T) {
	primary := &fakeProvider{id: "primary", completeErr: context.Canceled}
	secondary := &fakeProvider{id: "secondary", response: Response{Text: "must not run", FinishReason: "stop", Usage: Usage{InputTokens: 1, OutputTokens: 1}}}
	config := testRuntimeConfig()
	gateway, err := newGatewayWithProviders(config, map[string]*providerRuntime{
		"primary": {provider: primary}, "secondary": {provider: secondary},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	workload := config.Workloads[0].Workload
	request := validCanonicalRequest(false)
	request.ModelAlias = "governed-chat"
	_, _, err = gateway.Complete(context.Background(), workload, request)
	if !errors.Is(err, ErrCanceled) {
		t.Fatalf("cancellation error=%v", err)
	}
	secondaryComplete, _ := secondary.counts()
	if secondaryComplete != 0 {
		t.Fatalf("secondary provider called after cancellation: %d", secondaryComplete)
	}
}

func TestConfiguredDigestExample(t *testing.T) {
	// Keep the operator-facing SHA-256 bootstrap convention executable.
	digest := sha256.Sum256([]byte(testWorkloadKey))
	if len(hex.EncodeToString(digest[:])) != 64 {
		t.Fatal("workload digest is not SHA-256")
	}
}

func TestResponsesLengthStreamEndsIncomplete(t *testing.T) {
	usage := Usage{InputTokens: 7, OutputTokens: 64}
	primary := &fakeProvider{id: "primary", streamFactory: func() ProviderStream {
		return &fakeStream{events: []StreamEvent{
			{Type: StreamTextDelta, Text: "partial"},
			{Type: StreamFinish, FinishReason: "length"},
			{Type: StreamUsage, Usage: &usage},
			{Type: StreamDone},
		}}
	}}
	secondary := &fakeProvider{id: "secondary"}
	_, handler := newTestHTTPHandler(t, primary, secondary, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authorizedRequest(http.MethodPost, "/v1/responses", `{"model":"governed-chat","input":"hello","max_output_tokens":64,"stream":true}`))
	body := recorder.Body.String()
	if !strings.Contains(body, "event: response.incomplete") || strings.Contains(body, "event: response.completed") || !strings.Contains(body, `"reason":"max_output_tokens"`) {
		t.Fatalf("length stream lifecycle = %s", body)
	}
}

func TestResponsesContentFilterStreamEndsIncomplete(t *testing.T) {
	usage := Usage{InputTokens: 7, OutputTokens: 2}
	primary := &fakeProvider{id: "primary", streamFactory: func() ProviderStream {
		return &fakeStream{events: []StreamEvent{
			{Type: StreamTextDelta, Text: "partial"},
			{Type: StreamFinish, FinishReason: "content_filter"},
			{Type: StreamUsage, Usage: &usage},
			{Type: StreamDone},
		}}
	}}
	secondary := &fakeProvider{id: "secondary"}
	_, handler := newTestHTTPHandler(t, primary, secondary, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authorizedRequest(http.MethodPost, "/v1/responses", `{"model":"governed-chat","input":"hello","max_output_tokens":64,"stream":true}`))
	body := recorder.Body.String()
	if !strings.Contains(body, "event: response.incomplete") || strings.Contains(body, "event: response.completed") || !strings.Contains(body, `"reason":"content_filter"`) {
		t.Fatalf("content-filter stream lifecycle = %s", body)
	}
}

func TestResponsesContentFilterEndsIncomplete(t *testing.T) {
	usage := Usage{InputTokens: 7, OutputTokens: 2}
	primary := &fakeProvider{id: "primary", response: Response{
		Text: "partial", FinishReason: "content_filter", Usage: usage,
	}}
	secondary := &fakeProvider{id: "secondary"}
	_, handler := newTestHTTPHandler(t, primary, secondary, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authorizedRequest(http.MethodPost, "/v1/responses", `{"model":"governed-chat","input":"hello","max_output_tokens":64}`))
	body := recorder.Body.String()
	if recorder.Code != http.StatusOK || !strings.Contains(body, `"status":"incomplete"`) || !strings.Contains(body, `"reason":"content_filter"`) {
		t.Fatalf("content-filter response status=%d body=%s", recorder.Code, body)
	}
}

func TestResponsesRefusalRemainsCompleted(t *testing.T) {
	primary := &fakeProvider{id: "primary", response: Response{
		Refusal: "I cannot help with that.", FinishReason: "refusal", Usage: Usage{InputTokens: 7, OutputTokens: 2},
	}}
	secondary := &fakeProvider{id: "secondary"}
	_, handler := newTestHTTPHandler(t, primary, secondary, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authorizedRequest(http.MethodPost, "/v1/responses", `{"model":"governed-chat","input":"hello","max_output_tokens":64}`))
	body := recorder.Body.String()
	if recorder.Code != http.StatusOK || !strings.Contains(body, `"status":"completed"`) || !strings.Contains(body, `"type":"refusal"`) || strings.Contains(body, `"incomplete_details":{"reason"`) {
		t.Fatalf("refusal response status=%d body=%s", recorder.Code, body)
	}
}

func TestHTTPRejectsMissingJSONContentTypeBeforeProviderCall(t *testing.T) {
	primary := &fakeProvider{id: "primary", response: Response{Text: "must not run", FinishReason: "stop"}}
	secondary := &fakeProvider{id: "secondary"}
	_, handler := newTestHTTPHandler(t, primary, secondary, nil)
	request := authorizedRequest(http.MethodPost, "/v1/chat/completions", `{"model":"governed-chat","messages":[{"role":"user","content":"hello"}]}`)
	request.Header.Del("Content-Type")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"param":"Content-Type"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	complete, stream := primary.counts()
	if complete != 0 || stream != 0 {
		t.Fatalf("provider called before content-type validation: complete=%d stream=%d", complete, stream)
	}
}

func TestRecoveryDoesNotAppendJSONAfterResponseCommit(t *testing.T) {
	var logs bytes.Buffer
	handler := (&HTTPHandler{logger: slog.New(slog.NewJSONHandler(&logs, nil))}).recover(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
		panic("sensitive panic payload")
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if recorder.Code != http.StatusNoContent || recorder.Body.Len() != 0 {
		t.Fatalf("committed response was corrupted: status=%d body=%q", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(logs.String(), "sensitive panic payload") {
		t.Fatalf("panic payload leaked to logs: %s", logs.String())
	}
}
