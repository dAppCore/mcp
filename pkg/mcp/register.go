// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"

	core "dappco.re/go/core"
)

// Register is the service factory for core.WithService.
// Creates the MCP service, registers subsystems from other services
// already in the Core conclave, and wires notifiers.
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

	return core.Result{Value: svc, OK: true}
}

// OnStartup implements core.Startable — MCP is ready for tool calls.
// Transport is NOT started here — the CLI command (mcp/serve) starts
// the appropriate transport explicitly.
func (s *Service) OnStartup(ctx context.Context) error {
	return nil
}

// OnShutdown implements core.Stoppable — stops the MCP transport.
func (s *Service) OnShutdown(ctx context.Context) error {
	return s.Shutdown(ctx)
}
