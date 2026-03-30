// SPDX-License-Identifier: EUPL-1.2

// Notification broadcasting for the MCP service.
// Channel events use the claude/channel experimental capability
// via notifications/claude/channel JSON-RPC notifications.

package mcp

import (
	"context"
	"io"
	"iter"
	"os"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// lockedWriter wraps an io.Writer with a mutex.
// Both the SDK's transport and ChannelSend use this writer,
// ensuring channel notifications don't interleave with SDK messages.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (lw *lockedWriter) Write(p []byte) (int, error) {
	lw.mu.Lock()
	defer lw.mu.Unlock()
	return lw.w.Write(p)
}

func (lw *lockedWriter) Close() error { return nil }

// sharedStdout is the single writer for all stdio output.
// Created once when the MCP service enters stdio mode.
var sharedStdout = &lockedWriter{w: os.Stdout}

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

// ChannelSend pushes a channel event to all connected clients via
// the notifications/claude/channel JSON-RPC method.
//
//	s.ChannelSend(ctx, "agent.complete", map[string]any{"repo": "go-io", "workspace": "go-io-123"})
//	s.ChannelSend(ctx, "build.failed", map[string]any{"repo": "core", "error": "test timeout"})
func (s *Service) ChannelSend(ctx context.Context, channel string, data any) {
	payload := map[string]any{
		"channel": channel,
		"data":    data,
	}
	s.SendNotificationToAllClients(ctx, mcp.LoggingLevel("info"), "channel", payload)
}

// ChannelSendToSession pushes a channel event to a specific session.
//
//	s.ChannelSendToSession(ctx, session, "agent.progress", progressData)
func (s *Service) ChannelSendToSession(ctx context.Context, session *mcp.ServerSession, channel string, data any) {
	if session == nil {
		return
	}

	payload := map[string]any{
		"channel": channel,
		"data":    data,
	}
	if err := session.Log(ctx, &mcp.LoggingMessageParams{
		Level:  mcp.LoggingLevel("info"),
		Logger: "channel",
		Data:   payload,
	}); err != nil {
		s.logger.Debug("channel: failed to send to session", "session", session.ID(), "error", err)
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
				"brain.list.complete",
				"brain.forget.complete",
				"brain.remember.complete",
				"brain.recall.complete",
				"inbox.message",
				"process.exit",
				"harvest.complete",
				"test.result",
			},
		},
	}
}
