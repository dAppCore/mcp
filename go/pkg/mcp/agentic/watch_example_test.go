package agentic

import core "dappco.re/go"

func ExampleWatchInput() {
	var _ WatchInput
	core.Println("WatchInput")
	// Output: WatchInput
}

func ExampleWatchOutput() {
	var _ WatchOutput
	core.Println("WatchOutput")
	// Output: WatchOutput
}

func ExampleWatchResult() {
	var _ WatchResult
	core.Println("WatchResult")
	// Output: WatchResult
}
