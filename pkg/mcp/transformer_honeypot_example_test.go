package mcp

import core "dappco.re/go"

func ExampleHoneypotTransformer_Detect() {
	var subject HoneypotTransformer
	_ = subject.Detect
	core.Println("HoneypotTransformer.Detect")
	// Output: HoneypotTransformer.Detect
}

func ExampleHoneypotTransformer_Normalise() {
	var subject HoneypotTransformer
	_ = subject.Normalise
	core.Println("HoneypotTransformer.Normalise")
	// Output: HoneypotTransformer.Normalise
}

func ExampleHoneypotTransformer_Transform() {
	var subject HoneypotTransformer
	_ = subject.Transform
	core.Println("HoneypotTransformer.Transform")
	// Output: HoneypotTransformer.Transform
}
