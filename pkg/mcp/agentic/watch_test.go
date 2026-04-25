// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"
)

func TestWatchDefaults_Good_RFCThirtyMinuteTimeout(t *testing.T) {
	if defaultWatchTimeout != 30*time.Minute {
		t.Fatalf("expected default watch timeout to be 30m, got %s", defaultWatchTimeout)
	}
	if defaultWatchPollInterval != 5*time.Second {
		t.Fatalf("expected default poll interval to be 5s, got %s", defaultWatchPollInterval)
	}
}
