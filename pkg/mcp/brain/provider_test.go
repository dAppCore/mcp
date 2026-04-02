// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"testing"

	"dappco.re/go/mcp/pkg/mcp/ide"
)

func TestBrainProviderChannels_Good_IncludesListComplete(t *testing.T) {
	p := NewProvider(nil, nil)

	channels := p.Channels()
	found := false
	for _, channel := range channels {
		if channel == "brain.list.complete" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected brain.list.complete in provider channels: %#v", channels)
	}
}

func TestBrainProviderHandleBridgeMessage_Good_SupportsBrainEvents(t *testing.T) {
	p := NewProvider(nil, nil)
	for _, msg := range []ide.BridgeMessage{
		{Type: "brain_remember", Data: map[string]any{"type": "bug", "project": "core/mcp"}},
		{Type: "brain_recall", Data: map[string]any{"query": "test", "memories": []any{map[string]any{"id": "m1"}}}},
		{Type: "brain_forget", Data: map[string]any{"id": "mem-123", "reason": "outdated"}},
		{Type: "brain_list", Data: map[string]any{"project": "core/mcp", "limit": 10}},
	} {
		p.handleBridgeMessage(msg)
	}
}
