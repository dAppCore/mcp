// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ResumeInput is the input for agentic_resume.
type ResumeInput struct {
	Workspace string `json:"workspace"`           // workspace name (e.g. "go-scm-1773581173")
	Answer    string `json:"answer,omitempty"`     // answer to the blocked question (written to ANSWER.md)
	Agent     string `json:"agent,omitempty"`      // override agent type (default: same as original)
	DryRun    bool   `json:"dry_run,omitempty"`    // preview without executing
}

// ResumeOutput is the output for agentic_resume.
type ResumeOutput struct {
	Success      bool   `json:"success"`
	Workspace    string `json:"workspace"`
	Agent        string `json:"agent"`
	PID          int    `json:"pid,omitempty"`
	OutputFile   string `json:"output_file,omitempty"`
	Prompt       string `json:"prompt,omitempty"`
}

func (s *PrepSubsystem) registerResumeTool(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "agentic_resume",
		Description: "Resume a blocked agent workspace. Writes ANSWER.md if an answer is provided, then relaunches the agent with instructions to read it and continue.",
	}, s.resume)
}

func (s *PrepSubsystem) resume(ctx context.Context, _ *mcp.CallToolRequest, input ResumeInput) (*mcp.CallToolResult, ResumeOutput, error) {
	if input.Workspace == "" {
		return nil, ResumeOutput{}, fmt.Errorf("workspace is required")
	}

	home, _ := os.UserHomeDir()
	wsDir := filepath.Join(home, "Code", "host-uk", "core", ".core", "workspace", input.Workspace)
	srcDir := filepath.Join(wsDir, "src")

	// Verify workspace exists
	if _, err := os.Stat(srcDir); err != nil {
		return nil, ResumeOutput{}, fmt.Errorf("workspace not found: %s", input.Workspace)
	}

	// Read current status
	st, err := readStatus(wsDir)
	if err != nil {
		return nil, ResumeOutput{}, fmt.Errorf("no status.json in workspace: %w", err)
	}

	if st.Status != "blocked" && st.Status != "failed" && st.Status != "completed" {
		return nil, ResumeOutput{}, fmt.Errorf("workspace is %s, not resumable (must be blocked, failed, or completed)", st.Status)
	}

	// Determine agent
	agent := st.Agent
	if input.Agent != "" {
		agent = input.Agent
	}

	// Write ANSWER.md if answer provided
	if input.Answer != "" {
		answerPath := filepath.Join(srcDir, "ANSWER.md")
		content := fmt.Sprintf("# Answer\n\n%s\n", input.Answer)
		if err := os.WriteFile(answerPath, []byte(content), 0644); err != nil {
			return nil, ResumeOutput{}, fmt.Errorf("failed to write ANSWER.md: %w", err)
		}
	}

	// Build resume prompt
	prompt := "You are resuming previous work in this workspace. "
	if input.Answer != "" {
		prompt += "Read ANSWER.md for the response to your question. "
	}
	prompt += "Read PROMPT.md for the original task. Read BLOCKED.md to see what you were stuck on. Continue working."

	if input.DryRun {
		return nil, ResumeOutput{
			Success:   true,
			Workspace: input.Workspace,
			Agent:     agent,
			Prompt:    prompt,
		}, nil
	}

	// Spawn agent as detached process (survives parent death)
	outputFile := filepath.Join(wsDir, fmt.Sprintf("agent-%s-run%d.log", agent, st.Runs+1))

	command, args, err := agentCommand(agent, prompt)
	if err != nil {
		return nil, ResumeOutput{}, err
	}

	devNull, _ := os.Open(os.DevNull)
	outFile, _ := os.Create(outputFile)
	cmd := exec.Command(command, args...)
	cmd.Dir = srcDir
	cmd.Stdin = devNull
	cmd.Stdout = outFile
	cmd.Stderr = outFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		outFile.Close()
		return nil, ResumeOutput{}, fmt.Errorf("failed to spawn %s: %w", agent, err)
	}

	// Update status
	st.Status = "running"
	st.PID = cmd.Process.Pid
	st.Runs++
	st.Question = ""
	writeStatus(wsDir, st)

	go func() {
		cmd.Wait()
		outFile.Close()
	}()

	return nil, ResumeOutput{
		Success:    true,
		Workspace:  input.Workspace,
		Agent:      agent,
		PID:        cmd.Process.Pid,
		OutputFile: outputFile,
	}, nil
}
