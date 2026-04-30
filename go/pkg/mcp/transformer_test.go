// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"github.com/goccy/go-json"
	"testing"
)

func TestNegotiate_OpenAI_Good(t *testing.T) {
	body := []byte(`{"model":"gpt-4o-mini","messages":[{"role":"user","content":"hello"}]}`)

	OpenAI := NegotiateTransformer(body, "", "/v1/chat/completions")
	if _, ok := OpenAI.(OpenAITransformer); !ok {
		t.Fatal("expected OpenAITransformer for chat completions path")
	}
}

func TestNegotiate_Anthropic_Good(t *testing.T) {
	body := []byte(`{"model":"claude-3-5-sonnet","max_tokens":128,"messages":[{"role":"user","content":"hello"}]}`)

	Anthropic := NegotiateTransformer(body, "", "/v1/messages")
	if _, ok := Anthropic.(AnthropicTransformer); !ok {
		t.Fatal("expected AnthropicTransformer for messages path")
	}
}

func TestNegotiate_MCPNative_Good(t *testing.T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)

	MCPNative := NegotiateTransformer(body, "application/mcp+json", "/mcp")
	if _, ok := MCPNative.(MCPNativeTransformer); !ok {
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
	if args[`path`] != "README.md" {
		t.Fatalf("expected README.md path, got %v", args[`path`])
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
							"input": {` + "\"path\"" + `:"README.md"}
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
	if args[`path`] != "README.md" {
		t.Fatalf("expected README.md path, got %v", args[`path`])
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

	Priority := "application/openai+json"
	if _, ok := NegotiateTransformer(body, Priority, "/v1/messages").(OpenAITransformer); !ok {
		t.Fatal("expected explicit OpenAI media type to beat path/body inspection")
	}
}

// moved AX-7 triplet TestTransformer_MCPNativeTransformer_Detect_Good
func TestTransformer_MCPNativeTransformer_Detect_Good(t *T) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	got := (MCPNativeTransformer{}).Detect(body, "", "")
	AssertTrue(t, got)
}

// moved AX-7 triplet TestTransformer_MCPNativeTransformer_Detect_Bad
func TestTransformer_MCPNativeTransformer_Detect_Bad(t *T) {
	got := (MCPNativeTransformer{}).Detect([]byte(`{"method":"tools/list"}`), "", "")
	AssertFalse(t, got)
	AssertFalse(t, (MCPNativeTransformer{}).Detect([]byte(`bad`), "", ""))
}

// moved AX-7 triplet TestTransformer_MCPNativeTransformer_Detect_Ugly
func TestTransformer_MCPNativeTransformer_Detect_Ugly(t *T) {
	got := (MCPNativeTransformer{}).Detect(nil, "application/mcp+json", "")
	AssertTrue(t, got)
	AssertTrue(t, (MCPNativeTransformer{}).Detect(nil, "", "/mcp"))
}

// moved AX-7 triplet TestTransformer_MCPNativeTransformer_Normalise_Good
func TestTransformer_MCPNativeTransformer_Normalise_Good(t *T) {
	req, err := (MCPNativeTransformer{}).Normalise([]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	AssertNoError(t, err)
	AssertEqual(t, "tools/list", req.Method)
}

// moved AX-7 triplet TestTransformer_MCPNativeTransformer_Normalise_Bad
func TestTransformer_MCPNativeTransformer_Normalise_Bad(t *T) {
	req, err := (MCPNativeTransformer{}).Normalise([]byte(`bad`))
	AssertError(t, err)
	AssertEqual(t, MCPRequest{}, req)
}

// moved AX-7 triplet TestTransformer_MCPNativeTransformer_Normalise_Ugly
func TestTransformer_MCPNativeTransformer_Normalise_Ugly(t *T) {
	req, err := (MCPNativeTransformer{}).Normalise([]byte(`{"method":"ping"}`))
	AssertNoError(t, err)
	AssertEqual(t, "2.0", req.JSONRPC)
}

// moved AX-7 triplet TestTransformer_MCPNativeTransformer_Transform_Good
func TestTransformer_MCPNativeTransformer_Transform_Good(t *T) {
	out, err := (MCPNativeTransformer{}).Transform(MCPResult{ID: 1, Result: map[string]any{"ok": true}})
	AssertNoError(t, err)
	AssertContains(t, string(out), `"jsonrpc":"2.0"`)
}

// moved AX-7 triplet TestTransformer_MCPNativeTransformer_Transform_Bad
func TestTransformer_MCPNativeTransformer_Transform_Bad(t *T) {
	_, err := (MCPNativeTransformer{}).Transform(MCPResult{Result: make(chan int)})
	AssertError(t, err)
	AssertContains(t, err.Error(), "unsupported type")
}

// moved AX-7 triplet TestTransformer_MCPNativeTransformer_Transform_Ugly
func TestTransformer_MCPNativeTransformer_Transform_Ugly(t *T) {
	out, err := (MCPNativeTransformer{}).Transform(MCPResult{JSONRPC: "2.0", Error: map[string]any{"code": -1}})
	AssertNoError(t, err)
	AssertContains(t, string(out), `"error"`)
}

// moved AX-7 triplet TestTransformer_NegotiateTransformer_Good
func TestTransformer_NegotiateTransformer_Good(t *T) {
	body := []byte(`{"model":"gpt","messages":[{"role":"user","content":"hi"}]}`)
	got := NegotiateTransformer(body, "application/openai+json", "")
	AssertTrue(t, reflectTransformer[OpenAITransformer](got))
}

// moved AX-7 triplet TestTransformer_NegotiateTransformer_Bad
func TestTransformer_NegotiateTransformer_Bad(t *T) {
	got := NegotiateTransformer([]byte(`{}`), "", "")
	AssertTrue(t, reflectTransformer[MCPNativeTransformer](got))
	AssertFalse(t, reflectTransformer[OpenAITransformer](got))
}

// moved AX-7 triplet TestTransformer_NegotiateTransformer_Ugly
func TestTransformer_NegotiateTransformer_Ugly(t *T) {
	got := NegotiateTransformer([]byte(`not-json`), "", "/mcp")
	AssertTrue(t, reflectTransformer[HoneypotTransformer](got))
	AssertFalse(t, reflectTransformer[AnthropicTransformer](got))
}
