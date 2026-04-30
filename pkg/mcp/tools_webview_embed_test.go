// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"testing"
)

// TestToolsWebviewEmbed_WebviewRender_Good registers a view and verifies the
// registry keeps the rendered HTML and state.
func TestToolsWebviewEmbed_WebviewRender_Good(t *testing.T) {
	t.Cleanup(resetEmbeddedViews)

	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	WebviewRender := svc.webviewRender
	_, out, err := WebviewRender(context.Background(), nil, WebviewRenderInput{
		ViewID: "dashboard",
		HTML:   "<p>hello</p>",
		Title:  "Demo",
		State:  map[string]any{"count": 1},
	})
	if err != nil {
		t.Fatalf("webviewRender returned error: %v", err)
	}
	if !out.Success {
		t.Fatal("expected Success=true")
	}
	if out.ViewID != "dashboard" {
		t.Fatalf("expected view id 'dashboard', got %q", out.ViewID)
	}
	if out.UpdatedAt.IsZero() {
		t.Fatal("expected non-zero UpdatedAt")
	}

	view, ok := lookupEmbeddedView("dashboard")
	if !ok {
		t.Fatal("expected view to be stored in registry")
	}
	if view.HTML != "<p>hello</p>" {
		t.Fatalf("expected HTML '<p>hello</p>', got %q", view.HTML)
	}
	if view.State["count"] != 1 {
		t.Fatalf("expected state.count=1, got %v", view.State["count"])
	}
}

// TestToolsWebviewEmbed_WebviewRender_Bad ensures empty view IDs are rejected.
func TestToolsWebviewEmbed_WebviewRender_Bad(t *testing.T) {
	t.Cleanup(resetEmbeddedViews)

	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	WebviewRender := svc.webviewRender
	_, _, err = WebviewRender(context.Background(), nil, WebviewRenderInput{})
	if err == nil {
		t.Fatal("expected error for empty view_id")
	}
}

// TestToolsWebviewEmbed_WebviewUpdate_Good merges a state patch into the
// previously rendered view.
func TestToolsWebviewEmbed_WebviewUpdate_Good(t *testing.T) {
	t.Cleanup(resetEmbeddedViews)

	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = svc.webviewRender(context.Background(), nil, WebviewRenderInput{
		ViewID: "dashboard",
		HTML:   "<p>hello</p>",
		State:  map[string]any{"count": 1},
	})
	if err != nil {
		t.Fatalf("seed render failed: %v", err)
	}

	WebviewUpdate := svc.webviewUpdate
	_, out, err := WebviewUpdate(context.Background(), nil, WebviewUpdateInput{
		ViewID: "dashboard",
		State:  map[string]any{"theme": "dark"},
		Merge:  true,
	})
	if err != nil {
		t.Fatalf("webviewUpdate returned error: %v", err)
	}
	if !out.Success {
		t.Fatal("expected Success=true")
	}

	view, ok := lookupEmbeddedView("dashboard")
	if !ok {
		t.Fatal("expected view to exist after update")
	}
	if view.State["count"] != 1 {
		t.Fatalf("expected count to persist after merge, got %v", view.State["count"])
	}
	if view.State["theme"] != "dark" {
		t.Fatalf("expected theme 'dark' after merge, got %v", view.State["theme"])
	}
}

// TestToolsWebviewEmbed_WebviewUpdate_Ugly updates a view that was never
// rendered and verifies a fresh registry entry is created.
func TestToolsWebviewEmbed_WebviewUpdate_Ugly(t *testing.T) {
	t.Cleanup(resetEmbeddedViews)

	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	WebviewUpdate := svc.webviewUpdate
	_, out, err := WebviewUpdate(context.Background(), nil, WebviewUpdateInput{
		ViewID: "ghost",
		HTML:   "<p>new</p>",
	})
	if err != nil {
		t.Fatalf("webviewUpdate returned error: %v", err)
	}
	if !out.Success {
		t.Fatal("expected Success=true for lazy-create update")
	}

	view, ok := lookupEmbeddedView("ghost")
	if !ok {
		t.Fatal("expected ghost view to be created lazily")
	}
	if view.HTML != "<p>new</p>" {
		t.Fatalf("expected HTML '<p>new</p>', got %q", view.HTML)
	}
}
