package agentic

import core "dappco.re/go"

func ExampleEpicInput() {
	var _ EpicInput
	core.Println("EpicInput")
	// Output: EpicInput
}

func ExampleEpicOutput() {
	var _ EpicOutput
	core.Println("EpicOutput")
	// Output: EpicOutput
}

func ExampleChildRef() {
	var _ ChildRef
	core.Println("ChildRef")
	// Output: ChildRef
}
