package aigateway

import (
	"bytes"
	"encoding/json"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxMessages           = 256
	MaxTools              = 64
	MaxStops              = 4
	MaxTextBytes          = 1 << 20
	MaxToolArgumentsBytes = 1 << 20
	MaxToolSchemaBytes    = 1 << 20
	MaxOutputTokens       = 131072
)

type Protocol string

const (
	ProtocolChat      Protocol = "chat"
	ProtocolResponses Protocol = "responses"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleDeveloper Role = "developer"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type ToolChoiceMode = string

const (
	ToolChoiceAuto     = "auto"
	ToolChoiceNone     = "none"
	ToolChoiceRequired = "required"
	ToolChoiceNamed    = "named"
)

type ToolChoice struct {
	Mode ToolChoiceMode
	Name string
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type ToolCallDelta struct {
	Index     int
	ID        string
	Name      string
	Arguments string
}

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  json.RawMessage
	Strict      bool
}

type Message struct {
	Role       Role
	Text       string
	ToolCalls  []ToolCall
	ToolCallID string
}

type Request struct {
	ID                 string
	Protocol           Protocol
	ModelAlias         string
	RouteID            string
	Metadata           map[string]string
	Messages           []Message
	Tools              []ToolDefinition
	ToolChoice         ToolChoice
	MaxOutputTokens    int64
	Temperature        *float64
	TopP               *float64
	Stop               []string
	Stream             bool
	IncludeStreamUsage bool
}

type Usage struct {
	InputTokens       int64
	CachedInputTokens int64
	OutputTokens      int64
}

func (u Usage) TotalTokens() int64 {
	if u.InputTokens < 0 || u.OutputTokens < 0 || u.InputTokens > math.MaxInt64-u.OutputTokens {
		return math.MaxInt64
	}
	return u.InputTokens + u.OutputTokens
}

type Response struct {
	ID           string
	CreatedAt    time.Time
	Text         string
	Refusal      string
	ToolCalls    []ToolCall
	FinishReason string
	Usage        Usage
}

type TokenPrice struct {
	InputPerMillion  int64
	OutputPerMillion int64
}

type Route struct {
	ID         string
	ProviderID string
	Model      string
	Weight     int64
	Price      TokenPrice
}

type Workload struct {
	ID                    string
	TenantID              string
	Purpose               string
	Environment           string
	VerifiedMetadata      map[string]string
	Policy                PolicySnapshot
	AllowedModels         map[string]struct{}
	RequestsPerMinute     int64
	TokensPerMinute       int64
	CostMicroUSDPerMinute int64
	MaxConcurrent         int64
}

type StreamEventType string

const (
	StreamTextDelta    StreamEventType = "text_delta"
	StreamRefusalDelta StreamEventType = "refusal_delta"
	StreamToolDelta    StreamEventType = "tool_delta"
	StreamFinish       StreamEventType = "finish"
	StreamUsage        StreamEventType = "usage"
	StreamDone         StreamEventType = "done"
)

type StreamEvent struct {
	Type         StreamEventType
	Text         string
	Tool         *ToolCallDelta
	FinishReason string
	Usage        *Usage
}

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,255}$`)
var toolNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func validIdentifier(value string) bool {
	return identifierPattern.MatchString(strings.TrimSpace(value))
}

func validToolName(value string) bool {
	return toolNamePattern.MatchString(strings.TrimSpace(value))
}

func ValidateRequest(request Request) error {
	if !validIdentifier(request.ID) {
		return invalid("X-Request-ID", "The request identifier is invalid.")
	}
	if request.Protocol != ProtocolChat && request.Protocol != ProtocolResponses {
		return invalid("", "The request protocol is not supported.")
	}
	if !validIdentifier(request.ModelAlias) {
		return invalid("model", "The model alias is invalid.")
	}
	if request.RouteID != "" && !validIdentifier(request.RouteID) {
		return invalid("route", "The governed route identifier is invalid.")
	}
	if err := validateRequestMetadata(request.Metadata); err != nil {
		return err
	}
	if len(request.Messages) == 0 || len(request.Messages) > MaxMessages {
		return invalid("messages", "The request must contain between 1 and 256 messages.")
	}
	if len(request.Tools) > MaxTools {
		return invalid("tools", "The request contains too many function tools.")
	}
	if request.MaxOutputTokens < 1 || request.MaxOutputTokens > MaxOutputTokens {
		return invalid("max_output_tokens", "The output-token limit is outside the supported range.")
	}
	if request.Temperature != nil && (math.IsNaN(*request.Temperature) || math.IsInf(*request.Temperature, 0) || *request.Temperature < 0 || *request.Temperature > 1) {
		return invalid("temperature", "Temperature must be between 0 and 1.")
	}
	if request.TopP != nil && (math.IsNaN(*request.TopP) || math.IsInf(*request.TopP, 0) || *request.TopP <= 0 || *request.TopP > 1) {
		return invalid("top_p", "Top-p must be greater than 0 and no greater than 1.")
	}
	if len(request.Stop) > MaxStops {
		return invalid("stop", "The request contains too many stop sequences.")
	}
	for _, stop := range request.Stop {
		if stop == "" || len(stop) > 1024 || !utf8.ValidString(stop) {
			return invalid("stop", "A stop sequence is empty or outside the supported limit.")
		}
	}
	toolNames := make(map[string]struct{}, len(request.Tools))
	for _, tool := range request.Tools {
		if tool.Strict || !validToolName(tool.Name) || len(tool.Description) > 8192 || !utf8.ValidString(tool.Description) || len(tool.Parameters) == 0 || len(tool.Parameters) > MaxToolSchemaBytes || !json.Valid(tool.Parameters) || firstJSONByte(tool.Parameters) != '{' {
			return invalid("tools", "A function tool definition is invalid or outside the supported limits.")
		}
		if _, exists := toolNames[tool.Name]; exists {
			return invalid("tools", "Function tool names must be unique.")
		}
		toolNames[tool.Name] = struct{}{}
	}
	choice := request.ToolChoice
	if choice.Mode == "" {
		choice.Mode = ToolChoiceAuto
	}
	switch choice.Mode {
	case ToolChoiceAuto:
		if choice.Name != "" {
			return invalid("tool_choice", "Automatic tool choice cannot name a function.")
		}
	case ToolChoiceNone:
		if choice.Name != "" {
			return invalid("tool_choice", "Disabled tool choice cannot name a function.")
		}
	case ToolChoiceRequired:
		if len(request.Tools) == 0 || choice.Name != "" {
			return invalid("tool_choice", "Required tool choice needs at least one function tool.")
		}
	case ToolChoiceNamed:
		if _, exists := toolNames[choice.Name]; !exists {
			return invalid("tool_choice", "The named function is not present in the request tools.")
		}
	default:
		return invalid("tool_choice", "The tool-choice mode is not supported.")
	}

	var totalText int
	assistantCalls := make(map[string]struct{})
	resolvedCalls := make(map[string]struct{})
	for index, message := range request.Messages {
		if !validRole(message.Role) || !utf8.ValidString(message.Text) || len(message.Text) > MaxTextBytes-totalText || len(message.ToolCalls) > MaxTools {
			return invalid("messages", "A message is invalid or outside the supported limits.")
		}
		totalText += len(message.Text)
		switch message.Role {
		case RoleSystem, RoleDeveloper, RoleUser:
			if len(message.ToolCalls) > 0 || message.ToolCallID != "" || message.Text == "" {
				return invalid("messages", "The message role and content do not form a valid text conversation.")
			}
		case RoleAssistant:
			if message.ToolCallID != "" || (message.Text == "" && len(message.ToolCalls) == 0) {
				return invalid("messages", "An assistant message must contain text or function calls.")
			}
			for _, call := range message.ToolCalls {
				if err := validateRequestToolCall(call); err != nil {
					return err
				}
				if _, exists := assistantCalls[call.ID]; exists {
					return invalid("messages", "Function call identifiers must be unique.")
				}
				assistantCalls[call.ID] = struct{}{}
			}
		case RoleTool:
			if len(message.ToolCalls) > 0 || !validIdentifier(message.ToolCallID) || message.Text == "" {
				return invalid("messages", "A function result message is invalid.")
			}
			if _, exists := assistantCalls[message.ToolCallID]; !exists {
				return invalid("messages", "A function result must follow a matching function call.")
			}
			if _, duplicate := resolvedCalls[message.ToolCallID]; duplicate {
				return invalid("messages", "A function call cannot have more than one result.")
			}
			resolvedCalls[message.ToolCallID] = struct{}{}
		}
		_ = index
	}
	for callID := range assistantCalls {
		if _, resolved := resolvedCalls[callID]; !resolved {
			return invalid("messages", "Every function call must have one matching function result before generation continues.")
		}
	}
	if totalText == 0 {
		return invalid("messages", "The request contains no text or function input.")
	}
	return nil
}

func validateRequestToolCall(call ToolCall) error {
	arguments := strings.TrimSpace(call.Arguments)
	if !validIdentifier(call.ID) || !validToolName(call.Name) || len(arguments) == 0 || len(arguments) > MaxToolArgumentsBytes || !json.Valid([]byte(arguments)) || firstJSONByte([]byte(arguments)) != '{' {
		return invalid("messages", "A function call is invalid or outside the supported limits.")
	}
	return nil
}

func validRole(role Role) bool {
	switch role {
	case RoleSystem, RoleDeveloper, RoleUser, RoleAssistant, RoleTool:
		return true
	default:
		return false
	}
}

func firstJSONByte(value []byte) byte {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 {
		return 0
	}
	return trimmed[0]
}
