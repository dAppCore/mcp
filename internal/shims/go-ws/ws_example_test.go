package ws

import core "dappco.re/go"

func ExampleNewHub() {
	_ = NewHub
	core.Println("NewHub")
	// Output: NewHub
}

func ExampleHub_Run() {
	var subject Hub
	_ = subject.Run
	core.Println("Hub.Run")
	// Output: Hub.Run
}

func ExampleHub_Handler() {
	var subject Hub
	_ = subject.Handler
	core.Println("Hub.Handler")
	// Output: Hub.Handler
}

func ExampleHub_SendToChannel() {
	var subject Hub
	_ = subject.SendToChannel
	core.Println("Hub.SendToChannel")
	// Output: Hub.SendToChannel
}

func ExampleHub_Stats() {
	var subject Hub
	_ = subject.Stats
	core.Println("Hub.Stats")
	// Output: Hub.Stats
}

func ExampleHub_SendProcessOutput() {
	var subject Hub
	_ = subject.SendProcessOutput
	core.Println("Hub.SendProcessOutput")
	// Output: Hub.SendProcessOutput
}

func ExampleHub_SendProcessStatus() {
	var subject Hub
	_ = subject.SendProcessStatus
	core.Println("Hub.SendProcessStatus")
	// Output: Hub.SendProcessStatus
}
