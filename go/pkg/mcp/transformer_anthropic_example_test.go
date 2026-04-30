package mcp

import core "dappco.re/go"

func ExampleAnthropicTransformer_Detect() {
	var subject AnthropicTransformer
	_ = subject.Detect
	core.Println("AnthropicTransformer.Detect")
	// Output: AnthropicTransformer.Detect
}

func ExampleAnthropicTransformer_Normalise() {
	var subject AnthropicTransformer
	_ = subject.Normalise
	core.Println("AnthropicTransformer.Normalise")
	// Output: AnthropicTransformer.Normalise
}

func ExampleAnthropicTransformer_Transform() {
	var subject AnthropicTransformer
	_ = subject.Transform
	core.Println("AnthropicTransformer.Transform")
	// Output: AnthropicTransformer.Transform
}
