package ide

import core "dappco.re/go"

func ExampleNewBridge() {
	_ = NewBridge
	core.Println("NewBridge")
	// Output: NewBridge
}

func ExampleBridge_SetObserver() {
	var subject Bridge
	_ = subject.SetObserver
	core.Println("Bridge.SetObserver")
	// Output: Bridge.SetObserver
}

func ExampleBridge_AddObserver() {
	var subject Bridge
	_ = subject.AddObserver
	core.Println("Bridge.AddObserver")
	// Output: Bridge.AddObserver
}

func ExampleBridge_Start() {
	var subject Bridge
	_ = subject.Start
	core.Println("Bridge.Start")
	// Output: Bridge.Start
}

func ExampleBridge_Shutdown() {
	var subject Bridge
	_ = subject.Shutdown
	core.Println("Bridge.Shutdown")
	// Output: Bridge.Shutdown
}

func ExampleBridge_Connected() {
	var subject Bridge
	_ = subject.Connected
	core.Println("Bridge.Connected")
	// Output: Bridge.Connected
}

func ExampleBridge_Send() {
	var subject Bridge
	_ = subject.Send
	core.Println("Bridge.Send")
	// Output: Bridge.Send
}
