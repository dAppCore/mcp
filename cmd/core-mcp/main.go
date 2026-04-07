package main

import (
	"dappco.re/go/core/cli/pkg/cli"
	mcpcmd "dappco.re/go/mcp/cmd/mcpcmd"
)

func main() {
	cli.Main(
		cli.WithCommands("mcp", mcpcmd.AddMCPCommands),
	)
}
