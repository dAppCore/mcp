package mcp

import core "dappco.re/go"

func ExampleNegotiateTransformer() {
	_ = NegotiateTransformer
	core.Println("NegotiateTransformer")
	// Output: NegotiateTransformer
}

func ExampleMCPNativeTransformer_Detect() {
	var subject MCPNativeTransformer
	_ = subject.Detect
	core.Println("MCPNativeTransformer.Detect")
	// Output: MCPNativeTransformer.Detect
}

func ExampleMCPNativeTransformer_Normalise() {
	var subject MCPNativeTransformer
	_ = subject.Normalise
	core.Println("MCPNativeTransformer.Normalise")
	// Output: MCPNativeTransformer.Normalise
}

func ExampleMCPNativeTransformer_Transform() {
	var subject MCPNativeTransformer
	_ = subject.Transform
	core.Println("MCPNativeTransformer.Transform")
	// Output: MCPNativeTransformer.Transform
}
