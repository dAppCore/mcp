// SPDX-License-Identifier: EUPL-1.2

// Package brain provides an MCP subsystem that proxies OpenBrain knowledge
// store operations to the Laravel php-agentic backend via the IDE bridge.
package brain

import (
	"context"

	coreerr "dappco.re/go/log"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"dappco.re/go/mcp/pkg/mcp/ide"
)

// errBridgeNotAvailable is returned when a tool requires the Laravel bridge
// but it has not been initialised (headless mode).
var errBridgeNotAvailable = coreerr.E("brain", "bridge not available", nil)

// Subsystem implements mcp.Subsystem for OpenBrain knowledge store operations.
// It proxies brain_* tool calls to the Laravel backend via the shared IDE bridge.
type Subsystem struct {
	bridge   *ide.Bridge
	notifier coremcp.Notifier
}

var (
	_ coremcp.Subsystem             = (*Subsystem)(nil)
	_ coremcp.SubsystemWithShutdown = (*Subsystem)(nil)
	_ coremcp.SubsystemWithNotifier = (*Subsystem)(nil)
)

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

// SetNotifier stores the shared notifier so this subsystem can emit channel events.
func (s *Subsystem) SetNotifier(n coremcp.Notifier) {
	s.notifier = n
}

// RegisterTools implements mcp.Subsystem.
func (s *Subsystem) RegisterTools(svc *coremcp.Service) {
	s.registerBrainTools(svc)
}

func (s *Subsystem) handleBridgeMessage(msg ide.BridgeMessage) {
	switch msg.Type {
	case "brain_remember":
		emitBridgeChannel(context.Background(), s.notifier, coremcp.ChannelBrainRememberDone, bridgePayload(msg.Data, "org", "type", "project"))
	case "brain_recall":
		payload := bridgePayload(msg.Data, "query", "org", "project", "type", "agent_id")
		payload["count"] = bridgeCount(msg.Data)
		emitBridgeChannel(context.Background(), s.notifier, coremcp.ChannelBrainRecallDone, payload)
	case "brain_forget":
		emitBridgeChannel(context.Background(), s.notifier, coremcp.ChannelBrainForgetDone, bridgePayload(msg.Data, "id", "reason"))
	case "brain_list":
		emitBridgeChannel(context.Background(), s.notifier, coremcp.ChannelBrainListDone, bridgePayload(msg.Data, "org", "project", "type", "agent_id", "limit"))
	}
}

// Shutdown implements mcp.SubsystemWithShutdown.
func (s *Subsystem) Shutdown(_ context.Context) error {
	return nil
}
