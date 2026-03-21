// SPDX-License-Identifier: EUPL-1.2

// Notification broadcasting for the MCP service.
// Pushes events to connected MCP sessions via the logging protocol.
// Channel events use the claude/channel experimental capability.

package mcp

import (
	"context"
	"iter"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SendNotificationToAllClients broadcasts a log-level notification to every
// connected MCP session (stdio, HTTP, TCP, and Unix).
// Errors on individual sessions are logged but do not stop the broadcast.
//
//	s.SendNotificationToAllClients(ctx, "info", "monitor", map[string]any{"event": "build complete"})
func (s *Service) SendNotificationToAllClients(ctx context.Context, level mcp.LoggingLevel, logger string, data any) {
	for session := range s.server.Sessions() {
		if err := session.Log(ctx, &mcp.LoggingMessageParams{
			Level:  level,
			Logger: logger,
			Data:   data,
		}); err != nil {
			s.logger.Debug("notify: failed to send to session", "session", session.ID(), "error", err)
		}
	}
}

// ChannelSend pushes a channel event to all connected clients.
// Channel names follow "subsystem.event" convention.
//
//	s.ChannelSend(ctx, "agent.complete", map[string]any{"repo": "go-io", "workspace": "go-io-123"})
//	s.ChannelSend(ctx, "build.failed", map[string]any{"repo": "core", "error": "test timeout"})
func (s *Service) ChannelSend(ctx context.Context, channel string, data any) {
	payload := map[string]any{
		"channel": channel,
		"data":    data,
	}
	s.SendNotificationToAllClients(ctx, "info", "channel", payload)
}

// ChannelSendToSession pushes a channel event to a specific session.
//
//	s.ChannelSendToSession(ctx, session, "agent.progress", progressData)
func (s *Service) ChannelSendToSession(ctx context.Context, session *mcp.ServerSession, channel string, data any) {
	payload := map[string]any{
		"channel": channel,
		"data":    data,
	}
	if err := session.Log(ctx, &mcp.LoggingMessageParams{
		Level:  "info",
		Logger: "channel",
		Data:   payload,
	}); err != nil {
		s.logger.Debug("channel: failed to send to session", "session", session.ID(), "channel", channel, "error", err)
	}
}

// Sessions returns an iterator over all connected MCP sessions.
//
//	for session := range s.Sessions() {
//	    s.ChannelSendToSession(ctx, session, "status", data)
//	}
func (s *Service) Sessions() iter.Seq[*mcp.ServerSession] {
	return s.server.Sessions()
}

// channelCapability returns the experimental capability descriptor
// for claude/channel, registered during New().
func channelCapability() map[string]any {
	return map[string]any{
		"claude/channel": map[string]any{
			"version":     "1",
			"description": "Push events into client sessions via named channels",
			"channels": []string{
				"agent.complete",
				"agent.blocked",
				"agent.status",
				"build.complete",
				"build.failed",
				"brain.recall.complete",
				"inbox.message",
				"process.exit",
				"harvest.complete",
				"test.result",
			},
		},
	}
}
