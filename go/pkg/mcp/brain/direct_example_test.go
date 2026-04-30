package brain

import core "dappco.re/go"

func ExampleDirectSubsystem_OnChannel() {
	var subject DirectSubsystem
	_ = subject.OnChannel
	core.Println("DirectSubsystem.OnChannel")
	// Output: DirectSubsystem.OnChannel
}

func ExampleNewDirect() {
	_ = NewDirect
	core.Println("NewDirect")
	// Output: NewDirect
}

func ExampleNewDirectWithClient() {
	_ = NewDirectWithClient
	core.Println("NewDirectWithClient")
	// Output: NewDirectWithClient
}

func ExampleDirectSubsystem_Name() {
	var subject DirectSubsystem
	_ = subject.Name
	core.Println("DirectSubsystem.Name")
	// Output: DirectSubsystem.Name
}

func ExampleDirectSubsystem_RegisterTools() {
	var subject DirectSubsystem
	_ = subject.RegisterTools
	core.Println("DirectSubsystem.RegisterTools")
	// Output: DirectSubsystem.RegisterTools
}

func ExampleDirectSubsystem_Shutdown() {
	var subject DirectSubsystem
	_ = subject.Shutdown
	core.Println("DirectSubsystem.Shutdown")
	// Output: DirectSubsystem.Shutdown
}
