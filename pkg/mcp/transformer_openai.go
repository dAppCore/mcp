// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"encoding/json"
	"fmt"
)

// OpenAITransformer maps OpenAI Chat Completions requests and responses.
type OpenAITransformer struct{}

func (OpenAITransformer) Detect(body []byte, contentType, path string) bool {
	if headerHasMedia(contentType, "application/openai+json") {
		return true
	}
	if normaliseGatewayPath(path) == "/v1/chat/completions" {
		return true
	}
	return hasTopLevelFields(body, "model", "messages")
}

func (OpenAITransformer) Normalise(body []byte) (MCPRequest, error) {
	var req openAIChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return MCPRequest{}, err
	}
	if req.Model == "" {
		return MCPRequest{}, fmt.Errorf("openai chat completion request missing model")
	}
	if len(req.Messages) == 0 {
		return MCPRequest{}, fmt.Errorf("openai chat completion request missing messages")
	}

	params := map[string]any{
		"source_format": "openai",
		"model":         req.Model,
		"messages":      normaliseOpenAIMessages(req.Messages),
	}
	if len(req.Tools) > 0 {
		params["tools"] = normaliseOpenAITools(req.Tools)
	}
	if req.ToolChoice != nil {
		params["tool_choice"] = req.ToolChoice
	}
	if req.MaxTokens != nil {
		params["max_tokens"] = req.MaxTokens
	}
	if req.MaxCompletionTokens != nil {
		params["max_completion_tokens"] = req.MaxCompletionTokens
	}
	if req.Temperature != nil {
		params["temperature"] = req.Temperature
	}
	if req.Stream {
		params["stream"] = req.Stream
	}

	toolCalls := openAIToolCallsFromMessages(req.Messages)
	if len(toolCalls) > 0 {
		call := toolCalls[0]
		params["name"] = call.Name
		params["arguments"] = call.Arguments
		params["tool_calls"] = toolCalls
		return MCPRequest{JSONRPC: "2.0", Method: "tools/call", Params: params}, nil
	}

	return MCPRequest{JSONRPC: "2.0", Method: "sampling/createMessage", Params: params}, nil
}

func (OpenAITransformer) Transform(result MCPResult) ([]byte, error) {
	text := extractMCPText(result)
	toolCalls := extractMCPToolCalls(result)

	message := map[string]any{
		"role": "assistant",
	}
	if text != "" {
		message["content"] = text
	} else if len(toolCalls) > 0 {
		message["content"] = nil
	} else {
		message["content"] = ""
	}
	if len(toolCalls) > 0 {
		message["tool_calls"] = openAIToolCallsFromMCP(toolCalls)
	}

	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}
	if result.StopReason != "" {
		finishReason = result.StopReason
	}

	resp := map[string]any{
		"id":      openAIResponseID(result.ID),
		"object":  "chat.completion",
		"created": 0,
		"model":   "mcp-gateway",
		"choices": []map[string]any{
			{
				"index":         0,
				"message":       message,
				"finish_reason": finishReason,
			},
		},
	}
	return json.Marshal(resp)
}

type openAIChatCompletionRequest struct {
	Model               string          `json:"model"`
	Messages            []openAIMessage `json:"messages"`
	Tools               []openAITool    `json:"tools,omitempty"`
	ToolChoice          any             `json:"tool_choice,omitempty"`
	MaxTokens           any             `json:"max_tokens,omitempty"`
	MaxCompletionTokens any             `json:"max_completion_tokens,omitempty"`
	Temperature         any             `json:"temperature,omitempty"`
	Stream              bool            `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content,omitempty"`
	Name       string           `json:"name,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAITool struct {
	Type     string                 `json:"type"`
	Function openAIFunctionMetadata `json:"function"`
}

type openAIFunctionMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
}

type openAIToolCall struct {
	ID       string             `json:"id,omitempty"`
	Type     string             `json:"type,omitempty"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

func normaliseOpenAIMessages(messages []openAIMessage) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		item := map[string]any{
			"role": msg.Role,
		}
		if msg.Content != nil {
			item["content"] = msg.Content
		}
		if msg.Name != "" {
			item["name"] = msg.Name
		}
		if msg.ToolCallID != "" {
			item["tool_call_id"] = msg.ToolCallID
		}
		if len(msg.ToolCalls) > 0 {
			item["tool_calls"] = openAIToolCallsFromMessages([]openAIMessage{msg})
		}
		out = append(out, item)
	}
	return out
}

func normaliseOpenAITools(tools []openAITool) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		if tool.Type != "" && tool.Type != "function" {
			out = append(out, map[string]any{
				"type":     tool.Type,
				"function": tool.Function,
			})
			continue
		}
		item := map[string]any{
			"name":         tool.Function.Name,
			"description":  tool.Function.Description,
			"input_schema": tool.Function.Parameters,
		}
		out = append(out, item)
	}
	return out
}

func openAIToolCallsFromMessages(messages []openAIMessage) []MCPToolCall {
	var calls []MCPToolCall
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if len(msg.ToolCalls) == 0 {
			continue
		}
		for _, call := range msg.ToolCalls {
			if call.Function.Name == "" {
				continue
			}
			calls = append(calls, MCPToolCall{
				ID:        call.ID,
				Name:      call.Function.Name,
				Arguments: parseRawArgumentObject(call.Function.Arguments),
			})
		}
		break
	}
	return calls
}

func openAIToolCallsFromMCP(calls []MCPToolCall) []map[string]any {
	out := make([]map[string]any, 0, len(calls))
	for i, call := range calls {
		id := call.ID
		if id == "" {
			id = fmt.Sprintf("call_%d", i)
		}
		args, err := json.Marshal(call.Arguments)
		if err != nil {
			args = []byte("{}")
		}
		out = append(out, map[string]any{
			"id":   id,
			"type": "function",
			"function": map[string]any{
				"name":      call.Name,
				"arguments": string(args),
			},
		})
	}
	return out
}

func openAIResponseID(id any) string {
	if id == nil {
		return "chatcmpl-mcp"
	}
	return fmt.Sprintf("chatcmpl-%v", id)
}
