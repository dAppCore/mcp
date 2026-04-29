package mcp

import core "dappco.re/go"

func ExampleSubsystem() {
	var _ Subsystem
	core.Println("Subsystem")
	// Output: Subsystem
}

func ExampleSubsystemWithShutdown() {
	var _ SubsystemWithShutdown
	core.Println("SubsystemWithShutdown")
	// Output: SubsystemWithShutdown
}

func ExampleNotifier() {
	var _ Notifier
	core.Println("Notifier")
	// Output: Notifier
}

func ExampleChannelPush() {
	var _ ChannelPush
	core.Println("ChannelPush")
	// Output: ChannelPush
}

func ExampleSubsystemWithNotifier() {
	var _ SubsystemWithNotifier
	core.Println("SubsystemWithNotifier")
	// Output: SubsystemWithNotifier
}

func ExampleSubsystemWithChannelCallback() {
	var _ SubsystemWithChannelCallback
	core.Println("SubsystemWithChannelCallback")
	// Output: SubsystemWithChannelCallback
}
