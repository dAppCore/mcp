// SPDX-License-Identifier: EUPL-1.2

package mcpcmd

import (
	"context"
	"testing"

	core "dappco.re/go"
	"dappco.re/go/mcp/pkg/mcp"
)

func TestCmdMCP_RunServe_Good_ShutsDownService(t *testing.T) {
	restore := stubMCPService(t)
	defer restore()

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

func TestCmdMCP_RunServeAction_Good_PropagatesFlags(t *testing.T) {
	restore := stubMCPService(t)
	defer restore()

	workspaceFlag = ""
	unrestrictedFlag = false

	var gotOpts mcp.Options
	newMCPService = func(opts mcp.Options) (*mcp.Service, error) {
		gotOpts = opts
		return mcp.New(mcp.Options{WorkspaceRoot: t.TempDir()})
	}
	runMCPService = func(svc *mcp.Service, ctx context.Context) error {
		return nil
	}
	shutdownMCPService = func(svc *mcp.Service, ctx context.Context) error {
		return nil
	}

	tmp := t.TempDir()
	opts := core.NewOptions(core.Option{Key: "workspace", Value: tmp})

	result := runServeAction(opts)
	if !result.OK {
		t.Fatalf("expected OK, got %+v", result)
	}
	if gotOpts.WorkspaceRoot != tmp {
		t.Fatalf("expected workspace root %q, got %q", tmp, gotOpts.WorkspaceRoot)
	}
	if gotOpts.Unrestricted {
		t.Fatal("expected Unrestricted=false when --workspace is set")
	}
}

func TestCmdMCP_RunServeAction_Good_UnrestrictedFlag(t *testing.T) {
	restore := stubMCPService(t)
	defer restore()

	workspaceFlag = ""
	unrestrictedFlag = false

	var gotOpts mcp.Options
	newMCPService = func(opts mcp.Options) (*mcp.Service, error) {
		gotOpts = opts
		return mcp.New(mcp.Options{Unrestricted: true})
	}
	runMCPService = func(svc *mcp.Service, ctx context.Context) error {
		return nil
	}
	shutdownMCPService = func(svc *mcp.Service, ctx context.Context) error {
		return nil
	}

	opts := core.NewOptions(core.Option{Key: "unrestricted", Value: true})

	result := runServeAction(opts)
	if !result.OK {
		t.Fatalf("expected OK, got %+v", result)
	}
	if !gotOpts.Unrestricted {
		t.Fatal("expected Unrestricted=true when --unrestricted is set")
	}
}

func TestCmdMCP_RunServe_Bad_CreateServiceFails(t *testing.T) {
	restore := stubMCPService(t)
	defer restore()

	workspaceFlag = ""
	unrestrictedFlag = false

	sentinel := core.E("mcpcmd.test", "boom", nil)
	newMCPService = func(opts mcp.Options) (*mcp.Service, error) {
		return nil, sentinel
	}
	runMCPService = func(svc *mcp.Service, ctx context.Context) error {
		t.Fatal("runMCPService should not be called when New fails")
		return nil
	}
	shutdownMCPService = func(svc *mcp.Service, ctx context.Context) error {
		t.Fatal("shutdownMCPService should not be called when New fails")
		return nil
	}

	err := runServe()
	if err == nil {
		t.Fatal("expected error when newMCPService fails")
	}
}

func TestCmdMCP_RunServeAction_Bad_PropagatesFailure(t *testing.T) {
	restore := stubMCPService(t)
	defer restore()

	workspaceFlag = ""
	unrestrictedFlag = false

	newMCPService = func(opts mcp.Options) (*mcp.Service, error) {
		return nil, core.E("mcpcmd.test", "construction failed", nil)
	}
	runMCPService = func(svc *mcp.Service, ctx context.Context) error {
		return nil
	}
	shutdownMCPService = func(svc *mcp.Service, ctx context.Context) error {
		return nil
	}

	result := runServeAction(core.NewOptions())
	if result.OK {
		t.Fatal("expected runServeAction to fail when service creation fails")
	}
	if result.Value == nil {
		t.Fatal("expected error value on failure")
	}
}

func TestCmdMCP_FirstNonEmpty_Ugly_HandlesAllVariants(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{"no args", nil, ""},
		{"empty string", []string{""}, ""},
		{"all empty", []string{"", "", ""}, ""},
		{"first non-empty", []string{"foo", "bar"}, "foo"},
		{"skip empty", []string{"", "baz"}, "baz"},
		{"mixed", []string{"", "", "last"}, "last"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := firstNonEmpty(tc.values...)
			if got != tc.want {
				t.Fatalf("firstNonEmpty(%v) = %q, want %q", tc.values, got, tc.want)
			}
		})
	}
}

func TestCmdMCP_AddMCPCommands_Good_RegistersMcpTree(t *testing.T) {
	c := core.New()
	AddMCPCommands(c)

	commands := c.Commands()
	if len(commands) == 0 {
		t.Fatal("expected at least one registered command")
	}

	mustHave := map[string]bool{
		"mcp":       false,
		"mcp/serve": false,
	}
	for _, path := range commands {
		if _, ok := mustHave[path]; ok {
			mustHave[path] = true
		}
	}
	for path, present := range mustHave {
		if !present {
			t.Fatalf("expected command %q to be registered", path)
		}
	}
}

// stubMCPService captures the package-level function pointers and returns a
// restore hook so each test can mutate them without leaking into siblings.
func stubMCPService(t *testing.T) func() {
	t.Helper()
	oldNew := newMCPService
	oldRun := runMCPService
	oldShutdown := shutdownMCPService
	oldWorkspace := workspaceFlag
	oldUnrestricted := unrestrictedFlag

	return func() {
		newMCPService = oldNew
		runMCPService = oldRun
		shutdownMCPService = oldShutdown
		workspaceFlag = oldWorkspace
		unrestrictedFlag = oldUnrestricted
	}
}
