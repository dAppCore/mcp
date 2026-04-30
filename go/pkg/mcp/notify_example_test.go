package mcp

import core "dappco.re/go"

type Writer = lockedWriter
type Error = notificationError

func ExampleWriter_Write() {
	var subject Writer
	_ = subject.Write
	core.Println("Writer.Write")
	// Output: Writer.Write
}

func ExampleWriter_Close() {
	var subject Writer
	_ = subject.Close
	core.Println("Writer.Close")
	// Output: Writer.Close
}

func ExampleChannelCapabilitySpec_Map() {
	var subject ChannelCapabilitySpec
	_ = subject.Map
	core.Println("ChannelCapabilitySpec.Map")
	// Output: ChannelCapabilitySpec.Map
}

func ExampleService_SendNotificationToAllClients() {
	var subject Service
	_ = subject.SendNotificationToAllClients
	core.Println("Service.SendNotificationToAllClients")
	// Output: Service.SendNotificationToAllClients
}

func ExampleService_SendNotificationToSession() {
	var subject Service
	_ = subject.SendNotificationToSession
	core.Println("Service.SendNotificationToSession")
	// Output: Service.SendNotificationToSession
}

func ExampleService_SendNotificationToClient() {
	var subject Service
	_ = subject.SendNotificationToClient
	core.Println("Service.SendNotificationToClient")
	// Output: Service.SendNotificationToClient
}

func ExampleService_ChannelSend() {
	var subject Service
	_ = subject.ChannelSend
	core.Println("Service.ChannelSend")
	// Output: Service.ChannelSend
}

func ExampleService_ChannelSendToSession() {
	var subject Service
	_ = subject.ChannelSendToSession
	core.Println("Service.ChannelSendToSession")
	// Output: Service.ChannelSendToSession
}

func ExampleService_ChannelSendToClient() {
	var subject Service
	_ = subject.ChannelSendToClient
	core.Println("Service.ChannelSendToClient")
	// Output: Service.ChannelSendToClient
}

func ExampleService_Sessions() {
	var subject Service
	_ = subject.Sessions
	core.Println("Service.Sessions")
	// Output: Service.Sessions
}

func ExampleNotifySession() {
	_ = NotifySession
	core.Println("NotifySession")
	// Output: NotifySession
}

func ExampleError_Error() {
	var subject Error
	_ = subject.Error
	core.Println("Error.Error")
	// Output: Error.Error
}

func ExampleClaudeChannelCapability() {
	_ = ClaudeChannelCapability
	core.Println("ClaudeChannelCapability")
	// Output: ClaudeChannelCapability
}

func ExampleChannelCapability() {
	_ = ChannelCapability
	core.Println("ChannelCapability")
	// Output: ChannelCapability
}

func ExampleChannelCapabilityChannels() {
	_ = ChannelCapabilityChannels
	core.Println("ChannelCapabilityChannels")
	// Output: ChannelCapabilityChannels
}
