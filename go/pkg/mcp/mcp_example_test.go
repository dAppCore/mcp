package mcp

import core "dappco.re/go"

func ExampleNew() {
	_ = New
	core.Println("New")
	// Output: New
}

func ExampleService_Subsystems() {
	var subject Service
	_ = subject.Subsystems
	core.Println("Service.Subsystems")
	// Output: Service.Subsystems
}

func ExampleService_SubsystemsSeq() {
	var subject Service
	_ = subject.SubsystemsSeq
	core.Println("Service.SubsystemsSeq")
	// Output: Service.SubsystemsSeq
}

func ExampleService_Tools() {
	var subject Service
	_ = subject.Tools
	core.Println("Service.Tools")
	// Output: Service.Tools
}

func ExampleService_ToolsSeq() {
	var subject Service
	_ = subject.ToolsSeq
	core.Println("Service.ToolsSeq")
	// Output: Service.ToolsSeq
}

func ExampleService_Shutdown() {
	var subject Service
	_ = subject.Shutdown
	core.Println("Service.Shutdown")
	// Output: Service.Shutdown
}

func ExampleService_WSHub() {
	var subject Service
	_ = subject.WSHub
	core.Println("Service.WSHub")
	// Output: Service.WSHub
}

func ExampleService_ProcessService() {
	var subject Service
	_ = subject.ProcessService
	core.Println("Service.ProcessService")
	// Output: Service.ProcessService
}

func ExampleService_Run() {
	var subject Service
	_ = subject.Run
	core.Println("Service.Run")
	// Output: Service.Run
}

func ExampleService_Server() {
	var subject Service
	_ = subject.Server
	core.Println("Service.Server")
	// Output: Service.Server
}
