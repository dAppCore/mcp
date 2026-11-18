package mcp

import core "github.com/dappcore/go"

func ExampleService_ServeStdio() {
	var subject Service
	_ = subject.ServeStdio
	core.Println("Service.ServeStdio")
	// Output: Service.ServeStdio
}
