package mcp

import core "dappco.re/go"

func ExampleService_ServeStdio() {
	var subject Service
	_ = subject.ServeStdio
	core.Println("Service.ServeStdio")
	// Output: Service.ServeStdio
}
