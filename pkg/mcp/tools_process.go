package mcp

import (
	"context"
	"time"

	"forge.lthn.ai/core/go-log"
	"forge.lthn.ai/core/go-process"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// errIDEmpty is returned when a process tool call omits the required ID.
var errIDEmpty = log.E("process", "id cannot be empty", nil)

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
	s.logger.Security("MCP tool execution", "tool", "process_start", "command", input.Command, "args", input.Args, "dir", input.Dir, "user", log.Username())

	if input.Command == "" {
		return nil, ProcessStartOutput{}, log.E("processStart", "command cannot be empty", nil)
	}

	opts := process.RunOptions{
		Command: input.Command,
		Args:    input.Args,
		Dir:     input.Dir,
		Env:     input.Env,
	}

	proc, err := s.processService.StartWithOptions(ctx, opts)
	if err != nil {
		log.Error("mcp: process start failed", "command", input.Command, "err", err)
		return nil, ProcessStartOutput{}, log.E("processStart", "failed to start process", err)
	}

	info := proc.Info()
	output := ProcessStartOutput{
		ID:        proc.ID,
		PID:       info.PID,
		Command:   proc.Command,
		Args:      proc.Args,
		StartedAt: proc.StartedAt,
	}
	s.ChannelSend(ctx, "process.start", map[string]any{
		"id": output.ID, "pid": output.PID, "command": output.Command,
	})
	return nil, output, nil
}

// processStop handles the process_stop tool call.
func (s *Service) processStop(ctx context.Context, req *mcp.CallToolRequest, input ProcessStopInput) (*mcp.CallToolResult, ProcessStopOutput, error) {
	s.logger.Security("MCP tool execution", "tool", "process_stop", "id", input.ID, "user", log.Username())

	if input.ID == "" {
		return nil, ProcessStopOutput{}, errIDEmpty
	}

	proc, err := s.processService.Get(input.ID)
	if err != nil {
		log.Error("mcp: process stop failed", "id", input.ID, "err", err)
		return nil, ProcessStopOutput{}, log.E("processStop", "process not found", err)
	}

	// Use the process service's graceful shutdown path first so callers get
	// a real stop signal before we fall back to a hard kill internally.
	if err := proc.Shutdown(); err != nil {
		log.Error("mcp: process stop failed", "id", input.ID, "err", err)
		return nil, ProcessStopOutput{}, log.E("processStop", "failed to stop process", err)
	}

	s.ChannelSend(ctx, "process.exit", map[string]any{"id": input.ID, "signal": "stop"})
	return nil, ProcessStopOutput{
		ID:      input.ID,
		Success: true,
		Message: "Process stop signal sent",
	}, nil
}

// processKill handles the process_kill tool call.
func (s *Service) processKill(ctx context.Context, req *mcp.CallToolRequest, input ProcessKillInput) (*mcp.CallToolResult, ProcessKillOutput, error) {
	s.logger.Security("MCP tool execution", "tool", "process_kill", "id", input.ID, "user", log.Username())

	if input.ID == "" {
		return nil, ProcessKillOutput{}, errIDEmpty
	}

	if err := s.processService.Kill(input.ID); err != nil {
		log.Error("mcp: process kill failed", "id", input.ID, "err", err)
		return nil, ProcessKillOutput{}, log.E("processKill", "failed to kill process", err)
	}

	s.ChannelSend(ctx, "process.exit", map[string]any{"id": input.ID, "signal": "kill"})
	return nil, ProcessKillOutput{
		ID:      input.ID,
		Success: true,
		Message: "Process killed",
	}, nil
}

// processList handles the process_list tool call.
func (s *Service) processList(ctx context.Context, req *mcp.CallToolRequest, input ProcessListInput) (*mcp.CallToolResult, ProcessListOutput, error) {
	s.logger.Info("MCP tool execution", "tool", "process_list", "running_only", input.RunningOnly, "user", log.Username())

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
	s.logger.Info("MCP tool execution", "tool", "process_output", "id", input.ID, "user", log.Username())

	if input.ID == "" {
		return nil, ProcessOutputOutput{}, errIDEmpty
	}

	output, err := s.processService.Output(input.ID)
	if err != nil {
		log.Error("mcp: process output failed", "id", input.ID, "err", err)
		return nil, ProcessOutputOutput{}, log.E("processOutput", "failed to get process output", err)
	}

	return nil, ProcessOutputOutput{
		ID:     input.ID,
		Output: output,
	}, nil
}

// processInput handles the process_input tool call.
func (s *Service) processInput(ctx context.Context, req *mcp.CallToolRequest, input ProcessInputInput) (*mcp.CallToolResult, ProcessInputOutput, error) {
	s.logger.Security("MCP tool execution", "tool", "process_input", "id", input.ID, "user", log.Username())

	if input.ID == "" {
		return nil, ProcessInputOutput{}, errIDEmpty
	}
	if input.Input == "" {
		return nil, ProcessInputOutput{}, log.E("processInput", "input cannot be empty", nil)
	}

	proc, err := s.processService.Get(input.ID)
	if err != nil {
		log.Error("mcp: process input get failed", "id", input.ID, "err", err)
		return nil, ProcessInputOutput{}, log.E("processInput", "process not found", err)
	}

	if err := proc.SendInput(input.Input); err != nil {
		log.Error("mcp: process input send failed", "id", input.ID, "err", err)
		return nil, ProcessInputOutput{}, log.E("processInput", "failed to send input", err)
	}

	return nil, ProcessInputOutput{
		ID:      input.ID,
		Success: true,
		Message: "Input sent successfully",
	}, nil
}
