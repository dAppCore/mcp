package mcp

import core "dappco.re/go"

type Transport = connTransport
type Connection = connConnection

func ExampleNewTCPTransport() {
	_ = NewTCPTransport
	core.Println("NewTCPTransport")
	// Output: NewTCPTransport
}

func ExampleService_ServeTCP() {
	var subject Service
	_ = subject.ServeTCP
	core.Println("Service.ServeTCP")
	// Output: Service.ServeTCP
}

func ExampleTransport_Connect() {
	var subject Transport
	_ = subject.Connect
	core.Println("Transport.Connect")
	// Output: Transport.Connect
}

func ExampleConnection_Read() {
	var subject Connection
	_ = subject.Read
	core.Println("Connection.Read")
	// Output: Connection.Read
}

func ExampleConnection_Write() {
	var subject Connection
	_ = subject.Write
	core.Println("Connection.Write")
	// Output: Connection.Write
}

func ExampleConnection_Close() {
	var subject Connection
	_ = subject.Close
	core.Println("Connection.Close")
	// Output: Connection.Close
}

func ExampleConnection_SessionID() {
	var subject Connection
	_ = subject.SessionID
	core.Println("Connection.SessionID")
	// Output: Connection.SessionID
}
