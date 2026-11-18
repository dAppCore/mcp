package mcp

import core "github.com/dappcore/go"

func ExampleService_ServeUnix() {
	var subject Service
	_ = subject.ServeUnix
	core.Println("Service.ServeUnix")
	// Output: Service.ServeUnix
}
