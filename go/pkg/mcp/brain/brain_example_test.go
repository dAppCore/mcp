package brain

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

func ExampleSubsystem_SetNotifier() {
	var subject Subsystem
	_ = subject.SetNotifier
	core.Println("Subsystem.SetNotifier")
	// Output: Subsystem.SetNotifier
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
