package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"dappco.re/go/core"
	"forge.lthn.ai/core/go-process"
	"forge.lthn.ai/core/go-ws"
)

func TestRegister_Good_WiresOptionalServices(t *testing.T) {
	c := core.New()

	ps := &process.Service{}
	hub := ws.NewHub()

	if r := c.RegisterService("process", ps); !r.OK {
		t.Fatalf("failed to register process service: %v", r.Value)
	}
	if r := c.RegisterService("ws", hub); !r.OK {
		t.Fatalf("failed to register ws hub: %v", r.Value)
	}

	result := Register(c)
	if !result.OK {
		t.Fatalf("Register() failed: %v", result.Value)
	}

	svc, ok := result.Value.(*Service)
	if !ok {
		t.Fatalf("expected *Service, got %T", result.Value)
	}

	if svc.ProcessService() != ps {
		t.Fatalf("expected process service to be wired")
	}
	if svc.WSHub() != hub {
		t.Fatalf("expected ws hub to be wired")
	}

	tools := map[string]bool{}
	for _, rec := range svc.Tools() {
		tools[rec.Name] = true
	}
	if !tools["process_start"] {
		t.Fatal("expected process tools to be registered when process service is available")
	}
	if !tools["ws_start"] {
		t.Fatal("expected ws tools to be registered when ws hub is available")
	}
}

func TestHandleIPCEvents_Good_ForwardsProcessActions(t *testing.T) {
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
	received := make(chan map[string]any, 8)
	errCh := make(chan error, 1)
	go func() {
		for scanner.Scan() {
			var msg map[string]any
			if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
				errCh <- err
				return
			}
			received <- msg
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
			return
		}
		close(received)
	}()

	result := svc.HandleIPCEvents(nil, process.ActionProcessStarted{
		ID:      "proc-1",
		Command: "go",
		Args:    []string{"test", "./..."},
		Dir:     "/workspace",
		PID:     1234,
	})
	if !result.OK {
		t.Fatalf("HandleIPCEvents() returned non-OK result: %#v", result.Value)
	}

	deadline := time.NewTimer(5 * time.Second)
	defer deadline.Stop()

	for {
		select {
		case err := <-errCh:
			t.Fatalf("failed to read notification: %v", err)
		case msg, ok := <-received:
			if !ok {
				t.Fatal("notification stream closed before expected message arrived")
			}
			if msg["method"] != channelNotificationMethod {
				continue
			}

			params, ok := msg["params"].(map[string]any)
			if !ok {
				t.Fatalf("expected params object, got %T", msg["params"])
			}
			if params["channel"] != ChannelProcessStart {
				continue
			}

			payload, ok := params["data"].(map[string]any)
			if !ok {
				t.Fatalf("expected data object, got %T", params["data"])
			}
			if payload["id"] != "proc-1" || payload["command"] != "go" {
				t.Fatalf("unexpected payload: %#v", payload)
			}
			return
		case <-deadline.C:
			t.Fatal("timed out waiting for process start notification")
		}
	}
}
