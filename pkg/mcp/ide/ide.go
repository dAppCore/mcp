package ide

import (
	"context"
	"sync"
	"time"

	core "dappco.re/go/core"
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

	stateMu      sync.Mutex
	sessionOrder []string
	sessions     map[string]Session
	chats        map[string][]ChatMessage
	activity     []ActivityEvent
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
	return &Subsystem{
		cfg:      cfg,
		bridge:   bridge,
		hub:      hub,
		sessions: make(map[string]Session),
		chats:    make(map[string][]ChatMessage),
	}
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

func (s *Subsystem) addSession(session Session) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if s.sessions == nil {
		s.sessions = make(map[string]Session)
	}
	if s.chats == nil {
		s.chats = make(map[string][]ChatMessage)
	}
	if _, exists := s.sessions[session.ID]; !exists {
		s.sessionOrder = append(s.sessionOrder, session.ID)
	}
	s.sessions[session.ID] = session
}

func (s *Subsystem) listSessions() []Session {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if len(s.sessionOrder) == 0 {
		return []Session{}
	}

	result := make([]Session, 0, len(s.sessionOrder))
	for _, id := range s.sessionOrder {
		if session, ok := s.sessions[id]; ok {
			result = append(result, session)
		}
	}
	return result
}

func (s *Subsystem) appendChatMessage(sessionID, role, content string) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if s.chats == nil {
		s.chats = make(map[string][]ChatMessage)
	}
	s.chats[sessionID] = append(s.chats[sessionID], ChatMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})
}

func (s *Subsystem) chatMessages(sessionID string) []ChatMessage {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	history := s.chats[sessionID]
	if len(history) == 0 {
		return []ChatMessage{}
	}
	out := make([]ChatMessage, len(history))
	copy(out, history)
	return out
}

func (s *Subsystem) recordActivity(typ, msg string) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	s.activity = append(s.activity, ActivityEvent{
		Type:      typ,
		Message:   msg,
		Timestamp: time.Now(),
	})
}

func (s *Subsystem) activityFeed(limit int) []ActivityEvent {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	if limit <= 0 || limit > len(s.activity) {
		limit = len(s.activity)
	}
	if limit == 0 {
		return []ActivityEvent{}
	}

	start := len(s.activity) - limit
	out := make([]ActivityEvent, limit)
	copy(out, s.activity[start:])
	return out
}

func newSessionID() string {
	return core.ID()
}
