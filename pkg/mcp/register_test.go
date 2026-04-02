package mcp

import (
	"testing"

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
