// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"path/filepath"
	"time"

	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// WatchInput is the input for agentic_watch.
type WatchInput struct {
	Workspaces   []string `json:"workspaces,omitempty"`
	PollInterval int      `json:"poll_interval,omitempty"`
	Timeout      int      `json:"timeout,omitempty"`
}

// WatchOutput is the result of watching one or more workspaces.
type WatchOutput struct {
	Success   bool          `json:"success"`
	Completed []WatchResult `json:"completed"`
	Failed    []WatchResult `json:"failed,omitempty"`
	Duration  string        `json:"duration"`
}

// WatchResult describes one workspace result.
type WatchResult struct {
	Workspace string `json:"workspace"`
	Agent     string `json:"agent"`
	Repo      string `json:"repo"`
	Status    string `json:"status"`
	Branch    string `json:"branch,omitempty"`
	Issue     int    `json:"issue,omitempty"`
	PRURL     string `json:"pr_url,omitempty"`
}

func (s *PrepSubsystem) registerWatchTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_watch",
		Description: "Watch running or queued agent workspaces until they finish and return a completion summary.",
	}, s.watch)
}

func (s *PrepSubsystem) watch(ctx context.Context, req *mcp.CallToolRequest, input WatchInput) (*mcp.CallToolResult, WatchOutput, error) {
	pollInterval := time.Duration(input.PollInterval) * time.Second
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}

	timeout := time.Duration(input.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}

	start := time.Now()
	deadline := start.Add(timeout)

	targets := input.Workspaces
	if len(targets) == 0 {
		targets = s.findActiveWorkspaces()
	}

	if len(targets) == 0 {
		return nil, WatchOutput{Success: true, Duration: "0s"}, nil
	}

	remaining := make(map[string]struct{}, len(targets))
	for _, workspace := range targets {
		remaining[workspace] = struct{}{}
	}

	completed := make([]WatchResult, 0, len(targets))
	failed := make([]WatchResult, 0)

	for len(remaining) > 0 {
		if time.Now().After(deadline) {
			for workspace := range remaining {
				failed = append(failed, WatchResult{
					Workspace: workspace,
					Status:    "timeout",
				})
			}
			break
		}

		select {
		case <-ctx.Done():
			return nil, WatchOutput{}, coreerr.E("watch", "cancelled", ctx.Err())
		case <-time.After(pollInterval):
		}

		_, statusOut, err := s.status(ctx, req, StatusInput{})
		if err != nil {
			return nil, WatchOutput{}, coreerr.E("watch", "failed to refresh status", err)
		}

		for _, info := range statusOut.Workspaces {
			if _, ok := remaining[info.Name]; !ok {
				continue
			}

			switch info.Status {
			case "completed", "merged", "ready-for-review":
				completed = append(completed, WatchResult{
					Workspace: info.Name,
					Agent:     info.Agent,
					Repo:      info.Repo,
					Status:    info.Status,
					Branch:    info.Branch,
					Issue:     info.Issue,
					PRURL:     info.PRURL,
				})
				delete(remaining, info.Name)
			case "failed", "blocked":
				failed = append(failed, WatchResult{
					Workspace: info.Name,
					Agent:     info.Agent,
					Repo:      info.Repo,
					Status:    info.Status,
					Branch:    info.Branch,
					Issue:     info.Issue,
					PRURL:     info.PRURL,
				})
				delete(remaining, info.Name)
			}
		}
	}

	return nil, WatchOutput{
		Success:   len(failed) == 0,
		Completed: completed,
		Failed:    failed,
		Duration:  time.Since(start).Round(time.Second).String(),
	}, nil
}

func (s *PrepSubsystem) findActiveWorkspaces() []string {
	wsDirs := s.listWorkspaceDirs()
	if len(wsDirs) == 0 {
		return nil
	}

	active := make([]string, 0, len(wsDirs))
	for _, wsDir := range wsDirs {
		st, err := readStatus(wsDir)
		if err != nil {
			continue
		}
		switch st.Status {
		case "running", "queued":
			active = append(active, filepath.Base(wsDir))
		}
	}
	return active
}

func (s *PrepSubsystem) resolveWorkspaceDir(name string) string {
	if filepath.IsAbs(name) {
		return name
	}
	return filepath.Join(s.workspaceRoot(), name)
}
