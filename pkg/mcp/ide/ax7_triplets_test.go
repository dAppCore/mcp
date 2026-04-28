// SPDX-License-Identifier: EUPL-1.2

package ide

import (
	"context"
	"time"

	core "dappco.re/go"
	coremcp "dappco.re/go/mcp/pkg/mcp"
	"dappco.re/go/ws"
)

type T = core.T

var (
	AssertEqual     = core.AssertEqual
	AssertError     = core.AssertError
	AssertFalse     = core.AssertFalse
	AssertLen       = core.AssertLen
	AssertNil       = core.AssertNil
	AssertNoError   = core.AssertNoError
	AssertNotEmpty  = core.AssertNotEmpty
	AssertNotNil    = core.AssertNotNil
	AssertNotPanics = core.AssertNotPanics
	AssertPanics    = core.AssertPanics
	AssertTrue      = core.AssertTrue
	RequireNoError  = core.RequireNoError
)

func coremcpTestService(t *T) *coremcp.Service {
	t.Helper()
	svc, err := coremcp.New(coremcp.Options{WorkspaceRoot: t.TempDir()})
	RequireNoError(t, err)
	return svc
}

func TestAX7_DefaultConfig_Good(t *T) {
	cfg := DefaultConfig()
	AssertEqual(t, "ws://localhost:9876/ws", cfg.LaravelWSURL)
	AssertEqual(t, ".", cfg.WorkspaceRoot)
}
func TestAX7_DefaultConfig_Bad(t *T) {
	cfg := DefaultConfig()
	AssertNotEmpty(t, cfg.LaravelWSURL)
	AssertEqual(t, 2*time.Second, cfg.ReconnectInterval)
}
func TestAX7_DefaultConfig_Ugly(t *T) {
	cfg := DefaultConfig()
	AssertEqual(t, 30*time.Second, cfg.MaxReconnectInterval)
	AssertEqual(t, "", cfg.Token)
}
func TestAX7_Config_WithDefaults_Good(t *T) {
	cfg := (Config{ReconnectInterval: time.Millisecond}).WithDefaults()
	AssertEqual(t, time.Millisecond, cfg.ReconnectInterval)
	AssertNotEmpty(t, cfg.LaravelWSURL)
}
func TestAX7_Config_WithDefaults_Bad(t *T) {
	cfg := (Config{}).WithDefaults()
	AssertEqual(t, "ws://localhost:9876/ws", cfg.LaravelWSURL)
	AssertEqual(t, ".", cfg.WorkspaceRoot)
}
func TestAX7_Config_WithDefaults_Ugly(t *T) {
	cfg := (Config{LaravelWSURL: "ws://custom", WorkspaceRoot: "/tmp/work"}).WithDefaults()
	AssertEqual(t, "ws://custom", cfg.LaravelWSURL)
	AssertEqual(t, "/tmp/work", cfg.WorkspaceRoot)
}
func TestAX7_New_Good(t *T) {
	hub := ws.NewHub()
	sub := New(hub, Config{WorkspaceRoot: "/tmp/work"})
	AssertEqual(t, hub, sub.hub)
	AssertNotNil(t, sub.bridge)
}
func TestAX7_New_Bad(t *T) {
	sub := New(nil, Config{})
	AssertNil(t, sub.hub)
	AssertNil(t, sub.bridge)
}
func TestAX7_New_Ugly(t *T) {
	sub := New(ws.NewHub(), Config{})
	AssertNotNil(t, sub.sessions)
	AssertNotNil(t, sub.builds)
}
func TestAX7_NewBridge_Good(t *T) {
	bridge := NewBridge(ws.NewHub(), DefaultConfig())
	AssertNotNil(t, bridge)
	AssertFalse(t, bridge.Connected())
}
func TestAX7_NewBridge_Bad(t *T) {
	bridge := NewBridge(nil, Config{})
	AssertNil(t, bridge.hub)
	AssertEqual(t, "", bridge.cfg.LaravelWSURL)
}
func TestAX7_NewBridge_Ugly(t *T) {
	cfg := Config{LaravelWSURL: "ws://custom"}
	bridge := NewBridge(ws.NewHub(), cfg)
	AssertEqual(t, "ws://custom", bridge.cfg.LaravelWSURL)
	AssertNil(t, bridge.conn)
}
func TestAX7_Bridge_SetObserver_Good(t *T) {
	bridge := NewBridge(ws.NewHub(), DefaultConfig())
	called := false
	bridge.SetObserver(func(BridgeMessage) { called = true })
	for _, observer := range bridge.snapshotObservers() {
		observer(BridgeMessage{Type: "x"})
	}
	AssertTrue(t, called)
}
func TestAX7_Bridge_SetObserver_Bad(t *T) {
	bridge := NewBridge(ws.NewHub(), DefaultConfig())
	bridge.SetObserver(nil)
	AssertLen(t, bridge.snapshotObservers(), 0)
}
func TestAX7_Bridge_SetObserver_Ugly(t *T) {
	bridge := NewBridge(ws.NewHub(), DefaultConfig())
	bridge.SetObserver(func(BridgeMessage) {})
	bridge.SetObserver(func(BridgeMessage) {})
	AssertLen(t, bridge.snapshotObservers(), 1)
}
func TestAX7_Bridge_AddObserver_Good(t *T) {
	bridge := NewBridge(ws.NewHub(), DefaultConfig())
	count := 0
	bridge.AddObserver(func(BridgeMessage) { count++ })
	bridge.AddObserver(func(BridgeMessage) { count++ })
	for _, observer := range bridge.snapshotObservers() {
		observer(BridgeMessage{Type: "x"})
	}
	AssertEqual(t, 2, count)
}
func TestAX7_Bridge_AddObserver_Bad(t *T) {
	bridge := NewBridge(ws.NewHub(), DefaultConfig())
	bridge.AddObserver(nil)
	AssertLen(t, bridge.snapshotObservers(), 0)
}
func TestAX7_Bridge_AddObserver_Ugly(t *T) {
	bridge := NewBridge(ws.NewHub(), DefaultConfig())
	bridge.AddObserver(func(BridgeMessage) {})
	AssertLen(t, bridge.snapshotObservers(), 1)
}
func TestAX7_Bridge_Start_Good(t *T) {
	bridge := NewBridge(nil, Config{LaravelWSURL: "ws://127.0.0.1:1", ReconnectInterval: time.Millisecond, MaxReconnectInterval: time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	bridge.Start(ctx)
	cancel()
	bridge.Shutdown()
	AssertFalse(t, bridge.Connected())
}
func TestAX7_Bridge_Start_Bad(t *T) {
	bridge := NewBridge(nil, DefaultConfig())
	AssertPanics(t, func() { bridge.Start(nil) })
	AssertFalse(t, bridge.Connected())
}
func TestAX7_Bridge_Start_Ugly(t *T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	bridge := NewBridge(nil, DefaultConfig())
	bridge.Start(ctx)
	bridge.Shutdown()
	AssertFalse(t, bridge.Connected())
}
func TestAX7_Bridge_Shutdown_Good(t *T) {
	bridge := NewBridge(nil, DefaultConfig())
	bridge.connected = true
	bridge.Shutdown()
	AssertFalse(t, bridge.Connected())
}
func TestAX7_Bridge_Shutdown_Bad(t *T) {
	var bridge *Bridge
	AssertPanics(t, func() { bridge.Shutdown() })
	AssertNil(t, bridge)
}
func TestAX7_Bridge_Shutdown_Ugly(t *T) {
	bridge := NewBridge(nil, DefaultConfig())
	bridge.Shutdown()
	bridge.Shutdown()
	AssertFalse(t, bridge.Connected())
}
func TestAX7_Bridge_Connected_Good(t *T) {
	bridge := NewBridge(nil, DefaultConfig())
	bridge.connected = true
	AssertTrue(t, bridge.Connected())
}
func TestAX7_Bridge_Connected_Bad(t *T) {
	bridge := NewBridge(nil, DefaultConfig())
	AssertFalse(t, bridge.Connected())
	AssertNil(t, bridge.conn)
}
func TestAX7_Bridge_Connected_Ugly(t *T) {
	bridge := NewBridge(nil, DefaultConfig())
	bridge.connected = false
	AssertFalse(t, bridge.Connected())
}
func TestAX7_Bridge_Send_Good(t *T) {
	ts := echoServer(t)
	defer ts.Close()
	bridge := NewBridge(ws.NewHub(), Config{LaravelWSURL: wsURL(ts), ReconnectInterval: time.Millisecond, MaxReconnectInterval: time.Millisecond})
	bridge.Start(context.Background())
	waitConnected(t, bridge, 2*time.Second)
	err := bridge.Send(BridgeMessage{Type: "test", Data: "hello"})
	AssertNoError(t, err)
	bridge.Shutdown()
}
func TestAX7_Bridge_Send_Bad(t *T) {
	bridge := NewBridge(ws.NewHub(), DefaultConfig())
	err := bridge.Send(BridgeMessage{Type: "test"})
	AssertError(t, err)
	AssertFalse(t, bridge.Connected())
}
func TestAX7_Bridge_Send_Ugly(t *T) {
	ts := echoServer(t)
	defer ts.Close()
	bridge := NewBridge(ws.NewHub(), Config{LaravelWSURL: wsURL(ts), ReconnectInterval: time.Millisecond, MaxReconnectInterval: time.Millisecond})
	bridge.Start(context.Background())
	waitConnected(t, bridge, 2*time.Second)
	AssertNoError(t, bridge.Send(BridgeMessage{}))
	bridge.Shutdown()
}
func TestAX7_Subsystem_Name_Good(t *T) {
	sub := &Subsystem{}
	AssertEqual(t, "ide", sub.Name())
	AssertNil(t, sub.bridge)
}
func TestAX7_Subsystem_Name_Bad(t *T) {
	var sub *Subsystem
	AssertEqual(t, "ide", sub.Name())
	AssertNil(t, sub)
}
func TestAX7_Subsystem_Name_Ugly(t *T) {
	sub := New(nil, Config{})
	AssertEqual(t, "ide", sub.Name())
	AssertNil(t, sub.bridge)
}
func TestAX7_Subsystem_RegisterTools_Good(t *T) {
	svc := coremcpTestService(t)
	sub := New(nil, DefaultConfig())
	sub.RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}
func TestAX7_Subsystem_RegisterTools_Bad(t *T) {
	svc := coremcpTestService(t)
	sub := New(nil, Config{})
	sub.RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}
func TestAX7_Subsystem_RegisterTools_Ugly(t *T) {
	svc := coremcpTestService(t)
	sub := New(ws.NewHub(), DefaultConfig())
	sub.RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}
func TestAX7_Subsystem_Shutdown_Good(t *T) {
	sub := New(ws.NewHub(), DefaultConfig())
	err := sub.Shutdown(context.Background())
	AssertNoError(t, err)
}
func TestAX7_Subsystem_Shutdown_Bad(t *T) {
	var sub *Subsystem
	AssertPanics(t, func() { _ = sub.Shutdown(context.Background()) })
	AssertNil(t, sub)
}
func TestAX7_Subsystem_Shutdown_Ugly(t *T) {
	sub := New(nil, DefaultConfig())
	AssertNoError(t, sub.Shutdown(nil))
	AssertNil(t, sub.bridge)
}
func TestAX7_Subsystem_SetNotifier_Good(t *T) {
	sub := New(nil, DefaultConfig())
	svc := coremcpTestService(t)
	sub.SetNotifier(svc)
	AssertEqual(t, svc, sub.notifier)
}
func TestAX7_Subsystem_SetNotifier_Bad(t *T) {
	sub := New(nil, DefaultConfig())
	sub.SetNotifier(nil)
	AssertNil(t, sub.notifier)
}
func TestAX7_Subsystem_SetNotifier_Ugly(t *T) {
	sub := &Subsystem{}
	sub.SetNotifier(coremcpTestService(t))
	AssertNotNil(t, sub.notifier)
}
func TestAX7_Subsystem_Bridge_Good(t *T) {
	sub := New(ws.NewHub(), DefaultConfig())
	AssertNotNil(t, sub.Bridge())
	AssertEqual(t, sub.bridge, sub.Bridge())
}
func TestAX7_Subsystem_Bridge_Bad(t *T) {
	sub := New(nil, DefaultConfig())
	AssertNil(t, sub.Bridge())
	AssertNil(t, sub.bridge)
}
func TestAX7_Subsystem_Bridge_Ugly(t *T) {
	var sub *Subsystem
	AssertPanics(t, func() { _ = sub.Bridge() })
	AssertNil(t, sub)
}
func TestAX7_Subsystem_StartBridge_Good(t *T) {
	sub := New(nil, DefaultConfig())
	sub.StartBridge(context.Background())
	AssertNil(t, sub.Bridge())
}
func TestAX7_Subsystem_StartBridge_Bad(t *T) {
	var sub *Subsystem
	AssertPanics(t, func() { sub.StartBridge(context.Background()) })
	AssertNil(t, sub)
}
func TestAX7_Subsystem_StartBridge_Ugly(t *T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sub := New(ws.NewHub(), DefaultConfig())
	sub.StartBridge(ctx)
	AssertNotNil(t, sub.Bridge())
}
