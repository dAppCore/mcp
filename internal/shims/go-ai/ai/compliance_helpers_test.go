package ai

import (
	. "dappco.re/go"
)

// moved helpers from ax7_triplets_test.go
func resetAX7Events() {
	events.Lock()
	defer events.Unlock()
	events.items = nil
}
