// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"

	core "dappco.re/go/core"
	"forge.lthn.ai/core/go-log"
)

// Register is the service factory for core.WithService.
// Creates the MCP service, discovers subsystems from other Core services,
// and wires notifiers.
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
	for _, name := range c.Services() {
		r := c.Service(name)
		if !r.OK {
			continue
		}
		if sub, ok := r.Value.(Subsystem); ok {
			subsystems = append(subsystems, sub)
		}
	}

	svc, err := New(Options{
		Subsystems: subsystems,
	})
	if err != nil {
		return core.Result{Value: err, OK: false}
	}

	svc.coreRef = c

	return core.Result{Value: svc, OK: true}
}

// OnStartup implements core.Startable — registers MCP transport commands.
func (s *Service) OnStartup(ctx context.Context) core.Result {
	c, ok := s.coreRef.(*core.Core)
	if !ok || c == nil {
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
			log.Default().Info("MCP HTTP server starting")
			if err := s.Run(ctx); err != nil {
				return core.Result{Value: err, OK: false}
			}
			return core.Result{OK: true}
		},
	})

	return core.Result{OK: true}
}

// OnShutdown implements core.Stoppable — stops the MCP transport.
func (s *Service) OnShutdown(ctx context.Context) core.Result {
	if err := s.Shutdown(ctx); err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{OK: true}
}
