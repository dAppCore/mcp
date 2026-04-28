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
	"os"
	"os/signal"
	"syscall"

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
func AddMCPCommands(c *core.Core) {
	c.Command("mcp", core.Command{
		Description: "Model Context Protocol server (stdio, TCP, Unix socket, HTTP).",
		Action:      runServeAction,
		Flags: core.NewOptions(
			core.Option{Key: "workspace", Value: ""},
			core.Option{Key: "w", Value: ""},
			core.Option{Key: "unrestricted", Value: false},
		),
	})

	c.Command("mcp/serve", core.Command{
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
		return core.Result{Value: err, OK: false}
	}
	return core.Result{OK: true}
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
func runServe() error {
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
		_ = shutdownMCPService(svc, context.Background())
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()

	return runMCPService(svc, ctx)
}
