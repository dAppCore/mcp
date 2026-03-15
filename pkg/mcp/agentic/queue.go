// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"gopkg.in/yaml.v3"
)

// DispatchConfig controls agent dispatch behaviour.
type DispatchConfig struct {
	DefaultAgent    string `yaml:"default_agent"`
	DefaultTemplate string `yaml:"default_template"`
	WorkspaceRoot   string `yaml:"workspace_root"`
}

// AgentsConfig is the root of config/agents.yaml.
type AgentsConfig struct {
	Version     int            `yaml:"version"`
	Dispatch    DispatchConfig `yaml:"dispatch"`
	Concurrency map[string]int `yaml:"concurrency"` // per-agent type limits
}

// loadAgentsConfig reads config/agents.yaml from the code path.
func (s *PrepSubsystem) loadAgentsConfig() *AgentsConfig {
	paths := []string{
		filepath.Join(s.codePath, "core", "agent", "config", "agents.yaml"),
		filepath.Join(s.codePath, "core", "agent", ".core", "agents.yaml"),
		filepath.Join(s.codePath, "host-uk", "core", ".core", "agents.yaml"),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cfg AgentsConfig
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			continue
		}
		return &cfg
	}

	// Defaults: 1 claude, 3 gemini
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

// countRunningByAgent counts running workspaces for a specific agent type.
func (s *PrepSubsystem) countRunningByAgent(agent string) int {
	home, _ := os.UserHomeDir()
	wsRoot := filepath.Join(home, "Code", "host-uk", "core", ".core", "workspace")

	entries, err := os.ReadDir(wsRoot)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		st, err := readStatus(filepath.Join(wsRoot, entry.Name()))
		if err != nil || st.Status != "running" || st.Agent != agent {
			continue
		}

		// Verify PID is actually alive
		if st.PID > 0 {
			proc, err := os.FindProcess(st.PID)
			if err == nil && proc.Signal(syscall.Signal(0)) == nil {
				count++
			}
		}
	}

	return count
}

// canDispatchAgent checks if we're under the concurrency limit for a specific agent type.
func (s *PrepSubsystem) canDispatchAgent(agent string) bool {
	cfg := s.loadAgentsConfig()
	limit, ok := cfg.Concurrency[agent]
	if !ok || limit <= 0 {
		return true // no limit set or unlimited
	}
	return s.countRunningByAgent(agent) < limit
}

// canDispatch checks the legacy global limit (backwards compat).
func (s *PrepSubsystem) canDispatch() bool {
	return true // per-agent limits handle this now
}

// canDispatchFor checks per-agent concurrency.
func (s *PrepSubsystem) canDispatchFor(agent string) bool {
	return s.canDispatchAgent(agent)
}

// drainQueue finds the oldest queued workspace and spawns it if a slot is available.
func (s *PrepSubsystem) drainQueue() {
	home, _ := os.UserHomeDir()
	wsRoot := filepath.Join(home, "Code", "host-uk", "core", ".core", "workspace")

	entries, err := os.ReadDir(wsRoot)
	if err != nil {
		return
	}

	// Find oldest queued workspace that has a free slot for its agent type
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		wsDir := filepath.Join(wsRoot, entry.Name())
		st, err := readStatus(wsDir)
		if err != nil || st.Status != "queued" {
			continue
		}

		// Check per-agent limit
		if !s.canDispatchAgent(st.Agent) {
			continue
		}

		// Found a queued workspace with a free slot — spawn it
		srcDir := filepath.Join(wsDir, "src")
		prompt := "Read PROMPT.md for instructions. All context files (CLAUDE.md, TODO.md, CONTEXT.md, CONSUMERS.md, RECENT.md) are in the parent directory. Work in this directory."

		command, args, err := agentCommand(st.Agent, prompt)
		if err != nil {
			continue
		}

		outputFile := filepath.Join(wsDir, fmt.Sprintf("agent-%s.log", st.Agent))
		outFile, err := os.Create(outputFile)
		if err != nil {
			continue
		}

		cmd := exec.Command(command, args...)
		cmd.Dir = srcDir
		cmd.Stdout = outFile
		cmd.Stderr = outFile
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		if err := cmd.Start(); err != nil {
			outFile.Close()
			continue
		}

		// Update status to running
		st.Status = "running"
		st.PID = cmd.Process.Pid
		st.Runs++
		writeStatus(wsDir, st)

		// Monitor this one too
		go func() {
			cmd.Wait()
			outFile.Close()

			if st2, err := readStatus(wsDir); err == nil {
				st2.Status = "completed"
				st2.PID = 0
				writeStatus(wsDir, st2)
			}

			// Recursively drain — pick up next queued item
			s.drainQueue()
		}()

		return // Only spawn one at a time per drain call
	}
}
