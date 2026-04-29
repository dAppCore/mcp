package brain

import core "dappco.re/go"

func ExampleNewProvider() {
	_ = NewProvider
	core.Println("NewProvider")
	// Output: NewProvider
}

func ExampleBrainProvider_Name() {
	var subject BrainProvider
	_ = subject.Name
	core.Println("BrainProvider.Name")
	// Output: BrainProvider.Name
}

func ExampleBrainProvider_BasePath() {
	var subject BrainProvider
	_ = subject.BasePath
	core.Println("BrainProvider.BasePath")
	// Output: BrainProvider.BasePath
}

func ExampleBrainProvider_Channels() {
	var subject BrainProvider
	_ = subject.Channels
	core.Println("BrainProvider.Channels")
	// Output: BrainProvider.Channels
}

func ExampleBrainProvider_Element() {
	var subject BrainProvider
	_ = subject.Element
	core.Println("BrainProvider.Element")
	// Output: BrainProvider.Element
}

func ExampleBrainProvider_RegisterRoutes() {
	var subject BrainProvider
	_ = subject.RegisterRoutes
	core.Println("BrainProvider.RegisterRoutes")
	// Output: BrainProvider.RegisterRoutes
}

func ExampleBrainProvider_Describe() {
	var subject BrainProvider
	_ = subject.Describe
	core.Println("BrainProvider.Describe")
	// Output: BrainProvider.Describe
}
