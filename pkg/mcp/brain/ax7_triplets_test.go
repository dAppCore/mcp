// SPDX-License-Identifier: EUPL-1.2

package brain

import (
	"context"
	"net/http"
	"net/http/httptest"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	brainclient "dappco.re/go/mcp/pkg/mcp/brain/client"
	"dappco.re/go/mcp/pkg/mcp/ide"
	"dappco.re/go/ws"
	"github.com/gin-gonic/gin"
)

type T = core.T

var (
	AssertContains = core.AssertContains
	AssertEqual    = core.AssertEqual
	AssertLen      = core.AssertLen
	AssertNil      = core.AssertNil
	AssertNoError  = core.AssertNoError
	AssertNotEmpty = core.AssertNotEmpty
	AssertNotEqual = core.AssertNotEqual
	AssertNotNil   = core.AssertNotNil
	AssertPanics   = core.AssertPanics
	AssertTrue     = core.AssertTrue
	RequireNoError = core.RequireNoError
)

func ax7BrainService(t *T) *coremcp.Service {
	t.Helper()
	svc, err := coremcp.New(coremcp.Options{WorkspaceRoot: t.TempDir()})
	RequireNoError(t, err)
	return svc
}

func TestAX7_New_Good(t *T) {
	bridge := ide.NewBridge(ws.NewHub(), ide.DefaultConfig())
	sub := New(bridge)
	AssertEqual(t, bridge, sub.bridge)
	AssertEqual(t, "brain", sub.Name())
}

func TestAX7_New_Bad(t *T) {
	sub := New(nil)
	AssertNil(t, sub.bridge)
	AssertEqual(t, "brain", sub.Name())
}

func TestAX7_New_Ugly(t *T) {
	sub := New(ide.NewBridge(nil, ide.DefaultConfig()))
	AssertNotNil(t, sub.bridge)
	AssertNoError(t, sub.Shutdown(context.Background()))
}

func TestAX7_Subsystem_Name_Good(t *T) {
	sub := New(nil)
	AssertEqual(t, "brain", sub.Name())
	AssertNil(t, sub.bridge)
}

func TestAX7_Subsystem_Name_Bad(t *T) {
	var sub *Subsystem
	AssertEqual(t, "brain", sub.Name())
	AssertNil(t, sub)
}

func TestAX7_Subsystem_Name_Ugly(t *T) {
	sub := &Subsystem{}
	AssertEqual(t, "brain", sub.Name())
	AssertNil(t, sub.notifier)
}

func TestAX7_Subsystem_RegisterTools_Good(t *T) {
	svc := ax7BrainService(t)
	New(nil).RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}

func TestAX7_Subsystem_RegisterTools_Bad(t *T) {
	sub := New(nil)
	AssertPanics(t, func() { sub.RegisterTools(nil) })
	AssertEqual(t, "brain", sub.Name())
}

func TestAX7_Subsystem_RegisterTools_Ugly(t *T) {
	svc := ax7BrainService(t)
	(&Subsystem{}).RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}

func TestAX7_Subsystem_SetNotifier_Good(t *T) {
	sub := New(nil)
	notifier := &recordingNotifier{}
	sub.SetNotifier(notifier)
	AssertEqual(t, notifier, sub.notifier)
}

func TestAX7_Subsystem_SetNotifier_Bad(t *T) {
	sub := New(nil)
	sub.SetNotifier(nil)
	AssertNil(t, sub.notifier)
}

func TestAX7_Subsystem_SetNotifier_Ugly(t *T) {
	sub := &Subsystem{}
	sub.SetNotifier(&recordingNotifier{})
	AssertNotNil(t, sub.notifier)
}

func TestAX7_Subsystem_Shutdown_Good(t *T) {
	sub := New(nil)
	err := sub.Shutdown(context.Background())
	AssertNoError(t, err)
}

func TestAX7_Subsystem_Shutdown_Bad(t *T) {
	sub := New(nil)
	err := sub.Shutdown(nil)
	AssertNoError(t, err)
}

func TestAX7_Subsystem_Shutdown_Ugly(t *T) {
	var sub *Subsystem
	err := sub.Shutdown(context.Background())
	AssertNoError(t, err)
}

func TestAX7_NewDirect_Good(t *T) {
	t.Setenv("HOME", t.TempDir())
	sub := NewDirect()
	AssertNotNil(t, sub)
	AssertEqual(t, "brain", sub.Name())
}

func TestAX7_NewDirect_Bad(t *T) {
	t.Setenv("CORE_BRAIN_URL", "://bad")
	sub := NewDirect()
	AssertNotNil(t, sub.apiClient)
	AssertEqual(t, "brain", sub.Name())
}

func TestAX7_NewDirect_Ugly(t *T) {
	t.Setenv("HOME", "")
	sub := NewDirect()
	AssertNotNil(t, sub.client())
	AssertNil(t, sub.onChannel)
}

func TestAX7_NewDirectWithClient_Good(t *T) {
	client := brainclient.New(brainclient.Options{URL: brainclient.DefaultURL, Key: "test"})
	sub := NewDirectWithClient(client)
	AssertEqual(t, client, sub.apiClient)
	AssertEqual(t, "brain", sub.Name())
}

func TestAX7_NewDirectWithClient_Bad(t *T) {
	sub := NewDirectWithClient(nil)
	AssertNotNil(t, sub.apiClient)
	AssertEqual(t, "brain", sub.Name())
}

func TestAX7_NewDirectWithClient_Ugly(t *T) {
	client := brainclient.New(brainclient.Options{})
	sub := NewDirectWithClient(client)
	AssertEqual(t, client, sub.client())
	AssertNil(t, sub.onChannel)
}

func TestAX7_DirectSubsystem_OnChannel_Good(t *T) {
	sub := NewDirectWithClient(brainclient.New(brainclient.Options{}))
	called := false
	sub.OnChannel(func(_ context.Context, channel string, data any) {
		called = channel == coremcp.ChannelBrainRememberDone && data != nil
	})
	sub.onChannel(context.Background(), coremcp.ChannelBrainRememberDone, map[string]any{"id": "m1"})
	AssertTrue(t, called)
}

func TestAX7_DirectSubsystem_OnChannel_Bad(t *T) {
	sub := NewDirectWithClient(brainclient.New(brainclient.Options{}))
	sub.OnChannel(nil)
	AssertNil(t, sub.onChannel)
}

func TestAX7_DirectSubsystem_OnChannel_Ugly(t *T) {
	sub := NewDirectWithClient(brainclient.New(brainclient.Options{}))
	sub.OnChannel(func(context.Context, string, any) {})
	sub.OnChannel(func(context.Context, string, any) {})
	AssertNotNil(t, sub.onChannel)
}

func TestAX7_DirectSubsystem_Name_Good(t *T) {
	sub := NewDirectWithClient(brainclient.New(brainclient.Options{}))
	AssertEqual(t, "brain", sub.Name())
	AssertNotNil(t, sub.apiClient)
}

func TestAX7_DirectSubsystem_Name_Bad(t *T) {
	var sub *DirectSubsystem
	AssertEqual(t, "brain", sub.Name())
	AssertNil(t, sub)
}

func TestAX7_DirectSubsystem_Name_Ugly(t *T) {
	sub := &DirectSubsystem{}
	AssertEqual(t, "brain", sub.Name())
	AssertNotNil(t, sub.client())
}

func TestAX7_DirectSubsystem_RegisterTools_Good(t *T) {
	svc := ax7BrainService(t)
	NewDirectWithClient(brainclient.New(brainclient.Options{})).RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}

func TestAX7_DirectSubsystem_RegisterTools_Bad(t *T) {
	sub := NewDirectWithClient(brainclient.New(brainclient.Options{}))
	AssertPanics(t, func() { sub.RegisterTools(nil) })
	AssertEqual(t, "brain", sub.Name())
}

func TestAX7_DirectSubsystem_RegisterTools_Ugly(t *T) {
	svc := ax7BrainService(t)
	(&DirectSubsystem{}).RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}

func TestAX7_DirectSubsystem_Shutdown_Good(t *T) {
	sub := NewDirect()
	err := sub.Shutdown(context.Background())
	AssertNoError(t, err)
}

func TestAX7_DirectSubsystem_Shutdown_Bad(t *T) {
	sub := NewDirect()
	err := sub.Shutdown(nil)
	AssertNoError(t, err)
}

func TestAX7_DirectSubsystem_Shutdown_Ugly(t *T) {
	var sub *DirectSubsystem
	err := sub.Shutdown(context.Background())
	AssertNoError(t, err)
}

func TestAX7_NewProvider_Good(t *T) {
	hub := ws.NewHub()
	provider := NewProvider(nil, hub)
	AssertEqual(t, hub, provider.hub)
	AssertEqual(t, "brain", provider.Name())
}

func TestAX7_NewProvider_Bad(t *T) {
	provider := NewProvider(nil, nil)
	AssertNil(t, provider.bridge)
	AssertNil(t, provider.hub)
}

func TestAX7_NewProvider_Ugly(t *T) {
	bridge := ide.NewBridge(nil, ide.DefaultConfig())
	provider := NewProvider(bridge, nil)
	AssertEqual(t, bridge, provider.bridge)
	AssertEqual(t, "/api/brain", provider.BasePath())
}

func TestAX7_BrainProvider_Name_Good(t *T) {
	provider := NewProvider(nil, nil)
	AssertEqual(t, "brain", provider.Name())
	AssertEqual(t, "/api/brain", provider.BasePath())
}

func TestAX7_BrainProvider_Name_Bad(t *T) {
	var provider *BrainProvider
	AssertEqual(t, "brain", provider.Name())
	AssertNil(t, provider)
}

func TestAX7_BrainProvider_Name_Ugly(t *T) {
	provider := &BrainProvider{}
	AssertEqual(t, "brain", provider.Name())
	AssertNil(t, provider.hub)
}

func TestAX7_BrainProvider_BasePath_Good(t *T) {
	provider := NewProvider(nil, nil)
	AssertEqual(t, "/api/brain", provider.BasePath())
	AssertEqual(t, "brain", provider.Name())
}

func TestAX7_BrainProvider_BasePath_Bad(t *T) {
	var provider *BrainProvider
	AssertEqual(t, "/api/brain", provider.BasePath())
	AssertNil(t, provider)
}

func TestAX7_BrainProvider_BasePath_Ugly(t *T) {
	provider := &BrainProvider{}
	AssertEqual(t, "/api/brain", provider.BasePath())
	AssertNil(t, provider.bridge)
}

func TestAX7_BrainProvider_Channels_Good(t *T) {
	channels := NewProvider(nil, nil).Channels()
	AssertContains(t, channels, coremcp.ChannelBrainRecallDone)
	AssertLen(t, channels, 4)
}

func TestAX7_BrainProvider_Channels_Bad(t *T) {
	var provider *BrainProvider
	channels := provider.Channels()
	AssertLen(t, channels, 4)
	AssertNil(t, provider)
}

func TestAX7_BrainProvider_Channels_Ugly(t *T) {
	channels := (&BrainProvider{}).Channels()
	channels[0] = "mutated"
	AssertNotEqual(t, "mutated", (&BrainProvider{}).Channels()[0])
}

func TestAX7_BrainProvider_Element_Good(t *T) {
	element := NewProvider(nil, nil).Element()
	AssertEqual(t, "core-brain-panel", element.Tag)
	AssertEqual(t, "/assets/brain-panel.js", element.Source)
}

func TestAX7_BrainProvider_Element_Bad(t *T) {
	var provider *BrainProvider
	AssertEqual(t, "core-brain-panel", provider.Element().Tag)
	AssertNil(t, provider)
}

func TestAX7_BrainProvider_Element_Ugly(t *T) {
	element := (&BrainProvider{}).Element()
	AssertNotEmpty(t, element.Tag)
	AssertNotEmpty(t, element.Source)
}

func TestAX7_BrainProvider_RegisterRoutes_Good(t *T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewProvider(nil, nil).RegisterRoutes(router.Group("/api/brain"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/brain/status", nil))
	AssertEqual(t, http.StatusOK, rr.Code)
}

func TestAX7_BrainProvider_RegisterRoutes_Bad(t *T) {
	provider := NewProvider(nil, nil)
	AssertPanics(t, func() { provider.RegisterRoutes(nil) })
	AssertEqual(t, "brain", provider.Name())
}

func TestAX7_BrainProvider_RegisterRoutes_Ugly(t *T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	(&BrainProvider{}).RegisterRoutes(router.Group("/brain"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/brain/remember", nil))
	AssertEqual(t, http.StatusServiceUnavailable, rr.Code)
}

func TestAX7_BrainProvider_Describe_Good(t *T) {
	descs := NewProvider(nil, nil).Describe()
	AssertLen(t, descs, 5)
	AssertEqual(t, "/remember", descs[0].Path)
}

func TestAX7_BrainProvider_Describe_Bad(t *T) {
	var provider *BrainProvider
	descs := provider.Describe()
	AssertLen(t, descs, 5)
	AssertNil(t, provider)
}

func TestAX7_BrainProvider_Describe_Ugly(t *T) {
	descs := (&BrainProvider{}).Describe()
	AssertEqual(t, "/status", descs[len(descs)-1].Path)
	AssertContains(t, descs[len(descs)-1].Tags, "brain")
}
