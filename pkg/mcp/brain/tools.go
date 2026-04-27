// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"time"
	"unicode/utf8"

	coreerr "dappco.re/go/log"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"dappco.re/go/mcp/pkg/mcp/ide"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const brainOrgMaxLength = 128

// emitChannel pushes a brain event through the shared notifier.
func (s *Subsystem) emitChannel(ctx context.Context, channel string, data any) {
	if s.notifier != nil {
		s.notifier.ChannelSend(ctx, channel, data)
	}
}

// -- Input/Output types -------------------------------------------------------

// RememberInput is the input for brain_remember.
//
//	input := RememberInput{Content: "Use Qdrant for vector search", Type: "decision", Org: "core"}
type RememberInput struct {
	Content    string   `json:"content"`
	Type       string   `json:"type"`
	Tags       []string `json:"tags,omitempty"`
	Org        string   `json:"org,omitempty"`
	Project    string   `json:"project,omitempty"`
	Confidence float64  `json:"confidence,omitempty"`
	Supersedes string   `json:"supersedes,omitempty"`
	ExpiresIn  int      `json:"expires_in,omitempty"`
}

// RememberOutput is the output for brain_remember.
//
//	// out.Success == true
type RememberOutput struct {
	Success   bool      `json:"success"`
	MemoryID  string    `json:"memoryId,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// RecallInput is the input for brain_recall.
//
//	input := RecallInput{Query: "vector search", TopK: 5}
type RecallInput struct {
	Query  string       `json:"query"`
	TopK   int          `json:"top_k,omitempty"`
	Filter RecallFilter `json:"filter,omitempty"`
}

// RecallFilter holds optional filter criteria for brain_recall.
//
//	filter := RecallFilter{Org: "core", Project: "core/mcp", MinConfidence: 0.5}
type RecallFilter struct {
	Org           string  `json:"org,omitempty"`
	Project       string  `json:"project,omitempty"`
	Type          any     `json:"type,omitempty"`
	AgentID       string  `json:"agent_id,omitempty"`
	MinConfidence float64 `json:"min_confidence,omitempty"`
}

// RecallOutput is the output for brain_recall.
//
//	// out.Memories contains ranked matches
type RecallOutput struct {
	Success  bool     `json:"success"`
	Count    int      `json:"count"`
	Memories []Memory `json:"memories"`
}

// Memory is a single memory entry returned by recall or list.
//
//	mem := Memory{ID: "m1", Type: "bug", Content: "Fix timeout handling"}
type Memory struct {
	ID           string   `json:"id"`
	AgentID      string   `json:"agent_id"`
	Type         string   `json:"type"`
	Content      string   `json:"content"`
	Tags         []string `json:"tags,omitempty"`
	Org          string   `json:"org,omitempty"`
	Project      string   `json:"project,omitempty"`
	Confidence   float64  `json:"confidence"`
	SupersedesID string   `json:"supersedes_id,omitempty"`
	ExpiresAt    string   `json:"expires_at,omitempty"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

// ForgetInput is the input for brain_forget.
//
//	input := ForgetInput{ID: "m1"}
type ForgetInput struct {
	ID     string `json:"id"`
	Reason string `json:"reason,omitempty"`
}

// ForgetOutput is the output for brain_forget.
//
//	// out.Forgotten contains the deleted memory ID
type ForgetOutput struct {
	Success   bool      `json:"success"`
	Forgotten string    `json:"forgotten"`
	Timestamp time.Time `json:"timestamp"`
}

// ListInput is the input for brain_list.
//
//	input := ListInput{Org: "core", Project: "core/mcp", Limit: 50}
type ListInput struct {
	Org     string `json:"org,omitempty"`
	Project string `json:"project,omitempty"`
	Type    string `json:"type,omitempty"`
	AgentID string `json:"agent_id,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

// ListOutput is the output for brain_list.
//
//	// out.Count reports how many memories were returned
type ListOutput struct {
	Success  bool     `json:"success"`
	Count    int      `json:"count"`
	Memories []Memory `json:"memories"`
}

func validateBrainOrg(org string) error {
	if utf8.RuneCountInString(org) > brainOrgMaxLength {
		return coreerr.E("brain.validate", "org exceeds maximum length of 128 characters", nil)
	}
	return nil
}

func validateRememberInput(input RememberInput) error {
	return validateBrainOrg(input.Org)
}

func validateRecallInput(input RecallInput) error {
	return validateBrainOrg(input.Filter.Org)
}

func validateListInput(input ListInput) error {
	return validateBrainOrg(input.Org)
}

// -- Tool registration --------------------------------------------------------

func (s *Subsystem) registerBrainTools(svc *coremcp.Service) {
	server := svc.Server()
	coremcp.AddToolRecorded(svc, server, "brain", &mcp.Tool{
		Name:        "brain_remember",
		Description: "Store a memory in the shared OpenBrain knowledge store. Persists decisions, observations, conventions, research, plans, bugs, or architecture knowledge for other agents.",
	}, s.brainRemember)

	coremcp.AddToolRecorded(svc, server, "brain", &mcp.Tool{
		Name:        "brain_recall",
		Description: "Semantic search across the shared OpenBrain knowledge store. Returns memories ranked by similarity to your query, with optional filtering.",
	}, s.brainRecall)

	coremcp.AddToolRecorded(svc, server, "brain", &mcp.Tool{
		Name:        "brain_forget",
		Description: "Remove a memory from the shared OpenBrain knowledge store. Permanently deletes from both database and vector index.",
	}, s.brainForget)

	coremcp.AddToolRecorded(svc, server, "brain", &mcp.Tool{
		Name:        "brain_list",
		Description: "List memories in the shared OpenBrain knowledge store. Supports filtering by org, project, type, and agent. No vector search -- use brain_recall for semantic queries.",
	}, s.brainList)
}

// -- Tool handlers ------------------------------------------------------------

func (s *Subsystem) brainRemember(ctx context.Context, _ *mcp.CallToolRequest, input RememberInput) (*mcp.CallToolResult, RememberOutput, error) {
	if err := validateRememberInput(input); err != nil {
		return nil, RememberOutput{}, err
	}
	if s.bridge == nil {
		return nil, RememberOutput{}, errBridgeNotAvailable
	}

	err := s.bridge.Send(ide.BridgeMessage{
		Type: "brain_remember",
		Data: map[string]any{
			"content":    input.Content,
			"type":       input.Type,
			"tags":       input.Tags,
			"org":        input.Org,
			"project":    input.Project,
			"confidence": input.Confidence,
			"supersedes": input.Supersedes,
			"expires_in": input.ExpiresIn,
		},
	})
	if err != nil {
		return nil, RememberOutput{}, coreerr.E("brain.remember", "failed to send brain_remember", err)
	}

	s.emitChannel(ctx, coremcp.ChannelBrainRememberDone, map[string]any{
		"org":     input.Org,
		"type":    input.Type,
		"project": input.Project,
	})

	return nil, RememberOutput{
		Success:   true,
		Timestamp: time.Now(),
	}, nil
}

func (s *Subsystem) brainRecall(ctx context.Context, _ *mcp.CallToolRequest, input RecallInput) (*mcp.CallToolResult, RecallOutput, error) {
	if err := validateRecallInput(input); err != nil {
		return nil, RecallOutput{}, err
	}
	if s.bridge == nil {
		return nil, RecallOutput{}, errBridgeNotAvailable
	}

	err := s.bridge.Send(ide.BridgeMessage{
		Type: "brain_recall",
		Data: map[string]any{
			"query":  input.Query,
			"top_k":  input.TopK,
			"filter": input.Filter,
		},
	})
	if err != nil {
		return nil, RecallOutput{}, coreerr.E("brain.recall", "failed to send brain_recall", err)
	}

	return nil, RecallOutput{
		Success:  true,
		Memories: []Memory{},
	}, nil
}

func (s *Subsystem) brainForget(ctx context.Context, _ *mcp.CallToolRequest, input ForgetInput) (*mcp.CallToolResult, ForgetOutput, error) {
	if s.bridge == nil {
		return nil, ForgetOutput{}, errBridgeNotAvailable
	}

	err := s.bridge.Send(ide.BridgeMessage{
		Type: "brain_forget",
		Data: map[string]any{
			"id":     input.ID,
			"reason": input.Reason,
		},
	})
	if err != nil {
		return nil, ForgetOutput{}, coreerr.E("brain.forget", "failed to send brain_forget", err)
	}

	s.emitChannel(ctx, coremcp.ChannelBrainForgetDone, map[string]any{
		"id": input.ID,
	})

	return nil, ForgetOutput{
		Success:   true,
		Forgotten: input.ID,
		Timestamp: time.Now(),
	}, nil
}

func (s *Subsystem) brainList(ctx context.Context, _ *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, ListOutput, error) {
	if err := validateListInput(input); err != nil {
		return nil, ListOutput{}, err
	}
	if s.bridge == nil {
		return nil, ListOutput{}, errBridgeNotAvailable
	}

	limit := input.Limit
	if limit == 0 {
		limit = 50 // sensible default — backend clamps 0 to 1
	}
	err := s.bridge.Send(ide.BridgeMessage{
		Type: "brain_list",
		Data: map[string]any{
			"org":      input.Org,
			"project":  input.Project,
			"type":     input.Type,
			"agent_id": input.AgentID,
			"limit":    limit,
		},
	})
	if err != nil {
		return nil, ListOutput{}, coreerr.E("brain.list", "failed to send brain_list", err)
	}

	s.emitChannel(ctx, coremcp.ChannelBrainListDone, map[string]any{
		"org":      input.Org,
		"project":  input.Project,
		"type":     input.Type,
		"agent_id": input.AgentID,
		"limit":    limit,
	})

	return nil, ListOutput{
		Success:  true,
		Memories: []Memory{},
	}, nil
}
