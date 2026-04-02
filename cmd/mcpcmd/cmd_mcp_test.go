package mcpcmd

import (
	"context"
	"testing"

	"dappco.re/go/mcp/pkg/mcp"
)

func TestRunServe_Good_ShutsDownService(t *testing.T) {
	oldNew := newMCPService
	oldRun := runMCPService
	oldShutdown := shutdownMCPService
	oldWorkspace := workspaceFlag
	oldUnrestricted := unrestrictedFlag

	t.Cleanup(func() {
		newMCPService = oldNew
		runMCPService = oldRun
		shutdownMCPService = oldShutdown
		workspaceFlag = oldWorkspace
		unrestrictedFlag = oldUnrestricted
	})

	workspaceFlag = ""
	unrestrictedFlag = false

	var runCalled bool
	var shutdownCalled bool

	newMCPService = func(opts mcp.Options) (*mcp.Service, error) {
		return mcp.New(mcp.Options{})
	}
	runMCPService = func(svc *mcp.Service, ctx context.Context) error {
		runCalled = true
		return nil
	}
	shutdownMCPService = func(svc *mcp.Service, ctx context.Context) error {
		shutdownCalled = true
		return nil
	}

	if err := runServe(); err != nil {
		t.Fatalf("runServe() returned error: %v", err)
	}
	if !runCalled {
		t.Fatal("expected runMCPService to be called")
	}
	if !shutdownCalled {
		t.Fatal("expected shutdownMCPService to be called")
	}
}
