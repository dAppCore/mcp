// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"time"

	core "dappco.re/go"
	"dappco.re/go/process"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// errIDEmpty is returned when a process tool call omits the required ID.
var errIDEmpty = core.E("process", "id cannot be empty", nil)

// ProcessStartInput contains parameters for starting a new process.
//
//	input := ProcessStartInput{
//	    Command: "go",
//	    Args:    []string{"test", "./..."},
//	    Dir:     "/home/user/project",
//	    Env:     []string{"CGO_ENABLED=0"},
//	}
type ProcessStartInput struct {
	Command string   `json:"command"`        // e.g. "go"
	Args    []string `json:"args,omitempty"` // e.g. ["test", "./..."]
	Dir     string   `json:"dir,omitempty"`  // e.g. "/home/user/project"
	Env     []string `json:"env,omitempty"`  // e.g. ["CGO_ENABLED=0"]
}

// ProcessRunInput contains parameters for running a command to completion
// and returning its captured output.
//
//	input := ProcessRunInput{
//	    Command: "go",
//	    Args:    []string{"test", "./..."},
//	    Dir:     "/home/user/project",
//	    Env:     []string{"CGO_ENABLED=0"},
//	}
type ProcessRunInput struct {
	Command string   `json:"command"`        // e.g. "go"
	Args    []string `json:"args,omitempty"` // e.g. ["test", "./..."]
	Dir     string   `json:"dir,omitempty"`  // e.g. "/home/user/project"
	Env     []string `json:"env,omitempty"`  // e.g. ["CGO_ENABLED=0"]
}

// ProcessRunOutput contains the result of running a process to completion.
//
//	// out.ID == "proc-abc123", out.ExitCode == 0, out.Output == "PASS\n..."
type ProcessRunOutput struct {
	ID       string `json:"id"`       // e.g. "proc-abc123"
	ExitCode int    `json:"exitCode"` // 0 on success
	Output   string `json:"output"`   // combined stdout/stderr
	Command  string `json:"command"`  // e.g. "go"
}

// ProcessStartOutput contains the result of starting a process.
//
//	// out.ID == "proc-abc123", out.PID == 54321, out.Command == "go"
type ProcessStartOutput struct {
	ID        string    `json:"id"`        // e.g. "proc-abc123"
	PID       int       `json:"pid"`       // OS process ID
	Command   string    `json:"command"`   // e.g. "go"
	Args      []string  `json:"args"`      // e.g. ["test", "./..."]
	StartedAt time.Time `json:"startedAt"` // when the process was started
}

// ProcessStopInput contains parameters for gracefully stopping a process.
//
//	input := ProcessStopInput{ID: "proc-abc123"}
type ProcessStopInput struct {
	ID string `json:"id"` // e.g. "proc-abc123"
}

// ProcessStopOutput contains the result of stopping a process.
//
//	// out.Success == true, out.Message == "Process stop signal sent"
type ProcessStopOutput struct {
	ID      string `json:"id"`                // e.g. "proc-abc123"
	Success bool   `json:"success"`           // true when stop signal was sent
	Message string `json:"message,omitempty"` // e.g. "Process stop signal sent"
}

// ProcessKillInput contains parameters for force killing a process.
//
//	input := ProcessKillInput{ID: "proc-abc123"}
type ProcessKillInput struct {
	ID string `json:"id"` // e.g. "proc-abc123"
}

// ProcessKillOutput contains the result of killing a process.
//
//	// out.Success == true, out.Message == "Process killed"
type ProcessKillOutput struct {
	ID      string `json:"id"`                // e.g. "proc-abc123"
	Success bool   `json:"success"`           // true when the process was killed
	Message string `json:"message,omitempty"` // e.g. "Process killed"
}

// ProcessListInput contains parameters for listing processes.
//
//	input := ProcessListInput{RunningOnly: true}
type ProcessListInput struct {
	RunningOnly bool `json:"running_only,omitempty"` // true to filter to running processes only
}

// ProcessListOutput contains the list of processes.
//
//	// out.Total == 3, len(out.Processes) == 3
type ProcessListOutput struct {
	Processes []ProcessInfo `json:"processes"` // one entry per managed process
	Total     int           `json:"total"`     // number of processes returned
}

// ProcessInfo represents information about a managed process.
//
//	// info.ID == "proc-abc123", info.Status == "running", info.Command == "go"
type ProcessInfo struct {
	ID        string        `json:"id"`        // e.g. "proc-abc123"
	Command   string        `json:"command"`   // e.g. "go"
	Args      []string      `json:"args"`      // e.g. ["test", "./..."]
	Dir       string        `json:"dir"`       // e.g. "/home/user/project"
	Status    string        `json:"status"`    // "running", "exited", "killed"
	PID       int           `json:"pid"`       // OS process ID
	ExitCode  int           `json:"exitCode"`  // 0 on success
	StartedAt time.Time     `json:"startedAt"` // when the process was started
	Duration  time.Duration `json:"duration"`  // how long the process has run
}

// ProcessOutputInput contains parameters for getting process output.
//
//	input := ProcessOutputInput{ID: "proc-abc123"}
type ProcessOutputInput struct {
	ID string `json:"id"` // e.g. "proc-abc123"
}

// ProcessOutputOutput contains the captured output of a process.
//
//	// out.ID == "proc-abc123", out.Output == "PASS\nok core/pkg 1.234s\n"
type ProcessOutputOutput struct {
	ID     string `json:"id"`     // e.g. "proc-abc123"
	Output string `json:"output"` // combined stdout/stderr
}

// ProcessInputInput contains parameters for sending input to a process.
//
//	input := ProcessInputInput{ID: "proc-abc123", Input: "yes\n"}
type ProcessInputInput struct {
	ID    string `json:"id"`    // e.g. "proc-abc123"
	Input string `json:"input"` // e.g. "yes\n"
}

// ProcessInputOutput contains the result of sending input to a process.
//
//	// out.Success == true, out.Message == "Input sent successfully"
type ProcessInputOutput struct {
	ID      string `json:"id"`                // e.g. "proc-abc123"
	Success bool   `json:"success"`           // true when input was delivered
	Message string `json:"message,omitempty"` // e.g. "Input sent successfully"
}

// registerProcessTools adds process management tools to the MCP server.
// Returns false if process service is not available.
func (s *Service) registerProcessTools(server *mcp.Server) bool {
	if s.processService == nil {
		return false
	}

	addToolRecorded(s, server, "process", &mcp.Tool{
		Name:        "process_start",
		Description: "Start a new external process. Returns process ID for tracking.",
	}, s.processStart)

	addToolRecorded(s, server, "process", &mcp.Tool{
		Name:        "process_run",
		Description: "Run a command to completion and return the captured output. Blocks until the process exits.",
	}, s.processRun)

	addToolRecorded(s, server, "process", &mcp.Tool{
		Name:        "process_stop",
		Description: "Gracefully stop a running process by ID.",
	}, s.processStop)

	addToolRecorded(s, server, "process", &mcp.Tool{
		Name:        "process_kill",
		Description: "Force kill a process by ID. Use when process_stop doesn't work.",
	}, s.processKill)

	addToolRecorded(s, server, "process", &mcp.Tool{
		Name:        "process_list",
		Description: "List all managed processes. Use running_only=true for only active processes.",
	}, s.processList)

	addToolRecorded(s, server, "process", &mcp.Tool{
		Name:        "process_output",
		Description: "Get the captured output of a process by ID.",
	}, s.processOutput)

	addToolRecorded(s, server, "process", &mcp.Tool{
		Name:        "process_input",
		Description: "Send input to a running process stdin.",
	}, s.processInput)

	return true
}

// processStart handles the process_start tool call.
func (s *Service) processStart(ctx context.Context, req *mcp.CallToolRequest, input ProcessStartInput) (*mcp.CallToolResult, ProcessStartOutput, error) {
	if s.processService == nil {
		return nil, ProcessStartOutput{}, core.E("processStart", "process service unavailable", nil)
	}

	s.logger.Security("MCP tool execution", "tool", "process_start", "command", input.Command, "args", input.Args, "dir", input.Dir, "user", core.Username())

	if input.Command == "" {
		return nil, ProcessStartOutput{}, core.E("processStart", "command cannot be empty", nil)
	}

	opts := process.RunOptions{
		Command: input.Command,
		Args:    input.Args,
		Dir:     s.resolveWorkspacePath(input.Dir),
		Env:     input.Env,
	}

	proc, err := s.processService.StartWithOptions(ctx, opts)
	if err != nil {
		core.Error("mcp: process start failed", "command", input.Command, "err", err)
		return nil, ProcessStartOutput{}, core.E("processStart", "failed to start process", err)
	}

	info := proc.Info()
	output := ProcessStartOutput{
		ID:        proc.ID,
		PID:       info.PID,
		Command:   proc.Command,
		Args:      proc.Args,
		StartedAt: proc.StartedAt,
	}
	s.recordProcessRuntime(output.ID, processRuntime{
		Command:   output.Command,
		Args:      output.Args,
		Dir:       info.Dir,
		StartedAt: output.StartedAt,
	})
	s.ChannelSend(ctx, ChannelProcessStart, map[string]any{
		"id":        output.ID,
		"pid":       output.PID,
		"command":   output.Command,
		"args":      output.Args,
		"dir":       info.Dir,
		"startedAt": output.StartedAt,
	})
	return nil, output, nil
}

// processRun handles the process_run tool call.
// Executes the command to completion and returns the captured output.
func (s *Service) processRun(ctx context.Context, req *mcp.CallToolRequest, input ProcessRunInput) (*mcp.CallToolResult, ProcessRunOutput, error) {
	if s.processService == nil {
		return nil, ProcessRunOutput{}, core.E("processRun", "process service unavailable", nil)
	}

	progress := NewProgressNotifier(ctx, req)
	s.logger.Security("MCP tool execution", "tool", "process_run", "command", input.Command, "args", input.Args, "dir", input.Dir, "user", core.Username())

	if input.Command == "" {
		return nil, ProcessRunOutput{}, core.E("processRun", "command cannot be empty", nil)
	}

	opts := process.RunOptions{
		Command: input.Command,
		Args:    input.Args,
		Dir:     s.resolveWorkspacePath(input.Dir),
		Env:     input.Env,
	}

	sendToolProgress(progress, 0, 2, "starting process")
	proc, err := s.processService.StartWithOptions(ctx, opts)
	if err != nil {
		core.Error("mcp: process run start failed", "command", input.Command, "err", err)
		return nil, ProcessRunOutput{}, core.E("processRun", "failed to start process", err)
	}
	sendToolProgress(progress, 1, 2, "process started")

	info := proc.Info()
	s.recordProcessRuntime(proc.ID, processRuntime{
		Command:   proc.Command,
		Args:      proc.Args,
		Dir:       info.Dir,
		StartedAt: proc.StartedAt,
	})
	s.ChannelSend(ctx, ChannelProcessStart, map[string]any{
		"id":        proc.ID,
		"pid":       info.PID,
		"command":   proc.Command,
		"args":      proc.Args,
		"dir":       info.Dir,
		"startedAt": proc.StartedAt,
	})

	// Wait for completion (context-aware).
	select {
	case <-ctx.Done():
		sendToolProgress(progress, 2, 2, "process cancelled")
		return nil, ProcessRunOutput{}, core.E("processRun", "cancelled", ctx.Err())
	case <-proc.Done():
	}
	sendToolProgress(progress, 2, 2, "process completed")

	return nil, ProcessRunOutput{
		ID:       proc.ID,
		ExitCode: proc.ExitCode,
		Output:   proc.Output(),
		Command:  proc.Command,
	}, nil
}

func sendToolProgress(progress ProgressNotifier, current float64, total float64, message string) {
	if err := progress.Send(current, total, message); err != nil {
		core.Error("mcp: failed to send tool progress", "err", err)
	}
}

// processStop handles the process_stop tool call.
func (s *Service) processStop(ctx context.Context, req *mcp.CallToolRequest, input ProcessStopInput) (*mcp.CallToolResult, ProcessStopOutput, error) {
	if s.processService == nil {
		return nil, ProcessStopOutput{}, core.E("processStop", "process service unavailable", nil)
	}

	s.logger.Security("MCP tool execution", "tool", "process_stop", "id", input.ID, "user", core.Username())

	if input.ID == "" {
		return nil, ProcessStopOutput{}, errIDEmpty
	}

	proc, err := s.processService.Get(input.ID)
	if err != nil {
		core.Error("mcp: process stop failed", "id", input.ID, "err", err)
		return nil, ProcessStopOutput{}, core.E("processStop", "process not found", err)
	}

	// Use the process service's graceful shutdown path first so callers get
	// a real stop signal before we fall back to a hard kill internally.
	if err := proc.Shutdown(); err != nil {
		core.Error("mcp: process stop failed", "id", input.ID, "err", err)
		return nil, ProcessStopOutput{}, core.E("processStop", "failed to stop process", err)
	}

	info := proc.Info()
	s.ChannelSend(ctx, ChannelProcessExit, map[string]any{
		"id":        input.ID,
		"signal":    "stop",
		"command":   info.Command,
		"args":      info.Args,
		"dir":       info.Dir,
		"startedAt": info.StartedAt,
	})
	s.emitTestResult(ctx, input.ID, 0, 0, "stop", "")
	return nil, ProcessStopOutput{
		ID:      input.ID,
		Success: true,
		Message: "Process stop signal sent",
	}, nil
}

// processKill handles the process_kill tool call.
func (s *Service) processKill(ctx context.Context, req *mcp.CallToolRequest, input ProcessKillInput) (*mcp.CallToolResult, ProcessKillOutput, error) {
	if s.processService == nil {
		return nil, ProcessKillOutput{}, core.E("processKill", "process service unavailable", nil)
	}

	s.logger.Security("MCP tool execution", "tool", "process_kill", "id", input.ID, "user", core.Username())

	if input.ID == "" {
		return nil, ProcessKillOutput{}, errIDEmpty
	}

	proc, err := s.processService.Get(input.ID)
	if err != nil {
		core.Error("mcp: process kill failed", "id", input.ID, "err", err)
		return nil, ProcessKillOutput{}, core.E("processKill", "process not found", err)
	}

	if err := s.processService.Kill(input.ID); err != nil {
		core.Error("mcp: process kill failed", "id", input.ID, "err", err)
		return nil, ProcessKillOutput{}, core.E("processKill", "failed to kill process", err)
	}

	info := proc.Info()
	s.ChannelSend(ctx, ChannelProcessExit, map[string]any{
		"id":        input.ID,
		"signal":    "kill",
		"command":   info.Command,
		"args":      info.Args,
		"dir":       info.Dir,
		"startedAt": info.StartedAt,
	})
	s.emitTestResult(ctx, input.ID, 0, 0, "kill", "")
	return nil, ProcessKillOutput{
		ID:      input.ID,
		Success: true,
		Message: "Process killed",
	}, nil
}

// processList handles the process_list tool call.
func (s *Service) processList(ctx context.Context, req *mcp.CallToolRequest, input ProcessListInput) (*mcp.CallToolResult, ProcessListOutput, error) {
	if s.processService == nil {
		return nil, ProcessListOutput{}, core.E("processList", "process service unavailable", nil)
	}

	s.logger.Info("MCP tool execution", "tool", "process_list", "running_only", input.RunningOnly, "user", core.Username())

	var procs []*process.Process
	if input.RunningOnly {
		procs = s.processService.Running()
	} else {
		procs = s.processService.List()
	}

	result := make([]ProcessInfo, len(procs))
	for i, p := range procs {
		info := p.Info()
		result[i] = ProcessInfo{
			ID:        info.ID,
			Command:   info.Command,
			Args:      info.Args,
			Dir:       info.Dir,
			Status:    string(info.Status),
			PID:       info.PID,
			ExitCode:  info.ExitCode,
			StartedAt: info.StartedAt,
			Duration:  info.Duration,
		}
	}

	return nil, ProcessListOutput{
		Processes: result,
		Total:     len(result),
	}, nil
}

// processOutput handles the process_output tool call.
func (s *Service) processOutput(ctx context.Context, req *mcp.CallToolRequest, input ProcessOutputInput) (*mcp.CallToolResult, ProcessOutputOutput, error) {
	if s.processService == nil {
		return nil, ProcessOutputOutput{}, core.E("processOutput", "process service unavailable", nil)
	}

	s.logger.Info("MCP tool execution", "tool", "process_output", "id", input.ID, "user", core.Username())

	if input.ID == "" {
		return nil, ProcessOutputOutput{}, errIDEmpty
	}

	output, err := s.processService.Output(input.ID)
	if err != nil {
		core.Error("mcp: process output failed", "id", input.ID, "err", err)
		return nil, ProcessOutputOutput{}, core.E("processOutput", "failed to get process output", err)
	}

	return nil, ProcessOutputOutput{
		ID:     input.ID,
		Output: output,
	}, nil
}

// processInput handles the process_input tool call.
func (s *Service) processInput(ctx context.Context, req *mcp.CallToolRequest, input ProcessInputInput) (*mcp.CallToolResult, ProcessInputOutput, error) {
	if s.processService == nil {
		return nil, ProcessInputOutput{}, core.E("processInput", "process service unavailable", nil)
	}

	s.logger.Security("MCP tool execution", "tool", "process_input", "id", input.ID, "user", core.Username())

	if input.ID == "" {
		return nil, ProcessInputOutput{}, errIDEmpty
	}
	if input.Input == "" {
		return nil, ProcessInputOutput{}, core.E("processInput", "input cannot be empty", nil)
	}

	proc, err := s.processService.Get(input.ID)
	if err != nil {
		core.Error("mcp: process input get failed", "id", input.ID, "err", err)
		return nil, ProcessInputOutput{}, core.E("processInput", "process not found", err)
	}

	if err := proc.SendInput(input.Input); err != nil {
		core.Error("mcp: process input send failed", "id", input.ID, "err", err)
		return nil, ProcessInputOutput{}, core.E("processInput", "failed to send input", err)
	}

	return nil, ProcessInputOutput{
		ID:      input.ID,
		Success: true,
		Message: "Input sent successfully",
	}, nil
}
