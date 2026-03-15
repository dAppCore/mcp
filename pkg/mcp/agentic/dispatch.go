// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	process "forge.lthn.ai/core/go-process"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DispatchInput is the input for agentic_dispatch.
type DispatchInput struct {
	Repo         string            `json:"repo"`                    // Target repo (e.g. "go-io")
	Org          string            `json:"org,omitempty"`           // Forge org (default "core")
	Task         string            `json:"task"`                    // What the agent should do
	Agent        string            `json:"agent,omitempty"`         // "gemini" (default), "codex", "claude"
	Template     string            `json:"template,omitempty"`      // "conventions", "security", "coding" (default)
	PlanTemplate string            `json:"plan_template,omitempty"` // Plan template: bug-fix, code-review, new-feature, refactor, feature-port
	Variables    map[string]string `json:"variables,omitempty"`     // Template variable substitution
	Issue        int               `json:"issue,omitempty"`         // Forge issue to work from
	DryRun       bool              `json:"dry_run,omitempty"`       // Preview without executing
}

// DispatchOutput is the output for agentic_dispatch.
type DispatchOutput struct {
	Success      bool   `json:"success"`
	Agent        string `json:"agent"`
	Repo         string `json:"repo"`
	WorkspaceDir string `json:"workspace_dir"`
	Prompt       string `json:"prompt,omitempty"`
	PID          int    `json:"pid,omitempty"`
	ProcessID    string `json:"process_id,omitempty"` // go-process ID for lifecycle management
	OutputFile   string `json:"output_file,omitempty"`
}

func (s *PrepSubsystem) registerDispatchTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_dispatch",
		Description: "Dispatch a subagent (Gemini, Codex, or Claude) to work on a task. Preps a sandboxed workspace first, then spawns the agent inside it. Templates: conventions, security, coding.",
	}, s.dispatch)
}

// agentCommand returns the command and args for a given agent type.
func agentCommand(agent, prompt string) (string, []string, error) {
	switch agent {
	case "gemini":
		return "gemini", []string{"-p", prompt, "--yolo"}, nil
	case "codex":
		return "codex", []string{"--approval-mode", "full-auto", "-q", prompt}, nil
	case "claude":
		return "claude", []string{"-p", prompt, "--dangerously-skip-permissions"}, nil
	default:
		return "", nil, fmt.Errorf("unknown agent: %s", agent)
	}
}

func (s *PrepSubsystem) dispatch(ctx context.Context, req *mcp.CallToolRequest, input DispatchInput) (*mcp.CallToolResult, DispatchOutput, error) {
	if input.Repo == "" {
		return nil, DispatchOutput{}, fmt.Errorf("repo is required")
	}
	if input.Task == "" {
		return nil, DispatchOutput{}, fmt.Errorf("task is required")
	}
	if input.Org == "" {
		input.Org = "core"
	}
	if input.Agent == "" {
		input.Agent = "gemini"
	}
	if input.Template == "" {
		input.Template = "coding"
	}

	// Step 1: Prep the sandboxed workspace
	prepInput := PrepInput{
		Repo:         input.Repo,
		Org:          input.Org,
		Issue:        input.Issue,
		Task:         input.Task,
		Template:     input.Template,
		PlanTemplate: input.PlanTemplate,
		Variables:    input.Variables,
	}
	_, prepOut, err := s.prepWorkspace(ctx, req, prepInput)
	if err != nil {
		return nil, DispatchOutput{}, fmt.Errorf("prep workspace failed: %w", err)
	}

	wsDir := prepOut.WorkspaceDir
	srcDir := filepath.Join(wsDir, "src")

	// The prompt is just: read PROMPT.md and do the work
	prompt := "Read PROMPT.md for instructions. All context files (CLAUDE.md, TODO.md, CONTEXT.md, CONSUMERS.md, RECENT.md) are in the parent directory. Work in this directory."

	if input.DryRun {
		// Read PROMPT.md for the dry run output
		promptContent, _ := os.ReadFile(filepath.Join(wsDir, "PROMPT.md"))
		return nil, DispatchOutput{
			Success:      true,
			Agent:        input.Agent,
			Repo:         input.Repo,
			WorkspaceDir: wsDir,
			Prompt:       string(promptContent),
		}, nil
	}

	// Step 2: Spawn agent via go-process
	command, args, err := agentCommand(input.Agent, prompt)
	if err != nil {
		return nil, DispatchOutput{}, err
	}

	proc, err := process.StartWithOptions(ctx, process.RunOptions{
		Command: command,
		Args:    args,
		Dir:     srcDir,
	})
	if err != nil {
		return nil, DispatchOutput{}, fmt.Errorf("failed to spawn %s: %w", input.Agent, err)
	}

	info := proc.Info()

	// Write initial status
	writeStatus(wsDir, &WorkspaceStatus{
		Status:    "running",
		Agent:     input.Agent,
		Repo:      input.Repo,
		Org:       input.Org,
		Task:      input.Task,
		PID:       info.PID,
		StartedAt: time.Now(),
		Runs:      1,
	})

	// Write output to log file when process completes
	outputFile := filepath.Join(wsDir, fmt.Sprintf("agent-%s.log", input.Agent))
	go func() {
		proc.Wait()
		os.WriteFile(outputFile, proc.OutputBytes(), 0644)
	}()

	return nil, DispatchOutput{
		Success:      true,
		Agent:        input.Agent,
		Repo:         input.Repo,
		WorkspaceDir: wsDir,
		PID:          info.PID,
		ProcessID:    info.ID,
		OutputFile:   outputFile,
	}, nil
}
