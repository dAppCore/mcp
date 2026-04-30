package main

import (
	core "dappco.re/go"
	mcpcmd "dappco.re/go/mcp/cmd/mcpcmd"
)

func main() {
	c := core.New(core.WithService(core.CliRegister))
	mcpcmd.AddMCPCommands(c)
	c.Run()
}
