package agentic

import core "dappco.re/go"

func ExampleMirrorInput() {
	var _ MirrorInput
	core.Println("MirrorInput")
	// Output: MirrorInput
}

func ExampleMirrorOutput() {
	var _ MirrorOutput
	core.Println("MirrorOutput")
	// Output: MirrorOutput
}

func ExampleMirrorSync() {
	var _ MirrorSync
	core.Println("MirrorSync")
	// Output: MirrorSync
}
