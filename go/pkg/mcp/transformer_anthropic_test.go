package mcp

// moved AX-7 triplet TestTransformerAnthropic_AnthropicTransformer_Detect_Good
func TestTransformerAnthropic_AnthropicTransformer_Detect_Good(t *T) {
	body := []byte(`{"model":"claude","max_tokens":8,"messages":[{"role":"user","content":"hi"}]}`)
	got := (AnthropicTransformer{}).Detect(body, "", "")
	AssertTrue(t, got)
}

// moved AX-7 triplet TestTransformerAnthropic_AnthropicTransformer_Detect_Bad
func TestTransformerAnthropic_AnthropicTransformer_Detect_Bad(t *T) {
	body := []byte(`{"model":"claude","messages":[{"role":"system","content":"policy"}]}`)
	got := (AnthropicTransformer{}).Detect(body, "", "")
	AssertFalse(t, got)
}

// moved AX-7 triplet TestTransformerAnthropic_AnthropicTransformer_Detect_Ugly
func TestTransformerAnthropic_AnthropicTransformer_Detect_Ugly(t *T) {
	got := (AnthropicTransformer{}).Detect(nil, "application/anthropic+json", "")
	AssertTrue(t, got)
	AssertTrue(t, (AnthropicTransformer{}).Detect(nil, "", "/v1/messages"))
}

// moved AX-7 triplet TestTransformerAnthropic_AnthropicTransformer_Normalise_Good
func TestTransformerAnthropic_AnthropicTransformer_Normalise_Good(t *T) {
	body := []byte(`{"model":"claude","max_tokens":8,"messages":[{"role":"user","content":"hi"}]}`)
	req, err := (AnthropicTransformer{}).Normalise(body)
	AssertNoError(t, err)
	AssertEqual(t, "sampling/createMessage", req.Method)
}

// moved AX-7 triplet TestTransformerAnthropic_AnthropicTransformer_Normalise_Bad
func TestTransformerAnthropic_AnthropicTransformer_Normalise_Bad(t *T) {
	req, err := (AnthropicTransformer{}).Normalise([]byte(`{"messages":[]}`))
	AssertError(t, err)
	AssertEqual(t, MCPRequest{}, req)
}

// moved AX-7 triplet TestTransformerAnthropic_AnthropicTransformer_Normalise_Ugly
func TestTransformerAnthropic_AnthropicTransformer_Normalise_Ugly(t *T) {
	body := []byte(`{"model":"claude","messages":[{"role":"assistant","content":[{"type":"tool_use","id":"1","name":"echo","input":{"x":1}}]}]}`)
	req, err := (AnthropicTransformer{}).Normalise(body)
	AssertNoError(t, err)
	AssertEqual(t, "tools/call", req.Method)
}

// moved AX-7 triplet TestTransformerAnthropic_AnthropicTransformer_Transform_Good
func TestTransformerAnthropic_AnthropicTransformer_Transform_Good(t *T) {
	out, err := (AnthropicTransformer{}).Transform(MCPResult{ID: "1", Content: []MCPContent{{Type: "text", Text: "ok"}}})
	AssertNoError(t, err)
	AssertContains(t, string(out), `"type":"message"`)
}

// moved AX-7 triplet TestTransformerAnthropic_AnthropicTransformer_Transform_Bad
func TestTransformerAnthropic_AnthropicTransformer_Transform_Bad(t *T) {
	_, err := (AnthropicTransformer{}).Transform(MCPResult{ToolCalls: []MCPToolCall{{Name: "bad", Arguments: map[string]any{"bad": make(chan int)}}}})
	AssertError(t, err)
	AssertContains(t, err.Error(), "unsupported type")
}

// moved AX-7 triplet TestTransformerAnthropic_AnthropicTransformer_Transform_Ugly
func TestTransformerAnthropic_AnthropicTransformer_Transform_Ugly(t *T) {
	out, err := (AnthropicTransformer{}).Transform(MCPResult{})
	AssertNoError(t, err)
	AssertContains(t, string(out), `"text":""`)
}
