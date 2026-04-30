// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"

	coremcp "dappco.re/go/mcp/pkg/mcp"
)

func bridgePayload(data any, keys ...string) map[string]any {
	payload := make(map[string]any)

	m, ok := data.(map[string]any)
	if !ok {
		return payload
	}

	for _, key := range keys {
		if value, ok := m[key]; ok {
			payload[key] = value
		}
	}

	return payload
}

func bridgeCount(data any) int {
	m, ok := data.(map[string]any)
	if !ok {
		return 0
	}

	if count, ok := m["count"]; ok {
		switch v := count.(type) {
		case int:
			return v
		case int32:
			return int(v)
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}

	if memories, ok := m["memories"].([]any); ok {
		return len(memories)
	}

	return 0
}

func emitBridgeChannel(ctx context.Context, notifier coremcp.Notifier, channel string, data any) {
	if notifier == nil {
		return
	}
	notifier.ChannelSend(ctx, channel, data)
}
