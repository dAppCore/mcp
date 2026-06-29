// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"strings"
	"testing"
)

func TestToolsSubsystem_brainForget_Bad_NilBridge(t *testing.T) {
	sub := &Subsystem{}
	if _, _, err := sub.brainForget(context.Background(), nil, ForgetInput{ID: "m1"}); err == nil {
		t.Fatal("expected errBridgeNotAvailable with nil bridge")
	}
}

func TestToolsSubsystem_brainForget_Ugly_BridgeNotConnected(t *testing.T) {
	sub := New(disconnectedBridge())
	if _, _, err := sub.brainForget(context.Background(), nil, ForgetInput{ID: "mem-9", Reason: "stale"}); err == nil {
		t.Fatal("expected send failure with disconnected bridge")
	}
}

func TestToolsSubsystem_brainList_Bad_NilBridge(t *testing.T) {
	sub := &Subsystem{}
	if _, _, err := sub.brainList(context.Background(), nil, ListInput{Project: "core/mcp"}); err == nil {
		t.Fatal("expected errBridgeNotAvailable with nil bridge")
	}
}

func TestToolsSubsystem_brainList_Bad_OrgTooLong(t *testing.T) {
	sub := New(disconnectedBridge())
	// Validation runs before the bridge send, so an over-long org fails fast.
	longOrg := strings.Repeat("a", 200)
	if _, _, err := sub.brainList(context.Background(), nil, ListInput{Org: longOrg}); err == nil {
		t.Fatal("expected validation error for over-length org")
	}
}

func TestToolsSubsystem_brainList_Ugly_BridgeNotConnected(t *testing.T) {
	sub := New(disconnectedBridge())
	// Valid input (default limit applied), then Send fails on the dead bridge.
	if _, _, err := sub.brainList(context.Background(), nil, ListInput{Project: "core/mcp"}); err == nil {
		t.Fatal("expected send failure with disconnected bridge")
	}
}

func TestToolsSubsystem_validateListInput_Good_Bad(t *testing.T) {
	if err := validateListInput(ListInput{Org: "core"}); err != nil {
		t.Fatalf("expected valid short org, got %v", err)
	}
	if err := validateListInput(ListInput{Org: strings.Repeat("x", 200)}); err == nil {
		t.Fatal("expected error for over-length org")
	}
}
