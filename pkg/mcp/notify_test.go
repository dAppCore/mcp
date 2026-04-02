package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"slices"
	"testing"
	"time"
)

func TestSendNotificationToAllClients_Good(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx := context.Background()
	svc.SendNotificationToAllClients(ctx, "info", "test", map[string]any{
		"event": ChannelBuildComplete,
	})
}

func TestNotificationMethods_Good_NilService(t *testing.T) {
	var svc *Service

	ctx := context.Background()
	svc.SendNotificationToAllClients(ctx, "info", "test", map[string]any{"ok": true})
	svc.SendNotificationToSession(ctx, nil, "info", "test", map[string]any{"ok": true})
	svc.ChannelSend(ctx, ChannelBuildComplete, map[string]any{"ok": true})
	svc.ChannelSendToSession(ctx, nil, ChannelBuildComplete, map[string]any{"ok": true})

	for range svc.Sessions() {
		t.Fatal("expected no sessions from nil service")
	}
}

func TestNotificationMethods_Good_NilServer(t *testing.T) {
	svc := &Service{}

	ctx := context.Background()
	svc.SendNotificationToAllClients(ctx, "info", "test", map[string]any{"ok": true})
	svc.SendNotificationToSession(ctx, nil, "info", "test", map[string]any{"ok": true})
	svc.ChannelSend(ctx, ChannelBuildComplete, map[string]any{"ok": true})
	svc.ChannelSendToSession(ctx, nil, ChannelBuildComplete, map[string]any{"ok": true})

	for range svc.Sessions() {
		t.Fatal("expected no sessions from service without a server")
	}
}

func TestSendNotificationToAllClients_Good_CustomNotification(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := svc.server.Connect(ctx, &connTransport{conn: serverConn}, nil)
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer session.Close()

	clientConn.SetDeadline(time.Now().Add(5 * time.Second))
	scanner := bufio.NewScanner(clientConn)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	sent := make(chan struct{})
	go func() {
		svc.SendNotificationToAllClients(ctx, "info", "test", map[string]any{
			"event": ChannelBuildComplete,
		})
		close(sent)
	}()

	if !scanner.Scan() {
		t.Fatalf("failed to read notification: %v", scanner.Err())
	}

	select {
	case <-sent:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for notification send to complete")
	}

	var msg map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
		t.Fatalf("failed to unmarshal notification: %v", err)
	}
	if msg["method"] != loggingNotificationMethod {
		t.Fatalf("expected method %q, got %v", loggingNotificationMethod, msg["method"])
	}

	params, ok := msg["params"].(map[string]any)
	if !ok {
		t.Fatalf("expected params object, got %T", msg["params"])
	}
	if params["logger"] != "test" {
		t.Fatalf("expected logger test, got %v", params["logger"])
	}
	if params["level"] != "info" {
		t.Fatalf("expected level info, got %v", params["level"])
	}
	data, ok := params["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %T", params["data"])
	}
	if data["event"] != ChannelBuildComplete {
		t.Fatalf("expected event %s, got %v", ChannelBuildComplete, data["event"])
	}
}

func TestChannelSend_Good(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx := context.Background()
	svc.ChannelSend(ctx, ChannelBuildComplete, map[string]any{
		"repo": "go-io",
	})
}

func TestChannelSendToSession_Good_GuardNilSession(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx := context.Background()
	svc.ChannelSendToSession(ctx, nil, ChannelAgentStatus, map[string]any{
		"ok": true,
	})
}

func TestSendNotificationToSession_Good_GuardNilSession(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx := context.Background()
	svc.SendNotificationToSession(ctx, nil, "info", "test", map[string]any{
		"ok": true,
	})
}

func TestChannelSendToSession_Good_CustomNotification(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	session, err := svc.server.Connect(ctx, &connTransport{conn: serverConn}, nil)
	if err != nil {
		t.Fatalf("Connect() failed: %v", err)
	}
	defer session.Close()

	clientConn.SetDeadline(time.Now().Add(5 * time.Second))
	scanner := bufio.NewScanner(clientConn)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	sent := make(chan struct{})
	go func() {
		svc.ChannelSendToSession(ctx, session, ChannelBuildComplete, map[string]any{
			"repo": "go-io",
		})
		close(sent)
	}()

	if !scanner.Scan() {
		t.Fatalf("failed to read custom notification: %v", scanner.Err())
	}

	select {
	case <-sent:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for notification send to complete")
	}

	var msg map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
		t.Fatalf("failed to unmarshal notification: %v", err)
	}
	if msg["method"] != channelNotificationMethod {
		t.Fatalf("expected method %q, got %v", channelNotificationMethod, msg["method"])
	}

	params, ok := msg["params"].(map[string]any)
	if !ok {
		t.Fatalf("expected params object, got %T", msg["params"])
	}
	if params["channel"] != ChannelBuildComplete {
		t.Fatalf("expected channel %s, got %v", ChannelBuildComplete, params["channel"])
	}
	payload, ok := params["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %T", params["data"])
	}
	if payload["repo"] != "go-io" {
		t.Fatalf("expected repo go-io, got %v", payload["repo"])
	}
}

func TestChannelCapability_Good(t *testing.T) {
	caps := channelCapability()
	raw, ok := caps["claude/channel"]
	if !ok {
		t.Fatal("expected claude/channel capability entry")
	}

	cap, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("expected claude/channel to be a map, got %T", raw)
	}

	if cap["version"] == nil || cap["description"] == nil {
		t.Fatalf("expected capability to include version and description: %#v", cap)
	}

	channels, ok := cap["channels"].([]string)
	if !ok {
		t.Fatalf("expected channels to be []string, got %T", cap["channels"])
	}
	if len(channels) == 0 {
		t.Fatal("expected at least one channel in capability definition")
	}

	want := channelCapabilityChannels()
	if got, wantLen := len(channels), len(want); got != wantLen {
		t.Fatalf("expected %d channels, got %d", wantLen, got)
	}

	for _, channel := range want {
		if !slices.Contains(channels, channel) {
			t.Fatalf("expected channel %q to be advertised in capability definition", channel)
		}
	}
}
