package mcp

import core "dappco.re/go"

func ExampleNewProcessEventCallback() {
	_ = NewProcessEventCallback
	core.Println("NewProcessEventCallback")
	// Output: NewProcessEventCallback
}

func ExampleProcessEventCallback_OnProcessOutput() {
	var subject ProcessEventCallback
	_ = subject.OnProcessOutput
	core.Println("ProcessEventCallback.OnProcessOutput")
	// Output: ProcessEventCallback.OnProcessOutput
}

func ExampleProcessEventCallback_OnProcessStatus() {
	var subject ProcessEventCallback
	_ = subject.OnProcessStatus
	core.Println("ProcessEventCallback.OnProcessStatus")
	// Output: ProcessEventCallback.OnProcessStatus
}
