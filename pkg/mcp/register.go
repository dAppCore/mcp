// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"
	"time"

	core "dappco.re/go"
	"dappco.re/go/process"
	"dappco.re/go/ws"
)

// Register is the service factory for core.WithService.
// Creates the MCP service, discovers subsystems from other Core services,
// and wires optional process and WebSocket dependencies when they are
// already registered in Core.
//
//	core.New(
//	    core.WithService(agentic.Register),
//	    core.WithService(monitor.Register),
//	    core.WithService(brain.Register),
//	    core.WithService(mcp.Register),
//	)
func Register(c *core.Core) core.Result {
	// Collect subsystems from registered services
	var subsystems []Subsystem
	var processService *process.Service
	var wsHub *ws.Hub
	for _, name := range c.Services() {
		r := c.Service(name)
		if !r.OK {
			continue
		}
		if sub, ok := r.Value.(Subsystem); ok {
			subsystems = append(subsystems, sub)
			continue
		}
		switch v := r.Value.(type) {
		case *process.Service:
			processService = v
		case *ws.Hub:
			wsHub = v
		}
	}

	svc, err := New(Options{
		ProcessService: processService,
		WSHub:          wsHub,
		Subsystems:     subsystems,
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}

	svc.ServiceRuntime = core.NewServiceRuntime(c, struct{}{})

	return core.Result{Value: svc, OK: true}
}

// OnStartup implements core.Startable — registers MCP transport commands.
//
//	svc.OnStartup(context.Background())
//
//	core-agent mcp    — start MCP server on stdio
//	core-agent serve  — start MCP server on HTTP
func (s *Service) OnStartup(ctx context.Context) core.Result {
	c := s.Core()
	if c == nil {
		return core.Result{OK: true}
	}

	c.Command("mcp", core.Command{
		Description: "Start the MCP server on stdio",
		Action: func(opts core.Options) core.Result {
			s.logger.Info("MCP stdio server starting")
			if err := s.ServeStdio(ctx); err != nil {
				return core.Result{Value: err, OK: false}
			}
			return core.Result{OK: true}
		},
	})

	c.Command("serve", core.Command{
		Description: "Start the MCP server with auto-selected transport",
		Action: func(opts core.Options) core.Result {
			s.logger.Info("MCP server starting")
			if err := s.Run(ctx); err != nil {
				return core.Result{Value: err, OK: false}
			}
			return core.Result{OK: true}
		},
	})

	return core.Result{OK: true}
}

// HandleIPCEvents implements Core's IPC handler interface.
//
//	c.ACTION(mcp.ChannelPush{Channel: "agent.status", Data: statusMap})
//
// Catches ChannelPush messages from other services and pushes them to Claude Code sessions.
func (s *Service) HandleIPCEvents(c *core.Core, msg core.Message) core.Result {
	ctx := context.Background()
	if c != nil {
		if coreCtx := c.Context(); coreCtx != nil {
			ctx = coreCtx
		}
	}

	switch ev := msg.(type) {
	case ChannelPush:
		return s.handleChannelPushIPC(ctx, ev)
	case process.ActionProcessStarted:
		startedAt := time.Now()
		s.recordProcessRuntime(ev.ID, processRuntime{
			Command:   ev.Command,
			Args:      ev.Args,
			Dir:       ev.Dir,
			StartedAt: startedAt,
		})
		s.ChannelSend(ctx, ChannelProcessStart, map[string]any{
			"id":        ev.ID,
			"command":   ev.Command,
			"args":      ev.Args,
			"dir":       ev.Dir,
			"pid":       ev.PID,
			"startedAt": startedAt,
		})
	case process.ActionProcessOutput:
		s.ChannelSend(ctx, ChannelProcessOutput, map[string]any{
			"id":     ev.ID,
			"line":   ev.Line,
			"stream": ev.Stream,
		})
	case process.ActionProcessExited:
		meta, ok := s.processRuntimeFor(ev.ID)
		payload := map[string]any{
			"id":       ev.ID,
			"exitCode": ev.ExitCode,
			"duration": ev.Duration,
		}
		if ok {
			payload["command"] = meta.Command
			payload["args"] = meta.Args
			payload["dir"] = meta.Dir
			if !meta.StartedAt.IsZero() {
				payload["startedAt"] = meta.StartedAt
			}
		}
		if ev.Error != nil {
			payload["error"] = ev.Error.Error()
		}
		s.ChannelSend(ctx, ChannelProcessExit, payload)
		errText := ""
		if ev.Error != nil {
			errText = ev.Error.Error()
		}
		s.emitTestResult(ctx, ev.ID, ev.ExitCode, ev.Duration, "", errText)
	case process.ActionProcessKilled:
		meta, ok := s.processRuntimeFor(ev.ID)
		payload := map[string]any{
			"id":     ev.ID,
			"signal": ev.Signal,
		}
		if ok {
			payload["command"] = meta.Command
			payload["args"] = meta.Args
			payload["dir"] = meta.Dir
			if !meta.StartedAt.IsZero() {
				payload["startedAt"] = meta.StartedAt
			}
		}
		s.ChannelSend(ctx, ChannelProcessExit, payload)
		s.emitTestResult(ctx, ev.ID, 0, 0, ev.Signal, "")
	}
	return core.Result{OK: true}
}

// OnShutdown implements core.Stoppable — stops the MCP transport.
//
//	svc.OnShutdown(context.Background())
func (s *Service) OnShutdown(ctx context.Context) core.Result {
	if err := s.Shutdown(ctx); err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{OK: true}
}
