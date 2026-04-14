// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"errors"
	"testing"

	"dappco.re/go/core/process"
)

func TestToolRegistry_Good_RecordsTools(t *testing.T) {
	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	tools := svc.Tools()
	if len(tools) == 0 {
		t.Fatal("expected non-empty tool registry")
	}

	found := false
	for _, tr := range tools {
		if tr.Name == "file_read" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected file_read in tool registry")
	}
}

func TestToolRegistry_Good_SchemaExtraction(t *testing.T) {
	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	var record ToolRecord
	for _, tr := range svc.Tools() {
		if tr.Name == "file_read" {
			record = tr
			break
		}
	}
	if record.Name == "" {
		t.Fatal("file_read not found in registry")
	}

	if record.InputSchema == nil {
		t.Fatal("expected non-nil InputSchema for file_read")
	}

	props, ok := record.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties map in InputSchema")
	}

	if _, ok := props["path"]; !ok {
		t.Error("expected 'path' property in file_read InputSchema")
	}
}

func TestToolRegistry_Good_ToolCount(t *testing.T) {
	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	tools := svc.Tools()
	// Built-in tools (no ProcessService / WSHub / Subsystems):
	//   files (8):    file_read, file_write, file_delete, file_rename,
	//                 file_exists, file_edit, dir_list, dir_create
	//   language (2): lang_detect, lang_list
	//   metrics (2):  metrics_record, metrics_query
	//   rag (6):      rag_query, rag_search, rag_ingest, rag_index,
	//                 rag_retrieve, rag_collections
	//   webview (12): webview_connect, webview_disconnect, webview_navigate,
	//                 webview_click, webview_type, webview_query,
	//                 webview_console, webview_eval, webview_screenshot,
	//                 webview_wait, webview_render, webview_update
	//   ws (3):       ws_connect, ws_send, ws_close
	const expectedCount = 33
	if len(tools) != expectedCount {
		t.Errorf("expected %d tools, got %d", expectedCount, len(tools))
		for _, tr := range tools {
			t.Logf("  - %s (%s)", tr.Name, tr.Group)
		}
	}
}

func TestToolRegistry_Good_GroupAssignment(t *testing.T) {
	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	fileTools := []string{"file_read", "file_write", "file_delete", "file_rename", "file_exists", "file_edit", "dir_list", "dir_create"}
	langTools := []string{"lang_detect", "lang_list"}
	metricsTools := []string{"metrics_record", "metrics_query"}
	ragTools := []string{"rag_query", "rag_search", "rag_ingest", "rag_index", "rag_retrieve", "rag_collections"}
	webviewTools := []string{"webview_connect", "webview_disconnect", "webview_navigate", "webview_click", "webview_type", "webview_query", "webview_console", "webview_eval", "webview_screenshot", "webview_wait", "webview_render", "webview_update"}

	byName := make(map[string]ToolRecord)
	for _, tr := range svc.Tools() {
		byName[tr.Name] = tr
	}

	for _, name := range fileTools {
		tr, ok := byName[name]
		if !ok {
			t.Errorf("tool %s not found in registry", name)
			continue
		}
		if tr.Group != "files" {
			t.Errorf("tool %s: expected group 'files', got %q", name, tr.Group)
		}
	}

	for _, name := range langTools {
		tr, ok := byName[name]
		if !ok {
			t.Errorf("tool %s not found in registry", name)
			continue
		}
		if tr.Group != "language" {
			t.Errorf("tool %s: expected group 'language', got %q", name, tr.Group)
		}
	}

	for _, name := range metricsTools {
		tr, ok := byName[name]
		if !ok {
			t.Errorf("tool %s not found in registry", name)
			continue
		}
		if tr.Group != "metrics" {
			t.Errorf("tool %s: expected group 'metrics', got %q", name, tr.Group)
		}
	}

	for _, name := range ragTools {
		tr, ok := byName[name]
		if !ok {
			t.Errorf("tool %s not found in registry", name)
			continue
		}
		if tr.Group != "rag" {
			t.Errorf("tool %s: expected group 'rag', got %q", name, tr.Group)
		}
	}

	for _, name := range webviewTools {
		tr, ok := byName[name]
		if !ok {
			t.Errorf("tool %s not found in registry", name)
			continue
		}
		if tr.Group != "webview" {
			t.Errorf("tool %s: expected group 'webview', got %q", name, tr.Group)
		}
	}

	wsClientTools := []string{"ws_connect", "ws_send", "ws_close"}
	for _, name := range wsClientTools {
		tr, ok := byName[name]
		if !ok {
			t.Errorf("tool %s not found in registry", name)
			continue
		}
		if tr.Group != "ws" {
			t.Errorf("tool %s: expected group 'ws', got %q", name, tr.Group)
		}
	}
}

func TestToolRegistry_Good_ToolRecordFields(t *testing.T) {
	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	var record ToolRecord
	for _, tr := range svc.Tools() {
		if tr.Name == "file_write" {
			record = tr
			break
		}
	}
	if record.Name == "" {
		t.Fatal("file_write not found in registry")
	}

	if record.Name != "file_write" {
		t.Errorf("expected Name 'file_write', got %q", record.Name)
	}
	if record.Description == "" {
		t.Error("expected non-empty Description")
	}
	if record.Group == "" {
		t.Error("expected non-empty Group")
	}
	if record.InputSchema == nil {
		t.Error("expected non-nil InputSchema")
	}
	if record.OutputSchema == nil {
		t.Error("expected non-nil OutputSchema")
	}
}

func TestToolRegistry_Good_TimeSchemas(t *testing.T) {
	svc, err := New(Options{
		WorkspaceRoot:  t.TempDir(),
		ProcessService: &process.Service{},
	})
	if err != nil {
		t.Fatal(err)
	}

	byName := make(map[string]ToolRecord)
	for _, tr := range svc.Tools() {
		byName[tr.Name] = tr
	}

	metrics, ok := byName["metrics_record"]
	if !ok {
		t.Fatal("metrics_record not found in registry")
	}
	inputProps, ok := metrics.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected metrics_record input properties map")
	}
	dataSchema, ok := inputProps["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data schema for metrics_record input")
	}
	if got := dataSchema["type"]; got != "object" {
		t.Fatalf("expected metrics_record data type object, got %#v", got)
	}
	props, ok := metrics.OutputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected metrics_record output properties map")
	}
	timestamp, ok := props["timestamp"].(map[string]any)
	if !ok {
		t.Fatal("expected timestamp schema for metrics_record output")
	}
	if got := timestamp["type"]; got != "string" {
		t.Fatalf("expected metrics_record timestamp type string, got %#v", got)
	}
	if got := timestamp["format"]; got != "date-time" {
		t.Fatalf("expected metrics_record timestamp format date-time, got %#v", got)
	}

	processStart, ok := byName["process_start"]
	if !ok {
		t.Fatal("process_start not found in registry")
	}
	props, ok = processStart.OutputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected process_start output properties map")
	}
	startedAt, ok := props["startedAt"].(map[string]any)
	if !ok {
		t.Fatal("expected startedAt schema for process_start output")
	}
	if got := startedAt["type"]; got != "string" {
		t.Fatalf("expected process_start startedAt type string, got %#v", got)
	}
	if got := startedAt["format"]; got != "date-time" {
		t.Fatalf("expected process_start startedAt format date-time, got %#v", got)
	}
}

func TestToolRegistry_Bad_InvalidRESTInputIsClassified(t *testing.T) {
	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	var record ToolRecord
	for _, tr := range svc.Tools() {
		if tr.Name == "file_read" {
			record = tr
			break
		}
	}
	if record.Name == "" {
		t.Fatal("file_read not found in registry")
	}

	_, err = record.RESTHandler(context.Background(), []byte("{bad json"))
	if err == nil {
		t.Fatal("expected REST handler error for malformed JSON")
	}
	if !errors.Is(err, errInvalidRESTInput) {
		t.Fatalf("expected invalid REST input error, got %v", err)
	}
}
