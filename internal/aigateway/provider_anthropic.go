package aigateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type anthropicProvider struct {
	id               string
	baseURL          string
	secret           string
	apiVersion       string
	timeout          time.Duration
	maxBodyBytes     int64
	maxSSEEventBytes int64
	requireUsage     bool
	client           *http.Client
}

func newAnthropicProvider(config ResolvedProviderConfig, maxBodyBytes, maxSSEEventBytes int64) Provider {
	version := strings.TrimSpace(config.APIVersion)
	if version == "" {
		version = "2023-06-01"
	}
	return &anthropicProvider{
		id: config.ID, baseURL: config.BaseURL, secret: config.Secret, apiVersion: version,
		timeout: config.Timeout, maxBodyBytes: maxBodyBytes, maxSSEEventBytes: maxSSEEventBytes,
		requireUsage: config.RequireUsage, client: newProviderHTTPClient(config.Timeout),
	}
}

func (p *anthropicProvider) ID() string { return p.id }

func (p *anthropicProvider) Complete(ctx context.Context, request ProviderRequest) (Response, error) {
	payload, err := p.requestPayload(request, false)
	if err != nil {
		return Response{}, err
	}
	response, cancel, err := providerRequest(ctx, p.client, p.timeout, http.MethodPost, p.baseURL+"/v1/messages", payload, map[string]string{
		"x-api-key":         p.secret,
		"anthropic-version": p.apiVersion,
	})
	if err != nil {
		return Response{}, err
	}
	defer cancel()
	body, err := readProviderBody(response, p.maxBodyBytes)
	if err != nil {
		return Response{}, err
	}
	var decoded anthropicResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return Response{}, withCause(ErrProtocol, err)
	}
	return validateAnthropicResponse(decoded, p.requireUsage)
}

func (p *anthropicProvider) Stream(ctx context.Context, request ProviderRequest) (ProviderStream, error) {
	payload, err := p.requestPayload(request, true)
	if err != nil {
		return nil, err
	}
	response, cancel, err := providerRequest(ctx, p.client, p.timeout, http.MethodPost, p.baseURL+"/v1/messages", payload, map[string]string{
		"x-api-key":         p.secret,
		"anthropic-version": p.apiVersion,
		"Accept":            "text/event-stream",
	})
	if err != nil {
		return nil, err
	}
	if err := requireEventStream(response); err != nil {
		cancel()
		return nil, err
	}
	return &anthropicStream{
		body: response.Body, cancel: cancel, reader: newSSEReader(response.Body, p.maxSSEEventBytes),
		requireUsage: p.requireUsage, blocks: make(map[int]*anthropicBlockState), toolIDs: make(map[string]int),
	}, nil
}

func (p *anthropicProvider) requestPayload(request ProviderRequest, stream bool) ([]byte, error) {
	var system strings.Builder
	messages := make([]anthropicRequestMessage, 0, len(request.Messages))
	appendMessage := func(role string, content ...any) {
		if len(content) == 0 {
			return
		}
		if len(messages) > 0 && messages[len(messages)-1].Role == role {
			messages[len(messages)-1].Content = append(messages[len(messages)-1].Content, content...)
			return
		}
		messages = append(messages, anthropicRequestMessage{Role: role, Content: append([]any(nil), content...)})
	}
	for _, message := range request.Messages {
		switch message.Role {
		case RoleSystem, RoleDeveloper:
			if system.Len() > 0 {
				system.WriteString("\n\n")
			}
			system.WriteString(message.Text)
		case RoleUser:
			appendMessage("user", map[string]any{"type": "text", "text": message.Text})
		case RoleAssistant:
			content := make([]any, 0, 1+len(message.ToolCalls))
			if message.Text != "" {
				content = append(content, map[string]any{"type": "text", "text": message.Text})
			}
			for _, call := range message.ToolCalls {
				var input any
				if err := json.Unmarshal([]byte(call.Arguments), &input); err != nil {
					return nil, withCause(ErrInvalidRequest, err)
				}
				content = append(content, map[string]any{"type": "tool_use", "id": call.ID, "name": call.Name, "input": input})
			}
			appendMessage("assistant", content...)
		case RoleTool:
			appendMessage("user", map[string]any{"type": "tool_result", "tool_use_id": message.ToolCallID, "content": message.Text})
		}
	}
	tools := make([]anthropicTool, 0, len(request.Tools))
	for _, tool := range request.Tools {
		tools = append(tools, anthropicTool{Name: tool.Name, Description: tool.Description, InputSchema: tool.Parameters})
	}
	payload := anthropicRequest{
		Model: request.ProviderModel, MaxTokens: request.MaxOutputTokens, System: system.String(), Messages: messages,
		Tools: tools, Stream: stream, Temperature: request.Temperature, TopP: request.TopP, StopSequences: request.Stop,
	}
	if request.ToolChoice.Mode == ToolChoiceNone {
		payload.Tools = nil
	} else if len(payload.Tools) > 0 {
		payload.ToolChoice = anthropicToolChoice(request.ToolChoice)
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, withCause(ErrInternal, err)
	}
	return encoded, nil
}

func anthropicToolChoice(choice ToolChoice) any {
	switch choice.Mode {
	case ToolChoiceRequired:
		return map[string]string{"type": "any"}
	case ToolChoiceNamed:
		return map[string]string{"type": "tool", "name": choice.Name}
	default:
		return map[string]string{"type": "auto"}
	}
}

type anthropicRequest struct {
	Model         string                    `json:"model"`
	MaxTokens     int64                     `json:"max_tokens"`
	System        string                    `json:"system,omitempty"`
	Messages      []anthropicRequestMessage `json:"messages"`
	Tools         []anthropicTool           `json:"tools,omitempty"`
	ToolChoice    any                       `json:"tool_choice,omitempty"`
	Temperature   *float64                  `json:"temperature,omitempty"`
	TopP          *float64                  `json:"top_p,omitempty"`
	StopSequences []string                  `json:"stop_sequences,omitempty"`
	Stream        bool                      `json:"stream"`
}

type anthropicRequestMessage struct {
	Role    string `json:"role"`
	Content []any  `json:"content"`
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicUsage struct {
	InputTokens              *int64 `json:"input_tokens"`
	CacheCreationInputTokens int64  `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64  `json:"cache_read_input_tokens"`
	OutputTokens             *int64 `json:"output_tokens"`
}

type anthropicResponse struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Role       string            `json:"role"`
	Model      string            `json:"model"`
	Content    []json.RawMessage `json:"content"`
	StopReason string            `json:"stop_reason"`
	Usage      *anthropicUsage   `json:"usage"`
	Error      *struct {
		Type string `json:"type"`
	} `json:"error,omitempty"`
}

func validateAnthropicResponse(decoded anthropicResponse, requireUsage bool) (Response, error) {
	if decoded.Error != nil || decoded.Type != "message" || decoded.Role != "assistant" || decoded.ID == "" || len(decoded.ID) > 256 || decoded.Model == "" || len(decoded.Model) > 256 || decoded.StopReason == "" {
		return Response{}, ErrProtocol
	}
	if requireUsage && decoded.Usage == nil {
		return Response{}, ErrProtocol
	}
	result := Response{ID: decoded.ID, CreatedAt: time.Now().UTC(), FinishReason: normalizeFinishReason(decoded.StopReason)}
	toolIDs := make(map[string]struct{})
	for _, raw := range decoded.Content {
		var header struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(raw, &header); err != nil {
			return Response{}, withCause(ErrProtocol, err)
		}
		switch header.Type {
		case "text":
			var block struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if err := json.Unmarshal(raw, &block); err != nil {
				return Response{}, withCause(ErrProtocol, err)
			}
			if len(block.Text) > MaxTextBytes-len(result.Text)-len(result.Refusal) {
				return Response{}, ErrProtocol
			}
			result.Text += block.Text
		case "tool_use":
			if len(result.ToolCalls) >= MaxTools {
				return Response{}, ErrProtocol
			}
			var block struct {
				Type  string          `json:"type"`
				ID    string          `json:"id"`
				Name  string          `json:"name"`
				Input json.RawMessage `json:"input"`
			}
			if err := json.Unmarshal(raw, &block); err != nil {
				return Response{}, withCause(ErrProtocol, err)
			}
			input := strings.TrimSpace(string(block.Input))
			if !validIdentifier(block.ID) || !validToolName(block.Name) || len(input) == 0 || len(input) > MaxToolArgumentsBytes || !strings.HasPrefix(input, "{") || !json.Valid([]byte(input)) {
				return Response{}, ErrProtocol
			}
			if _, exists := toolIDs[block.ID]; exists {
				return Response{}, ErrProtocol
			}
			toolIDs[block.ID] = struct{}{}
			result.ToolCalls = append(result.ToolCalls, ToolCall{ID: block.ID, Name: block.Name, Arguments: input})
		case "refusal":
			var block struct {
				Refusal string `json:"refusal"`
				Text    string `json:"text"`
			}
			if err := json.Unmarshal(raw, &block); err != nil {
				return Response{}, withCause(ErrProtocol, err)
			}
			text := firstNonEmpty(block.Refusal, block.Text)
			if len(text) > MaxTextBytes-len(result.Text)-len(result.Refusal) {
				return Response{}, ErrProtocol
			}
			result.Refusal += text
		case "thinking", "redacted_thinking", "fallback":
			// Internal reasoning and provider fallback markers are intentionally not exposed.
		default:
			return Response{}, ErrProtocol
		}
	}
	if decoded.Usage != nil {
		usage, err := anthropicUsageValue(*decoded.Usage)
		if err != nil {
			return Response{}, err
		}
		result.Usage = usage
	}
	return result, nil
}

func anthropicUsageValue(value anthropicUsage) (Usage, error) {
	if value.InputTokens == nil || value.OutputTokens == nil {
		return Usage{}, ErrProtocol
	}
	inputTokens := *value.InputTokens
	outputTokens := *value.OutputTokens
	if inputTokens < 0 || value.CacheCreationInputTokens < 0 || value.CacheReadInputTokens < 0 || outputTokens < 0 {
		return Usage{}, ErrProtocol
	}
	input := inputTokens
	for _, add := range []int64{value.CacheCreationInputTokens, value.CacheReadInputTokens} {
		if input > int64(^uint64(0)>>1)-add {
			return Usage{}, ErrProtocol
		}
		input += add
	}
	return Usage{InputTokens: input, CachedInputTokens: value.CacheReadInputTokens, OutputTokens: outputTokens}, nil
}

type anthropicDeltaUsage struct {
	OutputTokens *int64 `json:"output_tokens"`
}

type anthropicBlockState struct {
	kind      string
	id        string
	name      string
	arguments strings.Builder
	open      bool
}

type anthropicStream struct {
	body          io.ReadCloser
	cancel        context.CancelFunc
	reader        *sseReader
	requireUsage  bool
	pending       []StreamEvent
	started       bool
	providerID    string
	providerModel string
	nextIndex     int
	blocks        map[int]*anthropicBlockState
	toolIDs       map[string]int
	inputUsage    anthropicUsage
	outputTokens  int64
	seenFinish    bool
	seenUsage     bool
	seenDone      bool
	outputBytes   int
}

func (s *anthropicStream) Close() error {
	s.cancel()
	return s.body.Close()
}

func (s *anthropicStream) Recv() (StreamEvent, error) {
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
		var envelope struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(event.Data), &envelope); err != nil {
			return StreamEvent{}, withCause(ErrProtocol, err)
		}
		if event.Name != "" && envelope.Type != "" && event.Name != envelope.Type {
			return StreamEvent{}, ErrProtocol
		}
		switch envelope.Type {
		case "ping":
			continue
		case "message_start":
			if s.started {
				return StreamEvent{}, ErrProtocol
			}
			var value struct {
				Type    string `json:"type"`
				Message struct {
					ID      string            `json:"id"`
					Type    string            `json:"type"`
					Role    string            `json:"role"`
					Model   string            `json:"model"`
					Content []json.RawMessage `json:"content"`
					Usage   *anthropicUsage   `json:"usage"`
				} `json:"message"`
			}
			if err := json.Unmarshal([]byte(event.Data), &value); err != nil || value.Message.ID == "" || len(value.Message.ID) > 256 || value.Message.Type != "message" || value.Message.Role != "assistant" || value.Message.Model == "" || len(value.Message.Model) > 256 || len(value.Message.Content) != 0 || value.Message.Usage == nil {
				return StreamEvent{}, ErrProtocol
			}
			if _, err := anthropicUsageValue(*value.Message.Usage); err != nil {
				return StreamEvent{}, err
			}
			s.started = true
			s.providerID = value.Message.ID
			s.providerModel = value.Message.Model
			s.inputUsage = *value.Message.Usage
			s.outputTokens = *value.Message.Usage.OutputTokens
			continue
		case "content_block_start":
			if !s.started || s.seenFinish || s.hasOpenBlock() {
				return StreamEvent{}, ErrProtocol
			}
			var value struct {
				Index        int             `json:"index"`
				ContentBlock json.RawMessage `json:"content_block"`
			}
			if err := json.Unmarshal([]byte(event.Data), &value); err != nil || value.Index != s.nextIndex {
				return StreamEvent{}, ErrProtocol
			}
			var block struct {
				Type    string          `json:"type"`
				Text    string          `json:"text"`
				Refusal string          `json:"refusal"`
				ID      string          `json:"id"`
				Name    string          `json:"name"`
				Input   json.RawMessage `json:"input"`
			}
			if err := json.Unmarshal(value.ContentBlock, &block); err != nil {
				return StreamEvent{}, withCause(ErrProtocol, err)
			}
			state := &anthropicBlockState{kind: block.Type, id: block.ID, name: block.Name, open: true}
			s.blocks[value.Index] = state
			s.nextIndex++
			switch block.Type {
			case "text":
				if len(block.Text) > MaxTextBytes-s.outputBytes {
					return StreamEvent{}, ErrProtocol
				}
				s.outputBytes += len(block.Text)
				if block.Text != "" {
					s.pending = append(s.pending, StreamEvent{Type: StreamTextDelta, Text: block.Text})
				}
			case "tool_use":
				if !validIdentifier(block.ID) || !validToolName(block.Name) || len(s.toolIDs) >= MaxTools {
					return StreamEvent{}, ErrProtocol
				}
				if _, exists := s.toolIDs[block.ID]; exists {
					return StreamEvent{}, ErrProtocol
				}
				s.toolIDs[block.ID] = value.Index
				if len(block.Input) > 0 {
					var initial map[string]json.RawMessage
					if err := json.Unmarshal(block.Input, &initial); err != nil || len(initial) != 0 {
						return StreamEvent{}, ErrProtocol
					}
				}
				s.pending = append(s.pending, StreamEvent{Type: StreamToolDelta, Tool: &ToolCallDelta{Index: value.Index, ID: block.ID, Name: block.Name}})
			case "refusal":
				text := firstNonEmpty(block.Refusal, block.Text)
				if len(text) > MaxTextBytes-s.outputBytes {
					return StreamEvent{}, ErrProtocol
				}
				s.outputBytes += len(text)
				if text != "" {
					s.pending = append(s.pending, StreamEvent{Type: StreamRefusalDelta, Text: text})
				}
			case "thinking", "redacted_thinking", "fallback":
			default:
				return StreamEvent{}, ErrProtocol
			}
		case "content_block_delta":
			if !s.started || s.seenFinish {
				return StreamEvent{}, ErrProtocol
			}
			var value struct {
				Index int `json:"index"`
				Delta struct {
					Type        string `json:"type"`
					Text        string `json:"text"`
					Refusal     string `json:"refusal"`
					PartialJSON string `json:"partial_json"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(event.Data), &value); err != nil {
				return StreamEvent{}, withCause(ErrProtocol, err)
			}
			state := s.blocks[value.Index]
			if state == nil || !state.open {
				return StreamEvent{}, ErrProtocol
			}
			switch value.Delta.Type {
			case "text_delta":
				if state.kind != "text" {
					return StreamEvent{}, ErrProtocol
				}
				if len(value.Delta.Text) > MaxTextBytes-s.outputBytes {
					return StreamEvent{}, ErrProtocol
				}
				s.outputBytes += len(value.Delta.Text)
				if value.Delta.Text != "" {
					s.pending = append(s.pending, StreamEvent{Type: StreamTextDelta, Text: value.Delta.Text})
				}
			case "input_json_delta":
				if state.kind != "tool_use" || state.arguments.Len()+len(value.Delta.PartialJSON) > MaxToolArgumentsBytes {
					return StreamEvent{}, ErrProtocol
				}
				state.arguments.WriteString(value.Delta.PartialJSON)
				s.pending = append(s.pending, StreamEvent{Type: StreamToolDelta, Tool: &ToolCallDelta{Index: value.Index, Arguments: value.Delta.PartialJSON}})
			case "refusal_delta":
				if state.kind != "refusal" {
					return StreamEvent{}, ErrProtocol
				}
				text := firstNonEmpty(value.Delta.Refusal, value.Delta.Text)
				if len(text) > MaxTextBytes-s.outputBytes {
					return StreamEvent{}, ErrProtocol
				}
				s.outputBytes += len(text)
				if text != "" {
					s.pending = append(s.pending, StreamEvent{Type: StreamRefusalDelta, Text: text})
				}
			case "thinking_delta", "signature_delta":
				if state.kind != "thinking" && state.kind != "redacted_thinking" {
					return StreamEvent{}, ErrProtocol
				}
			default:
				if state.kind != "thinking" && state.kind != "redacted_thinking" && state.kind != "fallback" {
					return StreamEvent{}, ErrProtocol
				}
			}
		case "content_block_stop":
			var value struct {
				Index int `json:"index"`
			}
			if err := json.Unmarshal([]byte(event.Data), &value); err != nil {
				return StreamEvent{}, withCause(ErrProtocol, err)
			}
			state := s.blocks[value.Index]
			if state == nil || !state.open {
				return StreamEvent{}, ErrProtocol
			}
			state.open = false
			if state.kind == "tool_use" {
				arguments := strings.TrimSpace(state.arguments.String())
				if arguments == "" || !strings.HasPrefix(arguments, "{") || !json.Valid([]byte(arguments)) {
					return StreamEvent{}, ErrProtocol
				}
			}
		case "message_delta":
			if !s.started || s.seenFinish || s.hasOpenBlock() {
				return StreamEvent{}, ErrProtocol
			}
			var value struct {
				Delta struct {
					StopReason string `json:"stop_reason"`
				} `json:"delta"`
				Usage anthropicDeltaUsage `json:"usage"`
			}
			if err := json.Unmarshal([]byte(event.Data), &value); err != nil {
				return StreamEvent{}, withCause(ErrProtocol, err)
			}
			if value.Usage.OutputTokens == nil || *value.Usage.OutputTokens < s.outputTokens {
				return StreamEvent{}, ErrProtocol
			}
			s.outputTokens = *value.Usage.OutputTokens
			if value.Delta.StopReason == "" {
				continue
			}
			s.seenFinish = true
			s.pending = append(s.pending, StreamEvent{Type: StreamFinish, FinishReason: normalizeFinishReason(value.Delta.StopReason)})
			usageInput := s.inputUsage
			usageInput.OutputTokens = &s.outputTokens
			usage, err := anthropicUsageValue(usageInput)
			if err != nil {
				return StreamEvent{}, err
			}
			s.seenUsage = true
			s.pending = append(s.pending, StreamEvent{Type: StreamUsage, Usage: &usage})
		case "message_stop":
			if !s.started || !s.seenFinish || (s.requireUsage && !s.seenUsage) || s.hasOpenBlock() {
				return StreamEvent{}, ErrProtocol
			}
			s.seenDone = true
			return StreamEvent{Type: StreamDone}, nil
		case "error":
			var value struct {
				Error struct {
					Type string `json:"type"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(event.Data), &value); err != nil {
				return StreamEvent{}, withCause(ErrProtocol, err)
			}
			switch value.Error.Type {
			case "rate_limit_error":
				return StreamEvent{}, ErrProviderRate
			case "overloaded_error", "api_error":
				return StreamEvent{}, ErrUnavailable
			default:
				return StreamEvent{}, ErrProvider
			}
		default:
			// Anthropic documents that new top-level event types may be added. Ignore
			// unknown envelopes unless they mutate a supported content block.
			continue
		}
	}
}

func (s *anthropicStream) hasOpenBlock() bool {
	for _, block := range s.blocks {
		if block.open {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
