// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"os"
	"os/exec"
	"syscall"

	core "dappco.re/go/core"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ResumeInput is the input for agentic_resume.
//
//	input := ResumeInput{Workspace: "go-mcp-1700000000", Answer: "Use the shared notifier"}
type ResumeInput struct {
	Workspace string `json:"workspace"`         // workspace name (e.g. "go-scm-1773581173")
	Answer    string `json:"answer,omitempty"`  // answer to the blocked question (written to ANSWER.md)
	Agent     string `json:"agent,omitempty"`   // override agent type (default: same as original)
	DryRun    bool   `json:"dry_run,omitempty"` // preview without executing
}

// ResumeOutput is the output for agentic_resume.
//
//	// out.Success == true, out.PID > 0
type ResumeOutput struct {
	Success    bool   `json:"success"`
	Workspace  string `json:"workspace"`
	Agent      string `json:"agent"`
	PID        int    `json:"pid,omitempty"`
	OutputFile string `json:"output_file,omitempty"`
	Prompt     string `json:"prompt,omitempty"`
}

func (s *PrepSubsystem) registerResumeTool(svc *coremcp.Service) {
	server := svc.Server()
	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_resume",
		Description: "Resume a blocked agent workspace. Writes ANSWER.md if an answer is provided, then relaunches the agent with instructions to read it and continue.",
	}, s.resume)
}

func (s *PrepSubsystem) resume(ctx context.Context, _ *mcp.CallToolRequest, input ResumeInput) (*mcp.CallToolResult, ResumeOutput, error) {
	if input.Workspace == "" {
		return nil, ResumeOutput{}, coreerr.E("resume", "workspace is required", nil)
	}

	wsDir := core.Path(s.workspaceRoot(), input.Workspace)
	srcDir := core.Path(wsDir, "src")

	// Verify workspace exists
	if _, err := coreio.Local.List(srcDir); err != nil {
		return nil, ResumeOutput{}, coreerr.E("resume", "workspace not found: "+input.Workspace, nil)
	}

	// Read current status
	st, err := readStatus(wsDir)
	if err != nil {
		return nil, ResumeOutput{}, coreerr.E("resume", "no status.json in workspace", err)
	}

	if st.Status != "blocked" && st.Status != "failed" && st.Status != "completed" {
		return nil, ResumeOutput{}, coreerr.E("resume", "workspace is "+st.Status+", not resumable (must be blocked, failed, or completed)", nil)
	}

	// Determine agent
	agent := st.Agent
	if input.Agent != "" {
		agent = input.Agent
	}

	// Write ANSWER.md if answer provided
	if input.Answer != "" {
		answerPath := core.Path(srcDir, "ANSWER.md")
		content := core.Sprintf("# Answer\n\n%s\n", input.Answer)
		if err := writeAtomic(answerPath, content); err != nil {
			return nil, ResumeOutput{}, coreerr.E("resume", "failed to write ANSWER.md", err)
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
	outputFile := core.Path(wsDir, core.Sprintf("agent-%s-run%d.log", agent, st.Runs+1))

	command, args, err := agentCommand(agent, prompt)
	if err != nil {
		return nil, ResumeOutput{}, err
	}

	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return nil, ResumeOutput{}, coreerr.E("resume", "failed to open /dev/null", err)
	}
	defer devNull.Close()

	outFile, err := os.Create(outputFile)
	if err != nil {
		return nil, ResumeOutput{}, coreerr.E("resume", "failed to create log file", err)
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
		return nil, ResumeOutput{}, coreerr.E("resume", "failed to spawn "+agent, err)
	}

	// Update status
	st.Status = "running"
	st.PID = cmd.Process.Pid
	st.Runs++
	st.Question = ""
	s.saveStatus(wsDir, st)

	go func() {
		cmd.Wait()
		outFile.Close()

		postCtx := context.WithoutCancel(ctx)
		status := "completed"
		channel := coremcp.ChannelAgentComplete
		payload := map[string]any{
			"workspace": input.Workspace,
			"agent":     agent,
			"repo":      st.Repo,
			"branch":    st.Branch,
		}

		if data, err := coreio.Local.Read(core.Path(srcDir, "BLOCKED.md")); err == nil {
			status = "blocked"
			channel = coremcp.ChannelAgentBlocked
			st.Question = core.Trim(data)
			if st.Question != "" {
				payload["question"] = st.Question
			}
		}

		st.Status = status
		st.PID = 0
		s.saveStatus(wsDir, st)

		payload["status"] = status
		s.emitChannel(postCtx, channel, payload)
		s.emitChannel(postCtx, coremcp.ChannelAgentStatus, payload)
	}()

	return nil, ResumeOutput{
		Success:    true,
		Workspace:  input.Workspace,
		Agent:      agent,
		PID:        cmd.Process.Pid,
		OutputFile: outputFile,
	}, nil
}
