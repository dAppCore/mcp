// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"net"
	"net/http"

	core "dappco.re/go"
	"dappco.re/go/ws"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WSStartInput contains parameters for starting the WebSocket server.
//
//	input := WSStartInput{Addr: ":9090"}
type WSStartInput struct {
	Addr string `json:"addr,omitempty"` // e.g. ":9090" (default: ":8080")
}

// WSStartOutput contains the result of starting the WebSocket server.
//
//	// out.Success == true, out.Addr == "127.0.0.1:9090"
type WSStartOutput struct {
	Success bool   `json:"success"`           // true when server started
	Addr    string `json:"addr"`              // actual listening address
	Message string `json:"message,omitempty"` // e.g. "WebSocket server started at ws://127.0.0.1:9090/ws"
}

// WSInfoInput takes no parameters.
//
//	input := WSInfoInput{}
type WSInfoInput struct{}

// WSInfoOutput contains WebSocket hub statistics.
//
//	// out.Clients == 3, out.Channels == 2
type WSInfoOutput struct {
	Clients  int `json:"clients"`  // number of connected WebSocket clients
	Channels int `json:"channels"` // number of active channels
}

// registerWSTools adds WebSocket tools to the MCP server.
// Returns false if WebSocket hub is not available.
func (s *Service) registerWSTools(server *mcp.Server) bool {
	if s.wsHub == nil {
		return false
	}

	addToolRecorded(s, server, "ws", &mcp.Tool{
		Name:        "ws_start",
		Description: "Start the WebSocket server for real-time process output streaming.",
	}, s.wsStart)

	addToolRecorded(s, server, "ws", &mcp.Tool{
		Name:        "ws_info",
		Description: "Get WebSocket hub statistics (connected clients and active channels).",
	}, s.wsInfo)

	return true
}

// wsStart handles the ws_start tool call.
func (s *Service) wsStart(ctx context.Context, req *mcp.CallToolRequest, input WSStartInput) (
	*mcp.CallToolResult,
	WSStartOutput,
	error,
) {
	if s.wsHub == nil {
		return nil, WSStartOutput{}, core.E("wsStart", "websocket hub unavailable", nil)
	}

	addr := input.Addr
	if addr == "" {
		addr = ":8080"
	}

	s.logger.Security("MCP tool execution", "tool", "ws_start", "addr", addr, "user", core.Username())

	s.wsMu.Lock()
	defer s.wsMu.Unlock()

	// Check if server is already running
	if s.wsServer != nil {
		return nil, WSStartOutput{
			Success: true,
			Addr:    s.wsAddr,
			Message: "WebSocket server already running",
		}, nil
	}

	// Create HTTP server with WebSocket handler
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.wsHub.Handler())

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start listener to get actual address
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		core.Error("mcp: ws start listen failed", "addr", addr, "err", err)
		return nil, WSStartOutput{}, core.E("wsStart", "failed to listen on "+addr, err)
	}

	actualAddr := ln.Addr().String()
	s.wsServer = server
	s.wsAddr = actualAddr

	// Start server in background
	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			core.Error("mcp: ws server error", "err", err)
		}
	}()

	return nil, WSStartOutput{
		Success: true,
		Addr:    actualAddr,
		Message: core.Sprintf("WebSocket server started at ws://%s/ws", actualAddr),
	}, nil
}

// wsInfo handles the ws_info tool call.
func (s *Service) wsInfo(ctx context.Context, req *mcp.CallToolRequest, input WSInfoInput) (
	*mcp.CallToolResult,
	WSInfoOutput,
	error,
) {
	if s.wsHub == nil {
		return nil, WSInfoOutput{}, core.E("wsInfo", "websocket hub unavailable", nil)
	}

	s.logger.Info("MCP tool execution", "tool", "ws_info", "user", core.Username())

	stats := s.wsHub.Stats()

	return nil, WSInfoOutput{
		Clients:  stats.Clients,
		Channels: stats.Channels,
	}, nil
}

// ProcessEventCallback forwards process lifecycle events to WebSocket clients.
//
//	cb := NewProcessEventCallback(hub)
//	cb.OnProcessOutput("proc-abc123", "build complete\n")
//	cb.OnProcessStatus("proc-abc123", "exited", 0)
type ProcessEventCallback struct {
	hub *ws.Hub
}

// NewProcessEventCallback creates a callback that forwards process events to WebSocket.
//
//	cb := NewProcessEventCallback(hub)
func NewProcessEventCallback(hub *ws.Hub) *ProcessEventCallback {
	return &ProcessEventCallback{hub: hub}
}

// OnProcessOutput forwards process output to WebSocket subscribers.
//
//	cb.OnProcessOutput("proc-abc123", "PASS\n")
func (c *ProcessEventCallback) OnProcessOutput(processID string, line string) {
	if c.hub != nil {
		if r := c.hub.SendProcessOutput(processID, line); !r.OK {
			core.Error("mcp: failed to send process output over websocket", "err", resultError(r))
		}
	}
}

// OnProcessStatus forwards process status changes to WebSocket subscribers.
//
//	cb.OnProcessStatus("proc-abc123", "exited", 0)
func (c *ProcessEventCallback) OnProcessStatus(processID string, status string, exitCode int) {
	if c.hub != nil {
		if r := c.hub.SendProcessStatus(processID, status, exitCode); !r.OK {
			core.Error("mcp: failed to send process status over websocket", "err", resultError(r))
		}
	}
}
