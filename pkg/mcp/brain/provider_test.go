// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"net/http"
	"net/http/httptest"
	"testing"

	coremcp "dappco.re/go/mcp/pkg/mcp"
	"dappco.re/go/mcp/pkg/mcp/ide"
	"dappco.re/go/ws"
	"github.com/gin-gonic/gin"
)

func TestBrainProviderChannels_Good_IncludesListComplete(t *testing.T) {
	p := NewProvider(nil, nil)

	channels := p.Channels()
	found := false
	for _, channel := range channels {
		if channel == "brain.list.complete" {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected brain.list.complete in provider channels: %#v", channels)
	}
}

func TestBrainProviderHandleBridgeMessage_Good_SupportsBrainEvents(t *testing.T) {
	p := NewProvider(nil, nil)
	for _, msg := range []ide.BridgeMessage{
		{Type: "brain_remember", Data: map[string]any{"type": "bug", "project": "core/mcp"}},
		{Type: "brain_recall", Data: map[string]any{"query": "test", "memories": []any{map[string]any{"id": "m1"}}}},
		{Type: "brain_forget", Data: map[string]any{"id": "mem-123", "reason": "outdated"}},
		{Type: "brain_list", Data: map[string]any{"project": "core/mcp", "limit": 10}},
	} {
		p.handleBridgeMessage(msg)
	}
}

// moved AX-7 triplet TestProvider_NewProvider_Good
func TestProvider_NewProvider_Good(t *T) {
	hub := ws.NewHub()
	provider := NewProvider(nil, hub)
	AssertEqual(t, hub, provider.hub)
	AssertEqual(t, "brain", provider.Name())
}

// moved AX-7 triplet TestProvider_NewProvider_Bad
func TestProvider_NewProvider_Bad(t *T) {
	provider := NewProvider(nil, nil)
	AssertNil(t, provider.bridge)
	AssertNil(t, provider.hub)
}

// moved AX-7 triplet TestProvider_NewProvider_Ugly
func TestProvider_NewProvider_Ugly(t *T) {
	bridge := ide.NewBridge(nil, ide.DefaultConfig())
	provider := NewProvider(bridge, nil)
	AssertEqual(t, bridge, provider.bridge)
	AssertEqual(t, "/api/brain", provider.BasePath())
}

// moved AX-7 triplet TestProvider_BrainProvider_Name_Good
func TestProvider_BrainProvider_Name_Good(t *T) {
	provider := NewProvider(nil, nil)
	AssertEqual(t, "brain", provider.Name())
	AssertEqual(t, "/api/brain", provider.BasePath())
}

// moved AX-7 triplet TestProvider_BrainProvider_Name_Bad
func TestProvider_BrainProvider_Name_Bad(t *T) {
	var provider *BrainProvider
	AssertEqual(t, "brain", provider.Name())
	AssertNil(t, provider)
}

// moved AX-7 triplet TestProvider_BrainProvider_Name_Ugly
func TestProvider_BrainProvider_Name_Ugly(t *T) {
	provider := &BrainProvider{}
	AssertEqual(t, "brain", provider.Name())
	AssertNil(t, provider.hub)
}

// moved AX-7 triplet TestProvider_BrainProvider_BasePath_Good
func TestProvider_BrainProvider_BasePath_Good(t *T) {
	provider := NewProvider(nil, nil)
	AssertEqual(t, "/api/brain", provider.BasePath())
	AssertEqual(t, "brain", provider.Name())
}

// moved AX-7 triplet TestProvider_BrainProvider_BasePath_Bad
func TestProvider_BrainProvider_BasePath_Bad(t *T) {
	var provider *BrainProvider
	AssertEqual(t, "/api/brain", provider.BasePath())
	AssertNil(t, provider)
}

// moved AX-7 triplet TestProvider_BrainProvider_BasePath_Ugly
func TestProvider_BrainProvider_BasePath_Ugly(t *T) {
	provider := &BrainProvider{}
	AssertEqual(t, "/api/brain", provider.BasePath())
	AssertNil(t, provider.bridge)
}

// moved AX-7 triplet TestProvider_BrainProvider_Channels_Good
func TestProvider_BrainProvider_Channels_Good(t *T) {
	channels := NewProvider(nil, nil).Channels()
	AssertContains(t, channels, coremcp.ChannelBrainRecallDone)
	AssertLen(t, channels, 4)
}

// moved AX-7 triplet TestProvider_BrainProvider_Channels_Bad
func TestProvider_BrainProvider_Channels_Bad(t *T) {
	var provider *BrainProvider
	channels := provider.Channels()
	AssertLen(t, channels, 4)
	AssertNil(t, provider)
}

// moved AX-7 triplet TestProvider_BrainProvider_Channels_Ugly
func TestProvider_BrainProvider_Channels_Ugly(t *T) {
	channels := (&BrainProvider{}).Channels()
	channels[0] = "mutated"
	AssertNotEqual(t, "mutated", (&BrainProvider{}).Channels()[0])
}

// moved AX-7 triplet TestProvider_BrainProvider_Element_Good
func TestProvider_BrainProvider_Element_Good(t *T) {
	element := NewProvider(nil, nil).Element()
	AssertEqual(t, "core-brain-panel", element.Tag)
	AssertEqual(t, "/assets/brain-panel.js", element.Source)
}

// moved AX-7 triplet TestProvider_BrainProvider_Element_Bad
func TestProvider_BrainProvider_Element_Bad(t *T) {
	var provider *BrainProvider
	AssertEqual(t, "core-brain-panel", provider.Element().Tag)
	AssertNil(t, provider)
}

// moved AX-7 triplet TestProvider_BrainProvider_Element_Ugly
func TestProvider_BrainProvider_Element_Ugly(t *T) {
	element := (&BrainProvider{}).Element()
	AssertNotEmpty(t, element.Tag)
	AssertNotEmpty(t, element.Source)
}

// moved AX-7 triplet TestProvider_BrainProvider_RegisterRoutes_Good
func TestProvider_BrainProvider_RegisterRoutes_Good(t *T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewProvider(nil, nil).RegisterRoutes(router.Group("/api/brain"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/brain/status", nil))
	AssertEqual(t, http.StatusOK, rr.Code)
}

// moved AX-7 triplet TestProvider_BrainProvider_RegisterRoutes_Bad
func TestProvider_BrainProvider_RegisterRoutes_Bad(t *T) {
	provider := NewProvider(nil, nil)
	AssertPanics(t, func() { provider.RegisterRoutes(nil) })
	AssertEqual(t, "brain", provider.Name())
}

// moved AX-7 triplet TestProvider_BrainProvider_RegisterRoutes_Ugly
func TestProvider_BrainProvider_RegisterRoutes_Ugly(t *T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	(&BrainProvider{}).RegisterRoutes(router.Group("/brain"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/brain/remember", nil))
	AssertEqual(t, http.StatusServiceUnavailable, rr.Code)
}

// moved AX-7 triplet TestProvider_BrainProvider_Describe_Good
func TestProvider_BrainProvider_Describe_Good(t *T) {
	descs := NewProvider(nil, nil).Describe()
	AssertLen(t, descs, 5)
	AssertEqual(t, "/remember", descs[0].Path)
}

// moved AX-7 triplet TestProvider_BrainProvider_Describe_Bad
func TestProvider_BrainProvider_Describe_Bad(t *T) {
	var provider *BrainProvider
	descs := provider.Describe()
	AssertLen(t, descs, 5)
	AssertNil(t, provider)
}

// moved AX-7 triplet TestProvider_BrainProvider_Describe_Ugly
func TestProvider_BrainProvider_Describe_Ugly(t *T) {
	descs := (&BrainProvider{}).Describe()
	AssertEqual(t, "/status", descs[len(descs)-1].Path)
	AssertContains(t, descs[len(descs)-1].Tags, "brain")
}
