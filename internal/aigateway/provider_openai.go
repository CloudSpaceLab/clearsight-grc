package aigateway

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

type openAIProvider struct {
	id               string
	baseURL          string
	secret           string
	timeout          time.Duration
	maxBodyBytes     int64
	maxSSEEventBytes int64
	requireUsage     bool
	client           *http.Client
}

func newOpenAIProvider(config ResolvedProviderConfig, maxBodyBytes, maxSSEEventBytes int64) Provider {
	return &openAIProvider{
		id: config.ID, baseURL: config.BaseURL, secret: config.Secret, timeout: config.Timeout,
		maxBodyBytes: maxBodyBytes, maxSSEEventBytes: maxSSEEventBytes, requireUsage: config.RequireUsage,
		client: newProviderHTTPClient(config.Timeout),
	}
}

func (p *openAIProvider) ID() string { return p.id }

func (p *openAIProvider) Complete(ctx context.Context, request ProviderRequest) (Response, error) {
	payload, err := p.requestPayload(request, false)
	if err != nil {
		return Response{}, err
	}
	response, cancel, err := providerRequest(ctx, p.client, p.timeout, http.MethodPost, p.baseURL+"/v1/chat/completions", payload, map[string]string{"Authorization": "Bearer " + p.secret})
	if err != nil {
		return Response{}, err
	}
	defer cancel()
	body, err := readProviderBody(response, p.maxBodyBytes)
	if err != nil {
		return Response{}, err
	}
	var decoded openAIChatResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return Response{}, withCause(ErrProtocol, err)
	}
	return validateOpenAIResponse(decoded, p.requireUsage)
}

func (p *openAIProvider) Stream(ctx context.Context, request ProviderRequest) (ProviderStream, error) {
	payload, err := p.requestPayload(request, true)
	if err != nil {
		return nil, err
	}
	response, cancel, err := providerRequest(ctx, p.client, p.timeout, http.MethodPost, p.baseURL+"/v1/chat/completions", payload, map[string]string{
		"Authorization": "Bearer " + p.secret,
		"Accept":        "text/event-stream",
	})
	if err != nil {
		return nil, err
	}
	if err := requireEventStream(response); err != nil {
		cancel()
		return nil, err
	}
	return &openAIStream{body: response.Body, cancel: cancel, reader: newSSEReader(response.Body, p.maxSSEEventBytes), requireUsage: p.requireUsage, toolStates: make(map[int]*openAIToolState), toolIDs: make(map[string]int)}, nil
}

func (p *openAIProvider) requestPayload(request ProviderRequest, stream bool) ([]byte, error) {
	messages := make([]openAIRequestMessage, 0, len(request.Messages))
	for _, message := range request.Messages {
		translated := openAIRequestMessage{Role: string(message.Role), Content: message.Text, ToolCallID: message.ToolCallID}
		for _, call := range message.ToolCalls {
			translated.ToolCalls = append(translated.ToolCalls, openAIRequestToolCall{ID: call.ID, Type: "function", Function: openAIFunctionCall{Name: call.Name, Arguments: call.Arguments}})
		}
		messages = append(messages, translated)
	}
	tools := make([]openAIRequestTool, 0, len(request.Tools))
	for _, tool := range request.Tools {
		tools = append(tools, openAIRequestTool{Type: "function", Function: openAIRequestFunction{Name: tool.Name, Description: tool.Description, Parameters: tool.Parameters, Strict: tool.Strict}})
	}
	payload := openAIChatRequest{
		Model: request.ProviderModel, Messages: messages, Tools: tools, MaxCompletionTokens: request.MaxOutputTokens,
		Temperature: request.Temperature, TopP: request.TopP, Stop: request.Stop, Stream: stream,
	}
	if len(payload.Tools) > 0 {
		payload.ToolChoice = openAIToolChoice(request.ToolChoice)
	}
	if stream {
		payload.StreamOptions = &openAIStreamOptions{IncludeUsage: true}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, withCause(ErrInternal, err)
	}
	return encoded, nil
}

func openAIToolChoice(choice ToolChoice) any {
	switch choice.Mode {
	case ToolChoiceNone:
		return "none"
	case ToolChoiceRequired:
		return "required"
	case ToolChoiceNamed:
		return map[string]any{"type": "function", "function": map[string]string{"name": choice.Name}}
	default:
		return "auto"
	}
}

type openAIChatRequest struct {
	Model               string                 `json:"model"`
	Messages            []openAIRequestMessage `json:"messages"`
	Tools               []openAIRequestTool    `json:"tools,omitempty"`
	ToolChoice          any                    `json:"tool_choice,omitempty"`
	MaxCompletionTokens int64                  `json:"max_completion_tokens"`
	Temperature         *float64               `json:"temperature,omitempty"`
	TopP                *float64               `json:"top_p,omitempty"`
	Stop                []string               `json:"stop,omitempty"`
	Stream              bool                   `json:"stream"`
	StreamOptions       *openAIStreamOptions   `json:"stream_options,omitempty"`
}

type openAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type openAIRequestMessage struct {
	Role       string                  `json:"role"`
	Content    string                  `json:"content,omitempty"`
	ToolCallID string                  `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIRequestToolCall `json:"tool_calls,omitempty"`
}

type openAIRequestTool struct {
	Type     string                `json:"type"`
	Function openAIRequestFunction `json:"function"`
}

type openAIRequestFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
	Strict      bool            `json:"strict,omitempty"`
}

type openAIRequestToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIUsage struct {
	PromptTokens        *int64 `json:"prompt_tokens"`
	CompletionTokens    *int64 `json:"completion_tokens"`
	TotalTokens         *int64 `json:"total_tokens"`
	PromptTokensDetails *struct {
		CachedTokens int64 `json:"cached_tokens"`
	} `json:"prompt_tokens_details,omitempty"`
}

type openAIError struct {
	Code string `json:"code"`
	Type string `json:"type"`
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role      string                  `json:"role"`
			Content   string                  `json:"content"`
			Refusal   string                  `json:"refusal"`
			ToolCalls []openAIRequestToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *openAIUsage `json:"usage"`
	Error *openAIError `json:"error,omitempty"`
}

func validateOpenAIResponse(decoded openAIChatResponse, requireUsage bool) (Response, error) {
	if decoded.Error != nil {
		return Response{}, classifyOpenAIError(*decoded.Error)
	}
	if decoded.Object != "chat.completion" || decoded.ID == "" || len(decoded.ID) > 256 || decoded.Created <= 0 || decoded.Model == "" || len(decoded.Model) > 256 || len(decoded.Choices) != 1 || decoded.Choices[0].Index != 0 || decoded.Choices[0].Message.Role != "assistant" || decoded.Choices[0].FinishReason == "" {
		return Response{}, ErrProtocol
	}
	if requireUsage && decoded.Usage == nil {
		return Response{}, ErrProtocol
	}
	choice := decoded.Choices[0]
	if len(choice.Message.Content)+len(choice.Message.Refusal) > MaxTextBytes {
		return Response{}, ErrProtocol
	}
	result := Response{ID: decoded.ID, CreatedAt: time.Unix(decoded.Created, 0).UTC(), Text: choice.Message.Content, Refusal: choice.Message.Refusal, FinishReason: normalizeFinishReason(choice.FinishReason)}
	if len(choice.Message.ToolCalls) > MaxTools {
		return Response{}, ErrProtocol
	}
	toolIDs := make(map[string]struct{}, len(choice.Message.ToolCalls))
	for _, call := range choice.Message.ToolCalls {
		arguments := strings.TrimSpace(call.Function.Arguments)
		if call.Type != "function" || !validIdentifier(call.ID) || !validToolName(call.Function.Name) || len(arguments) > MaxToolArgumentsBytes || !strings.HasPrefix(arguments, "{") || !json.Valid([]byte(arguments)) {
			return Response{}, ErrProtocol
		}
		if _, exists := toolIDs[call.ID]; exists {
			return Response{}, ErrProtocol
		}
		toolIDs[call.ID] = struct{}{}
		result.ToolCalls = append(result.ToolCalls, ToolCall{ID: call.ID, Name: call.Function.Name, Arguments: arguments})
	}
	if decoded.Usage != nil {
		usage, err := openAIUsageValue(*decoded.Usage)
		if err != nil {
			return Response{}, err
		}
		result.Usage = usage
	}
	return result, nil
}

func openAIUsageValue(value openAIUsage) (Usage, error) {
	if value.PromptTokens == nil || value.CompletionTokens == nil || value.TotalTokens == nil {
		return Usage{}, ErrProtocol
	}
	promptTokens := *value.PromptTokens
	completionTokens := *value.CompletionTokens
	totalTokens := *value.TotalTokens
	if promptTokens < 0 || completionTokens < 0 || totalTokens < 0 || promptTokens > math.MaxInt64-completionTokens {
		return Usage{}, ErrProtocol
	}
	total := promptTokens + completionTokens
	if totalTokens != total {
		return Usage{}, ErrProtocol
	}
	cached := int64(0)
	if value.PromptTokensDetails != nil {
		cached = value.PromptTokensDetails.CachedTokens
		if cached < 0 || cached > promptTokens {
			return Usage{}, ErrProtocol
		}
	}
	return Usage{InputTokens: promptTokens, CachedInputTokens: cached, OutputTokens: completionTokens}, nil
}

type openAIChatChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role      string `json:"role"`
			Content   string `json:"content"`
			Refusal   string `json:"refusal"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Type     string `json:"type"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Usage *openAIUsage `json:"usage"`
	Error *openAIError `json:"error,omitempty"`
}

type openAIToolState struct {
	id        string
	name      string
	arguments strings.Builder
	closed    bool
}

type openAIStream struct {
	body            io.ReadCloser
	cancel          context.CancelFunc
	reader          *sseReader
	requireUsage    bool
	pending         []StreamEvent
	providerID      string
	providerModel   string
	providerCreated int64
	seenChunk       bool
	seenFinish      bool
	seenUsage       bool
	seenDone        bool
	outputBytes     int
	toolStates      map[int]*openAIToolState
	toolIDs         map[string]int
}

func (s *openAIStream) Close() error {
	s.cancel()
	return s.body.Close()
}

func (s *openAIStream) Recv() (StreamEvent, error) {
	for {
		if len(s.pending) > 0 {
			event := s.pending[0]
			s.pending = s.pending[1:]
			return event, nil
		}
		if s.seenDone {
			return StreamEvent{}, io.EOF
		}
		event, err := s.reader.next()
		if err != nil {
			if err == io.EOF {
				return StreamEvent{}, ErrStream
			}
			return StreamEvent{}, err
		}
		if event.Data == "[DONE]" {
			if !s.seenChunk || !s.seenFinish || (s.requireUsage && !s.seenUsage) {
				return StreamEvent{}, ErrProtocol
			}
			for _, state := range s.toolStates {
				arguments := strings.TrimSpace(state.arguments.String())
				if !validIdentifier(state.id) || !validToolName(state.name) || arguments == "" || !strings.HasPrefix(arguments, "{") || !json.Valid([]byte(arguments)) {
					return StreamEvent{}, ErrProtocol
				}
			}
			s.seenDone = true
			return StreamEvent{Type: StreamDone}, nil
		}
		var chunk openAIChatChunk
		if err := json.Unmarshal([]byte(event.Data), &chunk); err != nil {
			return StreamEvent{}, withCause(ErrProtocol, err)
		}
		if chunk.Error != nil {
			return StreamEvent{}, classifyOpenAIError(*chunk.Error)
		}
		if chunk.Object != "chat.completion.chunk" || chunk.ID == "" || len(chunk.ID) > 256 || chunk.Created <= 0 || chunk.Model == "" || len(chunk.Model) > 256 {
			return StreamEvent{}, ErrProtocol
		}
		if !s.seenChunk {
			s.seenChunk = true
			s.providerID = chunk.ID
			s.providerModel = chunk.Model
			s.providerCreated = chunk.Created
		} else if chunk.ID != s.providerID || chunk.Model != s.providerModel || chunk.Created != s.providerCreated {
			return StreamEvent{}, ErrProtocol
		}
		if len(chunk.Choices) == 0 {
			if !s.seenFinish || chunk.Usage == nil || s.seenUsage {
				return StreamEvent{}, ErrProtocol
			}
			usage, err := openAIUsageValue(*chunk.Usage)
			if err != nil {
				return StreamEvent{}, err
			}
			s.seenUsage = true
			s.pending = append(s.pending, StreamEvent{Type: StreamUsage, Usage: &usage})
			continue
		}
		if s.seenFinish {
			return StreamEvent{}, ErrProtocol
		}
		if len(chunk.Choices) != 1 || chunk.Choices[0].Index != 0 {
			return StreamEvent{}, ErrProtocol
		}
		choice := chunk.Choices[0]
		if choice.Delta.Role != "" && choice.Delta.Role != "assistant" {
			return StreamEvent{}, ErrProtocol
		}
		if len(choice.Delta.Content)+len(choice.Delta.Refusal) > MaxTextBytes-s.outputBytes {
			return StreamEvent{}, ErrProtocol
		}
		s.outputBytes += len(choice.Delta.Content) + len(choice.Delta.Refusal)
		if choice.Delta.Content != "" {
			s.pending = append(s.pending, StreamEvent{Type: StreamTextDelta, Text: choice.Delta.Content})
		}
		if choice.Delta.Refusal != "" {
			s.pending = append(s.pending, StreamEvent{Type: StreamRefusalDelta, Text: choice.Delta.Refusal})
		}
		for _, call := range choice.Delta.ToolCalls {
			if call.Index < 0 || call.Index >= MaxTools || call.Type != "" && call.Type != "function" {
				return StreamEvent{}, ErrProtocol
			}
			state := s.toolStates[call.Index]
			if state == nil {
				state = &openAIToolState{}
				s.toolStates[call.Index] = state
			}
			if call.ID != "" {
				if state.id != "" && state.id != call.ID {
					return StreamEvent{}, ErrProtocol
				}
				if prior, exists := s.toolIDs[call.ID]; exists && prior != call.Index {
					return StreamEvent{}, ErrProtocol
				}
				s.toolIDs[call.ID] = call.Index
				state.id = call.ID
			}
			if call.Function.Name != "" {
				if state.name != "" && state.name != call.Function.Name {
					return StreamEvent{}, ErrProtocol
				}
				state.name = call.Function.Name
			}
			if state.arguments.Len()+len(call.Function.Arguments) > MaxToolArgumentsBytes {
				return StreamEvent{}, ErrProtocol
			}
			state.arguments.WriteString(call.Function.Arguments)
			delta := &ToolCallDelta{Index: call.Index, ID: call.ID, Name: call.Function.Name, Arguments: call.Function.Arguments}
			s.pending = append(s.pending, StreamEvent{Type: StreamToolDelta, Tool: delta})
		}
		if choice.FinishReason != nil {
			if *choice.FinishReason == "" || s.seenFinish {
				return StreamEvent{}, ErrProtocol
			}
			s.seenFinish = true
			s.pending = append(s.pending, StreamEvent{Type: StreamFinish, FinishReason: normalizeFinishReason(*choice.FinishReason)})
		}
		if chunk.Usage != nil {
			if choice.FinishReason == nil || s.seenUsage {
				return StreamEvent{}, ErrProtocol
			}
			usage, err := openAIUsageValue(*chunk.Usage)
			if err != nil {
				return StreamEvent{}, err
			}
			s.seenUsage = true
			s.pending = append(s.pending, StreamEvent{Type: StreamUsage, Usage: &usage})
		}
	}
}

func classifyOpenAIError(value openAIError) *Error {
	code := strings.ToLower(strings.TrimSpace(value.Code + " " + value.Type))
	switch {
	case strings.Contains(code, "rate_limit") || strings.Contains(code, "rate limit"):
		return ErrProviderRate
	case strings.Contains(code, "server_error") || strings.Contains(code, "overload") || strings.Contains(code, "temporarily_unavailable"):
		return ErrUnavailable
	default:
		return ErrProvider
	}
}

func normalizeFinishReason(reason string) string {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "stop", "end_turn", "stop_sequence":
		return "stop"
	case "length", "max_tokens", "model_context_window_exceeded":
		return "length"
	case "tool_calls", "tool_use":
		return "tool_calls"
	case "content_filter":
		return "content_filter"
	case "refusal":
		return "refusal"
	default:
		return "unknown"
	}
}
