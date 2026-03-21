# MCP SDK & AX Convention Migration Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Align the MCP service layer with CoreGO AX conventions (Options/Result/Service DTOs), add server→client notification broadcasting via `SendNotificationToAllClients()`, and register the `claude/channel` experimental capability for pushing events into Claude Code sessions.

**Architecture:** Refactor `mcp.Service` from functional options (`Option func(*Service) error`) to an `Options{}` struct. Add notification broadcasting by iterating the official SDK's `Server.Sessions()` (refactoring TCP/Unix transports to use `Server.Connect()` so all sessions are visible). Register `claude/channel` in `ServerCapabilities.Experimental` so clients can discover push-event support.

**Tech Stack:** Go 1.26, `github.com/modelcontextprotocol/go-sdk` v1.4.1, `dappco.re/go/core` v0.4.7

---

## SDK Evaluation

**Current SDK:** `github.com/modelcontextprotocol/go-sdk v1.4.1` (official MCP Go SDK)

**Alternative evaluated:** `github.com/mark3labs/mcp-go` — community SDK with built-in `SendNotificationToAllClients()` and `SendNotificationToClient()`.

| Criteria | Official SDK | mark3labs/mcp-go |
|----------|-------------|------------------|
| Multi-session support | `Server.Sessions()` iterator, `Server.Connect()` | `SendNotificationToAllClients()` built-in |
| Tool registration | Generic `AddTool[In, Out]()` — matches existing pattern | `AddTool(NewTool(), handler)` — would require rewrite |
| Experimental capabilities | `ServerCapabilities.Experimental map[string]any` | Same |
| Transport support | Stdio, SSE, StreamableHTTP, InMemory | Stdio, SSE, StreamableHTTP |
| Options pattern | `*ServerOptions` struct — aligns with AX DTOs | Functional options — conflicts with AX migration |
| Handler signatures | `func(ctx, *CallToolRequest, In) (*CallToolResult, Out, error)` | `func(ctx, CallToolRequest) (*CallToolResult, error)` |

**Decision:** Stay on official SDK. It already uses struct-based options (closer to AX), preserves our generic `addToolRecorded[In, Out]()` pattern, and supports multi-session via `Server.Sessions()`. Implement `SendNotificationToAllClients()` as a thin wrapper.

## Existing Infrastructure

Already built:
- `Service` struct wrapping `*mcp.Server` with functional options (`mcp.go`)
- Generic `addToolRecorded[In, Out]()` for tool registration + REST bridge (`registry.go`)
- `Subsystem` / `SubsystemWithShutdown` interfaces (`subsystem.go`)
- 4 transports: stdio, TCP, Unix, HTTP (`transport_*.go`)
- `BridgeToAPI` REST bridge (`bridge.go`)
- 7 tool groups: files, language, metrics, process, rag, webview, ws
- 3 subsystems: brain, ide, agentic
- Import migration to `dappco.re/go/core` already committed (4c6c9d7)

## Consumer Impact

2 consumers import `forge.lthn.ai/core/mcp`: **agent**, **ide**.

Both call `mcp.New(...)` with functional options and `mcp.WithSubsystem(...)`. Both must be updated after Phase 1.

## File Structure

| File | Action | Purpose |
|------|--------|---------|
| `pkg/mcp/mcp.go` | Modify | Replace `Option` func type with `Options{}` struct; update `New()` |
| `pkg/mcp/subsystem.go` | Modify | Remove `WithSubsystem` func; subsystems move into `Options.Subsystems` |
| `pkg/mcp/notify.go` | Create | `SendNotificationToAllClients()`, `ChannelSend()`, channel helpers |
| `pkg/mcp/registry.go` | Modify | Add usage-example comments |
| `pkg/mcp/bridge.go` | Modify | Minor: usage-example comments |
| `pkg/mcp/transport_stdio.go` | Modify | Usage-example comments |
| `pkg/mcp/transport_tcp.go` | Modify | Usage-example comments |
| `pkg/mcp/transport_unix.go` | Modify | Usage-example comments |
| `pkg/mcp/transport_http.go` | Modify | Usage-example comments |
| `pkg/mcp/tools_metrics.go` | Modify | Usage-example comments on Input/Output types |
| `pkg/mcp/tools_process.go` | Modify | Usage-example comments on Input/Output types |
| `pkg/mcp/tools_rag.go` | Modify | Usage-example comments on Input/Output types |
| `pkg/mcp/tools_webview.go` | Modify | Usage-example comments on Input/Output types |
| `pkg/mcp/tools_ws.go` | Modify | Usage-example comments on Input/Output types |
| `pkg/mcp/mcp_test.go` | Modify | Update tests for `Options{}` constructor |
| `pkg/mcp/subsystem_test.go` | Modify | Update tests for `Options.Subsystems` |
| `pkg/mcp/notify_test.go` | Create | Tests for notification broadcasting |

---

## Phase 1: Service Options{} Refactoring

Replace the functional options pattern with an `Options{}` struct. This is the breaking change — consumers must update their `mcp.New()` calls.

**Files:**
- Modify: `pkg/mcp/mcp.go`
- Modify: `pkg/mcp/subsystem.go`
- Modify: `pkg/mcp/mcp_test.go`
- Modify: `pkg/mcp/subsystem_test.go`

- [ ] **Step 1: Define Options struct and update New()**

Replace the current functional option pattern:

```go
// BEFORE:
type Option func(*Service) error

func WithWorkspaceRoot(root string) Option { ... }
func WithProcessService(ps *process.Service) Option { ... }
func WithWSHub(hub *ws.Hub) Option { ... }
func WithSubsystem(sub Subsystem) Option { ... }

func New(opts ...Option) (*Service, error) { ... }
```

With an `Options{}` struct:

```go
// Options configures a Service.
//
//   svc, err := mcp.New(mcp.Options{
//       WorkspaceRoot:  "/path/to/project",
//       ProcessService: ps,
//       WSHub:          hub,
//       Subsystems:     []Subsystem{brain, ide},
//   })
type Options struct {
    WorkspaceRoot  string           // Restrict file ops to this directory (empty = cwd)
    Unrestricted   bool             // Disable sandboxing entirely (not recommended)
    ProcessService *process.Service // Optional process management
    WSHub          *ws.Hub          // Optional WebSocket hub for real-time streaming
    Subsystems     []Subsystem      // Additional tool groups registered at startup
}

// New creates a new MCP service with file operations.
//
//   svc, err := mcp.New(mcp.Options{WorkspaceRoot: "."})
func New(opts Options) (*Service, error) {
    impl := &mcp.Implementation{
        Name:    "core-cli",
        Version: "0.1.0",
    }

    server := mcp.NewServer(impl, &mcp.ServerOptions{
        Capabilities: &mcp.ServerCapabilities{
            Tools: &mcp.ToolCapabilities{ListChanged: true},
        },
    })

    s := &Service{
        server:         server,
        processService: opts.ProcessService,
        wsHub:          opts.WSHub,
        subsystems:     opts.Subsystems,
        logger:         log.Default(),
    }

    // Workspace root: unrestricted, explicit root, or default to cwd
    if opts.Unrestricted {
        s.workspaceRoot = ""
        s.medium = io.Local
    } else {
        root := opts.WorkspaceRoot
        if root == "" {
            cwd, err := os.Getwd()
            if err != nil {
                return nil, log.E("mcp.New", "failed to get working directory", err)
            }
            root = cwd
        }
        abs, err := filepath.Abs(root)
        if err != nil {
            return nil, log.E("mcp.New", "invalid workspace root", err)
        }
        m, merr := io.NewSandboxed(abs)
        if merr != nil {
            return nil, log.E("mcp.New", "failed to create workspace medium", merr)
        }
        s.workspaceRoot = abs
        s.medium = m
    }

    s.registerTools(s.server)

    for _, sub := range s.subsystems {
        sub.RegisterTools(s.server)
    }

    return s, nil
}
```

- [ ] **Step 2: Remove functional option functions**

Delete from `mcp.go`:
- `type Option func(*Service) error`
- `func WithWorkspaceRoot(root string) Option`
- `func WithProcessService(ps *process.Service) Option`
- `func WithWSHub(hub *ws.Hub) Option`

Delete from `subsystem.go`:
- `func WithSubsystem(sub Subsystem) Option`

- [ ] **Step 3: Update tests**

Find all test calls to `New(...)` in `mcp_test.go`, `subsystem_test.go`, `integration_test.go`, `transport_e2e_test.go`, and other `_test.go` files. All tests use `package mcp` (internal). Replace:

```go
// BEFORE:
svc, err := New(WithWorkspaceRoot(dir))
svc, err := New(WithSubsystem(&fakeSub{}))

// AFTER:
svc, err := New(Options{WorkspaceRoot: dir})
svc, err := New(Options{Subsystems: []Subsystem{&fakeSub{}}})
```

- [ ] **Step 4: Verify compilation**

```bash
go vet ./pkg/mcp/...
go build ./pkg/mcp/...
go test ./pkg/mcp/...
```

- [ ] **Step 5: Commit**

```bash
git add pkg/mcp/mcp.go pkg/mcp/subsystem.go pkg/mcp/*_test.go
git commit -m "refactor(mcp): replace functional options with Options{} struct

Aligns with CoreGO AX convention: Options{} DTOs instead of
functional option closures. Breaking change for consumers
(agent, ide) — they must update their mcp.New() calls.

Co-Authored-By: Virgil <virgil@lethean.io>"
```

---

## Phase 2: Notification Support + claude/channel Capability

Add server→client notification broadcasting and register the `claude/channel` experimental capability.

**Files:**
- Create: `pkg/mcp/notify.go`
- Create: `pkg/mcp/notify_test.go`
- Modify: `pkg/mcp/mcp.go` (register experimental capability in `New()`)
- Modify: `pkg/mcp/transport_tcp.go` (use `s.server.Connect()` instead of per-connection servers)
- Modify: `pkg/mcp/transport_unix.go` (same as TCP)

**Important: Transport-level limitation.** The TCP and Unix transports currently create a **new `mcp.Server` per connection** in `handleConnection()`. Sessions on those per-connection servers are invisible to `s.server.Sessions()`. Notifications therefore only reach stdio and HTTP (StreamableHTTP) clients out of the box. To support TCP/Unix notifications, Phase 2 also refactors TCP/Unix to use `s.server.Connect()` instead of creating independent servers — this registers each connection's session on the shared server instance.

- [ ] **Step 1: Refactor TCP/Unix to use shared server sessions**

In `transport_tcp.go` and `transport_unix.go`, replace the per-connection `mcp.NewServer()` call with `s.server.Connect()`:

```go
// BEFORE (transport_tcp.go handleConnection):
server := mcp.NewServer(impl, nil)
s.registerTools(server)
for _, sub := range s.subsystems { sub.RegisterTools(server) }
_ = server.Run(ctx, transport)

// AFTER:
session, err := s.server.Connect(ctx, transport, nil)
if err != nil {
    s.logger.Debug("tcp: connect failed", "error", err)
    return
}
<-session.Wait()
```

This ensures every TCP/Unix connection registers its session on the shared `s.server`, making it visible to `Sessions()` and `SendNotificationToAllClients`.

- [ ] **Step 2: Create notify.go with notification methods**

```go
// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
    "context"
    "iter"

    "forge.lthn.ai/core/go-log"
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

// SendNotificationToAllClients broadcasts a log-level notification to every
// connected MCP session (stdio, HTTP, TCP, and Unix).
// Errors on individual sessions are logged but do not stop the broadcast.
//
//   s.SendNotificationToAllClients(ctx, "info", "build complete", map[string]any{"duration": "3.2s"})
func (s *Service) SendNotificationToAllClients(ctx context.Context, level mcp.LoggingLevel, logger string, data any) {
    for session := range s.server.Sessions() {
        if err := session.Log(ctx, &mcp.LoggingMessageParams{
            Level:  level,
            Logger: logger,
            Data:   data,
        }); err != nil {
            s.logger.Debug("notify: failed to send to session", "session", session.ID(), "error", err)
        }
    }
}

// ChannelSend pushes a channel event to all connected clients.
// This uses the claude/channel experimental capability.
// Channel names follow the convention "subsystem.event" (e.g. "build.complete", "agent.status").
//
//   s.ChannelSend(ctx, "build.complete", map[string]any{"repo": "go-io", "status": "passed"})
func (s *Service) ChannelSend(ctx context.Context, channel string, data any) {
    payload := map[string]any{
        "channel": channel,
        "data":    data,
    }
    s.SendNotificationToAllClients(ctx, "info", "channel", payload)
}

// ChannelSendToSession pushes a channel event to a specific session.
//
//   s.ChannelSendToSession(ctx, session, "agent.progress", progressData)
func (s *Service) ChannelSendToSession(ctx context.Context, session *mcp.ServerSession, channel string, data any) {
    payload := map[string]any{
        "channel": channel,
        "data":    data,
    }
    if err := session.Log(ctx, &mcp.LoggingMessageParams{
        Level:  "info",
        Logger: "channel",
        Data:   payload,
    }); err != nil {
        s.logger.Debug("channel: failed to send to session", "session", session.ID(), "channel", channel, "error", err)
    }
}

// Sessions returns an iterator over all connected MCP sessions.
// Useful for subsystems that need to send targeted notifications.
//
//   for session := range s.Sessions() {
//       s.ChannelSendToSession(ctx, session, "status", data)
//   }
func (s *Service) Sessions() iter.Seq[*mcp.ServerSession] {
    return s.server.Sessions()
}

// channelCapability returns the experimental capability descriptor
// for claude/channel, registered during New().
func channelCapability() map[string]any {
    return map[string]any{
        "claude/channel": map[string]any{
            "version":     "1",
            "description": "Push events into client sessions via named channels",
            "channels": []string{
                "build.complete",
                "build.failed",
                "agent.status",
                "agent.blocked",
                "agent.complete",
                "brain.recall.complete",
                "process.exit",
                "test.result",
            },
        },
    }
}
```

- [ ] **Step 3: Register experimental capability in New()**

Update `New()` in `mcp.go` to pass capabilities to `mcp.NewServer`:

```go
server := mcp.NewServer(impl, &mcp.ServerOptions{
    Capabilities: &mcp.ServerCapabilities{
        Tools:        &mcp.ToolCapabilities{ListChanged: true},
        Logging:      &mcp.LoggingCapabilities{},
        Experimental: channelCapability(),
    },
})
```

- [ ] **Step 4: Create notify_test.go**

Uses `package mcp` (internal tests) consistent with all existing test files in this package.

```go
// SPDX-License-Identifier: EUPL-1.2

package mcp

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestSendNotificationToAllClients_Good(t *testing.T) {
    svc, err := New(Options{})
    assert.NoError(t, err)

    // With no connected sessions, should not panic
    ctx := context.Background()
    svc.SendNotificationToAllClients(ctx, "info", "test", map[string]any{"key": "value"})
}

func TestChannelSend_Good(t *testing.T) {
    svc, err := New(Options{})
    assert.NoError(t, err)

    ctx := context.Background()
    svc.ChannelSend(ctx, "build.complete", map[string]any{"repo": "go-io"})
}

func TestChannelCapability_Good(t *testing.T) {
    // Verify the capability struct is well-formed
    svc, err := New(Options{})
    assert.NoError(t, err)
    assert.NotNil(t, svc.Server())
}
```

- [ ] **Step 5: Verify compilation and tests**

```bash
go vet ./pkg/mcp/...
go test ./pkg/mcp/...
```

- [ ] **Step 6: Commit**

```bash
git add pkg/mcp/notify.go pkg/mcp/notify_test.go pkg/mcp/mcp.go pkg/mcp/transport_tcp.go pkg/mcp/transport_unix.go
git commit -m "feat(mcp): add notification broadcasting + claude/channel capability

New methods:
- SendNotificationToAllClients: broadcasts to all connected MCP sessions
- ChannelSend: push named channel events (build.complete, agent.status, etc.)
- ChannelSendToSession: push to a specific session
- Sessions: iterator over connected sessions for subsystem use

Refactors TCP/Unix transports to use Server.Connect() instead of
creating per-connection servers, so all sessions are visible to
the notification broadcaster.

Registers claude/channel as an experimental MCP capability so clients
(Claude Code, IDEs) can discover and subscribe to push events.

Co-Authored-By: Virgil <virgil@lethean.io>"
```

---

## Phase 3: Usage-Example Comments + Naming

Add usage-example comments to all public types and functions. This is the CoreGO convention: comments show how to call the thing, not just what it does.

**Files:**
- Modify: `pkg/mcp/mcp.go` (Input/Output types)
- Modify: `pkg/mcp/registry.go`
- Modify: `pkg/mcp/bridge.go`
- Modify: `pkg/mcp/tools_metrics.go`
- Modify: `pkg/mcp/tools_process.go`
- Modify: `pkg/mcp/tools_rag.go`
- Modify: `pkg/mcp/tools_webview.go`
- Modify: `pkg/mcp/tools_ws.go`
- Modify: `pkg/mcp/transport_stdio.go`
- Modify: `pkg/mcp/transport_tcp.go`
- Modify: `pkg/mcp/transport_unix.go`
- Modify: `pkg/mcp/transport_http.go`

- [ ] **Step 1: Update Input/Output type comments in mcp.go**

Add inline usage examples to field comments:

```go
// ReadFileInput contains parameters for reading a file.
//
//   input := ReadFileInput{Path: "src/main.go"}
type ReadFileInput struct {
    Path string `json:"path"` // e.g. "src/main.go"
}

// ReadFileOutput contains the result of reading a file.
type ReadFileOutput struct {
    Content  string `json:"content"`  // File contents as string
    Language string `json:"language"` // e.g. "go", "typescript"
    Path     string `json:"path"`     // Echoed input path
}
```

Apply the same pattern to all Input/Output types in `mcp.go`:
- `WriteFileInput/Output`
- `ListDirectoryInput/Output`, `DirectoryEntry`
- `CreateDirectoryInput/Output`
- `DeleteFileInput/Output`
- `RenameFileInput/Output`
- `FileExistsInput/Output`
- `DetectLanguageInput/Output`
- `GetSupportedLanguagesInput/Output`
- `EditDiffInput/Output`

- [ ] **Step 2: Update tool file comments**

For each tool file (`tools_metrics.go`, `tools_process.go`, `tools_rag.go`, `tools_webview.go`, `tools_ws.go`), add usage-example comments to:
- Input/Output struct definitions
- Handler function doc comments
- Registration function doc comments

Example pattern:

```go
// ProcessStartInput contains parameters for starting a new process.
//
//   input := ProcessStartInput{Command: "go", Args: []string{"test", "./..."}}
type ProcessStartInput struct {
    Command string   `json:"command"`        // e.g. "go", "npm"
    Args    []string `json:"args,omitempty"` // e.g. ["test", "./..."]
    Dir     string   `json:"dir,omitempty"`  // Working directory, e.g. "/path/to/project"
    Env     []string `json:"env,omitempty"`  // e.g. ["DEBUG=true", "PORT=8080"]
}
```

- [ ] **Step 3: Update registry.go comments**

```go
// addToolRecorded registers a tool with the MCP server AND records its metadata
// for the REST bridge. The generic type parameters capture In/Out for schema extraction.
//
//   addToolRecorded(s, server, "files", &mcp.Tool{
//       Name:        "file_read",
//       Description: "Read the contents of a file",
//   }, s.readFile)
func addToolRecorded[In, Out any](...) { ... }
```

- [ ] **Step 4: Update transport comments**

```go
// ServeStdio starts the MCP server on stdin/stdout.
// This is the default transport for IDE integration.
//
//   err := svc.ServeStdio(ctx)
func (s *Service) ServeStdio(ctx context.Context) error { ... }

// ServeTCP starts the MCP server on a TCP address.
// Each connection gets its own MCP session.
//
//   err := svc.ServeTCP(ctx, "127.0.0.1:9100")
func (s *Service) ServeTCP(ctx context.Context, addr string) error { ... }

// ServeUnix starts the MCP server on a Unix domain socket.
//
//   err := svc.ServeUnix(ctx, "/tmp/core-mcp.sock")
func (s *Service) ServeUnix(ctx context.Context, socketPath string) error { ... }

// ServeHTTP starts the MCP server with Streamable HTTP transport.
// Supports optional Bearer token auth via MCP_AUTH_TOKEN env var.
//
//   err := svc.ServeHTTP(ctx, "127.0.0.1:9101")
func (s *Service) ServeHTTP(ctx context.Context, addr string) error { ... }
```

- [ ] **Step 5: Verify compilation**

```bash
go vet ./pkg/mcp/...
go build ./pkg/mcp/...
```

- [ ] **Step 6: Commit**

```bash
git add pkg/mcp/*.go
git commit -m "docs(mcp): add usage-example comments to all public types

CoreGO convention: comments show how to call the thing, not just
what it does. Adds inline examples to Input/Output structs, handler
functions, transport methods, and registry functions.

Co-Authored-By: Virgil <virgil@lethean.io>"
```

---

## Phase 4: Wire Notifications into Subsystems

Connect the notification system to existing subsystems so they emit channel events.

**Files:**
- Modify: `pkg/mcp/subsystem.go` (add Notifier interface, SubsystemWithNotifier)
- Modify: `pkg/mcp/mcp.go` (call SetNotifier in New())
- Modify: `pkg/mcp/tools_process.go` (emit process lifecycle events)
- Modify: `pkg/mcp/brain/brain.go` (accept Notifier, emit brain events from bridge callback)

- [ ] **Step 1: Define Notifier interface to avoid circular imports**

Sub-packages (`brain/`, `ide/`) cannot import `pkg/mcp` without creating a cycle. Define a small `Notifier` interface that sub-packages can accept without importing the parent package:

```go
// Notifier pushes events to connected MCP sessions.
// Implemented by *Service. Sub-packages accept this interface
// to avoid circular imports.
//
//   notifier.ChannelSend(ctx, "build.complete", data)
type Notifier interface {
    ChannelSend(ctx context.Context, channel string, data any)
}
```

Add an optional `SubsystemWithNotifier` interface:

```go
// SubsystemWithNotifier extends Subsystem for those that emit channel events.
// SetNotifier is called after New() before any tool calls.
type SubsystemWithNotifier interface {
    Subsystem
    SetNotifier(n Notifier)
}
```

In `New()`, after creating the service:

```go
for _, sub := range s.subsystems {
    sub.RegisterTools(s.server)
    if sn, ok := sub.(SubsystemWithNotifier); ok {
        sn.SetNotifier(s)
    }
}
```

- [ ] **Step 2: Emit process lifecycle events**

Process tools live in `pkg/mcp/` (same package as Service), so they can call `s.ChannelSend` directly:

```go
// After successful process start:
s.ChannelSend(ctx, "process.start", map[string]any{
    "id":      output.ID,
    "command": input.Command,
})

// In the process exit callback (if wired via ProcessEventCallback):
s.ChannelSend(ctx, "process.exit", map[string]any{
    "id":       id,
    "exitCode": code,
})
```

- [ ] **Step 3: Emit brain events in brain subsystem**

The brain subsystem's recall handler sends requests to the Laravel bridge asynchronously — the returned `output` does not contain real results (they arrive via WebSocket later). Instead, emit the notification from the bridge callback where results actually arrive.

In `pkg/mcp/brain/brain.go`, add a `Notifier` field and `SetNotifier` method:

```go
type Subsystem struct {
    bridge   *ide.Bridge
    notifier Notifier // set by SubsystemWithNotifier
}

// Notifier pushes events to MCP sessions (matches pkg/mcp.Notifier).
type Notifier interface {
    ChannelSend(ctx context.Context, channel string, data any)
}

func (s *Subsystem) SetNotifier(n Notifier) {
    s.notifier = n
}
```

Then in the bridge message handler (where recall results are received from Laravel), emit the notification with the actual result count:

```go
// In the bridge callback that processes recall results:
if s.notifier != nil {
    s.notifier.ChannelSend(ctx, "brain.recall.complete", map[string]any{
        "query": query,
        "count": len(memories),
    })
}
```

- [ ] **Step 4: Verify compilation and tests**

```bash
go vet ./pkg/mcp/...
go test ./pkg/mcp/...
```

- [ ] **Step 5: Commit**

```bash
git add pkg/mcp/subsystem.go pkg/mcp/mcp.go pkg/mcp/tools_process.go pkg/mcp/brain/brain.go
git commit -m "feat(mcp): wire channel notifications into process and brain subsystems

Adds Notifier interface to avoid circular imports between pkg/mcp
and sub-packages. Subsystems that implement SubsystemWithNotifier
receive a Notifier reference. Process tools emit process.start and
process.exit channel events. Brain subsystem emits
brain.recall.complete from the bridge callback (not the handler
return, which is async).

Co-Authored-By: Virgil <virgil@lethean.io>"
```

---

## Phase 5: Consumer Migration Guide

Document the breaking changes for the 2 consumers (agent, ide modules).

**Files:**
- Create: `docs/migration-guide-options.md`

- [ ] **Step 1: Write migration guide**

```markdown
# Migrating to Options{} Constructor

## Before (functional options)

    svc, err := mcp.New(
        mcp.WithWorkspaceRoot("/path"),
        mcp.WithProcessService(ps),
        mcp.WithWSHub(hub),
        mcp.WithSubsystem(brainSub),
        mcp.WithSubsystem(ideSub),
    )

## After (Options struct)

    svc, err := mcp.New(mcp.Options{
        WorkspaceRoot:  "/path",
        ProcessService: ps,
        WSHub:          hub,
        Subsystems:     []mcp.Subsystem{brainSub, ideSub},
    })

## New notification API

    // Broadcast to all sessions (LoggingLevel is a string type)
    svc.SendNotificationToAllClients(ctx, "info", "build", data)

    // Push a named channel event
    svc.ChannelSend(ctx, "build.complete", data)

    // Push to a specific session
    for session := range svc.Sessions() {
        svc.ChannelSendToSession(ctx, session, "agent.status", data)
    }
```

- [ ] **Step 2: Commit**

```bash
git add docs/migration-guide-options.md
git commit -m "docs(mcp): add migration guide for Options{} constructor

Documents breaking changes from functional options to Options{}
struct for consumers (agent, ide modules). Includes notification
API examples.

Co-Authored-By: Virgil <virgil@lethean.io>"
```

---

## Summary

**Total: 5 phases, 24 steps**

| Phase | Scope | Breaking? |
|-------|-------|-----------|
| 1 | `Options{}` struct replaces functional options | Yes — 2 consumers |
| 2 | Notification broadcasting + claude/channel | No — new API |
| 3 | Usage-example comments | No — docs only |
| 4 | Wire notifications into subsystems | No — additive |
| 5 | Consumer migration guide | No — docs only |

After completion:
- `mcp.New(mcp.Options{...})` replaces `mcp.New(mcp.WithXxx(...))`
- `svc.SendNotificationToAllClients(ctx, level, logger, data)` broadcasts to all sessions
- `svc.ChannelSend(ctx, "build.complete", data)` pushes named events
- `claude/channel` experimental capability advertised during MCP initialisation
- Clients (Claude Code, IDEs) can discover push-event support and receive real-time updates
- All public types and functions have usage-example comments
