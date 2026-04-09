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
	"slices"
	"sort"
	"sync"
	"unsafe"

	core "dappco.re/go/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func normalizeNotificationContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

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

// ChannelNotificationMethod is the JSON-RPC method used for named channel
// events sent through claude/channel.
const ChannelNotificationMethod = "notifications/claude/channel"

// LoggingNotificationMethod is the JSON-RPC method used for log messages sent
// to connected MCP clients.
const LoggingNotificationMethod = "notifications/message"

// ClaudeChannelCapabilityName is the experimental capability key advertised
// by the MCP server for channel-based client notifications.
const ClaudeChannelCapabilityName = "claude/channel"

// Shared channel names. Keeping them central avoids drift between emitters
// and the advertised claude/channel capability.
//
// Use these names when emitting structured events from subsystems:
//
//	s.ChannelSend(ctx, ChannelProcessStart, map[string]any{"id": "proc-1"})
const (
	ChannelBuildStart        = "build.start"
	ChannelBuildComplete     = "build.complete"
	ChannelBuildFailed       = "build.failed"
	ChannelAgentComplete     = "agent.complete"
	ChannelAgentBlocked      = "agent.blocked"
	ChannelAgentStatus       = "agent.status"
	ChannelBrainForgetDone   = "brain.forget.complete"
	ChannelBrainListDone     = "brain.list.complete"
	ChannelBrainRecallDone   = "brain.recall.complete"
	ChannelBrainRememberDone = "brain.remember.complete"
	ChannelHarvestComplete   = "harvest.complete"
	ChannelInboxMessage      = "inbox.message"
	ChannelProcessExit       = "process.exit"
	ChannelProcessStart      = "process.start"
	ChannelProcessOutput     = "process.output"
	ChannelTestResult        = "test.result"
)

var channelCapabilityList = []string{
	ChannelBuildStart,
	ChannelAgentComplete,
	ChannelAgentBlocked,
	ChannelAgentStatus,
	ChannelBuildComplete,
	ChannelBuildFailed,
	ChannelBrainForgetDone,
	ChannelBrainListDone,
	ChannelBrainRecallDone,
	ChannelBrainRememberDone,
	ChannelHarvestComplete,
	ChannelInboxMessage,
	ChannelProcessExit,
	ChannelProcessStart,
	ChannelProcessOutput,
	ChannelTestResult,
}

// ChannelCapabilitySpec describes the experimental claude/channel capability.
//
//	spec := ChannelCapabilitySpec{
//	    Version:     "1",
//	    Description: "Push events into client sessions via named channels",
//	    Channels:    ChannelCapabilityChannels(),
//	}
type ChannelCapabilitySpec struct {
	Version     string   `json:"version"`     // e.g. "1"
	Description string   `json:"description"` // capability summary shown to clients
	Channels    []string `json:"channels"`    // e.g. []string{"build.complete", "agent.status"}
}

// Map converts the typed capability into the wire-format map expected by the SDK.
//
//	caps := ChannelCapabilitySpec{
//	    Version:     "1",
//	    Description: "Push events into client sessions via named channels",
//	    Channels:    ChannelCapabilityChannels(),
//	}.Map()
func (c ChannelCapabilitySpec) Map() map[string]any {
	return map[string]any{
		"version":     c.Version,
		"description": c.Description,
		"channels":    slices.Clone(c.Channels),
	}
}

// ChannelNotification is the payload sent through the experimental channel
// notification method.
//
//	n := ChannelNotification{
//	    Channel: ChannelBuildComplete,
//	    Data:    map[string]any{"repo": "core/mcp"},
//	}
type ChannelNotification struct {
	Channel string `json:"channel"` // e.g. "build.complete"
	Data    any    `json:"data"`    // arbitrary payload for the named channel
}

// SendNotificationToAllClients broadcasts a log-level notification to every
// connected MCP session (stdio, HTTP, TCP, and Unix).
// Errors on individual sessions are logged but do not stop the broadcast.
//
//	s.SendNotificationToAllClients(ctx, "info", "monitor", map[string]any{"event": "build complete"})
func (s *Service) SendNotificationToAllClients(ctx context.Context, level mcp.LoggingLevel, logger string, data any) {
	if s == nil || s.server == nil {
		return
	}
	ctx = normalizeNotificationContext(ctx)
	s.broadcastToSessions(func(session *mcp.ServerSession) {
		s.sendLoggingNotificationToSession(ctx, session, level, logger, data)
	})
}

// SendNotificationToSession sends a log-level notification to one connected
// MCP session.
//
//	s.SendNotificationToSession(ctx, session, "info", "monitor", data)
func (s *Service) SendNotificationToSession(ctx context.Context, session *mcp.ServerSession, level mcp.LoggingLevel, logger string, data any) {
	if s == nil || s.server == nil {
		return
	}
	ctx = normalizeNotificationContext(ctx)
	s.sendLoggingNotificationToSession(ctx, session, level, logger, data)
}

// SendNotificationToClient sends a log-level notification to one connected
// MCP client.
//
//	s.SendNotificationToClient(ctx, client, "info", "monitor", data)
func (s *Service) SendNotificationToClient(ctx context.Context, client *mcp.ServerSession, level mcp.LoggingLevel, logger string, data any) {
	s.SendNotificationToSession(ctx, client, level, logger, data)
}

func (s *Service) sendLoggingNotificationToSession(ctx context.Context, session *mcp.ServerSession, level mcp.LoggingLevel, logger string, data any) {
	if s == nil || s.server == nil || session == nil {
		return
	}
	ctx = normalizeNotificationContext(ctx)

	if err := sendSessionNotification(ctx, session, LoggingNotificationMethod, &mcp.LoggingMessageParams{
		Level:  level,
		Logger: logger,
		Data:   data,
	}); err != nil {
		s.debugNotify("notify: failed to send to session", "session", session.ID(), "error", err)
	}
}

// ChannelSend pushes a channel event to all connected clients via
// the notifications/claude/channel JSON-RPC method.
//
//	s.ChannelSend(ctx, "agent.complete", map[string]any{"repo": "go-io", "workspace": "go-io-123"})
//	s.ChannelSend(ctx, "build.failed", map[string]any{"repo": "core", "error": "test timeout"})
func (s *Service) ChannelSend(ctx context.Context, channel string, data any) {
	if s == nil || s.server == nil {
		return
	}
	if core.Trim(channel) == "" {
		return
	}
	ctx = normalizeNotificationContext(ctx)
	payload := ChannelNotification{Channel: channel, Data: data}
	s.sendChannelNotificationToAllClients(ctx, payload)
}

// ChannelSendToSession pushes a channel event to a specific session.
//
//	s.ChannelSendToSession(ctx, session, "agent.progress", progressData)
func (s *Service) ChannelSendToSession(ctx context.Context, session *mcp.ServerSession, channel string, data any) {
	if s == nil || s.server == nil || session == nil {
		return
	}
	if core.Trim(channel) == "" {
		return
	}
	ctx = normalizeNotificationContext(ctx)
	payload := ChannelNotification{Channel: channel, Data: data}
	if err := sendSessionNotification(ctx, session, ChannelNotificationMethod, payload); err != nil {
		s.debugNotify("channel: failed to send to session", "session", session.ID(), "error", err)
	}
}

// ChannelSendToClient pushes a channel event to one connected MCP client.
//
//	s.ChannelSendToClient(ctx, client, "agent.progress", progressData)
func (s *Service) ChannelSendToClient(ctx context.Context, client *mcp.ServerSession, channel string, data any) {
	s.ChannelSendToSession(ctx, client, channel, data)
}

// Sessions returns an iterator over all connected MCP sessions.
//
//	for session := range s.Sessions() {
//	    s.ChannelSendToSession(ctx, session, "status", data)
//	}
func (s *Service) Sessions() iter.Seq[*mcp.ServerSession] {
	if s == nil || s.server == nil {
		return func(yield func(*mcp.ServerSession) bool) {}
	}
	return slices.Values(snapshotSessions(s.server))
}

func (s *Service) sendChannelNotificationToAllClients(ctx context.Context, payload ChannelNotification) {
	if s == nil || s.server == nil {
		return
	}
	ctx = normalizeNotificationContext(ctx)
	s.broadcastToSessions(func(session *mcp.ServerSession) {
		if err := sendSessionNotification(ctx, session, ChannelNotificationMethod, payload); err != nil {
			s.debugNotify("channel: failed to send to session", "session", session.ID(), "error", err)
		}
	})
}

func (s *Service) broadcastToSessions(fn func(*mcp.ServerSession)) {
	if s == nil || s.server == nil || fn == nil {
		return
	}
	for _, session := range snapshotSessions(s.server) {
		fn(session)
	}
}

func (s *Service) debugNotify(msg string, args ...any) {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.Debug(msg, args...)
}

// NotifySession sends a raw JSON-RPC notification to a specific MCP session.
//
//	coremcp.NotifySession(ctx, session, "notifications/claude/channel", map[string]any{
//	    "content": "build failed", "meta": map[string]string{"severity": "high"},
//	})
func NotifySession(ctx context.Context, session *mcp.ServerSession, method string, payload any) error {
	return sendSessionNotification(ctx, session, method, payload)
}

func sendSessionNotification(ctx context.Context, session *mcp.ServerSession, method string, payload any) error {
	if session == nil {
		return nil
	}
	ctx = normalizeNotificationContext(ctx)

	if conn, err := sessionMCPConnection(session); err == nil {
		if notifier, ok := conn.(interface {
			Notify(context.Context, string, any) error
		}); ok {
			if err := notifier.Notify(ctx, method, payload); err != nil {
				return err
			}
			return nil
		}
	}

	conn, err := sessionJSONRPCConnection(session)
	if err != nil {
		return err
	}
	notifier, ok := conn.(interface {
		Notify(context.Context, string, any) error
	})
	if !ok {
		return coreNotifyError("connection Notify method unavailable")
	}

	if err := notifier.Notify(ctx, method, payload); err != nil {
		return err
	}
	return nil
}

func sessionMCPConnection(session *mcp.ServerSession) (any, error) {
	value := reflect.ValueOf(session)
	if value.Kind() != reflect.Ptr || value.IsNil() {
		return nil, coreNotifyError("invalid session")
	}

	field := value.Elem().FieldByName("mcpConn")
	if !field.IsValid() {
		return nil, coreNotifyError("session mcp connection field unavailable")
	}

	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface(), nil
}

func sessionJSONRPCConnection(session *mcp.ServerSession) (any, error) {
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

func snapshotSessions(server *mcp.Server) []*mcp.ServerSession {
	if server == nil {
		return nil
	}

	sessions := make([]*mcp.ServerSession, 0)
	for session := range server.Sessions() {
		if session != nil {
			sessions = append(sessions, session)
		}
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ID() < sessions[j].ID()
	})

	return sessions
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
		ClaudeChannelCapabilityName: ClaudeChannelCapability().Map(),
	}
}

// ClaudeChannelCapability returns the typed experimental capability descriptor.
//
//	cap := ClaudeChannelCapability()
//	caps := cap.Map()
func ClaudeChannelCapability() ChannelCapabilitySpec {
	return ChannelCapabilitySpec{
		Version:     "1",
		Description: "Push events into client sessions via named channels",
		Channels:    channelCapabilityChannels(),
	}
}

// ChannelCapability returns the experimental capability descriptor registered
// during New(). Callers can reuse it when exposing server metadata.
//
//	caps := ChannelCapability()
func ChannelCapability() map[string]any {
	return channelCapability()
}

// channelCapabilityChannels lists the named channel events advertised by the
// experimental capability.
func channelCapabilityChannels() []string {
	return slices.Clone(channelCapabilityList)
}

// ChannelCapabilityChannels returns the named channel events advertised by the
// experimental capability.
//
//	channels := ChannelCapabilityChannels()
func ChannelCapabilityChannels() []string {
	return channelCapabilityChannels()
}
