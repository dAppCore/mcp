// SPDX-License-Identifier: EUPL-1.2

// Notification broadcasting for the MCP service.
// Channel events use the claude/channel experimental capability
// via notifications/claude/channel JSON-RPC notifications.

package mcp

import (
	"context"
	"iter"
	"os"
	"sync"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// stdoutMu protects stdout writes from concurrent goroutines.
var stdoutMu sync.Mutex

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

// channelNotification is the JSON-RPC notification format for claude/channel.
type channelNotification struct {
	JSONRPC string         `json:"jsonrpc"`
	Method  string         `json:"method"`
	Params  channelParams  `json:"params"`
}

type channelParams struct {
	Content string            `json:"content"`
	Meta    map[string]string `json:"meta,omitempty"`
}

// ChannelSend pushes a channel event to all connected clients via
// the notifications/claude/channel JSON-RPC method.
//
//	s.ChannelSend(ctx, "agent.complete", map[string]any{"repo": "go-io", "workspace": "go-io-123"})
//	s.ChannelSend(ctx, "build.failed", map[string]any{"repo": "core", "error": "test timeout"})
func (s *Service) ChannelSend(ctx context.Context, channel string, data any) {
	// Marshal the data payload as the content string
	content := core.JSONMarshalString(data)

	notification := channelNotification{
		JSONRPC: "2.0",
		Method:  "notifications/claude/channel",
		Params: channelParams{
			Content: content,
			Meta: map[string]string{
				"source":  "core-agent",
				"channel": channel,
			},
		},
	}

	msg := core.JSONMarshalString(notification)

	// Write directly to stdout (stdio transport) with newline delimiter.
	// The official SDK doesn't expose a way to send custom notification methods,
	// so we write the JSON-RPC notification directly to the transport.
	// Only write when running in stdio mode — HTTP/TCP transports don't use stdout.
	if !s.stdioMode {
		return
	}
	stdoutMu.Lock()
	os.Stdout.Write([]byte(core.Concat(msg, "\n")))
	stdoutMu.Unlock()
}

// ChannelSendToSession pushes a channel event to a specific session.
// Falls back to stdout for stdio transport.
//
//	s.ChannelSendToSession(ctx, session, "agent.progress", progressData)
func (s *Service) ChannelSendToSession(ctx context.Context, session *mcp.ServerSession, channel string, data any) {
	// For now, channel events go to all sessions via stdout
	s.ChannelSend(ctx, channel, data)
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
