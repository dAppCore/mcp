// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"

	core "dappco.re/go"
)

// writeWorkspaceStatus seeds a workspace directory under the prep root with a
// status.json describing a tracked run.
func writeWorkspaceStatus(t *testing.T, sub *PrepSubsystem, name string, st *WorkspaceStatus) string {
	t.Helper()
	wsDir := core.Path(sub.workspaceRoot(), name)
	if err := coreio.Local.EnsureDir(wsDir); err != nil {
		t.Fatalf("EnsureDir %s: %v", wsDir, err)
	}
	sub.saveStatus(wsDir, st)
	if !coreio.Local.IsFile(core.Path(wsDir, "status.json")) {
		t.Fatalf("expected status.json to be written for %s", name)
	}
	return wsDir
}

func TestQueue_loadAgentsConfig_Ugly_DefaultsWhenMissing(t *testing.T) {
	sub := newPlanSub(t) // empty temp codePath, no agents.yaml

	cfg := sub.loadAgentsConfig()
	if cfg == nil {
		t.Fatal("expected non-nil default config")
	}
	if cfg.Dispatch.DefaultAgent != "claude" {
		t.Fatalf("expected default agent claude, got %q", cfg.Dispatch.DefaultAgent)
	}
	if cfg.Concurrency["claude"] != 1 {
		t.Fatalf("expected claude concurrency 1, got %d", cfg.Concurrency["claude"])
	}
}

func TestQueue_loadAgentsConfig_Good_ParsesYAML(t *testing.T) {
	sub := newPlanSub(t)
	cfgDir := core.Path(sub.codePath, ".core")
	if err := coreio.Local.EnsureDir(cfgDir); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	yamlBody := "version: 1\n" +
		"dispatch:\n" +
		"  default_agent: codex\n" +
		"  default_template: security\n" +
		"concurrency:\n" +
		"  codex: 4\n" +
		"rates:\n" +
		"  codex:\n" +
		"    reset_utc: \"06:00\"\n" +
		"    sustained_delay: 30\n" +
		"    burst_window: 2\n" +
		"    burst_delay: 5\n"
	if err := writeAtomic(core.Path(cfgDir, "agents.yaml"), yamlBody); err != nil {
		t.Fatalf("write agents.yaml: %v", err)
	}

	cfg := sub.loadAgentsConfig()
	if cfg.Dispatch.DefaultAgent != "codex" {
		t.Fatalf("expected codex, got %q", cfg.Dispatch.DefaultAgent)
	}
	if cfg.Concurrency["codex"] != 4 {
		t.Fatalf("expected codex concurrency 4, got %d", cfg.Concurrency["codex"])
	}
	if cfg.Rates["codex"].SustainedDelay != 30 {
		t.Fatalf("expected sustained_delay 30, got %d", cfg.Rates["codex"].SustainedDelay)
	}
}

func TestQueue_delayForAgent_Ugly_NoRateConfigZero(t *testing.T) {
	sub := newPlanSub(t)
	// Default config has no rates → zero delay.
	if d := sub.delayForAgent("claude"); d != 0 {
		t.Fatalf("expected zero delay with no rate config, got %v", d)
	}
}

func TestQueue_delayForAgent_Good_SustainedDelay(t *testing.T) {
	sub := newPlanSub(t)
	cfgDir := core.Path(sub.codePath, ".core")
	if err := coreio.Local.EnsureDir(cfgDir); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	// burst_window 0 disables burst, so we always get sustained_delay.
	yamlBody := "dispatch:\n  default_agent: codex\n" +
		"rates:\n  codex:\n    reset_utc: \"06:00\"\n    sustained_delay: 45\n    burst_window: 0\n"
	if err := writeAtomic(core.Path(cfgDir, "agents.yaml"), yamlBody); err != nil {
		t.Fatalf("write agents.yaml: %v", err)
	}

	if d := sub.delayForAgent("codex"); d != 45*time.Second {
		t.Fatalf("expected 45s sustained delay, got %v", d)
	}
}

func TestQueue_listWorkspaceDirs_Good_FlatAndNested(t *testing.T) {
	sub := newPlanSub(t)

	// Flat workspace with status.json.
	writeWorkspaceStatus(t, sub, "go-mcp-1", &WorkspaceStatus{Status: "running", Agent: "codex"})

	// Nested workspace: <root>/core/go-io-2/status.json.
	nested := core.Path(sub.workspaceRoot(), "core", "go-io-2")
	if err := coreio.Local.EnsureDir(nested); err != nil {
		t.Fatalf("EnsureDir nested: %v", err)
	}
	sub.saveStatus(nested, &WorkspaceStatus{Status: "completed", Agent: "claude"})

	dirs := sub.listWorkspaceDirs()
	if len(dirs) != 2 {
		t.Fatalf("expected 2 workspace dirs, got %d (%v)", len(dirs), dirs)
	}
}

func TestQueue_listWorkspaceDirs_Ugly_NoRoot(t *testing.T) {
	sub := newPlanSub(t)
	// workspaceRoot does not exist yet.
	if dirs := sub.listWorkspaceDirs(); dirs != nil {
		t.Fatalf("expected nil for missing workspace root, got %v", dirs)
	}
}

func TestQueue_countRunningByAgent_Good_SelfPIDCounts(t *testing.T) {
	sub := newPlanSub(t)
	// A running workspace whose PID is this test process — kill(pid,0) succeeds.
	writeWorkspaceStatus(t, sub, "live", &WorkspaceStatus{
		Status: "running",
		Agent:  "codex:gpt-5.4",
		PID:    core.Getpid(),
	})
	// A running workspace for a different agent — should not count.
	writeWorkspaceStatus(t, sub, "other", &WorkspaceStatus{
		Status: "running",
		Agent:  "claude",
		PID:    core.Getpid(),
	})
	// A completed workspace — should not count even for codex.
	writeWorkspaceStatus(t, sub, "done", &WorkspaceStatus{
		Status: "completed",
		Agent:  "codex",
		PID:    core.Getpid(),
	})

	if n := sub.countRunningByAgent("codex"); n != 1 {
		t.Fatalf("expected 1 running codex workspace, got %d", n)
	}
}

func TestQueue_canDispatchAgent_Good_UnderLimit(t *testing.T) {
	sub := newPlanSub(t)
	cfgDir := core.Path(sub.codePath, ".core")
	if err := coreio.Local.EnsureDir(cfgDir); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	if err := writeAtomic(core.Path(cfgDir, "agents.yaml"),
		"concurrency:\n  codex: 2\n"); err != nil {
		t.Fatalf("write agents.yaml: %v", err)
	}

	// One running codex run, limit 2 → can dispatch.
	writeWorkspaceStatus(t, sub, "live", &WorkspaceStatus{
		Status: "running",
		Agent:  "codex",
		PID:    core.Getpid(),
	})
	if !sub.canDispatchAgent("codex") {
		t.Fatal("expected canDispatchAgent true when under limit")
	}
}

func TestQueue_canDispatchAgent_Ugly_AtLimitBlocks(t *testing.T) {
	sub := newPlanSub(t)
	cfgDir := core.Path(sub.codePath, ".core")
	if err := coreio.Local.EnsureDir(cfgDir); err != nil {
		t.Fatalf("EnsureDir: %v", err)
	}
	if err := writeAtomic(core.Path(cfgDir, "agents.yaml"),
		"concurrency:\n  codex: 1\n"); err != nil {
		t.Fatalf("write agents.yaml: %v", err)
	}

	writeWorkspaceStatus(t, sub, "live", &WorkspaceStatus{
		Status: "running",
		Agent:  "codex",
		PID:    core.Getpid(),
	})
	if sub.canDispatchAgent("codex") {
		t.Fatal("expected canDispatchAgent false at limit")
	}
}

func TestStatus_readStatus_RoundTrip_Good(t *testing.T) {
	sub := newPlanSub(t)
	wsDir := writeWorkspaceStatus(t, sub, "rt", &WorkspaceStatus{
		Status: "blocked",
		Agent:  "codex",
		Repo:   "core/mcp",
		Task:   "do the thing",
		Issue:  42,
	})

	st, err := readStatus(wsDir)
	if err != nil {
		t.Fatalf("readStatus: %v", err)
	}
	if st.Status != "blocked" || st.Repo != "core/mcp" || st.Issue != 42 {
		t.Fatalf("round-trip mismatch: %+v", st)
	}
	if st.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be stamped by saveStatus")
	}
}

func TestStatus_readStatus_Bad_Missing(t *testing.T) {
	if _, err := readStatus(t.TempDir()); err == nil {
		t.Fatal("expected error reading status from dir with no status.json")
	}
}

func TestStatus_readStatus_Ugly_Malformed(t *testing.T) {
	dir := t.TempDir()
	if err := writeAtomic(core.JoinPath(dir, "status.json"), "{not valid json"); err != nil {
		t.Fatalf("write malformed: %v", err)
	}
	if _, err := readStatus(dir); err == nil {
		t.Fatal("expected error parsing malformed status.json")
	}
}
