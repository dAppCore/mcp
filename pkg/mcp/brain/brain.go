// SPDX-License-Identifier: EUPL-1.2

// Package brain provides an MCP subsystem that proxies OpenBrain knowledge
// store operations to the Laravel php-agentic backend via the IDE bridge.
package brain

import (
	"context"

	coremcp "dappco.re/go/mcp/pkg/mcp"
	"dappco.re/go/mcp/pkg/mcp/ide"
	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// errBridgeNotAvailable is returned when a tool requires the Laravel bridge
// but it has not been initialised (headless mode).
var errBridgeNotAvailable = coreerr.E("brain", "bridge not available", nil)

// Subsystem implements mcp.Subsystem for OpenBrain knowledge store operations.
// It proxies brain_* tool calls to the Laravel backend via the shared IDE bridge.
type Subsystem struct {
	bridge   *ide.Bridge
	notifier Notifier
}

// New creates a brain subsystem that uses the given IDE bridge for Laravel communication.
//
//	brain := New(ideBridge)
//
// Pass nil if headless (tools will return errBridgeNotAvailable).
func New(bridge *ide.Bridge) *Subsystem {
	s := &Subsystem{bridge: bridge}
	if bridge != nil {
		bridge.AddObserver(func(msg ide.BridgeMessage) {
			s.handleBridgeMessage(msg)
		})
	}
	return s
}

// Name implements mcp.Subsystem.
func (s *Subsystem) Name() string { return "brain" }

// Notifier pushes events to MCP sessions (matches pkg/mcp.Notifier).
type Notifier interface {
	ChannelSend(ctx context.Context, channel string, data any)
}

// SetNotifier stores the shared notifier so this subsystem can emit channel events.
func (s *Subsystem) SetNotifier(n Notifier) {
	s.notifier = n
}

// RegisterTools implements mcp.Subsystem.
func (s *Subsystem) RegisterTools(server *mcp.Server) {
	s.registerBrainTools(server)
}

func (s *Subsystem) handleBridgeMessage(msg ide.BridgeMessage) {
	if msg.Type != "brain_recall" {
		return
	}

	payload := map[string]any{}
	if data, ok := msg.Data.(map[string]any); ok {
		for _, key := range []string{"query", "project", "type", "agent_id"} {
			if value, ok := data[key]; ok {
				payload[key] = value
			}
		}
		if count, ok := data["count"]; ok {
			payload["count"] = count
		} else if memories, ok := data["memories"].([]any); ok {
			payload["count"] = len(memories)
		}
	}
	if _, ok := payload["count"]; !ok {
		payload["count"] = 0
	}

	s.emitChannel(context.Background(), coremcp.ChannelBrainRecallDone, payload)
}

// Shutdown implements mcp.SubsystemWithShutdown.
func (s *Subsystem) Shutdown(_ context.Context) error {
	return nil
}
