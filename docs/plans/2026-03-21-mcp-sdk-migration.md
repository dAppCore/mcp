# Migration Plan: Official MCP SDK → mcp-go

**Date:** 2026-03-21
**Status:** Draft
**Motivation:** The official SDK (`github.com/modelcontextprotocol/go-sdk`) has unexported fields that make custom notifications impossible. We need `claude/channel` support (experimental capability) so the server can push events (inbox messages, dispatch completions, webhook events) into a running Claude Code session. `mcp-go` (`github.com/mark3labs/mcp-go`) exposes `SendNotificationToClient` and `SendNotificationToAllClients` plus full session management.

**Breaking change risk:** 2 consumers (`agent`, `ide`) import `forge.lthn.ai/core/mcp` — both expose the `Subsystem` interface which references `*mcp.Server`.

---

## 1. SDK Comparison — Key Differences

### Package layout

| Concept | Official SDK | mcp-go |
|---------|-------------|--------|
| Types import | `github.com/modelcontextprotocol/go-sdk/mcp` | `github.com/mark3labs/mcp-go/mcp` |
| Server import | same package | `github.com/mark3labs/mcp-go/server` |
| JSON-RPC | `github.com/modelcontextprotocol/go-sdk/jsonrpc` | Not exported (internal) |

### Server creation

| | Official SDK | mcp-go |
|-|-------------|--------|
| Constructor | `mcp.NewServer(&mcp.Implementation{Name, Version}, nil)` | `server.NewMCPServer("name", "version", ...options)` |
| Return type | `*mcp.Server` | `*server.MCPServer` |
| Capabilities | Set via `ServerOptions` (2nd arg) | `server.WithToolCapabilities(bool)`, `server.WithRecovery()`, etc. |

### Tool definition

| | Official SDK | mcp-go |
|-|-------------|--------|
| Struct | `&mcp.Tool{Name: "...", Description: "..."}` | `mcp.NewTool("name", mcp.WithDescription("..."), mcp.WithString("param", mcp.Required()), ...)` |
| Schema | Auto-generated from Go struct tags via reflection (our `addToolRecorded` pattern) | Must be declared explicitly with `WithString`/`WithNumber`/`WithBoolean`/`WithObject`/`WithArray` builders, OR supply raw `InputSchema` |

### Tool registration

| | Official SDK | mcp-go |
|-|-------------|--------|
| Function | `mcp.AddTool(server, tool, handler)` — package-level generic function | `s.AddTool(tool, handler)` — method on `*server.MCPServer` |
| Handler signature | `func(ctx, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)` — 3 returns, generic typed input | `func(ctx, mcp.CallToolRequest) (*mcp.CallToolResult, error)` — 2 returns, untyped input |
| Input access | Auto-deserialized into typed `In` struct | `request.RequireString("name")`, `request.GetString("name", default)`, or manual `request.Params.Arguments["key"]` |
| Result helpers | Return `*mcp.CallToolResult` with `Content` slice | `mcp.NewToolResultText("...")`, `mcp.NewToolResultError("...")` |

### Transports

| | Official SDK | mcp-go |
|-|-------------|--------|
| Stdio | `server.Run(ctx, &mcp.StdioTransport{})` | `server.ServeStdio(s)` — top-level function |
| HTTP | `mcp.NewStreamableHTTPHandler(factory, opts)` | `server.NewStreamableHTTPServer(s, opts...)` |
| SSE | N/A | `server.NewSSEServer(s, opts...)` |
| TCP | Custom `Transport`/`Connection` impl | No built-in TCP — must implement or wrap |

### Session & Notifications (the reason for migration)

| | Official SDK | mcp-go |
|-|-------------|--------|
| Session access | Not exposed | `server.ClientSessionFromContext(ctx)`, `server.GetSessionID(ctx)` |
| Server from handler | Not available | `server.ServerFromContext(ctx)` |
| Push to client | Not possible (unexported) | `mcpServer.SendNotificationToClient(ctx, method, params)` |
| Broadcast | Not possible | `mcpServer.SendNotificationToAllClients(method, params)` |
| Session hooks | None | `hooks.AddOnRegisterSession(...)`, `hooks.AddOnUnregisterSession(...)` |
| Per-session tools | None | `s.AddSessionTool(sessionID, tool, handler)`, `s.DeleteSessionTools(...)` |

---

## 2. Architectural Decisions

### 2a. Handler signature adapter

The biggest change: every tool handler must change from 3-return generic to 2-return untyped. Two approaches:

**Option A — Direct rewrite:** Change every handler to `func(ctx, mcp.CallToolRequest) (*mcp.CallToolResult, error)`. Manually unmarshal input from `request.Params.Arguments` and marshal output via `mcp.NewToolResultText(json)`.

**Option B — Adapter pattern (recommended):** Write a generic adapter that preserves the current handler signatures:

```go
// adaptHandler wraps a typed handler for use with mcp-go.
func adaptHandler[In, Out any](h func(ctx context.Context, req mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error)) server.ToolHandler {
    return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        var input In
        data, _ := json.Marshal(req.Params.Arguments)
        if err := json.Unmarshal(data, &input); err != nil {
            return mcp.NewToolResultError("invalid input: " + err.Error()), nil
        }
        result, output, err := h(ctx, req, input)
        if err != nil {
            return nil, err
        }
        if result != nil {
            return result, nil
        }
        // Marshal output to JSON text result
        outJSON, _ := json.Marshal(output)
        return mcp.NewToolResultText(string(outJSON)), nil
    }
}
```

This lets us keep every handler's current signature (including the REST bridge) and migrates only the registration layer. The handler `req` param changes from `*mcp.CallToolRequest` (pointer) to `mcp.CallToolRequest` (value) — a minor type change.

### 2b. Tool schema declaration

Current approach uses `addToolRecorded` with reflection-based `structSchema()`. With mcp-go, two options:

**Option A — Builder API:** Rewrite each tool's schema using `mcp.WithString(...)`, `mcp.WithNumber(...)`, etc. Tedious (40+ tools) but idiomatic.

**Option B — Raw InputSchema (recommended):** `mcp.NewTool` accepts a raw `InputSchema` option. Feed the existing `structSchema()` output directly:

```go
mcp.NewTool("file_read",
    mcp.WithDescription("Read the contents of a file"),
    mcp.WithRawSchema(structSchema(new(ReadFileInput))),
)
```

If `WithRawSchema` doesn't exist, the `mcp.Tool` struct likely has an `InputSchema` field we can set after construction. This preserves the reflection-based schema generation.

### 2c. REST bridge (`addToolRecorded`)

The `addToolRecorded` generic function currently:
1. Calls `mcp.AddTool(server, tool, handler)` — registers with MCP
2. Creates a `RESTHandler` closure — for the REST bridge
3. Reflects `In`/`Out` types for JSON Schema — for API docs

With mcp-go, step 1 changes to `s.AddTool(tool, adaptHandler(h))`. Steps 2 and 3 remain unchanged. The `ToolRecord` struct and `RESTHandler` pattern are internal to our code, not SDK types.

### 2d. TCP transport

The official SDK exposes `mcp.Transport` and `mcp.Connection` interfaces. Our `connTransport` implements these. mcp-go doesn't have equivalent interfaces. Options:

**Option A — Wrap stdio over TCP:** Pipe TCP conn's reader/writer into mcp-go's stdio transport.

**Option B — Use StreamableHTTP:** Replace TCP with HTTP transport (mcp-go has built-in support). This is architecturally cleaner for multi-client scenarios.

**Option C — Implement custom transport:** mcp-go's `server` package may expose transport interfaces. Investigate at implementation time.

### 2e. Subsystem interface

The `Subsystem` interface exposes `RegisterTools(server *mcp.Server)`. This must change to `RegisterTools(server *server.MCPServer)`. This is a **breaking change for consumers** (`agent`, `ide`).

---

## 3. File-by-File Migration Plan

### Phase 1 — Core types and registration (foundation)

#### `go.mod`
- **Remove:** `github.com/modelcontextprotocol/go-sdk v1.4.1`
- **Add:** `github.com/mark3labs/mcp-go v1.x.x` (latest stable)
- **Remove:** `github.com/modelcontextprotocol/go-sdk/jsonrpc` indirect (if present)

#### `pkg/mcp/subsystem.go`
- **Old import:** `github.com/modelcontextprotocol/go-sdk/mcp`
- **New imports:** `github.com/mark3labs/mcp-go/mcp` + `github.com/mark3labs/mcp-go/server`
- **Change:** `RegisterTools(server *mcp.Server)` → `RegisterTools(server *server.MCPServer)`
- **Impact:** Breaking change for `Subsystem` interface consumers (`agent`, `ide` modules)

#### `pkg/mcp/registry.go`
- **Old import:** `github.com/modelcontextprotocol/go-sdk/mcp`
- **New imports:** `github.com/mark3labs/mcp-go/mcp` + `github.com/mark3labs/mcp-go/server`
- **Changes:**
  - `addToolRecorded[In, Out]()` signature: `server *mcp.Server` → `server *server.MCPServer`
  - Replace `mcp.AddTool(server, t, h)` → `server.AddTool(tool, adaptHandler(h))`
  - Tool construction: `&mcp.Tool{Name, Description}` → `mcp.NewTool(name, mcp.WithDescription(desc))` with schema attached
  - `mcp.ToolHandlerFor[In, Out]` type param → custom type alias (mcp-go has no generic handler type)
  - `RESTHandler` closure stays (internal to our code, not SDK)
  - `structSchema()` stays (used for REST bridge schema generation)

#### `pkg/mcp/mcp.go`
- **Old import:** `github.com/modelcontextprotocol/go-sdk/mcp`
- **New imports:** `github.com/mark3labs/mcp-go/mcp` + `github.com/mark3labs/mcp-go/server`
- **Changes:**
  - `Service.server` field: `*mcp.Server` → `*server.MCPServer`
  - `New()`: replace `mcp.NewServer(impl, nil)` → `server.NewMCPServer("core-cli", "0.1.0", server.WithToolCapabilities(true))`
  - `Server()` return type: `*mcp.Server` → `*server.MCPServer`
  - `Run()`: replace `s.server.Run(ctx, &mcp.StdioTransport{})` → `server.ServeStdio(s.server)`
  - `registerTools()` param: `server *mcp.Server` → `server *server.MCPServer`
  - All `addToolRecorded(s, server, group, &mcp.Tool{...}, handler)` calls → updated tool construction
  - All handler signatures: `*mcp.CallToolRequest` → `mcp.CallToolRequest` (pointer → value)

### Phase 2 — Transports

#### `pkg/mcp/transport_stdio.go`
- **Old import:** `github.com/modelcontextprotocol/go-sdk/mcp`
- **New import:** `github.com/mark3labs/mcp-go/server`
- **Change:** `s.server.Run(ctx, &mcp.StdioTransport{})` → `server.ServeStdio(s.server)`
- **Note:** `ServeStdio` is a blocking function, same as `Run`. Context cancellation may need different handling (investigate mcp-go's stdio shutdown).

#### `pkg/mcp/transport_http.go`
- **Old import:** `github.com/modelcontextprotocol/go-sdk/mcp`
- **New import:** `github.com/mark3labs/mcp-go/server`
- **Changes:**
  - Replace `mcp.NewStreamableHTTPHandler(factory, opts)` → `server.NewStreamableHTTPServer(s.server, opts...)`
  - mcp-go's HTTP server may handle auth differently — investigate built-in auth options vs keeping our `withAuth` wrapper
  - `StreamableHTTPOptions{SessionTimeout}` → check mcp-go equivalent options
  - The factory function `func(r *http.Request) *mcp.Server` pattern may not exist — mcp-go likely uses a single server instance

#### `pkg/mcp/transport_tcp.go`
- **Old imports:** `github.com/modelcontextprotocol/go-sdk/jsonrpc` + `github.com/modelcontextprotocol/go-sdk/mcp`
- **New imports:** `github.com/mark3labs/mcp-go/server` (+ potentially mcp-go internals)
- **Changes:**
  - **Critical:** `connTransport` implements `mcp.Transport` interface — no equivalent in mcp-go
  - **Critical:** `connConnection` implements `mcp.Connection` with `Read`/`Write`/`Close`/`SessionID` — no equivalent
  - `handleConnection()` creates per-connection `mcp.NewServer` + registers tools — must use mcp-go equivalent
  - `jsonrpc.DecodeMessage`/`jsonrpc.EncodeMessage` — no public equivalent in mcp-go
  - **Decision needed:** Replace TCP with HTTP transport, OR implement custom transport adapter
  - Per-connection server instances: `mcp.NewServer()` → `server.NewMCPServer()` + re-register tools
  - `mcp.Implementation{Name, Version}` struct literal → string args to `NewMCPServer`

#### `pkg/mcp/transport_unix.go`
- **No direct SDK import** — delegates to `handleConnection()` from `transport_tcp.go`
- **Impact:** Inherits whatever transport approach we choose for TCP
- **No changes** if we keep the `handleConnection()` pattern

### Phase 3 — Tool files (mechanical changes)

All tool files follow the same pattern. For each:
1. Change import from `github.com/modelcontextprotocol/go-sdk/mcp` → `github.com/mark3labs/mcp-go/mcp` + `github.com/mark3labs/mcp-go/server`
2. Change handler param `*mcp.CallToolRequest` → `mcp.CallToolRequest` (pointer → value)
3. Change registration calls from `mcp.AddTool(server, &mcp.Tool{...}, handler)` → `server.AddTool(tool, adaptedHandler)`
4. Change `registerXTools(server *mcp.Server)` → `registerXTools(server *server.MCPServer)`

#### `pkg/mcp/tools_metrics.go`
- `registerMetricsTools(server *mcp.Server)` → `registerMetricsTools(server *server.MCPServer)`
- 2 tool registrations: `metrics_record`, `metrics_query`
- Uses `mcp.AddTool` directly (not `addToolRecorded`) — change to `server.AddTool`

#### `pkg/mcp/tools_process.go`
- `registerProcessTools(server *mcp.Server) bool` → `registerProcessTools(server *server.MCPServer) bool`
- 6 tool registrations: `process_start`, `process_stop`, `process_kill`, `process_list`, `process_output`, `process_input`
- Uses `mcp.AddTool` directly

#### `pkg/mcp/tools_rag.go`
- `registerRAGTools(server *mcp.Server)` → `registerRAGTools(server *server.MCPServer)`
- 3 tool registrations: `rag_query`, `rag_ingest`, `rag_collections`
- Uses `mcp.AddTool` directly

#### `pkg/mcp/tools_webview.go`
- `registerWebviewTools(server *mcp.Server)` → `registerWebviewTools(server *server.MCPServer)`
- 10 tool registrations: `webview_connect` through `webview_wait`
- Uses `mcp.AddTool` directly

#### `pkg/mcp/tools_ws.go`
- `registerWSTools(server *mcp.Server) bool` → `registerWSTools(server *server.MCPServer) bool`
- 2 tool registrations: `ws_start`, `ws_info`
- Uses `mcp.AddTool` directly

### Phase 4 — Subsystem packages

#### `pkg/mcp/ide/ide.go`
- **Old import:** `github.com/modelcontextprotocol/go-sdk/mcp`
- **New imports:** `github.com/mark3labs/mcp-go/mcp` + `github.com/mark3labs/mcp-go/server`
- `RegisterTools(server *mcp.Server)` → `RegisterTools(server *server.MCPServer)`

#### `pkg/mcp/ide/tools_build.go`
- `registerBuildTools(server *mcp.Server)` → `registerBuildTools(server *server.MCPServer)`
- 3 tools, handler signatures change `*mcp.CallToolRequest` → `mcp.CallToolRequest`

#### `pkg/mcp/ide/tools_chat.go`
- `registerChatTools(server *mcp.Server)` → `registerChatTools(server *server.MCPServer)`
- 5 tools, handler signatures change

#### `pkg/mcp/ide/tools_dashboard.go`
- `registerDashboardTools(server *mcp.Server)` → `registerDashboardTools(server *server.MCPServer)`
- 3 tools, handler signatures change

#### `pkg/mcp/brain/brain.go`
- `RegisterTools(server *mcp.Server)` → `RegisterTools(server *server.MCPServer)`

#### `pkg/mcp/brain/tools.go`
- `registerBrainTools(server *mcp.Server)` → `registerBrainTools(server *server.MCPServer)`
- 4 tools: `brain_remember`, `brain_recall`, `brain_forget`, `brain_list`
- Handler signatures change

#### `pkg/mcp/brain/direct.go`
- `RegisterTools(server *mcp.Server)` → `RegisterTools(server *server.MCPServer)`
- 3 tools: `brain_remember`, `brain_recall`, `brain_forget`
- Handler signatures change

### Phase 5 — Agentic subsystem

#### `pkg/mcp/agentic/prep.go`
- `RegisterTools(server *mcp.Server)` → `RegisterTools(server *server.MCPServer)`
- Registers `agentic_prep_workspace` + `agentic_scan` + delegates to sub-registration functions
- Handler signatures change

#### `pkg/mcp/agentic/dispatch.go`
- `registerDispatchTool(server *mcp.Server)` → `registerDispatchTool(server *server.MCPServer)`
- 1 tool: `agentic_dispatch`

#### `pkg/mcp/agentic/status.go`
- `registerStatusTool(server *mcp.Server)` → `registerStatusTool(server *server.MCPServer)`
- 1 tool: `agentic_status`

#### `pkg/mcp/agentic/scan.go`
- Handler signature change only (registration is in `prep.go`)

#### `pkg/mcp/agentic/resume.go`
- `registerResumeTool(server *mcp.Server)` → `registerResumeTool(server *server.MCPServer)`
- 1 tool: `agentic_resume`

#### `pkg/mcp/agentic/plan.go`
- `registerPlanTools(server *mcp.Server)` → `registerPlanTools(server *server.MCPServer)`
- 5 tools: `agentic_plan_create`, `agentic_plan_read`, `agentic_plan_update`, `agentic_plan_delete`, `agentic_plan_list`

#### `pkg/mcp/agentic/pr.go`
- `registerCreatePRTool(server *mcp.Server)` → `registerCreatePRTool(server *server.MCPServer)`
- `registerListPRsTool(server *mcp.Server)` → `registerListPRsTool(server *server.MCPServer)`
- 2 tools: `agentic_create_pr`, `agentic_list_prs`

#### `pkg/mcp/agentic/epic.go`
- `registerEpicTool(server *mcp.Server)` → `registerEpicTool(server *server.MCPServer)`
- 1 tool: `agentic_create_epic`

### Phase 6 — Bridge (no SDK changes)

#### `pkg/mcp/bridge.go`
- **No MCP SDK import** — uses `gin` and `api` only
- **No changes needed** — the REST bridge consumes `ToolRecord` which is our own type

### Phase 7 — Tests

All test files that import the SDK need the same import swap and type changes. Key test files:
- `pkg/mcp/subsystem_test.go` — references `*mcp.Server`
- `pkg/mcp/registry_test.go` — tests `addToolRecorded`
- `pkg/mcp/mcp_test.go` — creates `Service`
- `pkg/mcp/bridge_test.go` — tests REST bridge
- `pkg/mcp/transport_tcp_test.go` — tests TCP transport
- `pkg/mcp/transport_e2e_test.go` — end-to-end transport tests
- `pkg/mcp/tools_*_test.go` — tool handler tests
- `pkg/mcp/ide/bridge_test.go`, `pkg/mcp/ide/tools_test.go`
- `pkg/mcp/brain/brain_test.go`

---

## 4. Breaking Changes & Risks

### No direct equivalent

| Feature | Official SDK | mcp-go | Mitigation |
|---------|-------------|--------|------------|
| Generic typed handlers | `ToolHandlerFor[In, Out]` | None — untyped `ToolHandler` | Write `adaptHandler[In, Out]()` adapter (section 2a) |
| Auto input schema from structs | Via `addToolRecorded` reflection | Must declare or supply raw schema | Keep `structSchema()` + attach via raw schema option (section 2b) |
| TCP transport interfaces | `mcp.Transport`, `mcp.Connection` | Not exposed | Replace TCP with HTTP, or implement adapter (section 2d) |
| JSON-RPC codec | `jsonrpc.DecodeMessage`/`EncodeMessage` | Not exposed | Only needed for TCP — goes away if TCP is replaced |
| Per-connection server instances | `mcp.NewServer()` per TCP conn | Single `MCPServer` with sessions | Use mcp-go's session model (section 2d) |
| `*mcp.CallToolRequest` (pointer) | Used in all handlers | `mcp.CallToolRequest` (value) | Mechanical change in all handler signatures |

### Consumer impact

The `Subsystem` interface change (`*mcp.Server` → `*server.MCPServer`) breaks:
- `agent` module — must update its subsystem implementations
- `ide` module — must update its subsystem implementations

**Mitigation:** Coordinate the migration. Update `forge.lthn.ai/core/mcp` first, then update consumers to match.

### New capabilities unlocked

After migration, the following become possible:
- `server.ServerFromContext(ctx)` — access server from any tool handler
- `SendNotificationToClient(ctx, "claude/channel", payload)` — push events to Claude Code
- `SendNotificationToAllClients("claude/channel", payload)` — broadcast to all sessions
- Session hooks for connection tracking and cleanup
- Per-session tool registration (different tools for different clients)

---

## 5. Migration Order (Recommended)

1. **Phase 1:** Core types (`subsystem.go`, `registry.go`, `mcp.go`) — establishes the foundation
2. **Phase 2:** Transports (`transport_stdio.go`, `transport_http.go`, `transport_tcp.go`) — the riskiest phase
3. **Phase 3:** Tool files (mechanical, low risk) — `tools_metrics.go`, `tools_process.go`, `tools_rag.go`, `tools_webview.go`, `tools_ws.go`
4. **Phase 4:** IDE subsystem (`ide/`)
5. **Phase 5:** Brain subsystem (`brain/`)
6. **Phase 6:** Agentic subsystem (`agentic/`)
7. **Phase 7:** Tests — update in parallel with each phase
8. **Phase 8:** Consumer modules (`agent`, `ide`) — update after core module is published

Each phase should be a separate commit. Build must pass after each phase.

---

## 6. Estimated Scope

| Category | Files | Tools |
|----------|-------|-------|
| Core (mcp.go, registry.go, subsystem.go) | 3 | — |
| Transports | 4 | — |
| Tool files (pkg/mcp/) | 5 | 23 tools |
| IDE subsystem | 4 | 11 tools |
| Brain subsystem | 3 | 7 tools |
| Agentic subsystem | 8 | 14 tools |
| Tests | ~12 | — |
| **Total** | **~39 files** | **55 tools** |

---

## 7. Checklist

- [ ] Verify `mcp-go` latest version supports `InputSchema` raw attachment
- [ ] Confirm `mcp-go`'s `CallToolRequest` field layout matches our assumptions
- [ ] Investigate mcp-go's stdio shutdown/context cancellation behaviour
- [ ] Decide: TCP transport → HTTP replacement or custom adapter
- [ ] Investigate mcp-go's HTTP server auth options (keep `withAuth` or use built-in)
- [ ] Write `adaptHandler[In, Out]()` generic adapter
- [ ] Write tool schema attachment helper (raw JSON Schema → `mcp.NewTool`)
- [ ] Update `addToolRecorded` to use new registration API
- [ ] Migrate all 55 tool registrations
- [ ] Update all handler signatures (`*mcp.CallToolRequest` → `mcp.CallToolRequest`)
- [ ] Update `Subsystem` interface + all implementations
- [ ] Update consumer modules (`agent`, `ide`)
- [ ] Run full test suite
- [ ] Add notification support (the whole point of the migration)
