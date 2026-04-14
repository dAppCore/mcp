// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	core "dappco.re/go/core"
	"dappco.re/go/core/log"
	"github.com/gorilla/websocket"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WSConnectInput contains parameters for opening an outbound WebSocket
// connection from the MCP server. Each connection is given a stable ID that
// subsequent ws_send and ws_close calls use to address it.
//
//	input := WSConnectInput{URL: "wss://example.com/ws", Timeout: 10}
type WSConnectInput struct {
	URL     string            `json:"url"`                // e.g. "wss://example.com/ws"
	Headers map[string]string `json:"headers,omitempty"`  // custom request headers
	Timeout int               `json:"timeout,omitempty"`  // handshake timeout in seconds (default: 30)
}

// WSConnectOutput contains the result of opening a WebSocket connection.
//
//	// out.Success == true, out.ID == "ws-0af3…"
type WSConnectOutput struct {
	Success bool   `json:"success"` // true when the handshake completed
	ID      string `json:"id"`      // e.g. "ws-0af3…"
	URL     string `json:"url"`     // the URL that was dialled
}

// WSSendInput contains parameters for sending a message on an open
// WebSocket connection.
//
//	input := WSSendInput{ID: "ws-0af3…", Message: "ping"}
type WSSendInput struct {
	ID      string `json:"id"`                // e.g. "ws-0af3…"
	Message string `json:"message"`           // payload to send
	Binary  bool   `json:"binary,omitempty"`  // true to send a binary frame (payload is base64 text)
}

// WSSendOutput contains the result of sending a message.
//
//	// out.Success == true, out.ID == "ws-0af3…"
type WSSendOutput struct {
	Success bool   `json:"success"` // true when the message was written
	ID      string `json:"id"`      // e.g. "ws-0af3…"
	Bytes   int    `json:"bytes"`   // number of bytes written
}

// WSCloseInput contains parameters for closing a WebSocket connection.
//
//	input := WSCloseInput{ID: "ws-0af3…", Reason: "done"}
type WSCloseInput struct {
	ID     string `json:"id"`               // e.g. "ws-0af3…"
	Code   int    `json:"code,omitempty"`   // close code (default: 1000 - normal closure)
	Reason string `json:"reason,omitempty"` // human-readable reason
}

// WSCloseOutput contains the result of closing a WebSocket connection.
//
//	// out.Success == true, out.ID == "ws-0af3…"
type WSCloseOutput struct {
	Success bool   `json:"success"`           // true when the connection was closed
	ID      string `json:"id"`                // e.g. "ws-0af3…"
	Message string `json:"message,omitempty"` // e.g. "connection closed"
}

// wsClientConn tracks an outbound WebSocket connection tied to a stable ID.
type wsClientConn struct {
	ID        string
	URL       string
	conn      *websocket.Conn
	writeMu   sync.Mutex
	CreatedAt time.Time
}

// wsClientRegistry holds all live outbound WebSocket connections keyed by ID.
// Access is guarded by wsClientMu.
var (
	wsClientMu       sync.Mutex
	wsClientRegistry = map[string]*wsClientConn{}
)

// registerWSClientTools registers the outbound WebSocket client tools.
func (s *Service) registerWSClientTools(server *mcp.Server) {
	addToolRecorded(s, server, "ws", &mcp.Tool{
		Name:        "ws_connect",
		Description: "Open an outbound WebSocket connection. Returns a connection ID for subsequent ws_send and ws_close calls.",
	}, s.wsConnect)

	addToolRecorded(s, server, "ws", &mcp.Tool{
		Name:        "ws_send",
		Description: "Send a text or binary message on an open WebSocket connection identified by ID.",
	}, s.wsSend)

	addToolRecorded(s, server, "ws", &mcp.Tool{
		Name:        "ws_close",
		Description: "Close an open WebSocket connection identified by ID.",
	}, s.wsClose)
}

// wsConnect handles the ws_connect tool call.
func (s *Service) wsConnect(ctx context.Context, req *mcp.CallToolRequest, input WSConnectInput) (*mcp.CallToolResult, WSConnectOutput, error) {
	s.logger.Security("MCP tool execution", "tool", "ws_connect", "url", input.URL, "user", log.Username())

	if core.Trim(input.URL) == "" {
		return nil, WSConnectOutput{}, log.E("wsConnect", "url is required", nil)
	}

	timeout := time.Duration(input.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	dialer := websocket.Dialer{
		HandshakeTimeout: timeout,
	}

	headers := http.Header{}
	for k, v := range input.Headers {
		headers.Set(k, v)
	}

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, _, err := dialer.DialContext(dialCtx, input.URL, headers)
	if err != nil {
		log.Error("mcp: ws connect failed", "url", input.URL, "err", err)
		return nil, WSConnectOutput{}, log.E("wsConnect", "failed to connect", err)
	}

	id := newWSClientID()
	client := &wsClientConn{
		ID:        id,
		URL:       input.URL,
		conn:      conn,
		CreatedAt: time.Now(),
	}

	wsClientMu.Lock()
	wsClientRegistry[id] = client
	wsClientMu.Unlock()

	return nil, WSConnectOutput{
		Success: true,
		ID:      id,
		URL:     input.URL,
	}, nil
}

// wsSend handles the ws_send tool call.
func (s *Service) wsSend(ctx context.Context, req *mcp.CallToolRequest, input WSSendInput) (*mcp.CallToolResult, WSSendOutput, error) {
	s.logger.Info("MCP tool execution", "tool", "ws_send", "id", input.ID, "binary", input.Binary, "user", log.Username())

	if core.Trim(input.ID) == "" {
		return nil, WSSendOutput{}, log.E("wsSend", "id is required", nil)
	}

	client, ok := getWSClient(input.ID)
	if !ok {
		return nil, WSSendOutput{}, log.E("wsSend", "connection not found", nil)
	}

	messageType := websocket.TextMessage
	if input.Binary {
		messageType = websocket.BinaryMessage
	}

	client.writeMu.Lock()
	err := client.conn.WriteMessage(messageType, []byte(input.Message))
	client.writeMu.Unlock()
	if err != nil {
		log.Error("mcp: ws send failed", "id", input.ID, "err", err)
		return nil, WSSendOutput{}, log.E("wsSend", "failed to send message", err)
	}

	return nil, WSSendOutput{
		Success: true,
		ID:      input.ID,
		Bytes:   len(input.Message),
	}, nil
}

// wsClose handles the ws_close tool call.
func (s *Service) wsClose(ctx context.Context, req *mcp.CallToolRequest, input WSCloseInput) (*mcp.CallToolResult, WSCloseOutput, error) {
	s.logger.Info("MCP tool execution", "tool", "ws_close", "id", input.ID, "user", log.Username())

	if core.Trim(input.ID) == "" {
		return nil, WSCloseOutput{}, log.E("wsClose", "id is required", nil)
	}

	wsClientMu.Lock()
	client, ok := wsClientRegistry[input.ID]
	if ok {
		delete(wsClientRegistry, input.ID)
	}
	wsClientMu.Unlock()

	if !ok {
		return nil, WSCloseOutput{}, log.E("wsClose", "connection not found", nil)
	}

	code := input.Code
	if code == 0 {
		code = websocket.CloseNormalClosure
	}
	reason := input.Reason
	if reason == "" {
		reason = "closed"
	}

	client.writeMu.Lock()
	_ = client.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(code, reason),
		time.Now().Add(5*time.Second),
	)
	client.writeMu.Unlock()
	_ = client.conn.Close()

	return nil, WSCloseOutput{
		Success: true,
		ID:      input.ID,
		Message: "connection closed",
	}, nil
}

// newWSClientID returns a fresh identifier for an outbound WebSocket client.
//
//	id := newWSClientID() // "ws-0af3…"
func newWSClientID() string {
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	return "ws-" + hex.EncodeToString(buf[:])
}

// getWSClient returns a tracked outbound WebSocket client by ID, if any.
//
//	client, ok := getWSClient("ws-0af3…")
func getWSClient(id string) (*wsClientConn, bool) {
	wsClientMu.Lock()
	defer wsClientMu.Unlock()
	client, ok := wsClientRegistry[id]
	return client, ok
}

// resetWSClients drops all tracked outbound WebSocket clients. Intended for tests.
func resetWSClients() {
	wsClientMu.Lock()
	defer wsClientMu.Unlock()
	for id, client := range wsClientRegistry {
		_ = client.conn.Close()
		delete(wsClientRegistry, id)
	}
}
