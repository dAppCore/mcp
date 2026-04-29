package mcp

// moved AX-7 triplet TestTransformerOpenai_OpenAITransformer_Detect_Good
func TestTransformerOpenai_OpenAITransformer_Detect_Good(t *T) {
	body := []byte(`{"model":"gpt","messages":[{"role":"user","content":"hi"}]}`)
	got := (OpenAITransformer{}).Detect(body, "", "")
	AssertTrue(t, got)
}

// moved AX-7 triplet TestTransformerOpenai_OpenAITransformer_Detect_Bad
func TestTransformerOpenai_OpenAITransformer_Detect_Bad(t *T) {
	got := (OpenAITransformer{}).Detect([]byte(`{"model":"gpt"}`), "", "")
	AssertFalse(t, got)
	AssertFalse(t, (OpenAITransformer{}).Detect([]byte(`bad`), "", ""))
}

// moved AX-7 triplet TestTransformerOpenai_OpenAITransformer_Detect_Ugly
func TestTransformerOpenai_OpenAITransformer_Detect_Ugly(t *T) {
	got := (OpenAITransformer{}).Detect(nil, "application/openai+json", "")
	AssertTrue(t, got)
	AssertTrue(t, (OpenAITransformer{}).Detect(nil, "", "/v1/chat/completions"))
}

// moved AX-7 triplet TestTransformerOpenai_OpenAITransformer_Normalise_Good
func TestTransformerOpenai_OpenAITransformer_Normalise_Good(t *T) {
	body := []byte(`{"model":"gpt","messages":[{"role":"user","content":"hi"}]}`)
	req, err := (OpenAITransformer{}).Normalise(body)
	AssertNoError(t, err)
	AssertEqual(t, "sampling/createMessage", req.Method)
}

// moved AX-7 triplet TestTransformerOpenai_OpenAITransformer_Normalise_Bad
func TestTransformerOpenai_OpenAITransformer_Normalise_Bad(t *T) {
	req, err := (OpenAITransformer{}).Normalise([]byte(`{"messages":[]}`))
	AssertError(t, err)
	AssertEqual(t, MCPRequest{}, req)
}

// moved AX-7 triplet TestTransformerOpenai_OpenAITransformer_Normalise_Ugly
func TestTransformerOpenai_OpenAITransformer_Normalise_Ugly(t *T) {
	body := []byte(`{"model":"gpt","messages":[{"role":"assistant","tool_calls":[{"id":"1","type":"function","function":{"name":"echo","arguments":"{\"x\":1}"}}]}]}`)
	req, err := (OpenAITransformer{}).Normalise(body)
	AssertNoError(t, err)
	AssertEqual(t, "tools/call", req.Method)
}

// moved AX-7 triplet TestTransformerOpenai_OpenAITransformer_Transform_Good
func TestTransformerOpenai_OpenAITransformer_Transform_Good(t *T) {
	out, err := (OpenAITransformer{}).Transform(MCPResult{ID: "1", Content: []MCPContent{{Type: "text", Text: "ok"}}})
	AssertNoError(t, err)
	AssertContains(t, string(out), "chat.completion")
}

// moved AX-7 triplet TestTransformerOpenai_OpenAITransformer_Transform_Bad
func TestTransformerOpenai_OpenAITransformer_Transform_Bad(t *T) {
	out, err := (OpenAITransformer{}).Transform(MCPResult{ToolCalls: []MCPToolCall{{Name: "bad", Arguments: map[string]any{"bad": make(chan int)}}}})
	AssertNoError(t, err)
	AssertContains(t, string(out), `"arguments":"{}"`)
}

// moved AX-7 triplet TestTransformerOpenai_OpenAITransformer_Transform_Ugly
func TestTransformerOpenai_OpenAITransformer_Transform_Ugly(t *T) {
	out, err := (OpenAITransformer{}).Transform(MCPResult{})
	AssertNoError(t, err)
	AssertContains(t, string(out), `"content":""`)
}
