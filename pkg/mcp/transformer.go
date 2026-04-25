// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"bytes"
	"encoding/json"
	"mime"
	"strings"
)

// TransformerIn normalises an AI wire protocol request into a unified MCP
// request envelope.
type TransformerIn interface {
	Detect(body []byte, contentType, path string) bool
	Normalise(body []byte) (MCPRequest, error)
}

// TransformerOut converts an MCP result back into an AI wire protocol response.
type TransformerOut interface {
	Transform(result MCPResult) ([]byte, error)
}

// MCPRequest is the gateway's protocol-neutral JSON-RPC request shape.
type MCPRequest struct {
	JSONRPC string         `json:"jsonrpc,omitempty"`
	ID      any            `json:"id,omitempty"`
	Method  string         `json:"method,omitempty"`
	Params  map[string]any `json:"params,omitempty"`
}

// MCPResult is the gateway's protocol-neutral JSON-RPC result shape.
type MCPResult struct {
	JSONRPC    string        `json:"jsonrpc,omitempty"`
	ID         any           `json:"id,omitempty"`
	Result     any           `json:"result,omitempty"`
	Error      any           `json:"error,omitempty"`
	Content    []MCPContent  `json:"content,omitempty"`
	ToolCalls  []MCPToolCall `json:"tool_calls,omitempty"`
	StopReason string        `json:"stop_reason,omitempty"`
}

// MCPContent represents text and tool-use content blocks in the neutral result.
type MCPContent struct {
	Type      string         `json:"type,omitempty"`
	Text      string         `json:"text,omitempty"`
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Input     map[string]any `json:"input,omitempty"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// MCPToolCall captures a model-requested tool invocation.
type MCPToolCall struct {
	ID        string         `json:"id,omitempty"`
	Name      string         `json:"name,omitempty"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// TODO(#197 follow-up): add Ollama and LiteLLM concrete transformers once the
// OpenAI/Anthropic/MCP-native gateway surface has settled.

// NegotiateTransformer selects the inbound transformer using RFC §9.4 priority:
// explicit media type, path, body inspection, then MCP-native fallback. The
// honeypot is only selected for malformed or probe-like bodies that no concrete
// protocol claims.
func NegotiateTransformer(body []byte, contentType, path string) TransformerIn {
	if headerHasMedia(contentType, "application/openai+json") {
		return OpenAITransformer{}
	}
	if headerHasMedia(contentType, "application/anthropic+json") {
		return AnthropicTransformer{}
	}
	if headerHasMedia(contentType, "application/mcp+json", "application/json-rpc", "application/jsonrpc+json") {
		return MCPNativeTransformer{}
	}

	switch normaliseGatewayPath(path) {
	case "/v1/chat/completions":
		return OpenAITransformer{}
	case "/v1/messages":
		return AnthropicTransformer{}
	case "/mcp":
		if (HoneypotTransformer{}).Detect(body, contentType, path) {
			return HoneypotTransformer{}
		}
		return MCPNativeTransformer{}
	}

	if (MCPNativeTransformer{}).Detect(body, "", "") {
		return MCPNativeTransformer{}
	}
	if (OpenAITransformer{}).Detect(body, "", "") {
		if looksAnthropicBody(body) {
			return AnthropicTransformer{}
		}
		return OpenAITransformer{}
	}
	if (AnthropicTransformer{}).Detect(body, "", "") {
		return AnthropicTransformer{}
	}
	if (HoneypotTransformer{}).Detect(body, contentType, path) {
		return HoneypotTransformer{}
	}
	return MCPNativeTransformer{}
}

// MCPNativeTransformer is the identity transformer for native MCP JSON-RPC.
type MCPNativeTransformer struct{}

func (MCPNativeTransformer) Detect(body []byte, contentType, path string) bool {
	if headerHasMedia(contentType, "application/mcp+json", "application/json-rpc", "application/jsonrpc+json") {
		return true
	}
	if normaliseGatewayPath(path) == "/mcp" {
		return true
	}

	obj, ok := decodeJSONObject(body)
	if !ok {
		return false
	}
	_, hasMethod := obj["method"].(string)
	_, hasResult := obj["result"]
	_, hasError := obj["error"]
	return obj["jsonrpc"] == "2.0" && (hasMethod || hasResult || hasError)
}

func (MCPNativeTransformer) Normalise(body []byte) (MCPRequest, error) {
	var req MCPRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return MCPRequest{}, err
	}
	if req.JSONRPC == "" {
		req.JSONRPC = "2.0"
	}
	return req, nil
}

func (MCPNativeTransformer) Transform(result MCPResult) ([]byte, error) {
	if result.JSONRPC == "" {
		result.JSONRPC = "2.0"
	}
	return json.Marshal(result)
}

func headerHasMedia(header string, wants ...string) bool {
	header = strings.TrimSpace(header)
	if header == "" {
		return false
	}

	wantSet := make(map[string]struct{}, len(wants))
	for _, want := range wants {
		wantSet[strings.ToLower(strings.TrimSpace(want))] = struct{}{}
	}

	for _, part := range strings.Split(header, ",") {
		media := strings.TrimSpace(part)
		if parsed, _, err := mime.ParseMediaType(media); err == nil {
			media = parsed
		} else if semi := strings.IndexByte(media, ';'); semi >= 0 {
			media = media[:semi]
		}
		media = strings.ToLower(strings.TrimSpace(media))
		if _, ok := wantSet[media]; ok {
			return true
		}
	}
	return false
}

func normaliseGatewayPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	return path
}

func decodeJSONObject(body []byte) (map[string]any, bool) {
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		return nil, false
	}
	var obj map[string]any
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, false
	}
	return obj, true
}

func hasTopLevelFields(body []byte, fields ...string) bool {
	obj, ok := decodeJSONObject(body)
	if !ok {
		return false
	}
	for _, field := range fields {
		if _, ok := obj[field]; !ok {
			return false
		}
	}
	return true
}

func looksAnthropicBody(body []byte) bool {
	obj, ok := decodeJSONObject(body)
	if !ok {
		return false
	}
	if _, ok := obj["system"]; ok {
		return true
	}
	if _, ok := obj["max_tokens"]; ok {
		return true
	}
	if _, ok := obj["anthropic_version"]; ok {
		return true
	}

	messages, ok := obj["messages"].([]any)
	if !ok || len(messages) == 0 {
		return false
	}
	for _, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if role, _ := msg["role"].(string); role == "system" {
			return false
		}
		if blocks, ok := msg["content"].([]any); ok {
			for _, rawBlock := range blocks {
				block, ok := rawBlock.(map[string]any)
				if !ok {
					continue
				}
				switch block["type"] {
				case "tool_use", "tool_result":
					return true
				}
			}
		}
	}
	return false
}

func messagesHaveNoSystemRole(body []byte) bool {
	obj, ok := decodeJSONObject(body)
	if !ok {
		return false
	}
	messages, ok := obj["messages"].([]any)
	if !ok || len(messages) == 0 {
		return false
	}
	for _, raw := range messages {
		msg, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if role, _ := msg["role"].(string); role == "system" {
			return false
		}
	}
	return true
}

func parseRawArgumentObject(raw json.RawMessage) map[string]any {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return map[string]any{}
	}

	var encoded string
	if err := json.Unmarshal(raw, &encoded); err == nil {
		return parseArgumentString(encoded)
	}

	var args map[string]any
	if err := json.Unmarshal(raw, &args); err == nil && args != nil {
		return args
	}
	return map[string]any{"_raw": string(raw)}
}

func parseArgumentString(s string) map[string]any {
	s = strings.TrimSpace(s)
	if s == "" {
		return map[string]any{}
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(s), &args); err == nil && args != nil {
		return args
	}
	return map[string]any{"_raw": s}
}

func mapFromAny(v any) map[string]any {
	switch typed := v.(type) {
	case nil:
		return map[string]any{}
	case map[string]any:
		if typed == nil {
			return map[string]any{}
		}
		return typed
	case json.RawMessage:
		return parseRawArgumentObject(typed)
	case string:
		return parseArgumentString(typed)
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return map[string]any{"value": typed}
		}
		return parseRawArgumentObject(data)
	}
}

func extractMCPText(result MCPResult) string {
	var parts []string
	for _, block := range result.Content {
		if block.Text != "" && (block.Type == "" || block.Type == "text") {
			parts = append(parts, block.Text)
		}
	}
	parts = append(parts, extractTextFromAny(result.Result)...)
	return strings.Join(parts, "\n")
}

func extractTextFromAny(v any) []string {
	switch typed := v.(type) {
	case nil:
		return nil
	case string:
		if typed == "" {
			return nil
		}
		return []string{typed}
	case []byte:
		if len(typed) == 0 {
			return nil
		}
		return []string{string(typed)}
	case []MCPContent:
		var out []string
		for _, block := range typed {
			if block.Text != "" && (block.Type == "" || block.Type == "text") {
				out = append(out, block.Text)
			}
		}
		return out
	case []any:
		var out []string
		for _, item := range typed {
			out = append(out, extractTextFromAny(item)...)
		}
		return out
	case []map[string]any:
		var out []string
		for _, item := range typed {
			out = append(out, extractTextFromAny(item)...)
		}
		return out
	case map[string]any:
		for _, key := range []string{"text", "message", "output"} {
			if text, ok := typed[key].(string); ok && text != "" {
				return []string{text}
			}
		}
		if content, ok := typed["content"]; ok {
			return extractTextFromAny(content)
		}
		if result, ok := typed["result"]; ok {
			return extractTextFromAny(result)
		}
		return nil
	default:
		data, err := json.Marshal(typed)
		if err != nil || len(data) == 0 || bytes.Equal(data, []byte("null")) {
			return nil
		}
		return []string{string(data)}
	}
}

func extractMCPToolCalls(result MCPResult) []MCPToolCall {
	var calls []MCPToolCall
	calls = append(calls, result.ToolCalls...)
	for _, block := range result.Content {
		if block.Type != "tool_use" && block.Name == "" {
			continue
		}
		args := block.Input
		if len(args) == 0 {
			args = block.Arguments
		}
		calls = append(calls, MCPToolCall{ID: block.ID, Name: block.Name, Arguments: args})
	}
	calls = append(calls, extractToolCallsFromAny(result.Result)...)
	return calls
}

func extractToolCallsFromAny(v any) []MCPToolCall {
	switch typed := v.(type) {
	case nil:
		return nil
	case []MCPToolCall:
		return typed
	case []MCPContent:
		var calls []MCPToolCall
		for _, block := range typed {
			if block.Type == "tool_use" || block.Name != "" {
				args := block.Input
				if len(args) == 0 {
					args = block.Arguments
				}
				calls = append(calls, MCPToolCall{ID: block.ID, Name: block.Name, Arguments: args})
			}
		}
		return calls
	case []any:
		var calls []MCPToolCall
		for _, item := range typed {
			calls = append(calls, extractToolCallsFromAny(item)...)
		}
		return calls
	case []map[string]any:
		var calls []MCPToolCall
		for _, item := range typed {
			calls = append(calls, extractToolCallsFromAny(item)...)
		}
		return calls
	case map[string]any:
		for _, key := range []string{"tool_calls", "toolCalls"} {
			if raw, ok := typed[key]; ok {
				return extractToolCallsFromAny(raw)
			}
		}
		if raw, ok := typed["content"]; ok {
			return extractToolCallsFromAny(raw)
		}
		name, _ := typed["name"].(string)
		if name == "" {
			if fn, ok := typed["function"].(map[string]any); ok {
				name, _ = fn["name"].(string)
				args := mapFromAny(fn["arguments"])
				id, _ := typed["id"].(string)
				return []MCPToolCall{{ID: id, Name: name, Arguments: args}}
			}
			return nil
		}
		id, _ := typed["id"].(string)
		args := mapFromAny(typed["arguments"])
		if len(args) == 0 {
			args = mapFromAny(typed["input"])
		}
		return []MCPToolCall{{ID: id, Name: name, Arguments: args}}
	default:
		return nil
	}
}
