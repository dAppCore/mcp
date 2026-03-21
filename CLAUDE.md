# CLAUDE.md

Guidance for Claude Code and Codex when working with this repository.

## Module

`forge.lthn.ai/core/mcp` — Model Context Protocol server with file operations, tool registration, notification broadcasting, and channel events.

Licence: EUPL-1.2

## Build & Test

```bash
go test ./pkg/mcp/...            # run all tests
go build ./pkg/mcp/...           # verify compilation
go build ./cmd/core-mcp/         # build binary
```

Or via the Core CLI:

```bash
core go test
core go qa                       # fmt + vet + lint + test
```

## API Shape

Uses `Options{}` struct, not functional options:

```go
svc, err := mcp.New(mcp.Options{
    WorkspaceRoot:  "/path/to/project",
    ProcessService: ps,
    WSHub:          hub,
    Subsystems:     []mcp.Subsystem{brain, agentic, monitor},
})
```

**Do not use:** `WithWorkspaceRoot`, `WithSubsystem`, `WithProcessService`, `WithWSHub` — these no longer exist.

## Notification Broadcasting

```go
// Broadcast to all connected sessions
svc.SendNotificationToAllClients(ctx, "info", "monitor", data)

// Push a named channel event
svc.ChannelSend(ctx, "agent.complete", map[string]any{"repo": "go-io"})

// Push to a specific session
svc.ChannelSendToSession(ctx, session, "build.failed", data)
```

The `claude/channel` experimental capability is registered automatically.

## Tool Groups

| File | Group | Tools |
|------|-------|-------|
| `mcp.go` | files, language | file_read, file_write, file_delete, file_rename, file_exists, file_edit, dir_list, dir_create, lang_detect, lang_list |
| `tools_metrics.go` | metrics | metrics_record, metrics_query |
| `tools_process.go` | process | process_start, process_stop, process_kill, process_list, process_output, process_input |
| `tools_rag.go` | rag | rag_query, rag_ingest, rag_collections |
| `tools_webview.go` | webview | webview_connect, webview_navigate, etc. |
| `tools_ws.go` | ws | ws_start, ws_info |

## Subsystems

| Package | Name | Purpose |
|---------|------|---------|
| `pkg/mcp/brain/` | brain | OpenBrain recall, remember, forget |
| `pkg/mcp/ide/` | ide | IDE bridge to Laravel backend |
| `pkg/mcp/agentic/` | agentic | Dispatch, status, plans, PRs, scans |

## Adding a New Tool

```go
// 1. Define Input/Output structs
type MyInput struct {
    Name string `json:"name"`
}
type MyOutput struct {
    Result string `json:"result"`
}

// 2. Write handler
func (s *Service) myTool(ctx context.Context, req *mcp.CallToolRequest, input MyInput) (*mcp.CallToolResult, MyOutput, error) {
    return nil, MyOutput{Result: "done"}, nil
}

// 3. Register in registerTools()
addToolRecorded(s, server, "group", &mcp.Tool{
    Name:        "my_tool",
    Description: "Does something useful",
}, s.myTool)
```

## Adding a New Subsystem

```go
type MySubsystem struct{}

func (m *MySubsystem) Name() string { return "my-sub" }
func (m *MySubsystem) RegisterTools(server *mcp.Server) {
    // register tools here
}

// Register via Options
svc, err := mcp.New(mcp.Options{
    Subsystems: []mcp.Subsystem{&MySubsystem{}},
})
```

Subsystems that need to push channel events implement `SubsystemWithNotifier`.

## Transports

Selected by `Run()` in priority order:
1. Streamable HTTP (`MCP_HTTP_ADDR` env) — Bearer auth via `MCP_AUTH_TOKEN`
2. TCP (`MCP_ADDR` env)
3. Stdio (default) — used by Claude Code / IDEs

## Test Naming

`_Good` (happy path), `_Bad` (expected errors), `_Ugly` (panics/edge cases).

## Go Workspace

Part of `~/Code/go.work`. Use `GOWORK=off` to test in isolation.
