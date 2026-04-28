// SPDX-License-Identifier: EUPL-1.2

package ide

import (
	"context"
	"time"

	coreerr "dappco.re/go/log"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Chat tool input/output types.

// ChatSendInput is the input for ide_chat_send.
//
//	input := ChatSendInput{SessionID: "sess-42", Message: "hello"}
type ChatSendInput struct {
	SessionID string `json:"sessionId"`
	Message   string `json:"message"`
}

// ChatSendOutput is the output for ide_chat_send.
//
//	// out.Sent == true, out.SessionID == "sess-42"
type ChatSendOutput struct {
	Sent      bool      `json:"sent"`
	SessionID string    `json:"sessionId"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatHistoryInput is the input for ide_chat_history.
//
//	input := ChatHistoryInput{SessionID: "sess-42", Limit: 50}
type ChatHistoryInput struct {
	SessionID string `json:"sessionId"`
	Limit     int    `json:"limit,omitempty"`
}

// ChatMessage represents a single message in history.
//
//	msg := ChatMessage{Role: "user", Content: "hello"}
type ChatMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ChatHistoryOutput is the output for ide_chat_history.
//
//	// out.Messages contains the stored chat transcript
type ChatHistoryOutput struct {
	SessionID string        `json:"sessionId"`
	Messages  []ChatMessage `json:"messages"`
}

// SessionListInput is the input for ide_session_list.
//
//	input := SessionListInput{}
type SessionListInput struct{}

// Session represents an agent session.
//
//	session := Session{ID: "sess-42", Name: "draft", Status: "running"}
type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// SessionListOutput is the output for ide_session_list.
//
//	// out.Sessions contains every locally tracked session
type SessionListOutput struct {
	Sessions []Session `json:"sessions"`
}

// SessionCreateInput is the input for ide_session_create.
//
//	input := SessionCreateInput{Name: "draft"}
type SessionCreateInput struct {
	Name string `json:"name"`
}

// SessionCreateOutput is the output for ide_session_create.
//
//	// out.Session.ID is assigned by the backend or local store
type SessionCreateOutput struct {
	Session Session `json:"session"`
}

// PlanStatusInput is the input for ide_plan_status.
//
//	input := PlanStatusInput{SessionID: "sess-42"}
type PlanStatusInput struct {
	SessionID string `json:"sessionId"`
}

// PlanStep is a single step in an agent plan.
//
//	step := PlanStep{Name: "prep", Status: "done"}
type PlanStep struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// PlanStatusOutput is the output for ide_plan_status.
//
//	// out.Steps contains the current plan breakdown
type PlanStatusOutput struct {
	SessionID string     `json:"sessionId"`
	Status    string     `json:"status"`
	Steps     []PlanStep `json:"steps"`
}

func (s *Subsystem) registerChatTools(svc *coremcp.Service) {
	server := svc.Server()
	coremcp.AddToolRecorded(svc, server, "ide", &mcp.Tool{
		Name:        "ide_chat_send",
		Description: "Send a message to an agent chat session",
	}, s.chatSend)

	coremcp.AddToolRecorded(svc, server, "ide", &mcp.Tool{
		Name:        "ide_chat_history",
		Description: "Retrieve message history for a chat session",
	}, s.chatHistory)

	coremcp.AddToolRecorded(svc, server, "ide", &mcp.Tool{
		Name:        "ide_session_list",
		Description: "List active agent sessions",
	}, s.sessionList)

	coremcp.AddToolRecorded(svc, server, "ide", &mcp.Tool{
		Name:        "ide_session_create",
		Description: "Create a new agent session",
	}, s.sessionCreate)

	coremcp.AddToolRecorded(svc, server, "ide", &mcp.Tool{
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

// chatHistory returns the local message history for a session and refreshes
// the Laravel backend when the bridge is available.
func (s *Subsystem) chatHistory(_ context.Context, _ *mcp.CallToolRequest, input ChatHistoryInput) (*mcp.CallToolResult, ChatHistoryOutput, error) {
	if s.bridge != nil {
		// Request history via bridge when available; the local cache still
		// provides an immediate response in headless mode.
		_ = s.bridge.Send(BridgeMessage{
			Type:      "chat_history",
			SessionID: input.SessionID,
			Data:      map[string]any{"limit": input.Limit},
		})
	}
	return nil, ChatHistoryOutput{
		SessionID: input.SessionID,
		Messages:  s.chatMessages(input.SessionID),
	}, nil
}

// sessionList returns the local session cache and refreshes the Laravel
// backend when the bridge is available.
func (s *Subsystem) sessionList(_ context.Context, _ *mcp.CallToolRequest, _ SessionListInput) (*mcp.CallToolResult, SessionListOutput, error) {
	if s.bridge != nil {
		_ = s.bridge.Send(BridgeMessage{Type: "session_list"})
	}
	return nil, SessionListOutput{Sessions: s.listSessions()}, nil
}

// sessionCreate creates a local session record immediately and forwards the
// request to the Laravel backend when the bridge is available.
func (s *Subsystem) sessionCreate(_ context.Context, _ *mcp.CallToolRequest, input SessionCreateInput) (*mcp.CallToolResult, SessionCreateOutput, error) {
	if s.bridge != nil {
		if err := s.bridge.Send(BridgeMessage{
			Type: "session_create",
			Data: map[string]any{"name": input.Name},
		}); err != nil {
			return nil, SessionCreateOutput{}, err
		}
	}
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

// planStatus returns the local best-effort session status and refreshes the
// Laravel backend when the bridge is available.
func (s *Subsystem) planStatus(_ context.Context, _ *mcp.CallToolRequest, input PlanStatusInput) (*mcp.CallToolResult, PlanStatusOutput, error) {
	if s.bridge != nil {
		_ = s.bridge.Send(BridgeMessage{
			Type:      "plan_status",
			SessionID: input.SessionID,
		})
	}
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
