// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"net/http"
	"net/url"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	coremcp "dappco.re/go/mcp/pkg/mcp"
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
	apiURL    string
	apiKey    string
	client    *http.Client
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
	apiURL := core.Env("CORE_BRAIN_URL")
	if apiURL == "" {
		apiURL = "https://api.lthn.sh"
	}

	apiKey := core.Env("CORE_BRAIN_KEY")
	if apiKey == "" {
		home := core.Env("HOME")
		if data, err := coreio.Local.Read(core.Path(home, ".claude", "brain.key")); err == nil {
			apiKey = core.Trim(data)
		}
	}

	return &DirectSubsystem{
		apiURL: apiURL,
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
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
	if s.apiKey == "" {
		return nil, coreerr.E("brain.apiCall", "no API key (set CORE_BRAIN_KEY or create ~/.claude/brain.key)", nil)
	}

	var bodyStr string
	if body != nil {
		bodyStr = core.JSONMarshalString(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.apiURL+path, core.NewReader(bodyStr))
	if err != nil {
		return nil, coreerr.E("brain.apiCall", "create request", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, coreerr.E("brain.apiCall", "API call failed", err)
	}
	defer resp.Body.Close()

	r := core.ReadAll(resp.Body)
	if !r.OK {
		if readErr, ok := r.Value.(error); ok {
			return nil, coreerr.E("brain.apiCall", "read response", readErr)
		}
		return nil, coreerr.E("brain.apiCall", "read response failed", nil)
	}
	respData := r.Value.(string)

	if resp.StatusCode >= 400 {
		return nil, coreerr.E("brain.apiCall", "API returned "+respData, nil)
	}

	var result map[string]any
	if ur := core.JSONUnmarshal([]byte(respData), &result); !ur.OK {
		return nil, coreerr.E("brain.apiCall", "parse response", nil)
	}

	return result, nil
}

func (s *DirectSubsystem) remember(ctx context.Context, _ *mcp.CallToolRequest, input RememberInput) (*mcp.CallToolResult, RememberOutput, error) {
	result, err := s.apiCall(ctx, "POST", "/v1/brain/remember", map[string]any{
		"content":  input.Content,
		"type":     input.Type,
		"tags":     input.Tags,
		"org":      input.Org,
		"project":  input.Project,
		"agent_id": "cladius",
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
	body := map[string]any{
		"query":    input.Query,
		"top_k":    input.TopK,
		"agent_id": "cladius",
	}
	if input.Filter.Project != "" {
		body["project"] = input.Filter.Project
	}
	if input.Filter.Org != "" {
		body["org"] = input.Filter.Org
	}
	if input.Filter.Type != nil {
		body["type"] = input.Filter.Type
	}
	if input.TopK == 0 {
		body["top_k"] = 10
	}

	result, err := s.apiCall(ctx, "POST", "/v1/brain/recall", body)
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
	_, err := s.apiCall(ctx, "DELETE", "/v1/brain/forget/"+input.ID, nil)
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

	values := url.Values{}
	if input.Org != "" {
		values.Set("org", input.Org)
	}
	if input.Project != "" {
		values.Set("project", input.Project)
	}
	if input.Type != "" {
		values.Set("type", input.Type)
	}
	if input.AgentID != "" {
		values.Set("agent_id", input.AgentID)
	}
	values.Set("limit", core.Sprintf("%d", limit))

	result, err := s.apiCall(ctx, http.MethodGet, "/v1/brain/list?"+values.Encode(), nil)
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
