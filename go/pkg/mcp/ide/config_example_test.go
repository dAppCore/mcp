package ide

import core "dappco.re/go"

func ExampleDefaultConfig() {
	_ = DefaultConfig
	core.Println("DefaultConfig")
	// Output: DefaultConfig
}

func ExampleConfig_WithDefaults() {
	var subject Config
	_ = subject.WithDefaults
	core.Println("Config.WithDefaults")
	// Output: Config.WithDefaults
}
