// SPDX-License-Identifier: EUPL-1.2

// Package mcpcmd registers the `mcp` and `mcp serve` CLI commands.
//
// Wiring example:
//
//	cli.Main(cli.WithCommands("mcp", mcpcmd.AddMCPCommands))
//
// Commands:
//   - mcp           Start the MCP server on stdio (default transport).
//   - mcp serve     Start the MCP server with auto-selected transport.
package mcpcmd

import (
	"context"

	core "dappco.re/go"
	"dappco.re/go/mcp/pkg/mcp"
	"dappco.re/go/mcp/pkg/mcp/agentic"
	"dappco.re/go/mcp/pkg/mcp/brain"
)

// newMCPService is the service constructor, indirected for tests.
var newMCPService = mcp.New

// runMCPService starts the MCP server, indirected for tests.
var runMCPService = func(svc *mcp.Service, ctx context.Context) error {
	return svc.Run(ctx)
}

// shutdownMCPService performs graceful shutdown, indirected for tests.
var shutdownMCPService = func(svc *mcp.Service, ctx context.Context) error {
	return svc.Shutdown(ctx)
}

// workspaceFlag mirrors the --workspace CLI flag value.
var workspaceFlag string

// unrestrictedFlag mirrors the --unrestricted CLI flag value.
var unrestrictedFlag bool

// AddMCPCommands registers the `mcp` command tree on the Core instance.
//
//	cli.Main(cli.WithCommands("mcp", mcpcmd.AddMCPCommands))
//
// register reports a failed command registration instead of dropping it: an
// unregistered command is not a missing feature at startup, it is a "command
// not found" much later, a long way from the cause.
func register(c *core.Core, path string, cmd core.Command) {
	if r := c.Command(path, cmd); !r.OK {
		core.Warn("mcpcmd: registering command failed", "path", path, "reason", r.Value)
	}
}

func AddMCPCommands(c *core.Core) {
	register(c, "mcp", core.Command{
		Description: "Model Context Protocol server (stdio, TCP, Unix socket, HTTP).",
		Action:      runServeAction,
		Flags: core.NewOptions(
			core.Option{Key: "workspace", Value: ""},
			core.Option{Key: "w", Value: ""},
			core.Option{Key: "unrestricted", Value: false},
		),
	})

	register(c, "mcp/serve", core.Command{
		Description: "Start the MCP server with auto-selected transport (stdio, TCP, Unix, or HTTP).",
		Action:      runServeAction,
		Flags: core.NewOptions(
			core.Option{Key: "workspace", Value: ""},
			core.Option{Key: "w", Value: ""},
			core.Option{Key: "unrestricted", Value: false},
		),
	})
}

// runServeAction is the CLI entrypoint for `mcp` and `mcp serve`.
//
//	opts := core.NewOptions(core.Option{Key: "workspace", Value: "."})
//	result := runServeAction(opts)
func runServeAction(opts core.Options) core.Result {
	workspaceFlag = core.Trim(firstNonEmpty(opts.String("workspace"), opts.String("w")))
	unrestrictedFlag = opts.Bool("unrestricted")

	if err := runServe(); err != nil {
		return core.Fail(err)
	}
	return core.Ok(nil)
}

// firstNonEmpty returns the first non-empty string argument.
//
//	firstNonEmpty("", "foo") == "foo"
//	firstNonEmpty("bar", "baz") == "bar"
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// runServe wires the MCP service together and blocks until the context is
// cancelled by SIGINT/SIGTERM or a transport error.
//
//	if err := runServe(); err != nil {
//	    core.Error("mcp serve failed", "err", err)
//	}
func runServe() (
	_ error, // result
) {
	opts := mcp.Options{}

	if unrestrictedFlag {
		opts.Unrestricted = true
	} else if workspaceFlag != "" {
		opts.WorkspaceRoot = workspaceFlag
	}

	// Register OpenBrain and agentic subsystems.
	opts.Subsystems = []mcp.Subsystem{
		brain.NewDirect(),
		agentic.NewPrep(),
	}

	svc, err := newMCPService(opts)
	if err != nil {
		return core.E("mcpcmd.runServe", "create MCP service", err)
	}
	defer func() {
		if err := shutdownMCPService(svc, context.Background()); err != nil {
			core.Error("mcp serve: shutdown failed", "err", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return runMCPService(svc, ctx)
}
