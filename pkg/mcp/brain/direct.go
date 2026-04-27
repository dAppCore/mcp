// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"time"

	core "dappco.re/go/core"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	brainclient "dappco.re/go/mcp/pkg/mcp/brain/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// channelSender is the callback for pushing channel events.
//
//	fn := func(ctx context.Context, channel string, data any) { ... }
type channelSender func(ctx context.Context, channel string, data any)

// DirectSubsystem implements mcp.Subsystem for OpenBrain via direct HTTP calls.
// Unlike Subsystem (which uses the IDE WebSocket bridge), this calls the
// Laravel API directly — suitable for standalone core-mcp usage.
type DirectSubsystem struct {
	apiClient *brainclient.Client
	onChannel channelSender
}

var (
	_ coremcp.Subsystem                    = (*DirectSubsystem)(nil)
	_ coremcp.SubsystemWithShutdown        = (*DirectSubsystem)(nil)
	_ coremcp.SubsystemWithChannelCallback = (*DirectSubsystem)(nil)
)

// OnChannel sets a callback for channel event broadcasting.
// Called by the MCP service after creation to wire up notifications.
//
//	brain.OnChannel(func(ctx context.Context, ch string, data any) {
//	    mcpService.ChannelSend(ctx, ch, data)
//	})
func (s *DirectSubsystem) OnChannel(fn func(ctx context.Context, channel string, data any)) {
	s.onChannel = fn
}

// NewDirect creates a brain subsystem that calls the OpenBrain API directly.
//
//	brain := NewDirect()
//
// Reads CORE_BRAIN_URL and CORE_BRAIN_KEY from environment, or falls back
// to ~/.claude/brain.key for the API key.
func NewDirect() *DirectSubsystem {
	return NewDirectWithClient(brainclient.NewFromEnvironment())
}

// NewDirectWithClient creates a direct brain subsystem using the shared client.
//
//	brain := NewDirectWithClient(client.New(client.Options{URL: "http://127.0.0.1:8080", Key: "test"}))
func NewDirectWithClient(apiClient *brainclient.Client) *DirectSubsystem {
	if apiClient == nil {
		apiClient = brainclient.NewFromEnvironment()
	}
	return &DirectSubsystem{apiClient: apiClient}
}

// Name implements mcp.Subsystem.
func (s *DirectSubsystem) Name() string { return "brain" }

// RegisterTools implements mcp.Subsystem.
func (s *DirectSubsystem) RegisterTools(svc *coremcp.Service) {
	server := svc.Server()
	coremcp.AddToolRecorded(svc, server, "brain", &mcp.Tool{
		Name:        "brain_remember",
		Description: "Store a memory in OpenBrain. Types: fact, decision, observation, plan, convention, architecture, research, documentation, service, bug, pattern, context, procedure.",
	}, s.remember)

	coremcp.AddToolRecorded(svc, server, "brain", &mcp.Tool{
		Name:        "brain_recall",
		Description: "Semantic search across OpenBrain memories. Returns memories ranked by similarity. Use agent_id 'cladius' for Cladius's memories.",
	}, s.recall)

	coremcp.AddToolRecorded(svc, server, "brain", &mcp.Tool{
		Name:        "brain_forget",
		Description: "Remove a memory from OpenBrain by ID.",
	}, s.forget)

	coremcp.AddToolRecorded(svc, server, "brain", &mcp.Tool{
		Name:        "brain_list",
		Description: "List memories in OpenBrain with optional filtering by org, project, type, and agent.",
	}, s.list)
}

// Shutdown implements mcp.SubsystemWithShutdown.
func (s *DirectSubsystem) Shutdown(_ context.Context) error { return nil }

func (s *DirectSubsystem) apiCall(ctx context.Context, method, path string, body any) (map[string]any, error) {
	return s.client().Call(ctx, method, path, body)
}

func (s *DirectSubsystem) remember(ctx context.Context, _ *mcp.CallToolRequest, input RememberInput) (*mcp.CallToolResult, RememberOutput, error) {
	result, err := s.client().Remember(ctx, brainclient.RememberInput{
		Content:    input.Content,
		Type:       input.Type,
		Tags:       input.Tags,
		Org:        input.Org,
		Project:    input.Project,
		Confidence: input.Confidence,
		Supersedes: input.Supersedes,
		ExpiresIn:  input.ExpiresIn,
	})
	if err != nil {
		return nil, RememberOutput{}, err
	}

	id, _ := result["id"].(string)
	if s.onChannel != nil {
		s.onChannel(ctx, coremcp.ChannelBrainRememberDone, map[string]any{
			"id":      id,
			"org":     input.Org,
			"type":    input.Type,
			"project": input.Project,
		})
	}
	return nil, RememberOutput{
		Success:   true,
		MemoryID:  id,
		Timestamp: time.Now(),
	}, nil
}

func (s *DirectSubsystem) recall(ctx context.Context, _ *mcp.CallToolRequest, input RecallInput) (*mcp.CallToolResult, RecallOutput, error) {
	result, err := s.client().Recall(ctx, brainclient.RecallInput{
		Query:         input.Query,
		TopK:          input.TopK,
		Org:           input.Filter.Org,
		Project:       input.Filter.Project,
		Type:          input.Filter.Type,
		AgentID:       input.Filter.AgentID,
		MinConfidence: input.Filter.MinConfidence,
	})
	if err != nil {
		return nil, RecallOutput{}, err
	}

	memories := memoriesFromResult(result)

	if s.onChannel != nil {
		s.onChannel(ctx, coremcp.ChannelBrainRecallDone, map[string]any{
			"query":   input.Query,
			"org":     input.Filter.Org,
			"project": input.Filter.Project,
			"count":   len(memories),
		})
	}
	return nil, RecallOutput{
		Success:  true,
		Count:    len(memories),
		Memories: memories,
	}, nil
}

func (s *DirectSubsystem) forget(ctx context.Context, _ *mcp.CallToolRequest, input ForgetInput) (*mcp.CallToolResult, ForgetOutput, error) {
	_, err := s.client().Forget(ctx, brainclient.ForgetInput{ID: input.ID, Reason: input.Reason})
	if err != nil {
		return nil, ForgetOutput{}, err
	}

	if s.onChannel != nil {
		s.onChannel(ctx, coremcp.ChannelBrainForgetDone, map[string]any{
			"id":     input.ID,
			"reason": input.Reason,
		})
	}

	return nil, ForgetOutput{
		Success:   true,
		Forgotten: input.ID,
		Timestamp: time.Now(),
	}, nil
}

func (s *DirectSubsystem) list(ctx context.Context, _ *mcp.CallToolRequest, input ListInput) (*mcp.CallToolResult, ListOutput, error) {
	limit := input.Limit
	if limit == 0 {
		limit = 50
	}
	result, err := s.client().List(ctx, brainclient.ListInput{
		Org:     input.Org,
		Project: input.Project,
		Type:    input.Type,
		AgentID: input.AgentID,
		Limit:   limit,
	})
	if err != nil {
		return nil, ListOutput{}, err
	}

	memories := memoriesFromResult(result)

	if s.onChannel != nil {
		s.onChannel(ctx, coremcp.ChannelBrainListDone, map[string]any{
			"org":      input.Org,
			"project":  input.Project,
			"type":     input.Type,
			"agent_id": input.AgentID,
			"limit":    limit,
		})
	}

	return nil, ListOutput{
		Success:  true,
		Count:    len(memories),
		Memories: memories,
	}, nil
}

func (s *DirectSubsystem) client() *brainclient.Client {
	if s.apiClient == nil {
		s.apiClient = brainclient.NewFromEnvironment()
	}
	return s.apiClient
}

// memoriesFromResult extracts Memory entries from an API response map.
func memoriesFromResult(result map[string]any) []Memory {
	var memories []Memory
	mems, ok := result["memories"].([]any)
	if !ok {
		return memories
	}
	for _, m := range mems {
		mm, ok := m.(map[string]any)
		if !ok {
			continue
		}
		mem := Memory{
			Content:   stringFromMap(mm, "content"),
			Type:      stringFromMap(mm, "type"),
			Org:       stringFromMap(mm, "org"),
			Project:   stringFromMap(mm, "project"),
			AgentID:   stringFromMap(mm, "agent_id"),
			CreatedAt: stringFromMap(mm, "created_at"),
		}
		if id, ok := mm["id"].(string); ok {
			mem.ID = id
		}
		if score, ok := mm["score"].(float64); ok {
			mem.Confidence = score
		}
		if source, ok := mm["source"].(string); ok {
			mem.Tags = append(mem.Tags, "source:"+source)
		}
		memories = append(memories, mem)
	}
	return memories
}

// stringFromMap extracts a string value from a map, returning "" if missing or wrong type.
func stringFromMap(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return core.Sprintf("%v", v)
	}
	return s
}
