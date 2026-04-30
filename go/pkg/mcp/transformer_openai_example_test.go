package mcp

import core "dappco.re/go"

func ExampleOpenAITransformer_Detect() {
	var subject OpenAITransformer
	_ = subject.Detect
	core.Println("OpenAITransformer.Detect")
	// Output: OpenAITransformer.Detect
}

func ExampleOpenAITransformer_Normalise() {
	var subject OpenAITransformer
	_ = subject.Normalise
	core.Println("OpenAITransformer.Normalise")
	// Output: OpenAITransformer.Normalise
}

func ExampleOpenAITransformer_Transform() {
	var subject OpenAITransformer
	_ = subject.Transform
	core.Println("OpenAITransformer.Transform")
	// Output: OpenAITransformer.Transform
}
