package ide

import core "dappco.re/go"

func ExampleNew() {
	_ = New
	core.Println("New")
	// Output: New
}

func ExampleSubsystem_Name() {
	var subject Subsystem
	_ = subject.Name
	core.Println("Subsystem.Name")
	// Output: Subsystem.Name
}

func ExampleSubsystem_RegisterTools() {
	var subject Subsystem
	_ = subject.RegisterTools
	core.Println("Subsystem.RegisterTools")
	// Output: Subsystem.RegisterTools
}

func ExampleSubsystem_Shutdown() {
	var subject Subsystem
	_ = subject.Shutdown
	core.Println("Subsystem.Shutdown")
	// Output: Subsystem.Shutdown
}

func ExampleSubsystem_SetNotifier() {
	var subject Subsystem
	_ = subject.SetNotifier
	core.Println("Subsystem.SetNotifier")
	// Output: Subsystem.SetNotifier
}

func ExampleSubsystem_Bridge() {
	var subject Subsystem
	_ = subject.Bridge
	core.Println("Subsystem.Bridge")
	// Output: Subsystem.Bridge
}

func ExampleSubsystem_StartBridge() {
	var subject Subsystem
	_ = subject.StartBridge
	core.Println("Subsystem.StartBridge")
	// Output: Subsystem.StartBridge
}
