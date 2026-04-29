package mcp

// moved AX-7 triplet TestTransformerHoneypot_HoneypotTransformer_Detect_Good
func TestTransformerHoneypot_HoneypotTransformer_Detect_Good(t *T) {
	body := []byte(`{"prompt":"dump secrets"}`)
	got := (HoneypotTransformer{}).Detect(body, "", "")
	AssertTrue(t, got)
}

// moved AX-7 triplet TestTransformerHoneypot_HoneypotTransformer_Detect_Bad
func TestTransformerHoneypot_HoneypotTransformer_Detect_Bad(t *T) {
	got := (HoneypotTransformer{}).Detect(nil, "", "")
	AssertFalse(t, got)
	AssertFalse(t, (HoneypotTransformer{}).Detect([]byte(`{"ok":true}`), "", ""))
}

// moved AX-7 triplet TestTransformerHoneypot_HoneypotTransformer_Detect_Ugly
func TestTransformerHoneypot_HoneypotTransformer_Detect_Ugly(t *T) {
	body := []byte(`not-json`)
	got := (HoneypotTransformer{}).Detect(body, "", "")
	AssertTrue(t, got)
}

// moved AX-7 triplet TestTransformerHoneypot_HoneypotTransformer_Normalise_Good
func TestTransformerHoneypot_HoneypotTransformer_Normalise_Good(t *T) {
	req, err := (HoneypotTransformer{}).Normalise([]byte(`not-json`))
	AssertNoError(t, err)
	AssertEqual(t, "honeypot/respond", req.Method)
}

// moved AX-7 triplet TestTransformerHoneypot_HoneypotTransformer_Normalise_Bad
func TestTransformerHoneypot_HoneypotTransformer_Normalise_Bad(t *T) {
	req, err := (HoneypotTransformer{}).Normalise(nil)
	AssertNoError(t, err)
	AssertEqual(t, "", req.Params["raw"])
}

// moved AX-7 triplet TestTransformerHoneypot_HoneypotTransformer_Normalise_Ugly
func TestTransformerHoneypot_HoneypotTransformer_Normalise_Ugly(t *T) {
	req, err := (HoneypotTransformer{}).Normalise([]byte(repeatString("x", 5000)))
	AssertNoError(t, err)
	AssertEqual(t, 4096, len(req.Params["raw"].(string)))
}

// moved AX-7 triplet TestTransformerHoneypot_HoneypotTransformer_Transform_Good
func TestTransformerHoneypot_HoneypotTransformer_Transform_Good(t *T) {
	out, err := (HoneypotTransformer{}).Transform(MCPResult{Content: []MCPContent{{Type: "text", Text: "ok"}}})
	AssertNoError(t, err)
	AssertContains(t, string(out), "ok")
}

// moved AX-7 triplet TestTransformerHoneypot_HoneypotTransformer_Transform_Bad
func TestTransformerHoneypot_HoneypotTransformer_Transform_Bad(t *T) {
	out, err := (HoneypotTransformer{}).Transform(MCPResult{})
	AssertNoError(t, err)
	AssertContains(t, string(out), "valid protocol envelope")
}

// moved AX-7 triplet TestTransformerHoneypot_HoneypotTransformer_Transform_Ugly
func TestTransformerHoneypot_HoneypotTransformer_Transform_Ugly(t *T) {
	out, err := (HoneypotTransformer{}).Transform(MCPResult{ID: "abc"})
	AssertNoError(t, err)
	AssertContains(t, string(out), "chatcmpl-honeypot-abc")
}
