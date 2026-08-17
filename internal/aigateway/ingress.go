package aigateway

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
)

type textContent string

func (content *textContent) UnmarshalJSON(data []byte) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		*content = ""
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*content = textContent(text)
		return nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&parts); err != nil {
		return invalid("content", "Only text message content is supported in this gateway tranche.")
	}
	var builder strings.Builder
	for _, part := range parts {
		if part.Type != "text" && part.Type != "input_text" && part.Type != "output_text" {
			return invalid("content", "Only text message content is supported in this gateway tranche.")
		}
		builder.WriteString(part.Text)
	}
	*content = textContent(builder.String())
	return nil
}

type chatIngressRequest struct {
	Model               string               `json:"model"`
	Messages            []chatIngressMessage `json:"messages"`
	Tools               []chatIngressTool    `json:"tools,omitempty"`
	ToolChoice          json.RawMessage      `json:"tool_choice,omitempty"`
	MaxTokens           *int64               `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int64               `json:"max_completion_tokens,omitempty"`
	Temperature         *float64             `json:"temperature,omitempty"`
	TopP                *float64             `json:"top_p,omitempty"`
	Stop                json.RawMessage      `json:"stop,omitempty"`
	Stream              bool                 `json:"stream,omitempty"`
	StreamOptions       *struct {
		IncludeUsage bool `json:"include_usage"`
	} `json:"stream_options,omitempty"`
	N                 *int              `json:"n,omitempty"`
	ParallelToolCalls *bool             `json:"parallel_tool_calls,omitempty"`
	User              string            `json:"user,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	ResponseFormat    json.RawMessage   `json:"response_format,omitempty"`
	Logprobs          *bool             `json:"logprobs,omitempty"`
	Seed              *int64            `json:"seed,omitempty"`
}

type chatIngressMessage struct {
	Role       string                `json:"role"`
	Content    textContent           `json:"content"`
	Name       string                `json:"name,omitempty"`
	ToolCallID string                `json:"tool_call_id,omitempty"`
	ToolCalls  []chatIngressToolCall `json:"tool_calls,omitempty"`
}

type chatIngressToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatIngressTool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description,omitempty"`
		Parameters  json.RawMessage `json:"parameters"`
		Strict      bool            `json:"strict,omitempty"`
	} `json:"function"`
}

func decodeChatRequest(reader io.Reader, maxBytes int64, requestID string) (Request, error) {
	var input chatIngressRequest
	if err := decodeStrictJSON(reader, maxBytes, &input); err != nil {
		return Request{}, err
	}
	if input.N != nil && *input.N != 1 {
		return Request{}, invalid("n", "The gateway supports exactly one generated choice.")
	}
	if input.ParallelToolCalls != nil && !*input.ParallelToolCalls {
		return Request{}, invalid("parallel_tool_calls", "Disabling parallel function calls is not supported across both pilot providers.")
	}
	if (input.ResponseFormat != nil && string(input.ResponseFormat) != "null") || (input.Logprobs != nil && *input.Logprobs) || input.Seed != nil || strings.TrimSpace(input.User) != "" {
		return Request{}, invalid("", "The request includes a provider-specific option that is not supported by both pilot providers.")
	}
	maxOutput, err := chooseOutputLimit(input.MaxTokens, input.MaxCompletionTokens)
	if err != nil {
		return Request{}, err
	}
	request := Request{ID: requestID, Protocol: ProtocolChat, ModelAlias: strings.TrimSpace(input.Model), Metadata: cloneStringMap(input.Metadata), MaxOutputTokens: maxOutput, Temperature: input.Temperature, TopP: input.TopP, Stream: input.Stream}
	if input.StreamOptions != nil {
		request.IncludeStreamUsage = input.StreamOptions.IncludeUsage
	}
	request.Stop, err = parseStop(input.Stop)
	if err != nil {
		return Request{}, err
	}
	request.ToolChoice, err = parseToolChoice(input.ToolChoice, true)
	if err != nil {
		return Request{}, err
	}
	for _, message := range input.Messages {
		if strings.TrimSpace(message.Name) != "" {
			return Request{}, invalid("messages", "Named chat participants are not supported across both pilot providers.")
		}
		canonical := Message{Role: Role(message.Role), Text: string(message.Content), ToolCallID: message.ToolCallID}
		for _, call := range message.ToolCalls {
			if call.Type != "function" {
				return Request{}, invalid("messages", "Only function tool calls are supported.")
			}
			canonical.ToolCalls = append(canonical.ToolCalls, ToolCall{ID: call.ID, Name: call.Function.Name, Arguments: call.Function.Arguments})
		}
		request.Messages = append(request.Messages, canonical)
	}
	for _, tool := range input.Tools {
		if tool.Type != "function" {
			return Request{}, invalid("tools", "Only function tools are supported.")
		}
		request.Tools = append(request.Tools, ToolDefinition{Name: tool.Function.Name, Description: tool.Function.Description, Parameters: append(json.RawMessage(nil), tool.Function.Parameters...), Strict: tool.Function.Strict})
	}
	if err := ValidateRequest(request); err != nil {
		return Request{}, err
	}
	return request, nil
}

type responsesIngressRequest struct {
	Model             string            `json:"model"`
	Input             json.RawMessage   `json:"input"`
	Instructions      string            `json:"instructions,omitempty"`
	Tools             []responsesTool   `json:"tools,omitempty"`
	ToolChoice        json.RawMessage   `json:"tool_choice,omitempty"`
	MaxOutputTokens   *int64            `json:"max_output_tokens,omitempty"`
	Temperature       *float64          `json:"temperature,omitempty"`
	TopP              *float64          `json:"top_p,omitempty"`
	Stream            bool              `json:"stream,omitempty"`
	ParallelToolCalls *bool             `json:"parallel_tool_calls,omitempty"`
	Store             *bool             `json:"store,omitempty"`
	Background        *bool             `json:"background,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	Include           []string          `json:"include,omitempty"`
	Truncation        string            `json:"truncation,omitempty"`
	Text              json.RawMessage   `json:"text,omitempty"`
}

type responsesTool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
	Strict      bool            `json:"strict,omitempty"`
}

func decodeResponsesRequest(reader io.Reader, maxBytes int64, requestID string) (Request, error) {
	var input responsesIngressRequest
	if err := decodeStrictJSON(reader, maxBytes, &input); err != nil {
		return Request{}, err
	}
	if input.ParallelToolCalls != nil && !*input.ParallelToolCalls || input.Store != nil && *input.Store || input.Background != nil && *input.Background || len(input.Include) > 0 || input.Truncation != "" && input.Truncation != "disabled" || input.Text != nil && string(input.Text) != "null" {
		return Request{}, invalid("", "The request includes a stateful, background, multimodal or provider-specific option that is not supported in this stateless tranche.")
	}
	maxOutput := int64(1024)
	if input.MaxOutputTokens != nil {
		maxOutput = *input.MaxOutputTokens
	}
	request := Request{ID: requestID, Protocol: ProtocolResponses, ModelAlias: strings.TrimSpace(input.Model), Metadata: cloneStringMap(input.Metadata), MaxOutputTokens: maxOutput, Temperature: input.Temperature, TopP: input.TopP, Stream: input.Stream, IncludeStreamUsage: true}
	if input.Instructions != "" {
		request.Messages = append(request.Messages, Message{Role: RoleDeveloper, Text: input.Instructions})
	}
	messages, err := parseResponsesInput(input.Input)
	if err != nil {
		return Request{}, err
	}
	request.Messages = append(request.Messages, messages...)
	request.ToolChoice, err = parseToolChoice(input.ToolChoice, false)
	if err != nil {
		return Request{}, err
	}
	for _, tool := range input.Tools {
		if tool.Type != "function" {
			return Request{}, invalid("tools", "Only function tools are supported.")
		}
		request.Tools = append(request.Tools, ToolDefinition{Name: tool.Name, Description: tool.Description, Parameters: append(json.RawMessage(nil), tool.Parameters...), Strict: tool.Strict})
	}
	if err := ValidateRequest(request); err != nil {
		return Request{}, err
	}
	return request, nil
}

func chooseOutputLimit(legacy, current *int64) (int64, error) {
	if legacy != nil && current != nil && *legacy != *current {
		return 0, invalid("max_completion_tokens", "Specify one output-token limit or use the same value in both fields.")
	}
	if current != nil {
		return *current, nil
	}
	if legacy != nil {
		return *legacy, nil
	}
	return 1024, nil
}

func parseStop(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return []string{single}, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, invalid("stop", "Stop must be a string or an array of strings.")
	}
	return values, nil
}

func parseToolChoice(raw json.RawMessage, chat bool) (ToolChoice, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return ToolChoice{Mode: ToolChoiceAuto}, nil
	}
	var mode string
	if err := json.Unmarshal(raw, &mode); err == nil {
		switch mode {
		case ToolChoiceAuto, ToolChoiceNone, ToolChoiceRequired:
			return ToolChoice{Mode: mode}, nil
		default:
			return ToolChoice{}, invalid("tool_choice", "The tool-choice mode is not supported.")
		}
	}
	if chat {
		var value struct {
			Type     string `json:"type"`
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&value); err != nil || value.Type != "function" {
			return ToolChoice{}, invalid("tool_choice", "The named function tool choice is invalid.")
		}
		return ToolChoice{Mode: ToolChoiceNamed, Name: value.Function.Name}, nil
	}
	var value struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil || value.Type != "function" {
		return ToolChoice{}, invalid("tool_choice", "The named function tool choice is invalid.")
	}
	return ToolChoice{Mode: ToolChoiceNamed, Name: value.Name}, nil
}

func parseResponsesInput(raw json.RawMessage) ([]Message, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, invalid("input", "Responses input is required.")
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		return []Message{{Role: RoleUser, Text: text}}, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, invalid("input", "Responses input must be text or an array of supported input items.")
	}
	messages := make([]Message, 0, len(items))
	appendAssistantCall := func(call ToolCall) {
		if len(messages) > 0 && messages[len(messages)-1].Role == RoleAssistant && messages[len(messages)-1].Text == "" {
			messages[len(messages)-1].ToolCalls = append(messages[len(messages)-1].ToolCalls, call)
			return
		}
		messages = append(messages, Message{Role: RoleAssistant, ToolCalls: []ToolCall{call}})
	}
	for _, item := range items {
		var header struct {
			Type string `json:"type"`
			Role string `json:"role"`
		}
		if err := json.Unmarshal(item, &header); err != nil {
			return nil, invalid("input", "A Responses input item is invalid.")
		}
		switch header.Type {
		case "", "message":
			var value struct {
				Type    string      `json:"type,omitempty"`
				Role    string      `json:"role"`
				Content textContent `json:"content"`
			}
			decoder := json.NewDecoder(bytes.NewReader(item))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&value); err != nil {
				return nil, invalid("input", "A Responses message item is invalid.")
			}
			messages = append(messages, Message{Role: Role(value.Role), Text: string(value.Content)})
		case "function_call":
			var value struct {
				Type      string `json:"type"`
				CallID    string `json:"call_id"`
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
				ID        string `json:"id,omitempty"`
				Status    string `json:"status,omitempty"`
			}
			decoder := json.NewDecoder(bytes.NewReader(item))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&value); err != nil {
				return nil, invalid("input", "A function-call input item is invalid.")
			}
			appendAssistantCall(ToolCall{ID: value.CallID, Name: value.Name, Arguments: value.Arguments})
		case "function_call_output":
			var value struct {
				Type   string `json:"type"`
				CallID string `json:"call_id"`
				Output string `json:"output"`
				ID     string `json:"id,omitempty"`
				Status string `json:"status,omitempty"`
			}
			decoder := json.NewDecoder(bytes.NewReader(item))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&value); err != nil {
				return nil, invalid("input", "A function-call output item is invalid.")
			}
			messages = append(messages, Message{Role: RoleTool, ToolCallID: value.CallID, Text: value.Output})
		default:
			return nil, invalid("input", "This Responses input item type is not supported in the text-and-function transport tranche.")
		}
	}
	return messages, nil
}
