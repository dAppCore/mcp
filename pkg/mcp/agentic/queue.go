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
	MaxConcurrent  int    `yaml:"max_concurrent"`
	DefaultAgent   string `yaml:"default_agent"`
	DefaultTemplate string `yaml:"default_template"`
	WorkspaceRoot  string `yaml:"workspace_root"`
}

// AgentsConfig is the root of .core/agents.yaml.
type AgentsConfig struct {
	Version  int            `yaml:"version"`
	Dispatch DispatchConfig `yaml:"dispatch"`
}

// loadAgentsConfig reads .core/agents.yaml from the code path.
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

	// Defaults: unlimited concurrency
	return &AgentsConfig{
		Dispatch: DispatchConfig{
			MaxConcurrent:  0,
			DefaultAgent:   "claude",
			DefaultTemplate: "coding",
		},
	}
}

// countRunning counts how many agent workspaces have status "running"
// by checking if their PID is still alive.
func (s *PrepSubsystem) countRunning() int {
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
		if err != nil || st.Status != "running" {
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

// canDispatch checks if we're under the concurrency limit.
// Returns true if dispatch is allowed, false if it should be queued.
func (s *PrepSubsystem) canDispatch() bool {
	cfg := s.loadAgentsConfig()
	if cfg.Dispatch.MaxConcurrent <= 0 {
		return true // unlimited
	}
	return s.countRunning() < cfg.Dispatch.MaxConcurrent
}

// drainQueue finds the oldest queued workspace and spawns it if a slot is available.
func (s *PrepSubsystem) drainQueue() {
	if !s.canDispatch() {
		return
	}

	home, _ := os.UserHomeDir()
	wsRoot := filepath.Join(home, "Code", "host-uk", "core", ".core", "workspace")

	entries, err := os.ReadDir(wsRoot)
	if err != nil {
		return
	}

	// Find oldest queued workspace (entries are sorted by name which includes timestamp)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		wsDir := filepath.Join(wsRoot, entry.Name())
		st, err := readStatus(wsDir)
		if err != nil || st.Status != "queued" {
			continue
		}

		// Found a queued workspace — spawn it
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

		return // Only spawn one at a time
	}
}
