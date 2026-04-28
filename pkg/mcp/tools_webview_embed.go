// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"sync"
	"time"

	core "dappco.re/go"
	"dappco.re/go/log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WebviewRenderInput contains parameters for rendering an embedded
// HTML view. The named view is stored and broadcast so connected clients
// (Claude Code sessions, CoreGUI windows, HTTP/SSE subscribers) can
// display the content.
//
//	input := WebviewRenderInput{
//	    ViewID:  "dashboard",
//	    HTML:    "<div id='app'>Loading...</div>",
//	    Title:   "Agent Dashboard",
//	    Width:   1024,
//	    Height:  768,
//	    State:   map[string]any{"theme": "dark"},
//	}
type WebviewRenderInput struct {
	ViewID string         `json:"view_id"`          // e.g. "dashboard"
	HTML   string         `json:"html"`             // rendered markup
	Title  string         `json:"title,omitempty"`  // e.g. "Agent Dashboard"
	Width  int            `json:"width,omitempty"`  // preferred width in pixels
	Height int            `json:"height,omitempty"` // preferred height in pixels
	State  map[string]any `json:"state,omitempty"`  // initial view state
}

// WebviewRenderOutput reports the result of rendering an embedded view.
//
//	// out.Success == true, out.ViewID == "dashboard"
type WebviewRenderOutput struct {
	Success   bool      `json:"success"`   // true when the view was stored and broadcast
	ViewID    string    `json:"view_id"`   // echoed view identifier
	UpdatedAt time.Time `json:"updatedAt"` // when the view was rendered
}

// WebviewUpdateInput contains parameters for updating the state of an
// existing embedded view. Callers may provide HTML to replace the markup,
// patch fields in the view state, or do both.
//
//	input := WebviewUpdateInput{
//	    ViewID: "dashboard",
//	    HTML:   "<div id='app'>Ready</div>",
//	    State:  map[string]any{"count": 42},
//	    Merge:  true,
//	}
type WebviewUpdateInput struct {
	ViewID string         `json:"view_id"`         // e.g. "dashboard"
	HTML   string         `json:"html,omitempty"`  // replacement markup (optional)
	Title  string         `json:"title,omitempty"` // e.g. "Agent Dashboard"
	State  map[string]any `json:"state,omitempty"` // partial state update
	Merge  bool           `json:"merge,omitempty"` // merge state (default) or replace when false
}

// WebviewUpdateOutput reports the result of updating an embedded view.
//
//	// out.Success == true, out.ViewID == "dashboard"
type WebviewUpdateOutput struct {
	Success   bool      `json:"success"`   // true when the view was updated and broadcast
	ViewID    string    `json:"view_id"`   // echoed view identifier
	UpdatedAt time.Time `json:"updatedAt"` // when the view was last updated
}

// embeddedView captures the live state of a rendered UI view. Instances
// are kept per ViewID inside embeddedViewRegistry.
type embeddedView struct {
	ViewID    string
	Title     string
	HTML      string
	Width     int
	Height    int
	State     map[string]any
	UpdatedAt time.Time
}

// embeddedViewRegistry stores the most recent render/update state for each
// view so new subscribers can pick up the current UI on connection.
// Operations are guarded by embeddedViewMu.
var (
	embeddedViewMu       sync.RWMutex
	embeddedViewRegistry = map[string]*embeddedView{}
)

// ChannelWebviewRender is the channel used to broadcast webview_render events.
const ChannelWebviewRender = "webview.render"

// ChannelWebviewUpdate is the channel used to broadcast webview_update events.
const ChannelWebviewUpdate = "webview.update"

// webviewRender handles the webview_render tool call.
func (s *Service) webviewRender(ctx context.Context, req *mcp.CallToolRequest, input WebviewRenderInput) (*mcp.CallToolResult, WebviewRenderOutput, error) {
	s.logger.Info("MCP tool execution", "tool", "webview_render", "view", input.ViewID, "user", log.Username())

	if core.Trim(input.ViewID) == "" {
		return nil, WebviewRenderOutput{}, log.E("webviewRender", "view_id is required", nil)
	}

	now := time.Now()
	view := &embeddedView{
		ViewID:    input.ViewID,
		Title:     input.Title,
		HTML:      input.HTML,
		Width:     input.Width,
		Height:    input.Height,
		State:     cloneStateMap(input.State),
		UpdatedAt: now,
	}

	embeddedViewMu.Lock()
	embeddedViewRegistry[input.ViewID] = view
	embeddedViewMu.Unlock()

	s.ChannelSend(ctx, ChannelWebviewRender, map[string]any{
		"view_id":   view.ViewID,
		"title":     view.Title,
		"html":      view.HTML,
		"width":     view.Width,
		"height":    view.Height,
		"state":     cloneStateMap(view.State),
		"updatedAt": view.UpdatedAt,
	})

	return nil, WebviewRenderOutput{
		Success:   true,
		ViewID:    view.ViewID,
		UpdatedAt: view.UpdatedAt,
	}, nil
}

// webviewUpdate handles the webview_update tool call.
func (s *Service) webviewUpdate(ctx context.Context, req *mcp.CallToolRequest, input WebviewUpdateInput) (*mcp.CallToolResult, WebviewUpdateOutput, error) {
	s.logger.Info("MCP tool execution", "tool", "webview_update", "view", input.ViewID, "user", log.Username())

	if core.Trim(input.ViewID) == "" {
		return nil, WebviewUpdateOutput{}, log.E("webviewUpdate", "view_id is required", nil)
	}

	now := time.Now()

	embeddedViewMu.Lock()
	view, ok := embeddedViewRegistry[input.ViewID]
	if !ok {
		// Updating a view that was never rendered creates one lazily so
		// clients that reconnect mid-session get a consistent snapshot.
		view = &embeddedView{ViewID: input.ViewID, State: map[string]any{}}
		embeddedViewRegistry[input.ViewID] = view
	}

	if input.HTML != "" {
		view.HTML = input.HTML
	}
	if input.Title != "" {
		view.Title = input.Title
	}
	if input.State != nil {
		merge := input.Merge || len(view.State) == 0
		if merge {
			if view.State == nil {
				view.State = map[string]any{}
			}
			for k, v := range input.State {
				view.State[k] = v
			}
		} else {
			view.State = cloneStateMap(input.State)
		}
	}
	view.UpdatedAt = now
	snapshot := *view
	snapshot.State = cloneStateMap(view.State)
	embeddedViewMu.Unlock()

	s.ChannelSend(ctx, ChannelWebviewUpdate, map[string]any{
		"view_id":   snapshot.ViewID,
		"title":     snapshot.Title,
		"html":      snapshot.HTML,
		"width":     snapshot.Width,
		"height":    snapshot.Height,
		"state":     snapshot.State,
		"updatedAt": snapshot.UpdatedAt,
	})

	return nil, WebviewUpdateOutput{
		Success:   true,
		ViewID:    snapshot.ViewID,
		UpdatedAt: snapshot.UpdatedAt,
	}, nil
}

// cloneStateMap returns a shallow copy of a state map.
//
//	cloned := cloneStateMap(map[string]any{"a": 1}) // cloned["a"] == 1
func cloneStateMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// lookupEmbeddedView returns the current snapshot of an embedded view, if any.
//
//	view, ok := lookupEmbeddedView("dashboard")
func lookupEmbeddedView(id string) (*embeddedView, bool) {
	embeddedViewMu.RLock()
	defer embeddedViewMu.RUnlock()
	view, ok := embeddedViewRegistry[id]
	if !ok {
		return nil, false
	}
	snapshot := *view
	snapshot.State = cloneStateMap(view.State)
	return &snapshot, true
}

// resetEmbeddedViews clears the registry. Intended for tests.
func resetEmbeddedViews() {
	embeddedViewMu.Lock()
	defer embeddedViewMu.Unlock()
	embeddedViewRegistry = map[string]*embeddedView{}
}
