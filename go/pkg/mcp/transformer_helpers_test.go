// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"testing"

	"github.com/goccy/go-json"
)

func TestTransformerHelpers_mapFromAny_Good_AllShapes(t *testing.T) {
	// nil → empty map.
	if got := mapFromAny(nil); len(got) != 0 {
		t.Fatalf("mapFromAny(nil) = %v, want empty", got)
	}

	// map[string]any passes through unchanged.
	in := map[string]any{"a": 1.0}
	if got := mapFromAny(in); got["a"] != 1.0 {
		t.Fatalf("mapFromAny(map) lost data: %v", got)
	}

	// JSON RawMessage decodes to the object.
	raw := json.RawMessage(`{"k":"v"}`)
	if got := mapFromAny(raw); got["k"] != "v" {
		t.Fatalf("mapFromAny(RawMessage) = %v", got)
	}

	// String decodes JSON.
	if got := mapFromAny(`{"n":2}`); got["n"] != float64(2) {
		t.Fatalf("mapFromAny(string json) = %v", got)
	}

	// Arbitrary struct is marshalled then re-parsed.
	type payload struct {
		Field string `json:"field"`
	}
	if got := mapFromAny(payload{Field: "x"}); got["field"] != "x" {
		t.Fatalf("mapFromAny(struct) = %v", got)
	}
}

func TestTransformerHelpers_mapFromAny_Ugly_NonJSONString(t *testing.T) {
	// A non-JSON string is preserved under _raw rather than dropped.
	got := mapFromAny("not json at all")
	if got["_raw"] != "not json at all" {
		t.Fatalf("expected _raw fallback, got %v", got)
	}
}

func TestTransformerHelpers_parseArgumentString_Good_Bad(t *testing.T) {
	if got := parseArgumentString(""); len(got) != 0 {
		t.Fatalf("empty string should yield empty map, got %v", got)
	}
	if got := parseArgumentString(`{"a":1}`); got["a"] != float64(1) {
		t.Fatalf("parseArgumentString json = %v", got)
	}
	if got := parseArgumentString("garbage"); got["_raw"] != "garbage" {
		t.Fatalf("expected _raw fallback, got %v", got)
	}
}

func TestTransformerHelpers_indexByte_Good_Bad(t *testing.T) {
	if i := indexByte("hello", 'l'); i != 2 {
		t.Fatalf("indexByte = %d, want 2", i)
	}
	if i := indexByte("hello", 'z'); i != -1 {
		t.Fatalf("indexByte miss = %d, want -1", i)
	}
	if i := indexByte("", 'a'); i != -1 {
		t.Fatalf("indexByte empty = %d, want -1", i)
	}
}

func TestTransformerHelpers_indexAny_Good_Bad(t *testing.T) {
	if i := indexAny("abcdef", "xd"); i != 3 {
		t.Fatalf("indexAny = %d, want 3 (d)", i)
	}
	if i := indexAny("abc", "xyz"); i != -1 {
		t.Fatalf("indexAny miss = %d, want -1", i)
	}
}

func TestTransformerHelpers_trimBytes_Good(t *testing.T) {
	if got := string(trimBytes([]byte("  spaced  "))); got != "spaced" {
		t.Fatalf("trimBytes = %q", got)
	}
}

func TestTransformerHelpers_extractTextFromAny_Good_Variants(t *testing.T) {
	if got := extractTextFromAny(nil); got != nil {
		t.Fatalf("nil should yield nil, got %v", got)
	}
	if got := extractTextFromAny("hi"); len(got) != 1 || got[0] != "hi" {
		t.Fatalf("string = %v", got)
	}
	if got := extractTextFromAny(""); got != nil {
		t.Fatalf("empty string should yield nil, got %v", got)
	}
	if got := extractTextFromAny([]byte("bytes")); len(got) != 1 || got[0] != "bytes" {
		t.Fatalf("[]byte = %v", got)
	}
	// map with a "text" key.
	if got := extractTextFromAny(map[string]any{"text": "deep"}); len(got) != 1 || got[0] != "deep" {
		t.Fatalf("map text = %v", got)
	}
	// nested content slice.
	nested := map[string]any{"content": []any{"one", "two"}}
	if got := extractTextFromAny(nested); len(got) != 2 {
		t.Fatalf("nested content = %v", got)
	}
}

func TestTransformerHelpers_normaliseAnthropicTools_Good(t *testing.T) {
	out := normaliseAnthropicTools([]anthropicTool{
		{Name: "read_file", Description: "reads", InputSchema: map[string]any{"type": "object"}},
	})
	if len(out) != 1 {
		t.Fatalf("expected 1 normalised tool, got %d", len(out))
	}
	if out[0]["name"] != "read_file" || out[0]["description"] != "reads" {
		t.Fatalf("unexpected normalisation: %v", out[0])
	}
	if _, ok := out[0]["input_schema"]; !ok {
		t.Fatal("expected input_schema key")
	}
}

func TestTransformerHelpers_normaliseOpenAITools_Good_FunctionAndCustom(t *testing.T) {
	out := normaliseOpenAITools([]openAITool{
		{Type: "function", Function: openAIFunctionMetadata{Name: "search", Description: "find", Parameters: map[string]any{"type": "object"}}},
		{Type: "custom", Function: openAIFunctionMetadata{Name: "weird"}},
	})
	if len(out) != 2 {
		t.Fatalf("expected 2 normalised tools, got %d", len(out))
	}
	// Function tool maps to the MCP-native name/description/input_schema shape.
	if out[0]["name"] != "search" || out[0]["description"] != "find" {
		t.Fatalf("function tool normalisation: %v", out[0])
	}
	// Custom (non-function) tool keeps its type+function envelope.
	if out[1]["type"] != "custom" {
		t.Fatalf("custom tool should preserve type, got %v", out[1])
	}
}
