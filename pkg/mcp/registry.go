// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"reflect"
	"time"

	core "dappco.re/go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// RESTHandler handles a tool call from a REST endpoint.
// It receives raw JSON input and returns the typed output or an error.
//
//	var h RESTHandler = func(ctx context.Context, body []byte) (any, error) {
//	    var input ReadFileInput
//	    json.Unmarshal(body, &input)
//	    return ReadFileOutput{Content: "...", Path: input.Path}, nil
//	}
type RESTHandler func(ctx context.Context, body []byte) (any, error)

// errInvalidRESTInput marks malformed JSON bodies for the REST bridge.
var errInvalidRESTInput = &restInputError{}

// restInputError preserves invalid-REST-input identity without stdlib
// error constructors so bridge.go can keep using errors.Is.
type restInputError struct {
	cause error
}

func (e *restInputError) Error() string {
	if e == nil || e.cause == nil {
		return "invalid REST input"
	}
	return "invalid REST input: " + e.cause.Error()
}

func (e *restInputError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *restInputError) Is(target error) bool {
	_, ok := target.(*restInputError)
	return ok
}

func invalidRESTInputError(cause error) error {
	return &restInputError{cause: cause}
}

// ToolRecord captures metadata about a registered MCP tool.
//
//	for _, rec := range svc.Tools() {
//	    fmt.Printf("tool=%s group=%s desc=%s\n", rec.Name, rec.Group, rec.Description)
//	}
type ToolRecord struct {
	Name         string         // e.g. "file_read"
	Description  string         // e.g. "Read the contents of a file"
	Group        string         // e.g. "files", "rag", "process"
	InputSchema  map[string]any // JSON Schema from Go struct reflection
	OutputSchema map[string]any // JSON Schema from Go struct reflection
	RESTHandler  RESTHandler    // REST-callable handler created at registration time
}

// AddToolRecorded registers a tool with the MCP server and records its metadata.
// This is a generic function that captures the In/Out types for schema extraction.
// It also creates a RESTHandler closure that can unmarshal JSON to the correct
// input type and call the handler directly, enabling the MCP-to-REST bridge.
//
//	svc, _ := mcp.New(mcp.Options{})
//	mcp.AddToolRecorded(svc, svc.Server(), "files", &mcp.Tool{Name: "file_read"},
//	    func(context.Context, *mcp.CallToolRequest, ReadFileInput) (*mcp.CallToolResult, ReadFileOutput, error) {
//	        return nil, ReadFileOutput{Path: "src/main.go"}, nil
//	    })
func AddToolRecorded[In, Out any](s *Service, server *mcp.Server, group string, t *mcp.Tool, h mcp.ToolHandlerFor[In, Out]) {
	// Set inputSchema from struct reflection if not already set.
	// Use server.AddTool (non-generic) to avoid auto-generated outputSchema.
	// The go-sdk's generic mcp.AddTool generates outputSchema from the Out type,
	// but Claude Code's protocol (2025-03-26) doesn't support outputSchema.
	// Removing it reduces tools/list from 214KB to ~74KB.
	if t.InputSchema == nil {
		t.InputSchema = structSchema(new(In))
		if t.InputSchema == nil {
			t.InputSchema = map[string]any{"type": "object"}
		}
	}
	// Wrap the typed handler into a generic ToolHandler.
	wrapped := func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var input In
		if req != nil && len(req.Params.Arguments) > 0 {
			if r := core.JSONUnmarshal(req.Params.Arguments, &input); !r.OK {
				if err, ok := r.Value.(error); ok {
					return nil, err
				}
			}
		}
		if err := s.authorizeToolAccess(ctx, req, t.Name, input); err != nil {
			return nil, err
		}
		result, output, err := h(ctx, req, input)
		if err != nil {
			return nil, err
		}
		if result != nil {
			return result, nil
		}
		data := core.JSONMarshalString(output)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: data}},
		}, nil
	}
	server.AddTool(t, wrapped)

	restHandler := func(ctx context.Context, body []byte) (any, error) {
		var input In
		if len(body) > 0 {
			if r := core.JSONUnmarshal(body, &input); !r.OK {
				if err, ok := r.Value.(error); ok {
					return nil, invalidRESTInputError(err)
				}
				return nil, invalidRESTInputError(nil)
			}
		}
		// nil: REST callers have no MCP request context.
		// Tool handlers called via REST must not dereference CallToolRequest.
		_, output, err := h(ctx, nil, input)
		return output, err
	}

	s.tools = append(s.tools, ToolRecord{
		Name:         t.Name,
		Description:  t.Description,
		Group:        group,
		InputSchema:  structSchema(new(In)),
		OutputSchema: structSchema(new(Out)),
		RESTHandler:  restHandler,
	})
}

func addToolRecorded[In, Out any](s *Service, server *mcp.Server, group string, t *mcp.Tool, h mcp.ToolHandlerFor[In, Out]) {
	AddToolRecorded(s, server, group, t, h)
}

// structSchema builds a simple JSON Schema from a struct's json tags via reflection.
// Returns nil for non-struct types or empty structs.
func structSchema(v any) map[string]any {
	t := reflect.TypeOf(v)
	if t == nil {
		return nil
	}
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	return schemaForType(t, map[reflect.Type]bool{})
}

// splitTag splits a struct tag value by commas.
func splitTag(tag string) []string {
	return core.Split(tag, ",")
}

// goTypeToJSONType maps Go types to JSON Schema types.
func goTypeToJSONType(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map, reflect.Struct:
		return "object"
	default:
		return "string"
	}
}

func schemaForType(t reflect.Type, seen map[reflect.Type]bool) map[string]any {
	if t == nil {
		return nil
	}

	for t.Kind() == reflect.Pointer {
		t = t.Elem()
		if t == nil {
			return nil
		}
	}

	if isTimeType(t) {
		return map[string]any{
			"type":   "string",
			"format": "date-time",
		}
	}

	switch t.Kind() {
	case reflect.Interface:
		return map[string]any{}

	case reflect.Struct:
		if seen[t] {
			return map[string]any{"type": "object"}
		}
		seen[t] = true

		properties := make(map[string]any)
		required := make([]string, 0, t.NumField())

		for f := range t.Fields() {
			f := f
			if !f.IsExported() {
				continue
			}

			jsonTag := f.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}

			name := f.Name
			isOptional := false
			if jsonTag != "" {
				parts := splitTag(jsonTag)
				name = parts[0]
				for _, p := range parts[1:] {
					if p == "omitempty" {
						isOptional = true
					}
				}
			}

			prop := schemaForType(f.Type, cloneSeenSet(seen))
			if prop == nil {
				prop = map[string]any{"type": goTypeToJSONType(f.Type)}
			}
			properties[name] = prop

			if !isOptional {
				required = append(required, name)
			}
		}

		schema := map[string]any{
			"type":       "object",
			"properties": properties,
		}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema

	case reflect.Slice, reflect.Array:
		schema := map[string]any{
			"type":  "array",
			"items": schemaForType(t.Elem(), cloneSeenSet(seen)),
		}
		return schema

	case reflect.Map:
		schema := map[string]any{
			"type": "object",
		}
		if t.Key().Kind() == reflect.String {
			if valueSchema := schemaForType(t.Elem(), cloneSeenSet(seen)); valueSchema != nil {
				schema["additionalProperties"] = valueSchema
			}
		}
		return schema

	default:
		if typeName := goTypeToJSONType(t); typeName != "" {
			return map[string]any{"type": typeName}
		}
	}

	return nil
}

func cloneSeenSet(seen map[reflect.Type]bool) map[reflect.Type]bool {
	if len(seen) == 0 {
		return map[reflect.Type]bool{}
	}
	clone := make(map[reflect.Type]bool, len(seen))
	for t := range seen {
		clone[t] = true
	}
	return clone
}

func isTimeType(t reflect.Type) bool {
	return t == reflect.TypeOf(time.Time{})
}
