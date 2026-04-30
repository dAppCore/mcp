// SPDX-License-Identifier: EUPL-1.2

package agentic

import (
	"testing"
	"time"
)

func TestWatchDefaults_Good_RFCOneMinuteTimeout(t *testing.T) {
	if defaultWatchTimeout != 60*time.Second {
		t.Fatalf("expected default watch timeout to be 60s, got %s", defaultWatchTimeout)
	}
	if defaultWatchPollInterval != 5*time.Second {
		t.Fatalf("expected default poll interval to be 5s, got %s", defaultWatchPollInterval)
	}
	if maxWatchTimeout != 30*time.Minute {
		t.Fatalf("expected max watch timeout to be 30m, got %s", maxWatchTimeout)
	}
}

func TestResolveWatchTimeout_Good_HonorsInputTimeout(t *testing.T) {
	got := resolveWatchTimeout(WatchInput{Timeout: 10})
	if got != 10*time.Second {
		t.Fatalf("expected input timeout to be honored as 10s, got %s", got)
	}
}

func TestResolveWatchTimeout_Good_ClampsInputTimeout(t *testing.T) {
	got := resolveWatchTimeout(WatchInput{Timeout: int((10 * time.Hour) / time.Second)})
	if got != 30*time.Minute {
		t.Fatalf("expected input timeout to clamp to 30m, got %s", got)
	}
}

func TestResolveWatchTimeout_Good_ZeroUsesDefault(t *testing.T) {
	got := resolveWatchTimeout(WatchInput{Timeout: 0})
	if got != defaultWatchTimeout {
		t.Fatalf("expected zero timeout to use default %s, got %s", defaultWatchTimeout, got)
	}
}
