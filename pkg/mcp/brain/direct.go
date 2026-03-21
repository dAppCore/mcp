// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	coreio "forge.lthn.ai/core/go-io"
	coreerr "forge.lthn.ai/core/go-log"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// channelSender is the callback for pushing channel events.
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
// Reads CORE_BRAIN_URL and CORE_BRAIN_KEY from environment, or falls back
// to ~/.claude/brain.key for the API key.
func NewDirect() *DirectSubsystem {
	apiURL := os.Getenv("CORE_BRAIN_URL")
	if apiURL == "" {
		apiURL = "https://api.lthn.sh"
	}

	apiKey := os.Getenv("CORE_BRAIN_KEY")
	if apiKey == "" {
		if data, err := coreio.Local.Read(os.ExpandEnv("$HOME/.claude/brain.key")); err == nil {
			apiKey = strings.TrimSpace(data)
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
func (s *DirectSubsystem) RegisterTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "brain_remember",
		Description: "Store a memory in OpenBrain. Types: fact, decision, observation, plan, convention, architecture, research, documentation, service, bug, pattern, context, procedure.",
	}, s.remember)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "brain_recall",
		Description: "Semantic search across OpenBrain memories. Returns memories ranked by similarity. Use agent_id 'cladius' for Cladius's memories.",
	}, s.recall)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "brain_forget",
		Description: "Remove a memory from OpenBrain by ID.",
	}, s.forget)
}

// Shutdown implements mcp.SubsystemWithShutdown.
func (s *DirectSubsystem) Shutdown(_ context.Context) error { return nil }

func (s *DirectSubsystem) apiCall(ctx context.Context, method, path string, body any) (map[string]any, error) {
	if s.apiKey == "" {
		return nil, coreerr.E("brain.apiCall", "no API key (set CORE_BRAIN_KEY or create ~/.claude/brain.key)", nil)
	}

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, coreerr.E("brain.apiCall", "marshal request", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, s.apiURL+path, reqBody)
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

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, coreerr.E("brain.apiCall", "read response", err)
	}

	if resp.StatusCode >= 400 {
		return nil, coreerr.E("brain.apiCall", "API returned "+string(respData), nil)
	}

	var result map[string]any
	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, coreerr.E("brain.apiCall", "parse response", err)
	}

	return result, nil
}

func (s *DirectSubsystem) remember(ctx context.Context, _ *mcp.CallToolRequest, input RememberInput) (*mcp.CallToolResult, RememberOutput, error) {
	result, err := s.apiCall(ctx, "POST", "/v1/brain/remember", map[string]any{
		"content":  input.Content,
		"type":     input.Type,
		"tags":     input.Tags,
		"project":  input.Project,
		"agent_id": "cladius",
	})
	if err != nil {
		return nil, RememberOutput{}, err
	}

	id, _ := result["id"].(string)
	if s.onChannel != nil {
		s.onChannel(ctx, "brain.remember.complete", map[string]any{
			"id":      id,
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

	var memories []Memory
	if mems, ok := result["memories"].([]any); ok {
		for _, m := range mems {
			if mm, ok := m.(map[string]any); ok {
				mem := Memory{
					Content:   fmt.Sprintf("%v", mm["content"]),
					Type:      fmt.Sprintf("%v", mm["type"]),
					Project:   fmt.Sprintf("%v", mm["project"]),
					AgentID:   fmt.Sprintf("%v", mm["agent_id"]),
					CreatedAt: fmt.Sprintf("%v", mm["created_at"]),
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
		}
	}

	if s.onChannel != nil {
		s.onChannel(ctx, "brain.recall.complete", map[string]any{
			"query": input.Query,
			"count": len(memories),
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

	return nil, ForgetOutput{
		Success:   true,
		Forgotten: input.ID,
		Timestamp: time.Now(),
	}, nil
}
