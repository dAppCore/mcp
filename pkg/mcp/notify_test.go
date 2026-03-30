package mcp

import (
	"context"
	"testing"
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
}
