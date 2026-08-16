package aigateway

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCanonicalResponseValidationRejectsInvalidProviderSuccess(t *testing.T) {
	t.Parallel()

	valid := Response{
		CreatedAt:    time.Unix(1, 0).UTC(),
		Text:         "ok",
		FinishReason: "stop",
		Usage:        Usage{InputTokens: 3, OutputTokens: 1},
	}
	tests := []struct {
		name     string
		response Response
	}{
		{name: "unknown finish value", response: Response{FinishReason: "provider_new_reason"}},
		{name: "normalized unknown finish", response: Response{FinishReason: "unknown"}},
		{name: "invalid UTF-8", response: Response{Text: string([]byte{0xff}), FinishReason: "stop"}},
		{name: "duplicate tool IDs", response: Response{FinishReason: "tool_calls", ToolCalls: []ToolCall{
			{ID: "call_a", Name: "lookup", Arguments: `{}`},
			{ID: "call_a", Name: "lookup", Arguments: `{}`},
		}}},
		{name: "non-object arguments", response: Response{FinishReason: "tool_calls", ToolCalls: []ToolCall{
			{ID: "call_a", Name: "lookup", Arguments: `[]`},
		}}},
		{name: "cached tokens exceed input", response: Response{FinishReason: "stop", Usage: Usage{InputTokens: 1, CachedInputTokens: 2}}},
		{name: "tool finish without calls", response: Response{FinishReason: "tool_calls"}},
		{name: "calls without tool finish", response: Response{FinishReason: "stop", ToolCalls: []ToolCall{{ID: "call_a", Name: "lookup", Arguments: `{}`}}}},
	}
	if err := validateCanonicalResponse(valid); err != nil {
		t.Fatalf("valid response rejected: %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := validateCanonicalResponse(test.response); !errors.Is(err, ErrProtocol) {
				t.Fatalf("validation error = %v, want provider protocol error", err)
			}
		})
	}
}

func TestCanonicalStreamValidationRequiresTruthfulLifecycle(t *testing.T) {
	t.Parallel()

	usage := Usage{InputTokens: 3, OutputTokens: 1}
	tests := []struct {
		name         string
		requireUsage bool
		events       []StreamEvent
	}{
		{name: "done without finish", events: []StreamEvent{{Type: StreamDone}}},
		{name: "usage before finish", events: []StreamEvent{{Type: StreamUsage, Usage: &usage}}},
		{name: "content after finish", events: []StreamEvent{{Type: StreamFinish, FinishReason: "stop"}, {Type: StreamTextDelta, Text: "late"}}},
		{name: "required usage missing", requireUsage: true, events: []StreamEvent{{Type: StreamFinish, FinishReason: "stop"}, {Type: StreamDone}}},
		{name: "arguments before tool identity", events: []StreamEvent{{Type: StreamToolDelta, Tool: &ToolCallDelta{Index: 0, Arguments: `{}`}}}},
		{name: "tool finish without deltas", events: []StreamEvent{{Type: StreamFinish, FinishReason: "tool_calls"}, {Type: StreamDone}}},
		{name: "tool deltas with stop finish", events: []StreamEvent{
			{Type: StreamToolDelta, Tool: &ToolCallDelta{Index: 0, ID: "call_a", Name: "lookup", Arguments: `{}`}},
			{Type: StreamFinish, FinishReason: "stop"},
			{Type: StreamDone},
		}},
		{name: "invalid final tool arguments", events: []StreamEvent{
			{Type: StreamToolDelta, Tool: &ToolCallDelta{Index: 0, ID: "call_a", Name: "lookup", Arguments: `[]`}},
			{Type: StreamFinish, FinishReason: "tool_calls"},
			{Type: StreamDone},
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			validator := newCanonicalStreamValidator(test.requireUsage)
			var err error
			for _, event := range test.events {
				err = validator.accept(event)
				if err != nil {
					break
				}
			}
			if !errors.Is(err, ErrProtocol) {
				t.Fatalf("validation error = %v, want provider protocol error", err)
			}
		})
	}
}

func TestCompleteFallsBackWhenProviderReturnsMalformedSuccess(t *testing.T) {
	t.Parallel()

	config := testRuntimeConfig()
	primary := &fakeProvider{id: "primary", response: Response{Text: "invalid", FinishReason: ""}}
	secondary := &fakeProvider{id: "secondary", response: Response{
		CreatedAt: time.Unix(1, 0).UTC(), Text: "fallback", FinishReason: "stop", Usage: Usage{InputTokens: 2, OutputTokens: 1},
	}}
	gateway, err := newGatewayWithProviders(config, map[string]*providerRuntime{
		"primary": {provider: primary}, "secondary": {provider: secondary},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	workload := config.Workloads[0].Workload
	response, route, err := gateway.Complete(context.Background(), workload, Request{
		ID: "req_00112233445566778899aabbccddeeff", Protocol: ProtocolChat, ModelAlias: "governed-chat",
		Messages: []Message{{Role: RoleUser, Text: "hello"}}, MaxOutputTokens: 64,
	})
	if err != nil || route != "route-secondary" || response.Text != "fallback" {
		t.Fatalf("route=%q response=%#v err=%v", route, response, err)
	}
	primaryCalls, _ := primary.counts()
	secondaryCalls, _ := secondary.counts()
	if primaryCalls != 1 || secondaryCalls != 1 {
		t.Fatalf("complete calls primary=%d secondary=%d", primaryCalls, secondaryCalls)
	}
}

func TestStreamFallsBackWhenFirstProviderEventIsMalformed(t *testing.T) {
	t.Parallel()

	primary := &fakeProvider{id: "primary", streamFactory: func() ProviderStream {
		return &fakeStream{events: []StreamEvent{{Type: StreamDone}}}
	}}
	usage := Usage{InputTokens: 2, OutputTokens: 1}
	secondary := &fakeProvider{id: "secondary", streamFactory: func() ProviderStream {
		return &fakeStream{events: []StreamEvent{
			{Type: StreamTextDelta, Text: "fallback"},
			{Type: StreamFinish, FinishReason: "stop"},
			{Type: StreamUsage, Usage: &usage},
			{Type: StreamDone},
		}}
	}}
	_, handler := newTestHTTPHandler(t, primary, secondary, nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authorizedRequest(http.MethodPost, "/v1/chat/completions", `{"model":"governed-chat","messages":[{"role":"user","content":"hello"}],"max_completion_tokens":64,"stream":true,"stream_options":{"include_usage":true}}`))
	body := recorder.Body.String()
	if recorder.Code != http.StatusOK || recorder.Header().Get("X-ClearSight-Route") != "route-secondary" || !strings.Contains(body, "fallback") || !strings.Contains(body, "[DONE]") {
		t.Fatalf("status=%d route=%q body=%s", recorder.Code, recorder.Header().Get("X-ClearSight-Route"), body)
	}
	_, primaryStreams := primary.counts()
	_, secondaryStreams := secondary.counts()
	if primaryStreams != 1 || secondaryStreams != 1 {
		t.Fatalf("stream calls primary=%d secondary=%d", primaryStreams, secondaryStreams)
	}
}
