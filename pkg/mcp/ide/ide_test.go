package ide

import (
	"context"

	"dappco.re/go/ws"
)

// moved AX-7 triplet TestIde_New_Good
func TestIde_New_Good(t *T) {
	hub := ws.NewHub()
	sub := New(hub, Config{WorkspaceRoot: "/tmp/work"})
	AssertEqual(t, hub, sub.hub)
	AssertNotNil(t, sub.bridge)
}

// moved AX-7 triplet TestIde_New_Bad
func TestIde_New_Bad(t *T) {
	sub := New(nil, Config{})
	AssertNil(t, sub.hub)
	AssertNil(t, sub.bridge)
}

// moved AX-7 triplet TestIde_New_Ugly
func TestIde_New_Ugly(t *T) {
	sub := New(ws.NewHub(), Config{})
	AssertNotNil(t, sub.sessions)
	AssertNotNil(t, sub.builds)
}

// moved AX-7 triplet TestIde_Subsystem_Name_Good
func TestIde_Subsystem_Name_Good(t *T) {
	sub := &Subsystem{}
	AssertEqual(t, "ide", sub.Name())
	AssertNil(t, sub.bridge)
}

// moved AX-7 triplet TestIde_Subsystem_Name_Bad
func TestIde_Subsystem_Name_Bad(t *T) {
	var sub *Subsystem
	AssertEqual(t, "ide", sub.Name())
	AssertNil(t, sub)
}

// moved AX-7 triplet TestIde_Subsystem_Name_Ugly
func TestIde_Subsystem_Name_Ugly(t *T) {
	sub := New(nil, Config{})
	AssertEqual(t, "ide", sub.Name())
	AssertNil(t, sub.bridge)
}

// moved AX-7 triplet TestIde_Subsystem_RegisterTools_Good
func TestIde_Subsystem_RegisterTools_Good(t *T) {
	svc := coremcpTestService(t)
	sub := New(nil, DefaultConfig())
	sub.RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}

// moved AX-7 triplet TestIde_Subsystem_RegisterTools_Bad
func TestIde_Subsystem_RegisterTools_Bad(t *T) {
	svc := coremcpTestService(t)
	sub := New(nil, Config{})
	sub.RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}

// moved AX-7 triplet TestIde_Subsystem_RegisterTools_Ugly
func TestIde_Subsystem_RegisterTools_Ugly(t *T) {
	svc := coremcpTestService(t)
	sub := New(ws.NewHub(), DefaultConfig())
	sub.RegisterTools(svc)
	AssertTrue(t, len(svc.Tools()) > 0)
}

// moved AX-7 triplet TestIde_Subsystem_Shutdown_Good
func TestIde_Subsystem_Shutdown_Good(t *T) {
	sub := New(ws.NewHub(), DefaultConfig())
	err := sub.Shutdown(context.Background())
	AssertNoError(t, err)
}

// moved AX-7 triplet TestIde_Subsystem_Shutdown_Bad
func TestIde_Subsystem_Shutdown_Bad(t *T) {
	var sub *Subsystem
	AssertPanics(t, func() { _ = sub.Shutdown(context.Background()) })
	AssertNil(t, sub)
}

// moved AX-7 triplet TestIde_Subsystem_Shutdown_Ugly
func TestIde_Subsystem_Shutdown_Ugly(t *T) {
	sub := New(nil, DefaultConfig())
	AssertNoError(t, sub.Shutdown(nil))
	AssertNil(t, sub.bridge)
}

// moved AX-7 triplet TestIde_Subsystem_SetNotifier_Good
func TestIde_Subsystem_SetNotifier_Good(t *T) {
	sub := New(nil, DefaultConfig())
	svc := coremcpTestService(t)
	sub.SetNotifier(svc)
	AssertEqual(t, svc, sub.notifier)
}

// moved AX-7 triplet TestIde_Subsystem_SetNotifier_Bad
func TestIde_Subsystem_SetNotifier_Bad(t *T) {
	sub := New(nil, DefaultConfig())
	sub.SetNotifier(nil)
	AssertNil(t, sub.notifier)
}

// moved AX-7 triplet TestIde_Subsystem_SetNotifier_Ugly
func TestIde_Subsystem_SetNotifier_Ugly(t *T) {
	sub := &Subsystem{}
	sub.SetNotifier(coremcpTestService(t))
	AssertNotNil(t, sub.notifier)
}

// moved AX-7 triplet TestIde_Subsystem_Bridge_Good
func TestIde_Subsystem_Bridge_Good(t *T) {
	sub := New(ws.NewHub(), DefaultConfig())
	AssertNotNil(t, sub.Bridge())
	AssertEqual(t, sub.bridge, sub.Bridge())
}

// moved AX-7 triplet TestIde_Subsystem_Bridge_Bad
func TestIde_Subsystem_Bridge_Bad(t *T) {
	sub := New(nil, DefaultConfig())
	AssertNil(t, sub.Bridge())
	AssertNil(t, sub.bridge)
}

// moved AX-7 triplet TestIde_Subsystem_Bridge_Ugly
func TestIde_Subsystem_Bridge_Ugly(t *T) {
	var sub *Subsystem
	AssertPanics(t, func() { _ = sub.Bridge() })
	AssertNil(t, sub)
}

// moved AX-7 triplet TestIde_Subsystem_StartBridge_Good
func TestIde_Subsystem_StartBridge_Good(t *T) {
	sub := New(nil, DefaultConfig())
	sub.StartBridge(context.Background())
	AssertNil(t, sub.Bridge())
}

// moved AX-7 triplet TestIde_Subsystem_StartBridge_Bad
func TestIde_Subsystem_StartBridge_Bad(t *T) {
	var sub *Subsystem
	AssertPanics(t, func() { sub.StartBridge(context.Background()) })
	AssertNil(t, sub)
}

// moved AX-7 triplet TestIde_Subsystem_StartBridge_Ugly
func TestIde_Subsystem_StartBridge_Ugly(t *T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sub := New(ws.NewHub(), DefaultConfig())
	sub.StartBridge(ctx)
	AssertNotNil(t, sub.Bridge())
}
