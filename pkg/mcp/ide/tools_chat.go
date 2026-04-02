package ide

import (
	"context"
	"time"

	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Chat tool input/output types.

// ChatSendInput is the input for ide_chat_send.
type ChatSendInput struct {
	SessionID string `json:"sessionId"`
	Message   string `json:"message"`
}

// ChatSendOutput is the output for ide_chat_send.
type ChatSendOutput struct {
	Sent      bool      `json:"sent"`
	SessionID string    `json:"sessionId"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatHistoryInput is the input for ide_chat_history.
type ChatHistoryInput struct {
	SessionID string `json:"sessionId"`
	Limit     int    `json:"limit,omitempty"`
}

// ChatMessage represents a single message in history.
type ChatMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatHistoryOutput is the output for ide_chat_history.
type ChatHistoryOutput struct {
	SessionID string        `json:"sessionId"`
	Messages  []ChatMessage `json:"messages"`
}

// SessionListInput is the input for ide_session_list.
type SessionListInput struct{}

// Session represents an agent session.
type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// SessionListOutput is the output for ide_session_list.
type SessionListOutput struct {
	Sessions []Session `json:"sessions"`
}

// SessionCreateInput is the input for ide_session_create.
type SessionCreateInput struct {
	Name string `json:"name"`
}

// SessionCreateOutput is the output for ide_session_create.
type SessionCreateOutput struct {
	Session Session `json:"session"`
}

// PlanStatusInput is the input for ide_plan_status.
type PlanStatusInput struct {
	SessionID string `json:"sessionId"`
}

// PlanStep is a single step in an agent plan.
type PlanStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// PlanStatusOutput is the output for ide_plan_status.
type PlanStatusOutput struct {
	SessionID string     `json:"sessionId"`
	Status    string     `json:"status"`
	Steps     []PlanStep `json:"steps"`
}

func (s *Subsystem) registerChatTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ide_chat_send",
		Description: "Send a message to an agent chat session",
	}, s.chatSend)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ide_chat_history",
		Description: "Retrieve message history for a chat session",
	}, s.chatHistory)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ide_session_list",
		Description: "List active agent sessions",
	}, s.sessionList)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ide_session_create",
		Description: "Create a new agent session",
	}, s.sessionCreate)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ide_plan_status",
		Description: "Get the current plan status for a session",
	}, s.planStatus)
}

// chatSend forwards a chat message to the Laravel backend via bridge.
// The subsystem also stores the message locally so history lookups can
// return something useful before the backend answers.
func (s *Subsystem) chatSend(_ context.Context, _ *mcp.CallToolRequest, input ChatSendInput) (*mcp.CallToolResult, ChatSendOutput, error) {
	if s.bridge == nil {
		return nil, ChatSendOutput{}, errBridgeNotAvailable
	}
	err := s.bridge.Send(BridgeMessage{
		Type:      "chat_send",
		Channel:   "chat:" + input.SessionID,
		SessionID: input.SessionID,
		Data:      input.Message,
	})
	if err != nil {
		return nil, ChatSendOutput{}, coreerr.E("ide.chatSend", "failed to send message", err)
	}

	s.appendChatMessage(input.SessionID, "user", input.Message)
	s.recordActivity("chat_send", "forwarded chat message for session "+input.SessionID)

	return nil, ChatSendOutput{
		Sent:      true,
		SessionID: input.SessionID,
		Timestamp: time.Now(),
	}, nil
}

// chatHistory requests message history from the Laravel backend.
// The subsystem returns the local cache for the requested session while the
// backend response is still in flight.
func (s *Subsystem) chatHistory(_ context.Context, _ *mcp.CallToolRequest, input ChatHistoryInput) (*mcp.CallToolResult, ChatHistoryOutput, error) {
	if s.bridge == nil {
		return nil, ChatHistoryOutput{}, errBridgeNotAvailable
	}
	// Request history via bridge; for now return placeholder indicating the
	// request was forwarded. Real data arrives via WebSocket subscription.
	_ = s.bridge.Send(BridgeMessage{
		Type:      "chat_history",
		SessionID: input.SessionID,
		Data:      map[string]any{"limit": input.Limit},
	})
	return nil, ChatHistoryOutput{
		SessionID: input.SessionID,
		Messages:  s.chatMessages(input.SessionID),
	}, nil
}

// sessionList requests the session list from the Laravel backend.
// The local session cache is returned immediately so newly created sessions
// are visible without waiting for backend persistence.
func (s *Subsystem) sessionList(_ context.Context, _ *mcp.CallToolRequest, _ SessionListInput) (*mcp.CallToolResult, SessionListOutput, error) {
	if s.bridge == nil {
		return nil, SessionListOutput{}, errBridgeNotAvailable
	}
	_ = s.bridge.Send(BridgeMessage{Type: "session_list"})
	return nil, SessionListOutput{Sessions: s.listSessions()}, nil
}

// sessionCreate requests a new session from the Laravel backend.
// A local session record is created immediately so the caller receives a
// stable ID even before the backend finishes provisioning the session.
func (s *Subsystem) sessionCreate(_ context.Context, _ *mcp.CallToolRequest, input SessionCreateInput) (*mcp.CallToolResult, SessionCreateOutput, error) {
	if s.bridge == nil {
		return nil, SessionCreateOutput{}, errBridgeNotAvailable
	}
	_ = s.bridge.Send(BridgeMessage{
		Type: "session_create",
		Data: map[string]any{"name": input.Name},
	})
	session := Session{
		ID:        newSessionID(),
		Name:      input.Name,
		Status:    "creating",
		CreatedAt: time.Now(),
	}
	s.addSession(session)
	s.recordActivity("session_create", "created session "+session.ID)
	return nil, SessionCreateOutput{
		Session: session,
	}, nil
}

// planStatus requests plan status from the Laravel backend.
// When the backend has not populated plan state yet, the local session cache
// is used as a best-effort status source.
func (s *Subsystem) planStatus(_ context.Context, _ *mcp.CallToolRequest, input PlanStatusInput) (*mcp.CallToolResult, PlanStatusOutput, error) {
	if s.bridge == nil {
		return nil, PlanStatusOutput{}, errBridgeNotAvailable
	}
	_ = s.bridge.Send(BridgeMessage{
		Type:      "plan_status",
		SessionID: input.SessionID,
	})
	s.stateMu.Lock()
	session, ok := s.sessions[input.SessionID]
	s.stateMu.Unlock()

	status := "unknown"
	if ok && session.Status != "" {
		status = session.Status
	}
	return nil, PlanStatusOutput{
		SessionID: input.SessionID,
		Status:    status,
		Steps:     []PlanStep{},
	}, nil
}
