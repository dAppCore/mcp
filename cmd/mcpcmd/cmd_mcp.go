// Package mcpcmd provides the MCP server command.
//
// Commands:
//   - mcp serve: Start the MCP server for AI tool integration
package mcpcmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"dappco.re/go/mcp/pkg/mcp"
	"dappco.re/go/mcp/pkg/mcp/agentic"
	"dappco.re/go/mcp/pkg/mcp/brain"
	"forge.lthn.ai/core/cli/pkg/cli"
)

var workspaceFlag string
var unrestrictedFlag bool

var mcpCmd = &cli.Command{
	Use:   "mcp",
	Short: "MCP server for AI tool integration",
	Long:  "Model Context Protocol (MCP) server providing file operations, RAG, and metrics tools.",
}

var serveCmd = &cli.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	Long: `Start the MCP server on stdio (default), TCP, Unix socket, or HTTP.

The server provides file operations plus the brain and agentic subsystems
registered by this command.

Environment variables:
  MCP_ADDR    TCP address to listen on (e.g., "localhost:9999")
  MCP_UNIX_SOCKET
              Unix socket path to listen on (e.g., "/tmp/core-mcp.sock")
              Selected after MCP_ADDR and before stdio.
  MCP_HTTP_ADDR
              HTTP address to listen on (e.g., "127.0.0.1:9101")
              Selected before MCP_ADDR and stdio.

Examples:
  # Start with stdio transport (for Claude Code integration)
  core mcp serve

  # Start with workspace restriction
  core mcp serve --workspace /path/to/project

  # Start unrestricted (explicit opt-in)
  core mcp serve --unrestricted

  # Start TCP server
  MCP_ADDR=localhost:9999 core mcp serve`,
	RunE: func(cmd *cli.Command, args []string) error {
		return runServe()
	},
}

func initFlags() {
	cli.StringFlag(serveCmd, &workspaceFlag, "workspace", "w", "", "Restrict file operations to this directory")
	cli.BoolFlag(serveCmd, &unrestrictedFlag, "unrestricted", "", false, "Disable filesystem sandboxing entirely")
}

// AddMCPCommands registers the 'mcp' command and all subcommands.
func AddMCPCommands(root *cli.Command) {
	initFlags()
	mcpCmd.AddCommand(serveCmd)
	root.AddCommand(mcpCmd)
}

func runServe() error {
	opts := mcp.Options{}

	if unrestrictedFlag {
		opts.Unrestricted = true
	} else if workspaceFlag != "" {
		opts.WorkspaceRoot = workspaceFlag
	}

	// Register OpenBrain and agentic subsystems
	opts.Subsystems = []mcp.Subsystem{
		brain.NewDirect(),
		agentic.NewPrep(),
	}

	// Create the MCP service
	svc, err := mcp.New(opts)
	if err != nil {
		return cli.Wrap(err, "create MCP service")
	}

	// Set up signal handling for clean shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		cancel()
	}()

	// Run the server (blocks until context cancelled or error)
	return svc.Run(ctx)
}
