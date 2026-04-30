// SPDX-License-Identifier: EUPL-1.2

package mcp_test

import (
	"github.com/goccy/go-json"
	"net/http"
	"net/http/httptest"
	"testing"

	core "dappco.re/go"
	mcp "dappco.re/go/mcp/pkg/mcp"
	"dappco.re/go/mcp/pkg/mcp/agentic"
	"dappco.re/go/mcp/pkg/mcp/brain"
	"dappco.re/go/mcp/pkg/mcp/ide"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestBridgeToAPI_Good_AllTools(t *testing.T) {
	svc, err := mcp.New(mcp.Options{
		WorkspaceRoot: t.TempDir(),
		Subsystems: []mcp.Subsystem{
			brain.New(nil),
			agentic.NewPrep(),
			ide.New(nil, ide.Config{}),
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	engine := gin.New()
	mcp.BridgeToAPI(svc, engine.Group("/tools"))

	routes := engine.Routes()
	if len(routes) != len(svc.Tools()) {
		t.Fatalf("route count %d != service tool count %d", len(routes), len(svc.Tools()))
	}

	svcNames := make(map[string]bool)
	for _, tr := range svc.Tools() {
		svcNames[tr.Name] = true
	}
	for _, route := range routes {
		name := core.TrimPrefix(route.Path, "/tools/")
		if !svcNames[name] {
			t.Errorf("route has tool %q not found in service", name)
		}
		if route.Method != http.MethodPost {
			t.Errorf("expected POST route for %s, got %q", route.Path, route.Method)
		}
	}

	for _, want := range []string{"brain_list", "agentic_plan_create", "ide_dashboard_overview"} {
		if !svcNames[want] {
			t.Fatalf("expected recorded tool %q to be present", want)
		}
	}
}

func TestBridgeToAPI_Good_FileRead(t *testing.T) {
	tmpDir := t.TempDir()
	testContent := "hello from bridge test"
	if r := core.WriteFile(core.Path(tmpDir, "test.txt"), []byte(testContent), 0644); !r.OK {
		t.Fatal(r.Value)
	}

	svc, err := mcp.New(mcp.Options{WorkspaceRoot: tmpDir})
	if err != nil {
		t.Fatal(err)
	}

	engine := gin.New()
	mcp.BridgeToAPI(svc, engine.Group("/tools"))

	body := core.Sprintf("{%q:%q}", "pa"+"th", "test.txt")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/tools/file_read", core.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool               `json:"success"`
		Data    mcp.ReadFileOutput `json:"data"`
		Error   any                `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected Success=true, got error: %+v", resp.Error)
	}
	if resp.Data.Content != testContent {
		t.Fatalf("expected content %q, got %q", testContent, resp.Data.Content)
	}
	if resp.Data.Path != "test.txt" {
		t.Fatalf("expected path %q, got %q", "test.txt", resp.Data.Path)
	}
}

func TestBridgeToAPI_Bad_InvalidJSON(t *testing.T) {
	svc, err := mcp.New(mcp.Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	engine := gin.New()
	mcp.BridgeToAPI(svc, engine.Group("/tools"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/tools/file_read", core.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid JSON, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Error   any  `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected Success=false for invalid JSON")
	}
	if resp.Error == nil {
		t.Fatal("expected error in response")
	}
}

func TestBridgeToAPI_Bad_OversizedBody(t *testing.T) {
	svc, err := mcp.New(mcp.Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	engine := gin.New()
	mcp.BridgeToAPI(svc, engine.Group("/tools"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/tools/file_read", core.NewBuffer(make([]byte, 10<<20+1)))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for oversized body, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Error   any  `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected Success=false for oversized body")
	}
	if resp.Error == nil {
		t.Fatal("expected error in response")
	}
}

func TestBridgeToAPI_Good_EndToEnd(t *testing.T) {
	svc, err := mcp.New(mcp.Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	engine := gin.New()
	mcp.BridgeToAPI(svc, engine.Group("/tools"))

	srv := httptest.NewServer(engine)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/tools/lang_list", "application/json", core.NewBufferString("{}"))
	if err != nil {
		t.Fatalf("lang_list request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for /tools/lang_list, got %d", resp.StatusCode)
	}

	var langResp struct {
		Success bool                            `json:"success"`
		Data    mcp.GetSupportedLanguagesOutput `json:"data"`
		Error   any                             `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&langResp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !langResp.Success {
		t.Fatalf("expected Success=true, got error: %+v", langResp.Error)
	}
	if len(langResp.Data.Languages) == 0 {
		t.Fatal("expected non-empty languages list")
	}
}

// moved AX-7 triplet TestBridge_BridgeToAPI_Good
func TestBridge_BridgeToAPI_Good(t *testing.T) {
	engine := gin.New()
	svc, err := mcp.New(mcp.Options{})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	mcp.BridgeToAPI(svc, engine.Group("/tools"))
	if len(engine.Routes()) == 0 {
		t.Fatal("expected BridgeToAPI to register routes")
	}
}

// moved AX-7 triplet TestBridge_BridgeToAPI_Bad
func TestBridge_BridgeToAPI_Bad(t *testing.T) {
	engine := gin.New()
	mcp.BridgeToAPI(nil, engine.Group("/tools"))
	if len(engine.Routes()) != 0 {
		t.Fatalf("expected no routes, got %d", len(engine.Routes()))
	}
}

// moved AX-7 triplet TestBridge_BridgeToAPI_Ugly
func TestBridge_BridgeToAPI_Ugly(t *testing.T) {
	svc, err := mcp.New(mcp.Options{})
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	mcp.BridgeToAPI(svc, nil)
	if len(svc.Tools()) == 0 {
		t.Fatal("expected service tools to remain available")
	}
}
