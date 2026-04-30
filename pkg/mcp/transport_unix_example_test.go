package mcp

import core "dappco.re/go"

func ExampleService_ServeUnix() {
	var subject Service
	_ = subject.ServeUnix
	core.Println("Service.ServeUnix")
	// Output: Service.ServeUnix
}
