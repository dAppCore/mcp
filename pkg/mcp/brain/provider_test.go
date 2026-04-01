// SPDX-License-Identifier: EUPL-1.2

package brain

import "testing"

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
