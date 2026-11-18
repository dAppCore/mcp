package mcp

import core "github.com/dappcore/go"

func ExampleService_ServeHTTP() {
	var subject Service
	_ = subject.ServeHTTP
	core.Println("Service.ServeHTTP")
	// Output: Service.ServeHTTP
}
