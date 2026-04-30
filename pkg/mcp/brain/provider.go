// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"net/http"

	coremcp "dappco.re/go/mcp/pkg/mcp"
	"dappco.re/go/mcp/pkg/mcp/ide"
	"dappco.re/go/ws"
	"github.com/gin-gonic/gin"
)

// BrainProvider wraps the brain Subsystem as a service provider with REST
// endpoints. It delegates to the same IDE bridge that the MCP tools use.
type BrainProvider struct {
	bridge *ide.Bridge
	hub    *ws.Hub
}

// elementSpec describes the browser element that can render this provider.
type elementSpec struct {
	Tag    string `json:"tag"`
	Source string `json:"source"`
}

// routeDescription describes one direct Gin route.
type routeDescription struct {
	Method      string         `json:"method"`
	Path        string         "json:\"path\""
	Summary     string         `json:"summary"`
	Description string         `json:"description"`
	Tags        []string       `json:"tags"`
	RequestBody map[string]any `json:"requestBody,omitempty"`
	Response    map[string]any `json:"response,omitempty"`
}

type providerError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type providerResponse struct {
	Success bool           `json:"success"`
	Data    any            `json:"data,omitempty"`
	Error   *providerError `json:"error,omitempty"`
}

func providerOK(data any) providerResponse {
	return providerResponse{Success: true, Data: data}
}

func providerFail(code, message string) providerResponse {
	return providerResponse{
		Success: false,
		Error:   &providerError{Code: code, Message: message},
	}
}

// NewProvider creates a brain provider that proxies to Laravel via the IDE bridge.
// The WS hub is used to emit brain events. Pass nil for hub if not needed.
func NewProvider(bridge *ide.Bridge, hub *ws.Hub) *BrainProvider {
	p := &BrainProvider{
		bridge: bridge,
		hub:    hub,
	}
	if bridge != nil {
		bridge.AddObserver(func(msg ide.BridgeMessage) {
			p.handleBridgeMessage(msg)
		})
	}
	return p
}

// Name returns the provider name.
func (p *BrainProvider) Name() string { return "brain" }

// BasePath returns the direct Gin route base path.
func (p *BrainProvider) BasePath() string { return "/api/brain" }

// Channels returns the event channels emitted by this provider.
func (p *BrainProvider) Channels() []string {
	return []string{
		coremcp.ChannelBrainRememberDone,
		coremcp.ChannelBrainRecallDone,
		coremcp.ChannelBrainForgetDone,
		coremcp.ChannelBrainListDone,
	}
}

// Element returns the browser element that can render this provider.
func (p *BrainProvider) Element() elementSpec {
	return elementSpec{
		Tag:    "core-brain-panel",
		Source: "/assets/brain-panel.js",
	}
}

// RegisterRoutes mounts direct Gin routes for the brain provider.
func (p *BrainProvider) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/remember", p.remember)
	rg.POST("/recall", p.recall)
	rg.POST("/forget", p.forget)
	rg.GET("/list", p.list)
	rg.GET("/status", p.status)
}

// Describe returns route descriptions for direct Gin consumers.
func (p *BrainProvider) Describe() []routeDescription {
	return []routeDescription{
		{
			Method:      "POST",
			Path:        "/remember",
			Summary:     "Store a memory",
			Description: "Store a memory in the shared OpenBrain knowledge store via the Laravel backend.",
			Tags:        []string{"brain"},
			RequestBody: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"content":    map[string]any{"type": "string"},
					"type":       map[string]any{"type": "string"},
					"tags":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"org":        map[string]any{"type": "string"},
					"project":    map[string]any{"type": "string"},
					"confidence": map[string]any{"type": "number"},
				},
				"required": []string{"content", "type"},
			},
			Response: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"success":   map[string]any{"type": "boolean"},
					"memoryId":  map[string]any{"type": "string"},
					"timestamp": map[string]any{"type": "string", "format": "date-time"},
				},
			},
		},
		{
			Method:      "POST",
			Path:        "/recall",
			Summary:     "Semantic search memories",
			Description: "Semantic search across the shared OpenBrain knowledge store.",
			Tags:        []string{"brain"},
			RequestBody: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
					"top_k": map[string]any{"type": "integer"},
					"filter": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"org":     map[string]any{"type": "string"},
							"project": map[string]any{"type": "string"},
							"type":    map[string]any{"type": "string"},
						},
					},
				},
				"required": []string{"query"},
			},
			Response: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"success":  map[string]any{"type": "boolean"},
					"count":    map[string]any{"type": "integer"},
					"memories": map[string]any{"type": "array"},
				},
			},
		},
		{
			Method:      "POST",
			Path:        "/forget",
			Summary:     "Remove a memory",
			Description: "Permanently delete a memory from the knowledge store.",
			Tags:        []string{"brain"},
			RequestBody: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "string"},
					"reason": map[string]any{"type": "string"},
				},
				"required": []string{"id"},
			},
			Response: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"success":   map[string]any{"type": "boolean"},
					"forgotten": map[string]any{"type": "string"},
				},
			},
		},
		{
			Method:      "GET",
			Path:        "/list",
			Summary:     "List memories",
			Description: "List memories with optional filtering by org, project, type, and agent.",
			Tags:        []string{"brain"},
			Response: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"success":  map[string]any{"type": "boolean"},
					"count":    map[string]any{"type": "integer"},
					"memories": map[string]any{"type": "array"},
				},
			},
		},
		{
			Method:      "GET",
			Path:        "/status",
			Summary:     "Brain bridge status",
			Description: "Returns whether the Laravel bridge is connected.",
			Tags:        []string{"brain"},
			Response: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"connected": map[string]any{"type": "boolean"},
				},
			},
		},
	}
}

// -- Handlers -----------------------------------------------------------------

func (p *BrainProvider) remember(c *gin.Context) {
	if p.bridge == nil {
		c.JSON(http.StatusServiceUnavailable, providerFail("bridge_unavailable", "brain bridge not available"))
		return
	}

	var input RememberInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, providerFail("invalid_input", err.Error()))
		return
	}

	err := p.bridge.Send(ide.BridgeMessage{
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
		c.JSON(http.StatusInternalServerError, providerFail("bridge_error", err.Error()))
		return
	}

	p.emitEvent(coremcp.ChannelBrainRememberDone, map[string]any{
		"org":     input.Org,
		"type":    input.Type,
		"project": input.Project,
	})

	c.JSON(http.StatusOK, providerOK(map[string]any{"success": true}))
}

func (p *BrainProvider) recall(c *gin.Context) {
	if p.bridge == nil {
		c.JSON(http.StatusServiceUnavailable, providerFail("bridge_unavailable", "brain bridge not available"))
		return
	}

	var input RecallInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, providerFail("invalid_input", err.Error()))
		return
	}

	err := p.bridge.Send(ide.BridgeMessage{
		Type: "brain_recall",
		Data: map[string]any{
			"query":  input.Query,
			"top_k":  input.TopK,
			"filter": input.Filter,
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, providerFail("bridge_error", err.Error()))
		return
	}

	c.JSON(http.StatusOK, providerOK(RecallOutput{
		Success:  true,
		Memories: []Memory{},
	}))
}

func (p *BrainProvider) forget(c *gin.Context) {
	if p.bridge == nil {
		c.JSON(http.StatusServiceUnavailable, providerFail("bridge_unavailable", "brain bridge not available"))
		return
	}

	var input ForgetInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, providerFail("invalid_input", err.Error()))
		return
	}

	err := p.bridge.Send(ide.BridgeMessage{
		Type: "brain_forget",
		Data: map[string]any{
			"id":     input.ID,
			"reason": input.Reason,
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, providerFail("bridge_error", err.Error()))
		return
	}

	p.emitEvent(coremcp.ChannelBrainForgetDone, map[string]any{
		"id": input.ID,
	})

	c.JSON(http.StatusOK, providerOK(map[string]any{
		"success":   true,
		"forgotten": input.ID,
	}))
}

func (p *BrainProvider) list(c *gin.Context) {
	if p.bridge == nil {
		c.JSON(http.StatusServiceUnavailable, providerFail("bridge_unavailable", "brain bridge not available"))
		return
	}

	project := c.Query("project")
	org := c.Query("org")
	typ := c.Query("type")
	agentID := c.Query("agent_id")
	limit := c.Query("limit")

	err := p.bridge.Send(ide.BridgeMessage{
		Type: "brain_list",
		Data: map[string]any{
			"org":      org,
			"project":  project,
			"type":     typ,
			"agent_id": agentID,
			"limit":    limit,
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, providerFail("bridge_error", err.Error()))
		return
	}

	p.emitEvent(coremcp.ChannelBrainListDone, map[string]any{
		"org":      org,
		"project":  project,
		"type":     typ,
		"agent_id": agentID,
		"limit":    limit,
	})

	c.JSON(http.StatusOK, providerOK(ListOutput{
		Success:  true,
		Memories: []Memory{},
	}))
}

func (p *BrainProvider) status(c *gin.Context) {
	connected := false
	if p.bridge != nil {
		connected = p.bridge.Connected()
	}
	c.JSON(http.StatusOK, providerOK(map[string]any{
		"connected": connected,
	}))
}

// emitEvent sends a WS event if the hub is available.
func (p *BrainProvider) emitEvent(channel string, data any) {
	if p.hub == nil {
		return
	}
	if r := p.hub.SendToChannel(channel, ws.Message{
		Type: ws.TypeEvent,
		Data: data,
	}); !r.OK {
		return
	}
}

func (p *BrainProvider) handleBridgeMessage(msg ide.BridgeMessage) {
	switch msg.Type {
	case "brain_remember":
		p.emitEvent(coremcp.ChannelBrainRememberDone, bridgePayload(msg.Data, "org", "type", "project"))
	case "brain_recall":
		payload := bridgePayload(msg.Data, "query", "org", "project", "type", "agent_id")
		payload["count"] = bridgeCount(msg.Data)
		p.emitEvent(coremcp.ChannelBrainRecallDone, payload)
	case "brain_forget":
		p.emitEvent(coremcp.ChannelBrainForgetDone, bridgePayload(msg.Data, "id", "reason"))
	case "brain_list":
		p.emitEvent(coremcp.ChannelBrainListDone, bridgePayload(msg.Data, "org", "project", "type", "agent_id", "limit"))
	}
}
