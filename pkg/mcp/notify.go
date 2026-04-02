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
	"reflect"
	"sync"
	"unsafe"

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

const channelNotificationMethod = "notifications/claude/channel"
const loggingNotificationMethod = "notifications/message"

// ChannelNotification is the payload sent through the experimental channel
// notification method.
type ChannelNotification struct {
	Channel string `json:"channel"`
	Data    any    `json:"data"`
}

// SendNotificationToAllClients broadcasts a log-level notification to every
// connected MCP session (stdio, HTTP, TCP, and Unix).
// Errors on individual sessions are logged but do not stop the broadcast.
//
//	s.SendNotificationToAllClients(ctx, "info", "monitor", map[string]any{"event": "build complete"})
func (s *Service) SendNotificationToAllClients(ctx context.Context, level mcp.LoggingLevel, logger string, data any) {
	for session := range s.server.Sessions() {
		s.sendLoggingNotificationToSession(ctx, session, level, logger, data)
	}
}

// SendNotificationToSession sends a log-level notification to one connected
// MCP session.
//
//	s.SendNotificationToSession(ctx, session, "info", "monitor", data)
func (s *Service) SendNotificationToSession(ctx context.Context, session *mcp.ServerSession, level mcp.LoggingLevel, logger string, data any) {
	s.sendLoggingNotificationToSession(ctx, session, level, logger, data)
}

func (s *Service) sendLoggingNotificationToSession(ctx context.Context, session *mcp.ServerSession, level mcp.LoggingLevel, logger string, data any) {
	if session == nil {
		return
	}

	if err := sendSessionNotification(ctx, session, loggingNotificationMethod, &mcp.LoggingMessageParams{
		Level:  level,
		Logger: logger,
		Data:   data,
	}); err != nil {
		s.logger.Debug("notify: failed to send to session", "session", session.ID(), "error", err)
	}
}

// ChannelSend pushes a channel event to all connected clients via
// the notifications/claude/channel JSON-RPC method.
//
//	s.ChannelSend(ctx, "agent.complete", map[string]any{"repo": "go-io", "workspace": "go-io-123"})
//	s.ChannelSend(ctx, "build.failed", map[string]any{"repo": "core", "error": "test timeout"})
func (s *Service) ChannelSend(ctx context.Context, channel string, data any) {
	payload := ChannelNotification{Channel: channel, Data: data}
	s.sendChannelNotificationToAllClients(ctx, payload)
}

// ChannelSendToSession pushes a channel event to a specific session.
//
//	s.ChannelSendToSession(ctx, session, "agent.progress", progressData)
func (s *Service) ChannelSendToSession(ctx context.Context, session *mcp.ServerSession, channel string, data any) {
	if session == nil {
		return
	}

	payload := ChannelNotification{Channel: channel, Data: data}
	if err := sendSessionNotification(ctx, session, channelNotificationMethod, payload); err != nil {
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

func (s *Service) sendChannelNotificationToAllClients(ctx context.Context, payload ChannelNotification) {
	for session := range s.server.Sessions() {
		if err := sendSessionNotification(ctx, session, channelNotificationMethod, payload); err != nil {
			s.logger.Debug("channel: failed to send to session", "session", session.ID(), "error", err)
		}
	}
}

func sendSessionNotification(ctx context.Context, session *mcp.ServerSession, method string, payload any) error {
	if session == nil {
		return nil
	}

	conn, err := sessionConnection(session)
	if err != nil {
		return err
	}

	value := reflect.ValueOf(conn)
	call := value.MethodByName("Notify")
	if !call.IsValid() {
		return coreNotifyError("connection Notify method unavailable")
	}

	results := call.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(method),
		reflect.ValueOf(payload),
	})
	if len(results) != 1 {
		return coreNotifyError("unexpected Notify result shape")
	}
	if !results[0].IsNil() {
		if err, ok := results[0].Interface().(error); ok {
			return err
		}
		return coreNotifyError("Notify returned non-error result")
	}
	return nil
}

func sessionConnection(session *mcp.ServerSession) (any, error) {
	value := reflect.ValueOf(session)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return nil, coreNotifyError("invalid session")
	}

	field := value.Elem().FieldByName("conn")
	if !field.IsValid() {
		return nil, coreNotifyError("session connection field unavailable")
	}

	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface(), nil
}

func coreNotifyError(message string) error {
	return &notificationError{message: message}
}

type notificationError struct {
	message string
}

func (e *notificationError) Error() string {
	return e.message
}

// channelCapability returns the experimental capability descriptor
// for claude/channel, registered during New().
func channelCapability() map[string]any {
	return map[string]any{
		"claude/channel": map[string]any{
			"version":     "1",
			"description": "Push events into client sessions via named channels",
			"channels":    channelCapabilityChannels(),
		},
	}
}

// channelCapabilityChannels lists the named channel events advertised by the
// experimental capability.
func channelCapabilityChannels() []string {
	return []string{
		"agent.complete",
		"agent.blocked",
		"agent.status",
		"build.complete",
		"build.failed",
		"brain.forget.complete",
		"brain.list.complete",
		"brain.recall.complete",
		"brain.remember.complete",
		"harvest.complete",
		"inbox.message",
		"process.exit",
		"process.start",
		"test.result",
	}
}
