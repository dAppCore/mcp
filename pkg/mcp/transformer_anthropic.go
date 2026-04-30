// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	core "dappco.re/go"
	"github.com/goccy/go-json"
)

// AnthropicTransformer maps Anthropic Messages requests and responses.
type AnthropicTransformer struct{}

func (AnthropicTransformer) Detect(body []byte, contentType, path string) bool {
	if headerHasMedia(contentType, "application/anthropic+json") {
		return true
	}
	if normaliseGatewayPath(path) == "/v1/messages" {
		return true
	}
	if !hasTopLevelFields(body, "model", "messages") {
		return false
	}
	return looksAnthropicBody(body) || messagesHaveNoSystemRole(body)
}

func (AnthropicTransformer) Normalise(body []byte) (
	MCPRequest,
	error,
) {
	var req anthropicMessagesRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return MCPRequest{}, err
	}
	if req.Model == "" {
		return MCPRequest{}, core.NewError("anthropic messages request missing model")
	}
	if len(req.Messages) == 0 {
		return MCPRequest{}, core.NewError("anthropic messages request missing messages")
	}

	params := map[string]any{
		"source_format": "anthropic",
		"model":         req.Model,
		"messages":      normaliseAnthropicMessages(req.Messages),
	}
	if req.System != nil {
		params["system"] = req.System
	}
	if req.MaxTokens != nil {
		params["max_tokens"] = req.MaxTokens
	}
	if req.Temperature != nil {
		params["temperature"] = req.Temperature
	}
	if req.Stream {
		params["stream"] = req.Stream
	}
	if len(req.Tools) > 0 {
		params["tools"] = normaliseAnthropicTools(req.Tools)
	}

	toolCalls := anthropicToolUsesFromMessages(req.Messages)
	if len(toolCalls) > 0 {
		call := toolCalls[0]
		params["name"] = call.Name
		params["arguments"] = call.Arguments
		params["tool_calls"] = toolCalls
		return MCPRequest{JSONRPC: "2.0", Method: "tools/call", Params: params}, nil
	}

	return MCPRequest{JSONRPC: "2.0", Method: "sampling/createMessage", Params: params}, nil
}

func (AnthropicTransformer) Transform(result MCPResult) (
	[]byte,
	error,
) {
	text := extractMCPText(result)
	toolCalls := extractMCPToolCalls(result)

	content := make([]map[string]any, 0, 1+len(toolCalls))
	if text != "" {
		content = append(content, map[string]any{
			"type": "text",
			"text": text,
		})
	}
	for i, call := range toolCalls {
		id := call.ID
		if id == "" {
			id = core.Sprintf("toolu_%d", i)
		}
		content = append(content, map[string]any{
			"type":  "tool_use",
			"id":    id,
			"name":  call.Name,
			"input": call.Arguments,
		})
	}
	if len(content) == 0 {
		content = append(content, map[string]any{
			"type": "text",
			"text": "",
		})
	}

	stopReason := "end_turn"
	if len(toolCalls) > 0 {
		stopReason = "tool_use"
	}
	if result.StopReason != "" {
		stopReason = result.StopReason
	}

	resp := map[string]any{
		"id":            anthropicResponseID(result.ID),
		"type":          "message",
		"role":          "assistant",
		"model":         "mcp-gateway",
		"content":       content,
		"stop_reason":   stopReason,
		"stop_sequence": nil,
	}
	return json.Marshal(resp)
}

type anthropicMessagesRequest struct {
	Model       string             `json:"model"`
	MaxTokens   any                `json:"max_tokens,omitempty"`
	System      any                `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	Tools       []anthropicTool    `json:"tools,omitempty"`
	Temperature any                `json:"temperature,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content,omitempty"`
}

type anthropicTool struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"input_schema,omitempty"`
}

func normaliseAnthropicMessages(messages []anthropicMessage) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		item := map[string]any{
			"role": msg.Role,
		}
		if msg.Content != nil {
			item["content"] = msg.Content
		}
		out = append(out, item)
	}
	return out
}

func normaliseAnthropicTools(tools []anthropicTool) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, tool := range tools {
		out = append(out, map[string]any{
			"name":         tool.Name,
			"description":  tool.Description,
			"input_schema": tool.InputSchema,
		})
	}
	return out
}

func anthropicToolUsesFromMessages(messages []anthropicMessage) []MCPToolCall {
	var calls []MCPToolCall
	for i := len(messages) - 1; i >= 0; i-- {
		blocks := anthropicContentBlocks(messages[i].Content)
		for _, block := range blocks {
			if block.Type != "tool_use" || block.Name == "" {
				continue
			}
			calls = append(calls, MCPToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: block.Input,
			})
		}
		if len(calls) > 0 {
			break
		}
	}
	return calls
}

type anthropicContentBlock struct {
	Type  string         `json:"type"`
	Text  string         `json:"text,omitempty"`
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
}

func anthropicContentBlocks(content any) []anthropicContentBlock {
	switch typed := content.(type) {
	case nil:
		return nil
	case []anthropicContentBlock:
		return typed
	case []any:
		blocks := make([]anthropicContentBlock, 0, len(typed))
		for _, item := range typed {
			data, err := json.Marshal(item)
			if err != nil {
				continue
			}
			var block anthropicContentBlock
			if err := json.Unmarshal(data, &block); err == nil {
				blocks = append(blocks, block)
			}
		}
		return blocks
	case map[string]any:
		data, err := json.Marshal(typed)
		if err != nil {
			return nil
		}
		var block anthropicContentBlock
		if err := json.Unmarshal(data, &block); err != nil {
			return nil
		}
		return []anthropicContentBlock{block}
	case string:
		return []anthropicContentBlock{{Type: "text", Text: typed}}
	default:
		return nil
	}
}

func anthropicResponseID(id any) string {
	if id == nil {
		return "msg_mcp"
	}
	return core.Sprintf("msg_%v", id)
}
