// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"context"
	"os"
	"os/exec"
	"syscall"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
	coremcp "dappco.re/go/mcp/pkg/mcp"
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
	Persona      string            `json:"persona,omitempty"`       // Persona: engineering/backend-architect, testing/api-tester, etc.
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

func (s *PrepSubsystem) registerDispatchTool(svc *coremcp.Service) {
	server := svc.Server()
	coremcp.AddToolRecorded(svc, server, "agentic", &mcp.Tool{
		Name:        "agentic_dispatch",
		Description: "Dispatch a subagent (Gemini, Codex, or Claude) to work on a task. Preps a sandboxed workspace first, then spawns the agent inside it. Templates: conventions, security, coding.",
	}, s.dispatch)
}

// agentCommand returns the command and args for a given agent type.
// Supports model variants: "gemini", "gemini:flash", "gemini:pro", "claude", "claude:haiku".
func agentCommand(agent, prompt string) (string, []string, error) {
	parts := core.SplitN(agent, ":", 2)
	base := parts[0]
	model := ""
	if len(parts) > 1 {
		model = parts[1]
	}

	switch base {
	case "gemini":
		args := []string{"-p", prompt, "--yolo", "--sandbox"}
		if model != "" {
			args = append(args, "-m", "gemini-2.5-"+model)
		}
		return "gemini", args, nil
	case "codex":
		return "codex", []string{"--approval-mode", "full-auto", "-q", prompt}, nil
	case "claude":
		args := []string{"-p", prompt, "--dangerously-skip-permissions"}
		if model != "" {
			args = append(args, "--model", model)
		}
		return "claude", args, nil
	case "local":
		home, _ := os.UserHomeDir()
		script := core.Path(home, "Code", "core", "agent", "scripts", "local-agent.sh")
		return "bash", []string{script, prompt}, nil
	default:
		return "", nil, coreerr.E("agentCommand", "unknown agent: "+agent, nil)
	}
}

func (s *PrepSubsystem) dispatch(ctx context.Context, req *mcp.CallToolRequest, input DispatchInput) (*mcp.CallToolResult, DispatchOutput, error) {
	progress := coremcp.NewProgressNotifier(ctx, req)
	const dispatchProgressTotal = 4

	if input.Repo == "" {
		return nil, DispatchOutput{}, coreerr.E("dispatch", "repo is required", nil)
	}
	if input.Task == "" {
		return nil, DispatchOutput{}, coreerr.E("dispatch", "task is required", nil)
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

	_ = progress.Send(1, dispatchProgressTotal, "validated dispatch request")

	// Step 1: Prep the sandboxed workspace
	_ = progress.Send(2, dispatchProgressTotal, "preparing workspace")
	prepInput := PrepInput{
		Repo:         input.Repo,
		Org:          input.Org,
		Issue:        input.Issue,
		Task:         input.Task,
		Template:     input.Template,
		PlanTemplate: input.PlanTemplate,
		Variables:    input.Variables,
		Persona:      input.Persona,
	}
	_, prepOut, err := s.prepWorkspace(ctx, req, prepInput)
	if err != nil {
		return nil, DispatchOutput{}, coreerr.E("dispatch", "prep workspace failed", err)
	}
	_ = progress.Send(3, dispatchProgressTotal, core.Sprintf("workspace prepared for %s", prepOut.Branch))

	wsDir := prepOut.WorkspaceDir
	srcDir := core.Path(wsDir, "src")

	// The prompt is just: read PROMPT.md and do the work
	prompt := "Read PROMPT.md for instructions. All context files (CLAUDE.md, TODO.md, CONTEXT.md, CONSUMERS.md, RECENT.md) are in the parent directory. Work in this directory."

	if input.DryRun {
		// Read PROMPT.md for the dry run output
		promptRaw, _ := coreio.Local.Read(core.Path(wsDir, "PROMPT.md"))
		_ = progress.Send(dispatchProgressTotal, dispatchProgressTotal, "dry run complete")
		return nil, DispatchOutput{
			Success:      true,
			Agent:        input.Agent,
			Repo:         input.Repo,
			WorkspaceDir: wsDir,
			Prompt:       promptRaw,
		}, nil
	}

	// Step 2: Check per-agent concurrency limit
	if !s.canDispatchAgent(input.Agent) {
		// Queue the workspace — write status as "queued" and return
		s.saveStatus(wsDir, &WorkspaceStatus{
			Status:    "queued",
			Agent:     input.Agent,
			Repo:      input.Repo,
			Org:       input.Org,
			Task:      input.Task,
			Issue:     input.Issue,
			Branch:    prepOut.Branch,
			StartedAt: time.Now(),
			Runs:      0,
		})
		_ = progress.Send(dispatchProgressTotal, dispatchProgressTotal, "queued until an agent slot is available")
		return nil, DispatchOutput{
			Success:      true,
			Agent:        input.Agent,
			Repo:         input.Repo,
			WorkspaceDir: wsDir,
			OutputFile:   "queued — waiting for a slot",
		}, nil
	}

	// Step 3: Write status BEFORE spawning so concurrent dispatches
	// see this workspace as "running" during the concurrency check.
	s.saveStatus(wsDir, &WorkspaceStatus{
		Status:    "running",
		Agent:     input.Agent,
		Repo:      input.Repo,
		Org:       input.Org,
		Task:      input.Task,
		Issue:     input.Issue,
		Branch:    prepOut.Branch,
		StartedAt: time.Now(),
		Runs:      1,
	})
	_ = progress.Send(3.5, dispatchProgressTotal, "dispatch slot acquired")

	// Step 4: Spawn agent as a detached process
	_ = progress.Send(4, dispatchProgressTotal, core.Sprintf("spawning agent %s", input.Agent))
	// Uses Setpgid so the agent survives parent (MCP server) death.
	// Output goes directly to log file (not buffered in memory).
	command, args, err := agentCommand(input.Agent, prompt)
	if err != nil {
		return nil, DispatchOutput{}, err
	}

	outputFile := core.Path(wsDir, core.Sprintf("agent-%s.log", input.Agent))
	outFile, err := os.Create(outputFile)
	if err != nil {
		return nil, DispatchOutput{}, coreerr.E("dispatch", "failed to create log file", err)
	}

	// Fully detach from terminal:
	// - Setpgid: own process group
	// - Stdin from /dev/null
	// - TERM=dumb prevents terminal control sequences
	// - NO_COLOR=1 disables colour output
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		outFile.Close()
		return nil, DispatchOutput{}, coreerr.E("dispatch", "failed to open /dev/null", err)
	}
	defer devNull.Close()

	cmd := exec.Command(command, args...)
	cmd.Dir = srcDir
	cmd.Stdin = devNull
	cmd.Stdout = outFile
	cmd.Stderr = outFile
	cmd.Env = append(os.Environ(), "TERM=dumb", "NO_COLOR=1", "CI=true")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		outFile.Close()
		// Revert status so the slot is freed
		s.saveStatus(wsDir, &WorkspaceStatus{
			Status: "failed",
			Agent:  input.Agent,
			Repo:   input.Repo,
			Task:   input.Task,
			Issue:  input.Issue,
			Branch: prepOut.Branch,
		})
		return nil, DispatchOutput{}, coreerr.E("dispatch", "failed to spawn "+input.Agent, err)
	}

	pid := cmd.Process.Pid
	_ = progress.Send(dispatchProgressTotal, dispatchProgressTotal, "agent process started")

	// Update status with PID now that agent is running
	s.saveStatus(wsDir, &WorkspaceStatus{
		Status:    "running",
		Agent:     input.Agent,
		Repo:      input.Repo,
		Org:       input.Org,
		Task:      input.Task,
		Issue:     input.Issue,
		Branch:    prepOut.Branch,
		PID:       pid,
		StartedAt: time.Now(),
		Runs:      1,
	})

	// Background goroutine: close file handle when process exits,
	// update status, then drain queue if a slot opened up.
	go func() {
		cmd.Wait()
		outFile.Close()

		postCtx := context.WithoutCancel(ctx)
		status := "completed"
		channel := coremcp.ChannelAgentComplete
		payload := map[string]any{
			"workspace": core.PathBase(wsDir),
			"repo":      input.Repo,
			"org":       input.Org,
			"agent":     input.Agent,
			"branch":    prepOut.Branch,
		}

		// Update status to completed or blocked.
		if st, err := readStatus(wsDir); err == nil {
			st.PID = 0
			if data, err := coreio.Local.Read(core.Path(wsDir, "src", "BLOCKED.md")); err == nil {
				status = "blocked"
				channel = coremcp.ChannelAgentBlocked
				st.Status = status
				st.Question = core.Trim(data)
				if st.Question != "" {
					payload["question"] = st.Question
				}
			} else {
				st.Status = status
			}
			s.saveStatus(wsDir, st)
		}

		payload["status"] = status
		s.emitChannel(postCtx, channel, payload)
		s.emitChannel(postCtx, coremcp.ChannelAgentStatus, payload)

		// Ingest scan findings as issues
		s.ingestFindings(wsDir)

		// Drain queue: pop next queued workspace and spawn it
		s.drainQueue()
	}()

	return nil, DispatchOutput{
		Success:      true,
		Agent:        input.Agent,
		Repo:         input.Repo,
		WorkspaceDir: wsDir,
		PID:          pid,
		OutputFile:   outputFile,
	}, nil
}
