package agentic

import "testing"

func TestQueue_PublicTypes(t *testing.T) {
	cfg := AgentsConfig{Dispatch: DispatchConfig{DefaultAgent: "claude"}, Concurrency: map[string]int{"claude": 1}}
	cfg.Rates = map[string]RateConfig{"claude": {ResetUTC: "06:00", SustainedDelay: 60}}
	if cfg.Dispatch.DefaultAgent != "claude" || cfg.Concurrency["claude"] != 1 {
		t.Fatalf("unexpected agents config: %+v", cfg)
	}
}
