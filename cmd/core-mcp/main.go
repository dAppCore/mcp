package main

import (
	"forge.lthn.ai/core/cli/pkg/cli"
	mcpcmd "forge.lthn.ai/core/mcp/cmd/mcpcmd"
)

func main() {
	cli.Main(
		cli.WithCommands("mcp", mcpcmd.AddMCPCommands),
	)
}
