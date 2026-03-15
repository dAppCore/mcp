// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

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
	OutputFile   string `json:"output_file,omitempty"`
}

func (s *PrepSubsystem) registerDispatchTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_dispatch",
		Description: "Dispatch a subagent (Gemini, Codex, or Claude) to work on a task. Preps a sandboxed workspace first, then spawns the agent inside it. Templates: conventions, security, coding.",
	}, s.dispatch)
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

	// Step 2: Spawn agent in src/ directory
	outputFile := filepath.Join(wsDir, fmt.Sprintf("agent-%s.log", input.Agent))

	var cmd *exec.Cmd
	switch input.Agent {
	case "gemini":
		cmd = exec.Command("gemini", "-p", prompt, "--yolo")
	case "codex":
		cmd = exec.Command("codex", "--approval-mode", "full-auto", "-q", prompt)
	case "claude":
		cmd = exec.Command("claude", "-p", prompt, "--dangerously-skip-permissions")
	default:
		return nil, DispatchOutput{}, fmt.Errorf("unknown agent: %s", input.Agent)
	}

	cmd.Dir = srcDir

	outFile, _ := os.Create(outputFile)
	cmd.Stdout = outFile
	cmd.Stderr = outFile

	if err := cmd.Start(); err != nil {
		outFile.Close()
		return nil, DispatchOutput{}, fmt.Errorf("failed to spawn %s: %w", input.Agent, err)
	}

	// Write initial status
	writeStatus(wsDir, &WorkspaceStatus{
		Status:    "running",
		Agent:     input.Agent,
		Repo:      input.Repo,
		Task:      input.Task,
		PID:       cmd.Process.Pid,
		StartedAt: time.Now(),
		Runs:      1,
	})

	go func() {
		cmd.Wait()
		outFile.Close()
	}()

	return nil, DispatchOutput{
		Success:      true,
		Agent:        input.Agent,
		Repo:         input.Repo,
		WorkspaceDir: wsDir,
		PID:          cmd.Process.Pid,
		OutputFile:   outputFile,
	}, nil
}
