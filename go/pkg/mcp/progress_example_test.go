package mcp

import core "dappco.re/go"

func ExampleProgressTokenFromRequest() {
	_ = ProgressTokenFromRequest
	core.Println("ProgressTokenFromRequest")
	// Output: ProgressTokenFromRequest
}

func ExampleSendProgressNotification() {
	_ = SendProgressNotification
	core.Println("SendProgressNotification")
	// Output: SendProgressNotification
}

func ExampleNewProgressNotifier() {
	_ = NewProgressNotifier
	core.Println("NewProgressNotifier")
	// Output: NewProgressNotifier
}

func ExampleProgressNotifier_Send() {
	var subject ProgressNotifier
	_ = subject.Send
	core.Println("ProgressNotifier.Send")
	// Output: ProgressNotifier.Send
}
