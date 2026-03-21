// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Subsystem registers additional MCP tools at startup.
// Implementations should be safe to call concurrently.
//
//	type BrainSubsystem struct{}
//	func (b *BrainSubsystem) Name() string { return "brain" }
//	func (b *BrainSubsystem) RegisterTools(server *mcp.Server) { ... }
type Subsystem interface {
	Name() string
	RegisterTools(server *mcp.Server)
}

// SubsystemWithShutdown extends Subsystem with graceful cleanup.
type SubsystemWithShutdown interface {
	Subsystem
	Shutdown(ctx context.Context) error
}

// Notifier pushes events to connected MCP sessions.
// Implemented by *Service. Sub-packages accept this interface
// to avoid circular imports.
//
//	notifier.ChannelSend(ctx, "build.complete", data)
type Notifier interface {
	ChannelSend(ctx context.Context, channel string, data any)
}

// SubsystemWithNotifier extends Subsystem for those that emit channel events.
// SetNotifier is called after New() before any tool calls.
type SubsystemWithNotifier interface {
	Subsystem
	SetNotifier(n Notifier)
}
