package mcp

import (
	"testing"
	"time"
)

func TestIPC_HandleIPCEvents_Good(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	cancel, session, clientConn := connectNotificationSession(t, svc)
	defer cancel()
	defer session.Close()
	defer clientConn.Close()

	clientConn.SetDeadline(time.Now().Add(5 * time.Second))
	read := readNotificationMessageUntil(t, clientConn, func(msg map[string]any) bool {
		return msg["method"] == ChannelNotificationMethod
	})

	result := svc.HandleIPCEvents(nil, ChannelPush{
		Channel: "agent.completed",
		Data: map[string]any{
			"repo": "core/mcp",
			"ok":   true,
		},
	})
	if !result.OK {
		t.Fatalf("HandleIPCEvents() returned non-OK result: %#v", result.Value)
	}

	res := <-read
	if res.err != nil {
		t.Fatalf("failed to read channel notification: %v", res.err)
	}

	params, ok := res.msg["params"].(map[string]any)
	if !ok {
		t.Fatalf("expected params object, got %T", res.msg["params"])
	}
	if params["channel"] != "agent.completed" {
		t.Fatalf("expected channel agent.completed, got %#v", params["channel"])
	}

	payload, ok := params["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data object, got %T", params["data"])
	}
	if payload["repo"] != "core/mcp" || payload["ok"] != true {
		t.Fatalf("unexpected payload: %#v", payload)
	}
}

func TestIPC_HandleIPCEvents_Bad(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	result := svc.HandleIPCEvents(nil, ChannelPush{
		Channel: " \t ",
		Data:    map[string]any{"ok": false},
	})
	if result.OK {
		t.Fatal("expected empty ChannelPush channel to fail")
	}
	if _, ok := result.Value.(error); !ok {
		t.Fatalf("expected error result value, got %T", result.Value)
	}
}

func TestIPC_HandleIPCEvents_Ugly(t *testing.T) {
	svc, err := New(Options{})
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	cancel, session, clientConn := connectNotificationSession(t, svc)
	defer cancel()
	defer session.Close()
	defer clientConn.Close()

	clientConn.SetDeadline(time.Now().Add(5 * time.Second))
	read := readNotificationMessageUntil(t, clientConn, func(msg map[string]any) bool {
		params, ok := msg["params"].(map[string]any)
		return msg["method"] == ChannelNotificationMethod && ok && params["channel"] == "agent.edge"
	})

	result := svc.HandleIPCEvents(nil, ChannelPush{Channel: "agent.edge"})
	if !result.OK {
		t.Fatalf("HandleIPCEvents() returned non-OK result: %#v", result.Value)
	}

	res := <-read
	if res.err != nil {
		t.Fatalf("failed to read edge notification: %v", res.err)
	}
	params, ok := res.msg["params"].(map[string]any)
	if !ok {
		t.Fatalf("expected params object, got %T", res.msg["params"])
	}
	if _, ok := params["data"]; !ok {
		t.Fatalf("expected data key for nil ChannelPush data: %#v", params)
	}
	if params["data"] != nil {
		t.Fatalf("expected nil data, got %#v", params["data"])
	}
}
