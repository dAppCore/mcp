package mcpcmd

import . "dappco.re/go"

func TestAX7_AddMCPCommands_Good(t *T) {
	c := New()
	AddMCPCommands(c)
	commands := c.Commands()
	AssertContains(t, commands, "mcp")
	AssertContains(t, commands, "mcp/serve")
}

func TestAX7_AddMCPCommands_Bad(t *T) {
	var c *Core
	AssertPanics(t, func() { AddMCPCommands(c) })
	AssertNil(t, c)
}

func TestAX7_AddMCPCommands_Ugly(t *T) {
	c := New()
	AddMCPCommands(c)
	AddMCPCommands(c)
	commands := c.Commands()
	AssertContains(t, commands, "mcp")
	AssertContains(t, commands, "mcp/serve")
}
