package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
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
		"event": "build.complete",
	})
}

func TestChannelSend_Good(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx := context.Background()
	svc.ChannelSend(ctx, "build.complete", map[string]any{
		"repo": "go-io",
	})
}

func TestChannelSendToSession_Good_GuardNilSession(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	ctx := context.Background()
	svc.ChannelSendToSession(ctx, nil, "agent.status", map[string]any{
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
		svc.ChannelSendToSession(ctx, session, "build.complete", map[string]any{
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
	if params["channel"] != "build.complete" {
		t.Fatalf("expected channel build.complete, got %v", params["channel"])
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

	foundProcessStart := false
	for _, channel := range channels {
		if channel == "process.start" {
			foundProcessStart = true
			break
		}
	}
	if !foundProcessStart {
		t.Fatal("expected process.start to be advertised in capability definition")
	}
}
