package ide

import (
	"context"

	coreerr "forge.lthn.ai/core/go-log"
	"forge.lthn.ai/core/go-ws"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// errBridgeNotAvailable is returned when a tool requires the Laravel bridge
// but it has not been initialised (headless mode).
var errBridgeNotAvailable = coreerr.E("ide", "bridge not available", nil)

// Subsystem implements mcp.Subsystem and mcp.SubsystemWithShutdown for the IDE.
type Subsystem struct {
	cfg    Config
	bridge *Bridge
	hub    *ws.Hub
}

// New creates an IDE subsystem from a Config DTO.
//
// The ws.Hub is used for real-time forwarding; pass nil if headless
// (tools still work but real-time streaming is disabled).
func New(hub *ws.Hub, cfg Config) *Subsystem {
	cfg = cfg.WithDefaults()
	var bridge *Bridge
	if hub != nil {
		bridge = NewBridge(hub, cfg)
	}
	return &Subsystem{cfg: cfg, bridge: bridge, hub: hub}
}

// Name implements mcp.Subsystem.
func (s *Subsystem) Name() string { return "ide" }

// RegisterTools implements mcp.Subsystem.
func (s *Subsystem) RegisterTools(server *mcp.Server) {
	s.registerChatTools(server)
	s.registerBuildTools(server)
	s.registerDashboardTools(server)
}

// Shutdown implements mcp.SubsystemWithShutdown.
func (s *Subsystem) Shutdown(_ context.Context) error {
	if s.bridge != nil {
		s.bridge.Shutdown()
	}
	return nil
}

// Bridge returns the Laravel WebSocket bridge (may be nil in headless mode).
func (s *Subsystem) Bridge() *Bridge { return s.bridge }

// StartBridge begins the background connection to the Laravel backend.
func (s *Subsystem) StartBridge(ctx context.Context) {
	if s.bridge != nil {
		s.bridge.Start(ctx)
	}
}
