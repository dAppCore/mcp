package client

import core "dappco.re/go"

func ExampleNew() {
	_ = New
	core.Println("New")
	// Output: New
}

func ExampleNewFromEnvironment() {
	_ = NewFromEnvironment
	core.Println("NewFromEnvironment")
	// Output: NewFromEnvironment
}

func ExampleWriteBrainKey() {
	_ = WriteBrainKey
	core.Println("WriteBrainKey")
	// Output: WriteBrainKey
}

func ExampleNewCircuitBreaker() {
	_ = NewCircuitBreaker
	core.Println("NewCircuitBreaker")
	// Output: NewCircuitBreaker
}

func ExampleCircuitBreaker_State() {
	var subject CircuitBreaker
	_ = subject.State
	core.Println("CircuitBreaker.State")
	// Output: CircuitBreaker.State
}

func ExampleClient_Remember() {
	var subject Client
	_ = subject.Remember
	core.Println("Client.Remember")
	// Output: Client.Remember
}

func ExampleClient_Recall() {
	var subject Client
	_ = subject.Recall
	core.Println("Client.Recall")
	// Output: Client.Recall
}

func ExampleClient_Forget() {
	var subject Client
	_ = subject.Forget
	core.Println("Client.Forget")
	// Output: Client.Forget
}

func ExampleClient_List() {
	var subject Client
	_ = subject.List
	core.Println("Client.List")
	// Output: Client.List
}

func ExampleClient_Call() {
	var subject Client
	_ = subject.Call
	core.Println("Client.Call")
	// Output: Client.Call
}
