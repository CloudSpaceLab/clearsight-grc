package aigateway

import (
	"encoding/json"
	"math"
	"strings"
	"unicode/utf8"
)

func validateCanonicalResponse(response Response) error {
	if !validFinishReason(response.FinishReason) || len(response.Text)+len(response.Refusal) > MaxTextBytes || !utf8.ValidString(response.Text) || !utf8.ValidString(response.Refusal) || len(response.ToolCalls) > MaxTools {
		return ErrProtocol
	}
	if err := validateUsage(response.Usage); err != nil {
		return err
	}
	if (response.FinishReason == "tool_calls") != (len(response.ToolCalls) > 0) {
		return ErrProtocol
	}
	ids := make(map[string]struct{}, len(response.ToolCalls))
	for _, call := range response.ToolCalls {
		arguments := strings.TrimSpace(call.Arguments)
		if !validIdentifier(call.ID) || !validToolName(call.Name) || len(arguments) == 0 || len(arguments) > MaxToolArgumentsBytes || !strings.HasPrefix(arguments, "{") || !json.Valid([]byte(arguments)) {
			return ErrProtocol
		}
		if _, exists := ids[call.ID]; exists {
			return ErrProtocol
		}
		ids[call.ID] = struct{}{}
	}
	return nil
}

func validateUsage(usage Usage) error {
	if usage.InputTokens < 0 || usage.OutputTokens < 0 || usage.CachedInputTokens < 0 || usage.CachedInputTokens > usage.InputTokens || usage.InputTokens > math.MaxInt64-usage.OutputTokens {
		return ErrProtocol
	}
	return nil
}

func validFinishReason(reason string) bool {
	switch reason {
	case "stop", "length", "tool_calls", "content_filter", "refusal":
		return true
	default:
		return false
	}
}

type canonicalToolStreamState struct {
	id        string
	name      string
	arguments strings.Builder
}

type canonicalStreamValidator struct {
	requireUsage bool
	seenFinish   bool
	seenUsage    bool
	finishReason string
	outputBytes  int
	tools        map[int]*canonicalToolStreamState
	toolIDs      map[string]int
}

func newCanonicalStreamValidator(requireUsage bool) *canonicalStreamValidator {
	return &canonicalStreamValidator{
		requireUsage: requireUsage,
		tools:        make(map[int]*canonicalToolStreamState),
		toolIDs:      make(map[string]int),
	}
}

func (v *canonicalStreamValidator) accept(event StreamEvent) error {
	switch event.Type {
	case StreamTextDelta, StreamRefusalDelta:
		if v.seenFinish || event.Text == "" || !utf8.ValidString(event.Text) || len(event.Text) > MaxTextBytes-v.outputBytes {
			return ErrProtocol
		}
		v.outputBytes += len(event.Text)
	case StreamToolDelta:
		if v.seenFinish || event.Tool == nil || event.Tool.Index < 0 || event.Tool.Index >= MaxTools || event.Tool.ID == "" && event.Tool.Name == "" && event.Tool.Arguments == "" {
			return ErrProtocol
		}
		state := v.tools[event.Tool.Index]
		if state == nil {
			if event.Tool.ID == "" || event.Tool.Name == "" {
				return ErrProtocol
			}
			state = &canonicalToolStreamState{}
			v.tools[event.Tool.Index] = state
		}
		if event.Tool.ID != "" {
			if !validIdentifier(event.Tool.ID) || state.id != "" && state.id != event.Tool.ID {
				return ErrProtocol
			}
			if prior, exists := v.toolIDs[event.Tool.ID]; exists && prior != event.Tool.Index {
				return ErrProtocol
			}
			v.toolIDs[event.Tool.ID] = event.Tool.Index
			state.id = event.Tool.ID
		}
		if event.Tool.Name != "" {
			if !validToolName(event.Tool.Name) || state.name != "" && state.name != event.Tool.Name {
				return ErrProtocol
			}
			state.name = event.Tool.Name
		}
		if state.arguments.Len()+len(event.Tool.Arguments) > MaxToolArgumentsBytes {
			return ErrProtocol
		}
		state.arguments.WriteString(event.Tool.Arguments)
	case StreamFinish:
		if v.seenFinish || !validFinishReason(event.FinishReason) {
			return ErrProtocol
		}
		v.seenFinish = true
		v.finishReason = event.FinishReason
	case StreamUsage:
		if !v.seenFinish || v.seenUsage || event.Usage == nil {
			return ErrProtocol
		}
		if err := validateUsage(*event.Usage); err != nil {
			return err
		}
		v.seenUsage = true
	case StreamDone:
		if !v.seenFinish || v.requireUsage && !v.seenUsage || (v.finishReason == "tool_calls") != (len(v.tools) > 0) {
			return ErrProtocol
		}
		for _, state := range v.tools {
			arguments := strings.TrimSpace(state.arguments.String())
			if !validIdentifier(state.id) || !validToolName(state.name) || arguments == "" || !strings.HasPrefix(arguments, "{") || !json.Valid([]byte(arguments)) {
				return ErrProtocol
			}
		}
	default:
		return ErrProtocol
	}
	return nil
}
