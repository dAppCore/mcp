package provider

import core "dappco.re/go"

func ExampleProvider() {
	var _ Provider
	core.Println("Provider")
	// Output: Provider
}

func ExampleStreamable() {
	var _ Streamable
	core.Println("Streamable")
	// Output: Streamable
}

func ExampleDescribable() {
	var _ Describable
	core.Println("Describable")
	// Output: Describable
}

func ExampleElementSpec() {
	var _ ElementSpec
	core.Println("ElementSpec")
	// Output: ElementSpec
}

func ExampleRenderable() {
	var _ Renderable
	core.Println("Renderable")
	// Output: Renderable
}
