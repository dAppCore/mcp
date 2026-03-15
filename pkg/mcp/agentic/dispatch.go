// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DispatchInput is the input for agentic_dispatch.
type DispatchInput struct {
	Repo    string `json:"repo"`              // Target repo (e.g. "go-io")
	Org     string `json:"org,omitempty"`      // Forge org (default "core")
	Task    string `json:"task"`              // What the agent should do
	Agent   string `json:"agent,omitempty"`   // "gemini" (default), "codex", "claude"
	Issue   int    `json:"issue,omitempty"`   // Forge issue to work from
	DryRun  bool   `json:"dry_run,omitempty"` // Preview without executing
}

// DispatchOutput is the output for agentic_dispatch.
type DispatchOutput struct {
	Success    bool   `json:"success"`
	Agent      string `json:"agent"`
	Repo       string `json:"repo"`
	WorkDir    string `json:"work_dir"`
	Prompt     string `json:"prompt,omitempty"`
	PID        int    `json:"pid,omitempty"`
	OutputFile string `json:"output_file,omitempty"`
}

func (s *PrepSubsystem) registerDispatchTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_dispatch",
		Description: "Dispatch a subagent (Gemini, Codex, or Claude) to work on a task in a specific repo. Preps workspace context first, then spawns the agent with a structured prompt.",
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

	repoPath := filepath.Join(s.codePath, "core", input.Repo)
	outputDir := filepath.Join(repoPath, ".core")

	// Step 1: Prep workspace
	prepInput := PrepInput{
		Repo:  input.Repo,
		Org:   input.Org,
		Issue: input.Issue,
	}
	_, _, _ = s.prepWorkspace(ctx, req, prepInput)

	// Step 2: Build prompt from prepped context
	prompt := s.buildPrompt(input, outputDir)

	if input.DryRun {
		return nil, DispatchOutput{
			Success: true,
			Agent:   input.Agent,
			Repo:    input.Repo,
			WorkDir: repoPath,
			Prompt:  prompt,
		}, nil
	}

	// Step 3: Spawn agent
	switch input.Agent {
	case "gemini":
		return s.spawnGemini(repoPath, prompt, input)
	case "codex":
		return s.spawnCodex(repoPath, prompt, input)
	case "claude":
		return s.spawnClaude(repoPath, prompt, input)
	default:
		return nil, DispatchOutput{}, fmt.Errorf("unknown agent: %s (use gemini, codex, or claude)", input.Agent)
	}
}

func (s *PrepSubsystem) buildPrompt(input DispatchInput, outputDir string) string {
	var prompt strings.Builder

	prompt.WriteString("You are working on the " + input.Repo + " repository.\n\n")

	// Include CLAUDE.md context
	if data, err := os.ReadFile(filepath.Join(outputDir, "CLAUDE.md")); err == nil {
		prompt.WriteString("## Project Context (CLAUDE.md)\n\n")
		prompt.WriteString(string(data))
		prompt.WriteString("\n\n")
	}

	// Include OpenBrain context
	if data, err := os.ReadFile(filepath.Join(outputDir, "context.md")); err == nil {
		prompt.WriteString(string(data))
		prompt.WriteString("\n\n")
	}

	// Include consumers
	if data, err := os.ReadFile(filepath.Join(outputDir, "consumers.md")); err == nil {
		prompt.WriteString(string(data))
		prompt.WriteString("\n\n")
	}

	// Include recent changes
	if data, err := os.ReadFile(filepath.Join(outputDir, "recent.md")); err == nil {
		prompt.WriteString(string(data))
		prompt.WriteString("\n\n")
	}

	// Include TODO if from issue
	if data, err := os.ReadFile(filepath.Join(outputDir, "todo.md")); err == nil {
		prompt.WriteString(string(data))
		prompt.WriteString("\n\n")
	}

	// The actual task
	prompt.WriteString("## Your Task\n\n")
	prompt.WriteString(input.Task)
	prompt.WriteString("\n\n")

	// Conventions
	prompt.WriteString("## Conventions\n\n")
	prompt.WriteString("- UK English (colour, organisation, centre)\n")
	prompt.WriteString("- Conventional commits: type(scope): description\n")
	prompt.WriteString("- Co-Author: Co-Authored-By: Virgil <virgil@lethean.io>\n")
	prompt.WriteString("- Licence: EUPL-1.2\n")
	prompt.WriteString("- Push to forge: ssh://git@forge.lthn.ai:2223/" + input.Org + "/" + input.Repo + ".git\n")

	return prompt.String()
}

func (s *PrepSubsystem) spawnGemini(repoPath, prompt string, input DispatchInput) (*mcp.CallToolResult, DispatchOutput, error) {
	// Write prompt to temp file (gemini -p has length limits)
	promptFile := filepath.Join(repoPath, ".core", "dispatch-prompt.md")
	os.WriteFile(promptFile, []byte(prompt), 0644)

	// Output file for capturing results
	outputFile := filepath.Join(repoPath, ".core", fmt.Sprintf("dispatch-%s-%d.log", input.Agent, time.Now().Unix()))

	cmd := exec.Command("gemini", "-p", prompt, "--yolo")
	cmd.Dir = repoPath

	outFile, _ := os.Create(outputFile)
	cmd.Stdout = outFile
	cmd.Stderr = outFile

	if err := cmd.Start(); err != nil {
		outFile.Close()
		return nil, DispatchOutput{}, fmt.Errorf("failed to spawn gemini: %w", err)
	}

	// Don't wait — let it run in background
	go func() {
		cmd.Wait()
		outFile.Close()
	}()

	return nil, DispatchOutput{
		Success:    true,
		Agent:      "gemini",
		Repo:       input.Repo,
		WorkDir:    repoPath,
		PID:        cmd.Process.Pid,
		OutputFile: outputFile,
	}, nil
}

func (s *PrepSubsystem) spawnCodex(repoPath, prompt string, input DispatchInput) (*mcp.CallToolResult, DispatchOutput, error) {
	outputFile := filepath.Join(repoPath, ".core", fmt.Sprintf("dispatch-%s-%d.log", input.Agent, time.Now().Unix()))

	cmd := exec.Command("codex", "--approval-mode", "full-auto", "-q", prompt)
	cmd.Dir = repoPath

	outFile, _ := os.Create(outputFile)
	cmd.Stdout = outFile
	cmd.Stderr = outFile

	if err := cmd.Start(); err != nil {
		outFile.Close()
		return nil, DispatchOutput{}, fmt.Errorf("failed to spawn codex: %w", err)
	}

	go func() {
		cmd.Wait()
		outFile.Close()
	}()

	return nil, DispatchOutput{
		Success:    true,
		Agent:      "codex",
		Repo:       input.Repo,
		WorkDir:    repoPath,
		PID:        cmd.Process.Pid,
		OutputFile: outputFile,
	}, nil
}

func (s *PrepSubsystem) spawnClaude(repoPath, prompt string, input DispatchInput) (*mcp.CallToolResult, DispatchOutput, error) {
	outputFile := filepath.Join(repoPath, ".core", fmt.Sprintf("dispatch-%s-%d.log", input.Agent, time.Now().Unix()))

	cmd := exec.Command("claude", "-p", prompt, "--dangerously-skip-permissions")
	cmd.Dir = repoPath

	outFile, _ := os.Create(outputFile)
	cmd.Stdout = outFile
	cmd.Stderr = outFile

	if err := cmd.Start(); err != nil {
		outFile.Close()
		return nil, DispatchOutput{}, fmt.Errorf("failed to spawn claude: %w", err)
	}

	go func() {
		cmd.Wait()
		outFile.Close()
	}()

	return nil, DispatchOutput{
		Success:    true,
		Agent:      "claude",
		Repo:       input.Repo,
		WorkDir:    repoPath,
		PID:        cmd.Process.Pid,
		OutputFile: outputFile,
	}, nil
}
