// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
	"github.com/gorilla/websocket"
)

// TestToolsWSClient_WSConnect_Good dials a test WebSocket server and verifies
// the handshake completes and a client ID is assigned.
func TestToolsWSClient_WSConnect_Good(t *testing.T) {
	t.Cleanup(resetWSClients)

	server := startTestWSServer(t)
	defer server.Close()

	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	WSConnect := svc.wsConnect
	_, out, err := WSConnect(context.Background(), nil, WSConnectInput{
		URL:     "ws" + core.TrimPrefix(server.URL, "http") + "/ws",
		Timeout: 5,
	})
	if err != nil {
		t.Fatalf("wsConnect failed: %v", err)
	}
	if !out.Success {
		t.Fatal("expected Success=true")
	}
	if !core.HasPrefix(out.ID, "ws-") {
		t.Fatalf("expected ID prefix 'ws-', got %q", out.ID)
	}

	_, _, err = svc.wsClose(context.Background(), nil, WSCloseInput{ID: out.ID})
	if err != nil {
		t.Fatalf("wsClose failed: %v", err)
	}
}

// TestToolsWSClient_WSConnect_Bad rejects empty URLs.
func TestToolsWSClient_WSConnect_Bad(t *testing.T) {
	t.Cleanup(resetWSClients)

	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	WSConnect := svc.wsConnect
	_, _, err = WSConnect(context.Background(), nil, WSConnectInput{})
	if err == nil {
		t.Fatal("expected error for empty URL")
	}
}

// TestToolsWSClient_WSSendClose_Good sends a message on an open connection
// and then closes it.
func TestToolsWSClient_WSSendClose_Good(t *testing.T) {
	t.Cleanup(resetWSClients)

	server := startTestWSServer(t)
	defer server.Close()

	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	WSConnect := svc.wsConnect
	_, conn, err := WSConnect(context.Background(), nil, WSConnectInput{
		URL:     "ws" + core.TrimPrefix(server.URL, "http") + "/ws",
		Timeout: 5,
	})
	if err != nil {
		t.Fatalf("wsConnect failed: %v", err)
	}

	WSSend := svc.wsSend
	_, sendOut, err := WSSend(context.Background(), nil, WSSendInput{
		ID:      conn.ID,
		Message: "ping",
	})
	if err != nil {
		t.Fatalf("wsSend failed: %v", err)
	}
	if !sendOut.Success {
		t.Fatal("expected Success=true for wsSend")
	}
	if sendOut.Bytes != 4 {
		t.Fatalf("expected 4 bytes written, got %d", sendOut.Bytes)
	}

	WSSendClose := svc.wsClose
	_, closeOut, err := WSSendClose(context.Background(), nil, WSCloseInput{ID: conn.ID})
	if err != nil {
		t.Fatalf("wsClose failed: %v", err)
	}
	if !closeOut.Success {
		t.Fatal("expected Success=true for wsClose")
	}

	if _, ok := getWSClient(conn.ID); ok {
		t.Fatal("expected connection to be removed after close")
	}
}

// TestToolsWSClient_WSSend_Bad rejects unknown connection IDs.
func TestToolsWSClient_WSSend_Bad(t *testing.T) {
	t.Cleanup(resetWSClients)

	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	WSSend := svc.wsSend
	_, _, err = WSSend(context.Background(), nil, WSSendInput{ID: "ws-missing", Message: "x"})
	if err == nil {
		t.Fatal("expected error for unknown connection ID")
	}
}

// TestToolsWSClient_WSClose_Bad rejects closes for unknown connection IDs.
func TestToolsWSClient_WSClose_Bad(t *testing.T) {
	t.Cleanup(resetWSClients)

	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}

	WSClose := svc.wsClose
	_, _, err = WSClose(context.Background(), nil, WSCloseInput{ID: "ws-missing"})
	if err == nil {
		t.Fatal("expected error for unknown connection ID")
	}
}

// startTestWSServer returns an httptest.Server running a minimal echo WebSocket
// handler used by the ws_connect/ws_send tests.
func startTestWSServer(t *testing.T) *httptest.Server {
	t.Helper()

	upgrader := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	})
	return httptest.NewServer(mux)
}
