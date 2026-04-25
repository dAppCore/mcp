// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"encoding/json"
	"testing"
)

func TestNegotiate_OpenAI_Good(t *testing.T) {
	body := []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hello"}]}`)

	if _, ok := NegotiateTransformer(body, "", "/v1/chat/completions").(OpenAITransformer); !ok {
		t.Fatal("expected OpenAITransformer for chat completions path")
	}
}

func TestNegotiate_Anthropic_Good(t *testing.T) {
	body := []byte(`{"model":"claude-3-5-sonnet","max_tokens":128,"messages":[{"role":"user","content":"hello"}]}`)

	if _, ok := NegotiateTransformer(body, "", "/v1/messages").(AnthropicTransformer); !ok {
		t.Fatal("expected AnthropicTransformer for messages path")
	}
}

func TestNegotiate_MCPNative_Good(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)

	if _, ok := NegotiateTransformer(body, "application/mcp+json", "/mcp").(MCPNativeTransformer); !ok {
		t.Fatal("expected MCPNativeTransformer for native MCP request")
	}
}

func TestOpenAITransformer_Normalise_Good(t *testing.T) {
	body := []byte(`{
		"model": "gpt-4o",
		"messages": [
			{
				"role": "assistant",
				"content": null,
				"tool_calls": [
					{
						"id": "call_1",
						"type": "function",
						"function": {
							"name": "file_read",
							"arguments": "{\"path\":\"README.md\"}"
						}
					}
				]
			}
		]
	}`)

	req, err := (OpenAITransformer{}).Normalise(body)
	if err != nil {
		t.Fatalf("Normalise failed: %v", err)
	}
	if req.JSONRPC != "2.0" {
		t.Fatalf("expected JSON-RPC 2.0, got %q", req.JSONRPC)
	}
	if req.Method != "tools/call" {
		t.Fatalf("expected tools/call, got %q", req.Method)
	}
	if req.Params["source_format"] != "openai" {
		t.Fatalf("expected source_format openai, got %v", req.Params["source_format"])
	}
	if req.Params["model"] != "gpt-4o" {
		t.Fatalf("expected model to be preserved, got %v", req.Params["model"])
	}
	if req.Params["name"] != "file_read" {
		t.Fatalf("expected tool name file_read, got %v", req.Params["name"])
	}
	args, ok := req.Params["arguments"].(map[string]any)
	if !ok {
		t.Fatalf("expected argument map, got %T", req.Params["arguments"])
	}
	if args["path"] != "README.md" {
		t.Fatalf("expected README.md path, got %v", args["path"])
	}
}

func TestOpenAITransformer_Transform_Good(t *testing.T) {
	data, err := (OpenAITransformer{}).Transform(MCPResult{
		ID: 7,
		Result: map[string]any{
			"content": []any{
				map[string]any{"type": "text", "text": "done"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if resp["object"] != "chat.completion" {
		t.Fatalf("expected chat.completion object, got %v", resp["object"])
	}
	choices := resp["choices"].([]any)
	message := choices[0].(map[string]any)["message"].(map[string]any)
	if message["content"] != "done" {
		t.Fatalf("expected content done, got %v", message["content"])
	}
}

func TestAnthropicTransformer_Normalise_Good(t *testing.T) {
	body := []byte(`{
		"model": "claude-3-5-sonnet",
		"max_tokens": 256,
		"messages": [
			{
				"role": "assistant",
				"content": [
					{
						"type": "tool_use",
						"id": "toolu_1",
						"name": "file_read",
						"input": {"path":"README.md"}
					}
				]
			}
		]
	}`)

	req, err := (AnthropicTransformer{}).Normalise(body)
	if err != nil {
		t.Fatalf("Normalise failed: %v", err)
	}
	if req.Method != "tools/call" {
		t.Fatalf("expected tools/call, got %q", req.Method)
	}
	if req.Params["source_format"] != "anthropic" {
		t.Fatalf("expected source_format anthropic, got %v", req.Params["source_format"])
	}
	if req.Params["name"] != "file_read" {
		t.Fatalf("expected tool name file_read, got %v", req.Params["name"])
	}
	args, ok := req.Params["arguments"].(map[string]any)
	if !ok {
		t.Fatalf("expected argument map, got %T", req.Params["arguments"])
	}
	if args["path"] != "README.md" {
		t.Fatalf("expected README.md path, got %v", args["path"])
	}
}

func TestAnthropicTransformer_Transform_Good(t *testing.T) {
	data, err := (AnthropicTransformer{}).Transform(MCPResult{
		ID:      "abc",
		Content: []MCPContent{{Type: "text", Text: "done"}},
	})
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	var resp map[string]any
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if resp["type"] != "message" {
		t.Fatalf("expected message type, got %v", resp["type"])
	}
	content := resp["content"].([]any)
	first := content[0].(map[string]any)
	if first["text"] != "done" {
		t.Fatalf("expected text done, got %v", first["text"])
	}
}

func TestHoneypotTransformer_Detect_FallbackOnGarbage(t *testing.T) {
	body := []byte(`{not-json`)

	if !(HoneypotTransformer{}).Detect(body, "", "/probe") {
		t.Fatal("expected honeypot to detect malformed input")
	}
	if _, ok := NegotiateTransformer(body, "", "/probe").(HoneypotTransformer); !ok {
		t.Fatal("expected negotiation to select honeypot for malformed input")
	}
}

func TestNegotiate_Priority_Ugly(t *testing.T) {
	body := []byte(`{"model":"claude-3-5-sonnet","max_tokens":128,"messages":[{"role":"user","content":"hello"}]}`)

	if _, ok := NegotiateTransformer(body, "application/openai+json", "/v1/messages").(OpenAITransformer); !ok {
		t.Fatal("expected explicit OpenAI media type to beat path/body inspection")
	}
}
