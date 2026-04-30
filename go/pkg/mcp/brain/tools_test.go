// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core "dappco.re/go"
	"dappco.re/go/mcp/pkg/mcp/ide"
	"dappco.re/go/ws"
	"github.com/gorilla/websocket"
)

var brainToolTestUpgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool { return true },
}

func newConnectedBrainToolSubsystem(t *testing.T) (*Subsystem, <-chan ide.BridgeMessage) {
	t.Helper()

	messages := make(chan ide.BridgeMessage, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := brainToolTestUpgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}
		defer conn.Close()

		for {
			var msg ide.BridgeMessage
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			messages <- msg
		}
	}))

	ctx, cancel := context.WithCancel(context.Background())
	hub := ws.NewHub()
	go hub.Run(ctx)

	cfg := ide.DefaultConfig()
	cfg.LaravelWSURL = "ws" + core.TrimPrefix(srv.URL, "http")
	cfg.ReconnectInterval = 10 * time.Millisecond
	cfg.MaxReconnectInterval = 10 * time.Millisecond

	bridge := ide.NewBridge(hub, cfg)
	bridge.Start(ctx)
	waitBrainToolBridgeConnected(t, bridge)

	t.Cleanup(func() {
		bridge.Shutdown()
		cancel()
		srv.Close()
	})

	return New(bridge), messages
}

func waitBrainToolBridgeConnected(t *testing.T, bridge *ide.Bridge) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if bridge.Connected() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("bridge did not connect within timeout")
}

func readBrainToolBridgeMessage(t *testing.T, messages <-chan ide.BridgeMessage) ide.BridgeMessage {
	t.Helper()

	select {
	case msg := <-messages:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for bridge message")
		return ide.BridgeMessage{}
	}
}

func brainRepeatString(value string, count int) string {
	b := core.NewBuilder()
	for range count {
		b.WriteString(value)
	}
	return b.String()
}

func assertBrainOrgValidationError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected org validation error")
	}
	if !core.Contains(err.Error(), "org exceeds maximum length of 128 characters") {
		t.Fatalf("expected org length error, got %v", err)
	}
}

func TestBrainRemember_Good_OrgLengthBoundary(t *testing.T) {
	sub, messages := newConnectedBrainToolSubsystem(t)

	for _, tc := range []struct {
		name string
		org  string
	}{
		{name: "non_empty", org: "core"},
		{name: "empty", org: ""},
		{name: "boundary", org: brainRepeatString("a", brainOrgMaxLength)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, out, err := sub.brainRemember(context.Background(), nil, RememberInput{
				Content: "test memory",
				Type:    "observation",
				Org:     tc.org,
			})
			if err != nil {
				t.Fatalf("brainRemember failed: %v", err)
			}
			if !out.Success {
				t.Fatal("expected success=true")
			}

			msg := readBrainToolBridgeMessage(t, messages)
			if msg.Type != "brain_remember" {
				t.Fatalf("expected brain_remember message, got %q", msg.Type)
			}
			data, ok := msg.Data.(map[string]any)
			if !ok {
				t.Fatalf("expected bridge data map, got %T", msg.Data)
			}
			if data["org"] != tc.org {
				t.Fatalf("expected org %q, got %v", tc.org, data["org"])
			}
		})
	}
}

func TestBrainRemember_Bad_OrgTooLong(t *testing.T) {
	sub := New(nil)

	_, _, err := sub.brainRemember(context.Background(), nil, RememberInput{
		Content: "test memory",
		Type:    "observation",
		Org:     brainRepeatString("a", brainOrgMaxLength+1),
	})

	assertBrainOrgValidationError(t, err)
}

func TestBrainOrgValidation_Bad_RecallAndListRejectBeforeBridge(t *testing.T) {
	sub := New(nil)
	tooLong := brainRepeatString("a", brainOrgMaxLength+1)

	_, _, err := sub.brainRecall(context.Background(), nil, RecallInput{
		Query:  "test",
		Filter: RecallFilter{Org: tooLong},
	})
	assertBrainOrgValidationError(t, err)

	_, _, err = sub.brainList(context.Background(), nil, ListInput{
		Org: tooLong,
	})
	assertBrainOrgValidationError(t, err)
}
