// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	coreio "forge.lthn.ai/core/go-io"
	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Workspace status file convention:
//
//   {workspace}/status.json  — current state of the workspace
//   {workspace}/BLOCKED.md   — question the agent needs answered (written by agent)
//   {workspace}/ANSWER.md    — response from human (written by reviewer)
//
// Status lifecycle:
//   running → completed     (normal finish)
//   running → blocked       (agent wrote BLOCKED.md and exited)
//   blocked → running       (resume after ANSWER.md provided)
//   running → failed        (agent crashed / non-zero exit)

// WorkspaceStatus represents the current state of an agent workspace.
type WorkspaceStatus struct {
	Status    string    `json:"status"`              // running, completed, blocked, failed
	Agent     string    `json:"agent"`               // gemini, claude, codex
	Repo      string    `json:"repo"`                // target repo
	Org       string    `json:"org,omitempty"`       // forge org (e.g. "core")
	Task      string    `json:"task"`                // task description
	Branch    string    `json:"branch,omitempty"`    // git branch name
	Issue     int       `json:"issue,omitempty"`     // forge issue number
	PID       int       `json:"pid,omitempty"`       // process ID (if running)
	StartedAt time.Time `json:"started_at"`          // when dispatch started
	UpdatedAt time.Time `json:"updated_at"`          // last status change
	Question  string    `json:"question,omitempty"`  // from BLOCKED.md
	Runs      int       `json:"runs"`                // how many times dispatched/resumed
	PRURL     string    `json:"pr_url,omitempty"`    // pull request URL (after PR created)
}

func writeStatus(wsDir string, status *WorkspaceStatus) error {
	status.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	return coreio.Local.Write(filepath.Join(wsDir, "status.json"), string(data))
}

func readStatus(wsDir string) (*WorkspaceStatus, error) {
	data, err := coreio.Local.Read(filepath.Join(wsDir, "status.json"))
	if err != nil {
		return nil, err
	}
	var s WorkspaceStatus
	if err := json.Unmarshal([]byte(data), &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// --- agentic_status tool ---

type StatusInput struct {
	Workspace string `json:"workspace,omitempty"` // specific workspace name, or empty for all
}

type StatusOutput struct {
	Workspaces []WorkspaceInfo `json:"workspaces"`
	Count      int             `json:"count"`
}

type WorkspaceInfo struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	Agent     string `json:"agent"`
	Repo      string `json:"repo"`
	Task      string `json:"task"`
	Age       string `json:"age"`
	Question  string `json:"question,omitempty"`
	Runs      int    `json:"runs"`
}

func (s *PrepSubsystem) registerStatusTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_status",
		Description: "List agent workspaces and their status (running, completed, blocked, failed). Shows blocked agents with their questions.",
	}, s.status)
}

func (s *PrepSubsystem) status(ctx context.Context, _ *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, StatusOutput, error) {
	wsRoot := s.workspaceRoot()

	entries, err := coreio.Local.List(wsRoot)
	if err != nil {
		return nil, StatusOutput{}, coreerr.E("status", "no workspaces found", err)
	}

	var workspaces []WorkspaceInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Filter by specific workspace if requested
		if input.Workspace != "" && name != input.Workspace {
			continue
		}

		wsDir := filepath.Join(wsRoot, name)
		info := WorkspaceInfo{Name: name}

		// Try reading status.json
		st, err := readStatus(wsDir)
		if err != nil {
			// Legacy workspace (no status.json) — check for log file
			logFiles, _ := filepath.Glob(filepath.Join(wsDir, "agent-*.log"))
			if len(logFiles) > 0 {
				info.Status = "completed"
			} else {
				info.Status = "unknown"
			}
			fi, _ := entry.Info()
			if fi != nil {
				info.Age = time.Since(fi.ModTime()).Truncate(time.Minute).String()
			}
			workspaces = append(workspaces, info)
			continue
		}

		info.Status = st.Status
		info.Agent = st.Agent
		info.Repo = st.Repo
		info.Task = st.Task
		info.Runs = st.Runs
		info.Age = time.Since(st.StartedAt).Truncate(time.Minute).String()

		// If status is "running", check if PID is still alive
		if st.Status == "running" && st.PID > 0 {
			proc, err := os.FindProcess(st.PID)
			if err != nil || proc.Signal(nil) != nil {
				// Process died — check for BLOCKED.md
				blockedPath := filepath.Join(wsDir, "src", "BLOCKED.md")
				if data, err := coreio.Local.Read(blockedPath); err == nil {
					info.Status = "blocked"
					info.Question = strings.TrimSpace(data)
					st.Status = "blocked"
					st.Question = info.Question
				} else {
					info.Status = "completed"
					st.Status = "completed"
				}
				writeStatus(wsDir, st)
			}
		}

		if st.Status == "blocked" {
			info.Question = st.Question
		}

		workspaces = append(workspaces, info)
	}

	return nil, StatusOutput{
		Workspaces: workspaces,
		Count:      len(workspaces),
	}, nil
}
