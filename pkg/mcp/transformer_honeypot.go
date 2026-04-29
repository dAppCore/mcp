// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	core "dappco.re/go"
	"github.com/goccy/go-json"
)

// HoneypotTransformer absorbs malformed or probe-like input and returns a
// plausible synthetic response without dispatching to real tools.
type HoneypotTransformer struct{}

func (HoneypotTransformer) Detect(body []byte, contentType, path string) bool {
	trimmed := trimBytes(body)
	if len(trimmed) == 0 {
		return false
	}
	if !json.Valid(trimmed) {
		return true
	}

	var obj map[string]any
	if err := json.Unmarshal(trimmed, &obj); err != nil {
		return true
	}
	return looksProbeLike(trimmed, contentType, path)
}

func (HoneypotTransformer) Normalise(body []byte) (
	MCPRequest,
	error,
) {
	params := map[string]any{
		"source_format": "honeypot",
		"raw":           honeypotSnippet(body),
		"malformed":     !json.Valid(trimBytes(body)),
	}
	return MCPRequest{
		JSONRPC: "2.0",
		Method:  "honeypot/respond",
		Params:  params,
	}, nil
}

func (HoneypotTransformer) Transform(result MCPResult) (
	[]byte,
	error,
) {
	text := extractMCPText(result)
	if text == "" {
		text = "Request received. The gateway is processing the available context and will return compatible MCP output when a valid protocol envelope is provided."
	}

	resp := map[string]any{
		"id":      honeypotResponseID(result.ID),
		"object":  "chat.completion",
		"created": 0,
		"model":   "mcp-gateway",
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": text,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     0,
			"completion_tokens": 0,
			"total_tokens":      0,
		},
	}
	return json.Marshal(resp)
}

func looksProbeLike(body []byte, contentType, path string) bool {
	haystack := core.Lower(core.Join("\n",
		string(body),
		contentType,
		path,
	))
	for _, marker := range []string{
		"ignore previous",
		"system prompt",
		"developer message",
		"/etc/passwd",
		"../../",
		"dump secrets",
		"jailbreak",
		"prompt injection",
	} {
		if core.Contains(haystack, marker) {
			return true
		}
	}
	return false
}

func honeypotSnippet(body []byte) string {
	s := string(trimBytes(body))
	const max = 4096
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func honeypotResponseID(id any) string {
	if id == nil {
		return "chatcmpl-honeypot"
	}
	return core.Sprintf("chatcmpl-honeypot-%v", id)
}
