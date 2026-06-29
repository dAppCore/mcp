// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"testing"

	core "dappco.re/go"
)

func TestServiceBehaviour_NewService_Good_BuildsService(t *testing.T) {
	c := core.New()
	factory := NewService(Options{WorkspaceRoot: t.TempDir()})

	result := factory(c)
	if !result.OK {
		t.Fatalf("NewService factory failed: %v", result.Value)
	}
	svc, ok := result.Value.(*Service)
	if !ok {
		t.Fatalf("expected *Service, got %T", result.Value)
	}
	if svc.ServiceRuntime == nil {
		t.Fatal("expected ServiceRuntime to be wired")
	}
	if svc.Core() != c {
		t.Fatal("expected Core to be the one passed to the factory")
	}
}

func TestServiceBehaviour_OnStartup_Good_RegistersCommands(t *testing.T) {
	c := core.New()
	result := NewService(Options{WorkspaceRoot: t.TempDir()})(c)
	if !result.OK {
		t.Fatalf("NewService factory failed: %v", result.Value)
	}
	svc := result.Value.(*Service)

	if r := svc.OnStartup(context.Background()); !r.OK {
		t.Fatalf("OnStartup failed: %v", r.Value)
	}
	// The mcp and serve commands must now be present on the Core.
	cmds := map[string]bool{}
	for _, name := range c.Commands() {
		cmds[name] = true
	}
	if !cmds["mcp"] {
		t.Fatal("expected mcp command to be registered")
	}
	if !cmds["serve"] {
		t.Fatal("expected serve command to be registered")
	}
}

func TestServiceBehaviour_OnStartup_Ugly_NilRuntimeIsNoOp(t *testing.T) {
	// A Service without a ServiceRuntime must not panic and returns Ok(nil).
	svc, err := New(Options{WorkspaceRoot: t.TempDir()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if r := svc.OnStartup(context.Background()); !r.OK {
		t.Fatalf("expected Ok for nil-runtime OnStartup, got %v", r.Value)
	}
}

func TestNotifyBehaviour_debugNotify_Ugly_NilSafe(t *testing.T) {
	// Must not panic on a nil receiver or a logger-less service.
	var nilSvc *Service
	nilSvc.debugNotify("ignored")

	svc := &Service{}
	svc.debugNotify("also ignored", "k", "v")
}

func TestNotifyBehaviour_debugNotify_Good_WithLogger(t *testing.T) {
	svc := newTmpService(t) // has a logger
	svc.debugNotify("debug line", "tool", "metrics")
}

func TestNotifyBehaviour_coreNotifyError_Good(t *testing.T) {
	err := coreNotifyError("boom")
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if err.Error() != "boom" {
		t.Fatalf("error message = %q, want boom", err.Error())
	}
}

func TestNotifyBehaviour_snapshotSessions_Ugly_NilServer(t *testing.T) {
	if got := snapshotSessions(nil); got != nil {
		t.Fatalf("expected nil for nil server, got %v", got)
	}
}
