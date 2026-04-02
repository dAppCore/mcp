// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"

	core "dappco.re/go/core"
	"forge.lthn.ai/core/go-process"
	"forge.lthn.ai/core/go-ws"
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
		Description: "Start as a persistent HTTP daemon",
		Action: func(opts core.Options) core.Result {
			s.logger.Info("MCP HTTP server starting")
			if err := s.Run(ctx); err != nil {
				return core.Result{Value: err, OK: false}
			}
			return core.Result{OK: true}
		},
	})

	return core.Result{OK: true}
}

// HandleIPCEvents implements Core's IPC handler interface.
// Catches ChannelPush messages from other services and pushes them to Claude Code sessions.
//
//	c.ACTION(mcp.ChannelPush{Channel: "agent.status", Data: statusMap})
func (s *Service) HandleIPCEvents(c *core.Core, msg core.Message) core.Result {
	ctx := context.Background()
	if c != nil {
		if coreCtx := c.Context(); coreCtx != nil {
			ctx = coreCtx
		}
	}

	switch ev := msg.(type) {
	case ChannelPush:
		s.ChannelSend(ctx, ev.Channel, ev.Data)
	case process.ActionProcessStarted:
		s.ChannelSend(ctx, ChannelProcessStart, map[string]any{
			"id":      ev.ID,
			"command": ev.Command,
			"args":    ev.Args,
			"dir":     ev.Dir,
			"pid":     ev.PID,
		})
	case process.ActionProcessOutput:
		s.ChannelSend(ctx, ChannelProcessOutput, map[string]any{
			"id":     ev.ID,
			"line":   ev.Line,
			"stream": ev.Stream,
		})
	case process.ActionProcessExited:
		payload := map[string]any{
			"id":       ev.ID,
			"exitCode": ev.ExitCode,
			"duration": ev.Duration,
		}
		if ev.Error != nil {
			payload["error"] = ev.Error.Error()
		}
		s.ChannelSend(ctx, ChannelProcessExit, payload)
	case process.ActionProcessKilled:
		s.ChannelSend(ctx, ChannelProcessExit, map[string]any{
			"id":     ev.ID,
			"signal": ev.Signal,
		})
	}
	return core.Result{OK: true}
}

// OnShutdown implements core.Stoppable — stops the MCP transport.
func (s *Service) OnShutdown(ctx context.Context) core.Result {
	if err := s.Shutdown(ctx); err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{OK: true}
}
