// Package ide provides an MCP subsystem that bridges the desktop IDE to
// a Laravel core-agentic backend over WebSocket.
package ide

import "time"

// Config holds connection and workspace settings for the IDE subsystem.
//
//	cfg := Config{
//	    LaravelWSURL: "ws://localhost:9876/ws",
//	    WorkspaceRoot: "/workspace",
//	}
type Config struct {
	// LaravelWSURL is the WebSocket endpoint for the Laravel core-agentic backend.
	LaravelWSURL string

	// WorkspaceRoot is the local path used as the default workspace context.
	WorkspaceRoot string

	// Token is the Bearer token sent in the Authorization header during
	// WebSocket upgrade. When empty, no auth header is sent.
	Token string

	// ReconnectInterval controls how long to wait between reconnect attempts.
	ReconnectInterval time.Duration

	// MaxReconnectInterval caps exponential backoff for reconnection.
	MaxReconnectInterval time.Duration
}

// DefaultConfig returns sensible defaults for local development.
//
//	cfg := DefaultConfig()
func DefaultConfig() Config {
	return Config{}.WithDefaults()
}

// WithDefaults fills unset fields with the default development values.
//
//	cfg := Config{WorkspaceRoot: "/workspace"}.WithDefaults()
func (c Config) WithDefaults() Config {
	if c.LaravelWSURL == "" {
		c.LaravelWSURL = "ws://localhost:9876/ws"
	}
	if c.WorkspaceRoot == "" {
		c.WorkspaceRoot = "."
	}
	if c.ReconnectInterval == 0 {
		c.ReconnectInterval = 2 * time.Second
	}
	if c.MaxReconnectInterval == 0 {
		c.MaxReconnectInterval = 30 * time.Second
	}
	return c
}
