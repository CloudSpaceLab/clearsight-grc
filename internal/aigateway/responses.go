package aigateway

import "time"

func chatCompletionResponse(request Request, response Response) map[string]any {
	message := map[string]any{"role": "assistant", "content": nullableContent(response.Text)}
	if response.Refusal != "" {
		message["refusal"] = response.Refusal
	}
	if len(response.ToolCalls) > 0 {
		calls := make([]map[string]any, 0, len(response.ToolCalls))
		for _, call := range response.ToolCalls {
			calls = append(calls, map[string]any{"id": call.ID, "type": "function", "function": map[string]string{"name": call.Name, "arguments": call.Arguments}})
		}
		message["tool_calls"] = calls
	}
	return map[string]any{
		"id": response.ID, "object": "chat.completion", "created": response.CreatedAt.Unix(), "model": request.ModelAlias,
		"choices": []map[string]any{{"index": 0, "message": message, "finish_reason": chatFinishReason(response.FinishReason), "logprobs": nil}},
		"usage":   chatUsage(response.Usage),
	}
}

func responsesObject(request Request, response Response, status string) map[string]any {
	incomplete := any(nil)
	itemStatus := status
	if status == "completed" {
		status, incomplete = responsesTerminalState(response.FinishReason)
		itemStatus = status
	}
	output := make([]any, 0, 1+len(response.ToolCalls))
	if response.Text != "" || response.Refusal != "" {
		content := make([]any, 0, 2)
		if response.Text != "" {
			content = append(content, map[string]any{"type": "output_text", "text": response.Text, "annotations": []any{}})
		}
		if response.Refusal != "" {
			content = append(content, map[string]any{"type": "refusal", "refusal": response.Refusal})
		}
		output = append(output, map[string]any{"id": "msg_" + request.ID, "type": "message", "status": itemStatus, "role": "assistant", "content": content})
	}
	for index, call := range response.ToolCalls {
		output = append(output, map[string]any{"id": responseToolItemID(request.ID, index), "type": "function_call", "status": itemStatus, "call_id": call.ID, "name": call.Name, "arguments": call.Arguments})
	}
	return map[string]any{
		"id": response.ID, "object": "response", "created_at": response.CreatedAt.Unix(), "status": status,
		"error": nil, "incomplete_details": incomplete, "instructions": nil, "max_output_tokens": request.MaxOutputTokens,
		"model": request.ModelAlias, "output": output, "parallel_tool_calls": true, "tool_choice": responsesToolChoice(request.ToolChoice),
		"tools": responsesTools(request.Tools), "usage": responsesUsage(response.Usage),
	}
}

func chatFinishReason(reason string) string {
	if reason == "refusal" {
		return "content_filter"
	}
	return reason
}

func responsesTerminalState(reason string) (string, any) {
	switch reason {
	case "length":
		return "incomplete", map[string]string{"reason": "max_output_tokens"}
	case "content_filter":
		return "incomplete", map[string]string{"reason": "content_filter"}
	default:
		return "completed", nil
	}
}

func responsesTools(tools []ToolDefinition) []any {
	result := make([]any, 0, len(tools))
	for _, tool := range tools {
		result = append(result, map[string]any{"type": "function", "name": tool.Name, "description": tool.Description, "parameters": tool.Parameters, "strict": tool.Strict})
	}
	return result
}

func responsesToolChoice(choice ToolChoice) any {
	switch choice.Mode {
	case ToolChoiceNone, ToolChoiceRequired:
		return choice.Mode
	case ToolChoiceNamed:
		return map[string]string{"type": "function", "name": choice.Name}
	default:
		return "auto"
	}
}

func chatUsage(usage Usage) map[string]any {
	return map[string]any{
		"prompt_tokens": usage.InputTokens, "completion_tokens": usage.OutputTokens, "total_tokens": usage.TotalTokens(),
		"prompt_tokens_details": map[string]int64{"cached_tokens": usage.CachedInputTokens},
	}
}

func responsesUsage(usage Usage) map[string]any {
	return map[string]any{
		"input_tokens": usage.InputTokens, "output_tokens": usage.OutputTokens, "total_tokens": usage.TotalTokens(),
		"input_tokens_details":  map[string]int64{"cached_tokens": usage.CachedInputTokens},
		"output_tokens_details": map[string]int64{"reasoning_tokens": 0},
	}
}

func nullableContent(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func responseToolItemID(requestID string, index int) string {
	return "fc_" + requestID + "_" + itoa(index)
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = byte('0' + value%10)
		value /= 10
	}
	return string(buffer[index:])
}

func responseStartedAt(requestID string, protocol Protocol) Response {
	return Response{ID: responseID(protocol, requestID), CreatedAt: time.Now().UTC()}
}
