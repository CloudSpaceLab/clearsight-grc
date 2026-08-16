package aigateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func writeSSE(writer http.ResponseWriter, flusher http.Flusher, eventName string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if eventName != "" {
		if _, err := fmt.Fprintf(writer, "event: %s\n", eventName); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(writer, "data: %s\n\n", encoded); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func writeSSELiteral(writer http.ResponseWriter, flusher http.Flusher, value string) error {
	if _, err := fmt.Fprintf(writer, "data: %s\n\n", value); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func findFlusher(writer http.ResponseWriter) (http.Flusher, bool) {
	for writer != nil {
		if flusher, ok := writer.(http.Flusher); ok {
			return flusher, true
		}
		unwrapper, ok := writer.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			break
		}
		writer = unwrapper.Unwrap()
	}
	return nil, false
}

type chatStreamSink struct {
	writer       http.ResponseWriter
	flusher      http.Flusher
	request      Request
	responseID   string
	created      int64
	started      bool
	finished     bool
	includeUsage bool
}

func newChatStreamSink(writer http.ResponseWriter, request Request, requestID string) *chatStreamSink {
	return &chatStreamSink{writer: writer, request: request, responseID: responseID(ProtocolChat, requestID), created: time.Now().UTC().Unix(), includeUsage: request.IncludeStreamUsage}
}

func (s *chatStreamSink) Start(routeID string) error {
	flusher, ok := findFlusher(s.writer)
	if !ok {
		return ErrInternal
	}
	s.flusher = flusher
	s.writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	s.writer.Header().Set("Cache-Control", "no-cache, no-store")
	s.writer.Header().Set("X-Accel-Buffering", "no")
	s.writer.Header().Set("X-ClearSight-Route", routeID)
	s.writer.WriteHeader(http.StatusOK)
	s.started = true
	return s.chunk(map[string]any{"role": "assistant"}, nil, nil)
}

func (s *chatStreamSink) Emit(event StreamEvent) error {
	if !s.started || s.finished {
		return ErrProtocol
	}
	switch event.Type {
	case StreamTextDelta:
		return s.chunk(map[string]any{"content": event.Text}, nil, nil)
	case StreamRefusalDelta:
		return s.chunk(map[string]any{"refusal": event.Text}, nil, nil)
	case StreamToolDelta:
		if event.Tool == nil {
			return ErrProtocol
		}
		function := map[string]any{}
		if event.Tool.Name != "" {
			function["name"] = event.Tool.Name
		}
		if event.Tool.Arguments != "" {
			function["arguments"] = event.Tool.Arguments
		}
		call := map[string]any{"index": event.Tool.Index, "function": function}
		if event.Tool.ID != "" {
			call["id"] = event.Tool.ID
			call["type"] = "function"
		}
		return s.chunk(map[string]any{"tool_calls": []any{call}}, nil, nil)
	case StreamFinish:
		reason := chatFinishReason(event.FinishReason)
		return s.chunk(map[string]any{}, &reason, nil)
	case StreamUsage:
		if !s.includeUsage || event.Usage == nil {
			return nil
		}
		return s.chunk(nil, nil, chatUsage(*event.Usage))
	case StreamDone:
		s.finished = true
		return writeSSELiteral(s.writer, s.flusher, "[DONE]")
	default:
		return ErrProtocol
	}
}

func (s *chatStreamSink) Fail(gatewayErr *Error) error {
	if !s.started || s.finished {
		return nil
	}
	s.finished = true
	if gatewayErr == nil {
		gatewayErr = ErrInternal
	}
	if err := writeSSE(s.writer, s.flusher, "", errorObject(gatewayErr)); err != nil {
		return err
	}
	return writeSSELiteral(s.writer, s.flusher, "[DONE]")
}

func (s *chatStreamSink) chunk(delta any, finishReason *string, usage any) error {
	choices := []any{}
	if delta != nil || finishReason != nil {
		choices = append(choices, map[string]any{"index": 0, "delta": delta, "finish_reason": finishReason, "logprobs": nil})
	}
	payload := map[string]any{"id": s.responseID, "object": "chat.completion.chunk", "created": s.created, "model": s.request.ModelAlias, "choices": choices}
	if usage != nil {
		payload["usage"] = usage
	}
	return writeSSE(s.writer, s.flusher, "", payload)
}

type responseToolStreamState struct {
	id        string
	name      string
	arguments strings.Builder
	added     bool
}

type responsesStreamSink struct {
	writer       http.ResponseWriter
	flusher      http.Flusher
	request      Request
	response     Response
	responseID   string
	sequence     int64
	started      bool
	finished     bool
	textAdded    bool
	refusalAdded bool
	finishReason string
	tools        map[int]*responseToolStreamState
}

func newResponsesStreamSink(writer http.ResponseWriter, request Request, requestID string) *responsesStreamSink {
	response := responseStartedAt(requestID, ProtocolResponses)
	return &responsesStreamSink{writer: writer, request: request, response: response, responseID: response.ID, tools: make(map[int]*responseToolStreamState)}
}

func (s *responsesStreamSink) Start(routeID string) error {
	flusher, ok := findFlusher(s.writer)
	if !ok {
		return ErrInternal
	}
	s.flusher = flusher
	s.writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	s.writer.Header().Set("Cache-Control", "no-cache, no-store")
	s.writer.Header().Set("X-Accel-Buffering", "no")
	s.writer.Header().Set("X-ClearSight-Route", routeID)
	s.writer.WriteHeader(http.StatusOK)
	s.started = true
	created := responsesObject(s.request, s.response, "in_progress")
	if err := s.emit("response.created", map[string]any{"type": "response.created", "response": created}); err != nil {
		return err
	}
	return s.emit("response.in_progress", map[string]any{"type": "response.in_progress", "response": created})
}

func (s *responsesStreamSink) Emit(event StreamEvent) error {
	if !s.started || s.finished {
		return ErrProtocol
	}
	switch event.Type {
	case StreamTextDelta:
		if err := s.ensureTextItem(false); err != nil {
			return err
		}
		s.response.Text += event.Text
		return s.emit("response.output_text.delta", map[string]any{"type": "response.output_text.delta", "item_id": "msg_" + s.request.ID, "output_index": 0, "content_index": 0, "delta": event.Text})
	case StreamRefusalDelta:
		if err := s.ensureTextItem(true); err != nil {
			return err
		}
		s.response.Refusal += event.Text
		return s.emit("response.refusal.delta", map[string]any{"type": "response.refusal.delta", "item_id": "msg_" + s.request.ID, "output_index": 0, "content_index": s.refusalContentIndex(), "delta": event.Text})
	case StreamToolDelta:
		return s.emitToolDelta(event.Tool)
	case StreamFinish:
		s.finishReason = event.FinishReason
		s.response.FinishReason = event.FinishReason
		return nil
	case StreamUsage:
		if event.Usage == nil {
			return ErrProtocol
		}
		s.response.Usage = *event.Usage
		return nil
	case StreamDone:
		return s.complete()
	default:
		return ErrProtocol
	}
}

func (s *responsesStreamSink) Fail(gatewayErr *Error) error {
	if !s.started || s.finished {
		return nil
	}
	s.finished = true
	if gatewayErr == nil {
		gatewayErr = ErrInternal
	}
	if err := s.emit("error", map[string]any{"type": "error", "code": gatewayErr.Code, "message": gatewayErr.Message, "param": nullableString(gatewayErr.Param)}); err != nil {
		return err
	}
	failed := responsesObject(s.request, s.response, "failed")
	failed["error"] = map[string]any{"code": gatewayErr.Code, "message": gatewayErr.Message}
	return s.emit("response.failed", map[string]any{"type": "response.failed", "response": failed})
}

func (s *responsesStreamSink) ensureTextItem(refusal bool) error {
	if !s.textAdded && !s.refusalAdded {
		item := map[string]any{"id": "msg_" + s.request.ID, "type": "message", "status": "in_progress", "role": "assistant", "content": []any{}}
		if err := s.emit("response.output_item.added", map[string]any{"type": "response.output_item.added", "output_index": 0, "item": item}); err != nil {
			return err
		}
	}
	if refusal {
		if s.refusalAdded {
			return nil
		}
		s.refusalAdded = true
		return s.emit("response.content_part.added", map[string]any{"type": "response.content_part.added", "item_id": "msg_" + s.request.ID, "output_index": 0, "content_index": s.refusalContentIndex(), "part": map[string]any{"type": "refusal", "refusal": ""}})
	}
	if s.textAdded {
		return nil
	}
	s.textAdded = true
	return s.emit("response.content_part.added", map[string]any{"type": "response.content_part.added", "item_id": "msg_" + s.request.ID, "output_index": 0, "content_index": 0, "part": map[string]any{"type": "output_text", "text": "", "annotations": []any{}}})
}

func (s *responsesStreamSink) refusalContentIndex() int {
	if s.textAdded {
		return 1
	}
	return 0
}

func (s *responsesStreamSink) emitToolDelta(delta *ToolCallDelta) error {
	if delta == nil {
		return ErrProtocol
	}
	state := s.tools[delta.Index]
	if state == nil {
		state = &responseToolStreamState{}
		s.tools[delta.Index] = state
	}
	if delta.ID != "" {
		state.id = delta.ID
	}
	if delta.Name != "" {
		state.name = delta.Name
	}
	if !state.added {
		if state.id == "" || state.name == "" {
			return ErrProtocol
		}
		state.added = true
		item := map[string]any{"id": responseToolItemID(s.request.ID, delta.Index), "type": "function_call", "status": "in_progress", "call_id": state.id, "name": state.name, "arguments": ""}
		if err := s.emit("response.output_item.added", map[string]any{"type": "response.output_item.added", "output_index": s.toolOutputIndex(delta.Index), "item": item}); err != nil {
			return err
		}
	}
	if delta.Arguments == "" {
		return nil
	}
	state.arguments.WriteString(delta.Arguments)
	return s.emit("response.function_call_arguments.delta", map[string]any{"type": "response.function_call_arguments.delta", "item_id": responseToolItemID(s.request.ID, delta.Index), "output_index": s.toolOutputIndex(delta.Index), "delta": delta.Arguments})
}

func (s *responsesStreamSink) toolOutputIndex(index int) int {
	if s.textAdded || s.refusalAdded {
		return index + 1
	}
	return index
}

func (s *responsesStreamSink) complete() error {
	if s.finished || s.finishReason == "" {
		return ErrProtocol
	}
	if s.textAdded {
		if err := s.emit("response.output_text.done", map[string]any{"type": "response.output_text.done", "item_id": "msg_" + s.request.ID, "output_index": 0, "content_index": 0, "text": s.response.Text}); err != nil {
			return err
		}
		if err := s.emit("response.content_part.done", map[string]any{"type": "response.content_part.done", "item_id": "msg_" + s.request.ID, "output_index": 0, "content_index": 0, "part": map[string]any{"type": "output_text", "text": s.response.Text, "annotations": []any{}}}); err != nil {
			return err
		}
	}
	if s.refusalAdded {
		idx := s.refusalContentIndex()
		if err := s.emit("response.refusal.done", map[string]any{"type": "response.refusal.done", "item_id": "msg_" + s.request.ID, "output_index": 0, "content_index": idx, "refusal": s.response.Refusal}); err != nil {
			return err
		}
		if err := s.emit("response.content_part.done", map[string]any{"type": "response.content_part.done", "item_id": "msg_" + s.request.ID, "output_index": 0, "content_index": idx, "part": map[string]any{"type": "refusal", "refusal": s.response.Refusal}}); err != nil {
			return err
		}
	}
	if s.textAdded || s.refusalAdded {
		item := map[string]any{"id": "msg_" + s.request.ID, "type": "message", "status": "completed", "role": "assistant", "content": []any{}}
		if s.textAdded {
			item["content"] = append(item["content"].([]any), map[string]any{"type": "output_text", "text": s.response.Text, "annotations": []any{}})
		}
		if s.refusalAdded {
			item["content"] = append(item["content"].([]any), map[string]any{"type": "refusal", "refusal": s.response.Refusal})
		}
		if err := s.emit("response.output_item.done", map[string]any{"type": "response.output_item.done", "output_index": 0, "item": item}); err != nil {
			return err
		}
	}
	indexes := make([]int, 0, len(s.tools))
	for index := range s.tools {
		indexes = append(indexes, index)
	}
	for i := 0; i < len(indexes); i++ {
		for j := i + 1; j < len(indexes); j++ {
			if indexes[j] < indexes[i] {
				indexes[i], indexes[j] = indexes[j], indexes[i]
			}
		}
	}
	for _, index := range indexes {
		state := s.tools[index]
		arguments := state.arguments.String()
		if err := s.emit("response.function_call_arguments.done", map[string]any{"type": "response.function_call_arguments.done", "item_id": responseToolItemID(s.request.ID, index), "output_index": s.toolOutputIndex(index), "arguments": arguments}); err != nil {
			return err
		}
		s.response.ToolCalls = append(s.response.ToolCalls, ToolCall{ID: state.id, Name: state.name, Arguments: arguments})
		item := map[string]any{"id": responseToolItemID(s.request.ID, index), "type": "function_call", "status": "completed", "call_id": state.id, "name": state.name, "arguments": arguments}
		if err := s.emit("response.output_item.done", map[string]any{"type": "response.output_item.done", "output_index": s.toolOutputIndex(index), "item": item}); err != nil {
			return err
		}
	}
	s.finished = true
	status, _ := responsesTerminalState(s.finishReason)
	name := "response.completed"
	if status == "incomplete" {
		name = "response.incomplete"
	}
	return s.emit(name, map[string]any{"type": name, "response": responsesObject(s.request, s.response, "completed")})
}

func (s *responsesStreamSink) emit(name string, value map[string]any) error {
	value["sequence_number"] = s.sequence
	s.sequence++
	return writeSSE(s.writer, s.flusher, name, value)
}

func errorObject(gatewayErr *Error) map[string]any {
	return map[string]any{"error": map[string]any{"message": gatewayErr.Message, "type": gatewayErr.Code, "param": nullableString(gatewayErr.Param), "code": gatewayErr.Code}}
}
