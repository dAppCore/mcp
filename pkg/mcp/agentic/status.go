// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"encoding/json"
	"os"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	coremcp "dappco.re/go/mcp/pkg/mcp"
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
//
//	status := WorkspaceStatus{
//	    Status: "blocked",
//	    Agent:  "claude",
//	    Repo:   "go-mcp",
//	}
type WorkspaceStatus struct {
	Status    string    `json:"status"`             // running, completed, blocked, failed
	Agent     string    `json:"agent"`              // gemini, claude, codex
	Repo      string    `json:"repo"`               // target repo
	Org       string    `json:"org,omitempty"`      // forge org (e.g. "core")
	Task      string    `json:"task"`               // task description
	Branch    string    `json:"branch,omitempty"`   // git branch name
	Issue     int       `json:"issue,omitempty"`    // forge issue number
	PID       int       `json:"pid,omitempty"`      // process ID (if running)
	StartedAt time.Time `json:"started_at"`         // when dispatch started
	UpdatedAt time.Time `json:"updated_at"`         // last status change
	Question  string    `json:"question,omitempty"` // from BLOCKED.md
	Runs      int       `json:"runs"`               // how many times dispatched/resumed
	PRURL     string    `json:"pr_url,omitempty"`   // pull request URL (after PR created)
}

func writeStatus(wsDir string, status *WorkspaceStatus) error {
	status.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(core.JoinPath(wsDir, "status.json"), string(data))
}

func (s *PrepSubsystem) saveStatus(wsDir string, status *WorkspaceStatus) {
	if err := writeStatus(wsDir, status); err != nil {
		coreerr.Warn("failed to write workspace status", "workspace", core.PathBase(wsDir), "err", err)
	}
}

func readStatus(wsDir string) (*WorkspaceStatus, error) {
	data, err := coreio.Local.Read(core.JoinPath(wsDir, "status.json"))
	if err != nil {
		return nil, err
	}
	var s WorkspaceStatus
	if r := core.JSONUnmarshal([]byte(data), &s); !r.OK {
		return nil, coreerr.E("readStatus", "failed to parse status.json", nil)
	}
	return &s, nil
}

// --- agentic_status tool ---

// StatusInput is the input for agentic_status.
//
//	input := StatusInput{Workspace: "go-mcp-1700000000"}
type StatusInput struct {
	Workspace string `json:"workspace,omitempty"` // specific workspace name, or empty for all
}

// StatusOutput is the output for agentic_status.
//
//	// out.Count == 2, len(out.Workspaces) == 2
type StatusOutput struct {
	Workspaces []WorkspaceInfo `json:"workspaces"`
	Count      int             `json:"count"`
}

// WorkspaceInfo summarizes a tracked workspace.
//
//	// ws.Name == "go-mcp-1700000000", ws.Status == "running"
type WorkspaceInfo struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Agent    string `json:"agent"`
	Repo     string `json:"repo"`
	Branch   string `json:"branch,omitempty"`
	Issue    int    `json:"issue,omitempty"`
	PRURL    string `json:"pr_url,omitempty"`
	Task     string `json:"task"`
	Age      string `json:"age"`
	Question string `json:"question,omitempty"`
	Runs     int    `json:"runs"`
}

func (s *PrepSubsystem) registerStatusTool(svc *coremcp.Service) {
	server := svc.Server()
	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_status",
		Description: "List agent workspaces and their status (running, completed, blocked, failed). Shows blocked agents with their questions.",
	}, s.status)
}

func (s *PrepSubsystem) status(ctx context.Context, _ *mcp.CallToolRequest, input StatusInput) (*mcp.CallToolResult, StatusOutput, error) {
	wsDirs := s.listWorkspaceDirs()

	var workspaces []WorkspaceInfo

	for _, wsDir := range wsDirs {
		name := core.PathBase(wsDir)

		// Filter by specific workspace if requested
		if input.Workspace != "" && name != input.Workspace {
			continue
		}

		info := WorkspaceInfo{Name: name}

		// Try reading status.json
		st, err := readStatus(wsDir)
		if err != nil {
			// Legacy workspace (no status.json) — check for log file
			logFiles := core.PathGlob(core.Path(wsDir, "agent-*.log"))
			if len(logFiles) > 0 {
				info.Status = "completed"
			} else {
				info.Status = "unknown"
			}
			if fi, err := os.Stat(wsDir); err == nil {
				info.Age = time.Since(fi.ModTime()).Truncate(time.Minute).String()
			}
			workspaces = append(workspaces, info)
			continue
		}

		info.Status = st.Status
		info.Agent = st.Agent
		info.Repo = st.Repo
		info.Branch = st.Branch
		info.Issue = st.Issue
		info.PRURL = st.PRURL
		info.Task = st.Task
		info.Runs = st.Runs
		info.Age = time.Since(st.StartedAt).Truncate(time.Minute).String()

		// If status is "running", check if PID is still alive
		if st.Status == "running" && st.PID > 0 {
			proc, err := os.FindProcess(st.PID)
			if err != nil || proc.Signal(nil) != nil {
				prevStatus := st.Status
				status := "completed"
				channel := coremcp.ChannelAgentComplete
				payload := map[string]any{
					"workspace": name,
					"agent":     st.Agent,
					"repo":      st.Repo,
					"branch":    st.Branch,
				}

				// Process died — check for BLOCKED.md
				blockedPath := core.Path(wsDir, "src", "BLOCKED.md")
				if data, err := coreio.Local.Read(blockedPath); err == nil {
					info.Status = "blocked"
					info.Question = core.Trim(data)
					st.Status = "blocked"
					st.Question = info.Question
					status = "blocked"
					channel = coremcp.ChannelAgentBlocked
					if st.Question != "" {
						payload["question"] = st.Question
					}
				} else {
					info.Status = "completed"
					st.Status = "completed"
				}
				s.saveStatus(wsDir, st)

				if prevStatus != status {
					payload["status"] = status
					s.emitChannel(ctx, channel, payload)
					s.emitChannel(ctx, coremcp.ChannelAgentStatus, payload)
				}
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
