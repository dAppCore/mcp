// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	"gopkg.in/yaml.v3"
)

// os.Create, os.Open, os.DevNull, os.Environ, os.FindProcess are used for
// process spawning and management — no core equivalents for these OS primitives.

// DispatchConfig controls agent dispatch behaviour.
type DispatchConfig struct {
	DefaultAgent    string `yaml:"default_agent"`
	DefaultTemplate string `yaml:"default_template"`
	WorkspaceRoot   string `yaml:"workspace_root"`
}

// RateConfig controls pacing between task dispatches.
type RateConfig struct {
	ResetUTC       string `yaml:"reset_utc"`       // Daily quota reset time (UTC), e.g. "06:00"
	DailyLimit     int    `yaml:"daily_limit"`     // Max requests per day (0 = unknown)
	MinDelay       int    `yaml:"min_delay"`       // Minimum seconds between task starts
	SustainedDelay int    `yaml:"sustained_delay"` // Delay when pacing for full-day use
	BurstWindow    int    `yaml:"burst_window"`    // Hours before reset where burst kicks in
	BurstDelay     int    `yaml:"burst_delay"`     // Delay during burst window
}

// AgentsConfig is the root of config/agents.yaml.
type AgentsConfig struct {
	Version     int                   `yaml:"version"`
	Dispatch    DispatchConfig        `yaml:"dispatch"`
	Concurrency map[string]int        `yaml:"concurrency"`
	Rates       map[string]RateConfig `yaml:"rates"`
}

// loadAgentsConfig reads config/agents.yaml from the code path.
func (s *PrepSubsystem) loadAgentsConfig() *AgentsConfig {
	paths := []string{
		core.Path(s.codePath, ".core", "agents.yaml"),
	}

	for _, path := range paths {
		data, err := coreio.Local.Read(path)
		if err != nil {
			continue
		}
		var cfg AgentsConfig
		if err := yaml.Unmarshal([]byte(data), &cfg); err != nil {
			continue
		}
		return &cfg
	}

	return &AgentsConfig{
		Dispatch: DispatchConfig{
			DefaultAgent:    "claude",
			DefaultTemplate: "coding",
		},
		Concurrency: map[string]int{
			"claude": 1,
			"gemini": 3,
		},
	}
}

// delayForAgent calculates how long to wait before spawning the next task
// for a given agent type, based on rate config and time of day.
func (s *PrepSubsystem) delayForAgent(agent string) time.Duration {
	cfg := s.loadAgentsConfig()
	rate, ok := cfg.Rates[agent]
	if !ok || rate.SustainedDelay == 0 {
		return 0
	}

	// Parse reset time (format: "HH:MM")
	resetHour, resetMin := 6, 0
	if parts := core.Split(rate.ResetUTC, ":"); len(parts) == 2 {
		if h, ok := parseSimpleInt(parts[0]); ok {
			resetHour = h
		}
		if m, ok := parseSimpleInt(parts[1]); ok {
			resetMin = m
		}
	}

	now := time.Now().UTC()
	resetToday := time.Date(now.Year(), now.Month(), now.Day(), resetHour, resetMin, 0, 0, time.UTC)
	if now.Before(resetToday) {
		// Reset hasn't happened yet today — reset was yesterday
		resetToday = resetToday.AddDate(0, 0, -1)
	}
	nextReset := resetToday.AddDate(0, 0, 1)
	hoursUntilReset := nextReset.Sub(now).Hours()

	// Burst mode: if within burst window of reset, use burst delay
	if rate.BurstWindow > 0 && hoursUntilReset <= float64(rate.BurstWindow) {
		return time.Duration(rate.BurstDelay) * time.Second
	}

	// Sustained mode
	return time.Duration(rate.SustainedDelay) * time.Second
}

// listWorkspaceDirs returns all workspace directories, including those
// nested one level deep (e.g. workspace/core/go-io-123/).
func (s *PrepSubsystem) listWorkspaceDirs() []string {
	wsRoot := s.workspaceRoot()
	entries, err := coreio.Local.List(wsRoot)
	if err != nil {
		return nil
	}

	var dirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := core.Path(wsRoot, entry.Name())
		// Check if this dir has a status.json (it's a workspace)
		if coreio.Local.IsFile(core.Path(path, "status.json")) {
			dirs = append(dirs, path)
			continue
		}
		// Otherwise check one level deeper (org subdirectory)
		subEntries, err := coreio.Local.List(path)
		if err != nil {
			continue
		}
		for _, sub := range subEntries {
			if sub.IsDir() {
				subPath := core.Path(path, sub.Name())
				if coreio.Local.IsFile(core.Path(subPath, "status.json")) {
					dirs = append(dirs, subPath)
				}
			}
		}
	}
	return dirs
}

// countRunningByAgent counts running workspaces for a specific agent type.
func (s *PrepSubsystem) countRunningByAgent(agent string) int {
	count := 0
	for _, wsDir := range s.listWorkspaceDirs() {
		st, err := readStatus(wsDir)
		if err != nil || st.Status != "running" {
			continue
		}
		stBase := core.SplitN(st.Agent, ":", 2)[0]
		if stBase != agent {
			continue
		}
		if st.PID > 0 {
			proc, err := os.FindProcess(st.PID)
			if err == nil && proc.Signal(syscall.Signal(0)) == nil {
				count++
			}
		}
	}
	return count
}

// baseAgent strips the model variant (gemini:flash → gemini).
func baseAgent(agent string) string {
	return core.SplitN(agent, ":", 2)[0]
}

// canDispatchAgent checks if we're under the concurrency limit for a specific agent type.
func (s *PrepSubsystem) canDispatchAgent(agent string) bool {
	cfg := s.loadAgentsConfig()
	base := baseAgent(agent)
	limit, ok := cfg.Concurrency[base]
	if !ok || limit <= 0 {
		return true
	}
	return s.countRunningByAgent(base) < limit
}

// parseSimpleInt parses a small non-negative integer from a string.
// Returns (value, true) on success, (0, false) on failure.
func parseSimpleInt(s string) (int, bool) {
	s = core.Trim(s)
	if s == "" {
		return 0, false
	}
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

// canDispatch is kept for backwards compat.
func (s *PrepSubsystem) canDispatch() bool {
	return true
}

// drainQueue finds the oldest queued workspace and spawns it if a slot is available.
// Applies rate-based delay between spawns.
func (s *PrepSubsystem) drainQueue() {
	for _, wsDir := range s.listWorkspaceDirs() {
		st, err := readStatus(wsDir)
		if err != nil || st.Status != "queued" {
			continue
		}

		if !s.canDispatchAgent(st.Agent) {
			continue
		}

		// Apply rate delay before spawning
		delay := s.delayForAgent(st.Agent)
		if delay > 0 {
			time.Sleep(delay)
		}

		// Re-check concurrency after delay (another task may have started)
		if !s.canDispatchAgent(st.Agent) {
			continue
		}

		srcDir := core.Path(wsDir, "src")
		prompt := "Read PROMPT.md for instructions. All context files (CLAUDE.md, TODO.md, CONTEXT.md, CONSUMERS.md, RECENT.md) are in the parent directory. Work in this directory."

		command, args, err := agentCommand(st.Agent, prompt)
		if err != nil {
			continue
		}

		outputFile := core.Path(wsDir, core.Sprintf("agent-%s.log", st.Agent))
		outFile, err := os.Create(outputFile)
		if err != nil {
			continue
		}

		devNull, err := os.Open(os.DevNull)
		if err != nil {
			outFile.Close()
			continue
		}

		cmd := exec.Command(command, args...)
		cmd.Dir = srcDir
		cmd.Stdin = devNull
		cmd.Stdout = outFile
		cmd.Stderr = outFile
		cmd.Env = append(os.Environ(), "TERM=dumb", "NO_COLOR=1", "CI=true")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		if err := cmd.Start(); err != nil {
			outFile.Close()
			devNull.Close()
			continue
		}
		devNull.Close()

		st.Status = "running"
		st.PID = cmd.Process.Pid
		st.Runs++
		s.saveStatus(wsDir, st)

		go func() {
			cmd.Wait()
			outFile.Close()

			if st2, err := readStatus(wsDir); err == nil {
				st2.Status = "completed"
				st2.PID = 0
				s.saveStatus(wsDir, st2)
			}

			// Ingest scan findings as issues
			s.ingestFindings(wsDir)

			s.drainQueue()
		}()

		return
	}
}
