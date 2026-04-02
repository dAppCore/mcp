---
title: Architecture
description: Internals of the Go MCP server and PHP Laravel package -- types, data flow, subsystems, and security model.
---

# Architecture

Core MCP is split into two cooperating halves: a Go binary that speaks the
native MCP protocol over stdio/TCP/Unix sockets, and a PHP Laravel package
that exposes an HTTP MCP API with multi-tenant auth, quotas, and analytics.

## Go server (`pkg/mcp/`)

### Service

`mcp.Service` is the central type. It owns an `mcp.Server` from the official
Go SDK, a sandboxed filesystem `Medium`, optional subsystems, and an ordered
slice of `ToolRecord` metadata that powers the REST bridge.

```go
svc, err := mcp.New(
    mcp.Options{
        WorkspaceRoot:  "/home/user/project",
        ProcessService: processService,
        WSHub:         wsHub,
        Subsystems: []mcp.Subsystem{
            &MySubsystem{},
        },
    },
)
```

Options are provided via the `mcp.Options` DTO. `WorkspaceRoot` creates a
sandboxed `io.Medium` that confines all file operations to a single directory
tree. Passing an empty string disables sandboxing (not recommended for
untrusted clients).

### Tool registration

Every tool is registered through a single generic function:

```go
func addToolRecorded[In, Out any](
    s *Service,
    server *mcp.Server,
    group string,
    t *mcp.Tool,
    h mcp.ToolHandlerFor[In, Out],
)
```

This function does three things in one call:

1. Registers the handler with the MCP server for native protocol calls.
2. Reflects on the `In` and `Out` type parameters to build JSON Schemas.
3. Creates a `RESTHandler` closure that unmarshals raw JSON into the concrete
   `In` type, calls the handler, and returns the `Out` value -- enabling the
   REST bridge without any per-tool glue code.

The resulting `ToolRecord` structs are stored in `Service.tools` and exposed
via `Tools()` and `ToolsSeq()` (the latter returns a Go 1.23 `iter.Seq`).

### Built-in tool groups

The service registers the following tools at startup:

| Group | Tools | Source |
|-------|-------|--------|
| **files** | `file_read`, `file_write`, `file_delete`, `file_rename`, `file_exists`, `file_edit`, `dir_list`, `dir_create` | `mcp.go` |
| **language** | `lang_detect`, `lang_list` | `mcp.go` |
| **metrics** | `metrics_record`, `metrics_query` | `tools_metrics.go` |
| **rag** | `rag_query`, `rag_ingest`, `rag_collections` | `tools_rag.go` |
| **process** | `process_start`, `process_stop`, `process_kill`, `process_list`, `process_output`, `process_input` | `tools_process.go` |
| **webview** | `webview_connect`, `webview_disconnect`, `webview_navigate`, `webview_click`, `webview_type`, `webview_query`, `webview_console`, `webview_eval`, `webview_screenshot`, `webview_wait` | `tools_webview.go` |
| **ws** | `ws_start`, `ws_info` | `tools_ws.go` |

Process and WebSocket tools are conditionally registered -- they require
`ProcessService` and `WSHub` in `Options` respectively.

### Subsystem interface

Additional tool groups are plugged in via the `Subsystem` interface:

```go
type Subsystem interface {
    Name() string
    RegisterTools(server *mcp.Server)
}
```

Subsystems that need teardown implement `SubsystemWithShutdown`:

```go
type SubsystemWithShutdown interface {
    Subsystem
    Shutdown(ctx context.Context) error
}
```

Three subsystems ship with this repo:

#### Agentic subsystem (`pkg/mcp/agentic/`)

`agentic` tools prepare workspaces, dispatch agents, and track execution
status for issue-driven task workflows.

#### Brain subsystem (`pkg/mcp/brain/`)

Proxies OpenBrain knowledge-store operations to the Laravel backend via the IDE
bridge. Four tools:

- `brain_remember` -- store a memory (decision, observation, bug, etc.).
- `brain_recall` -- semantic search across stored memories.
- `brain_forget` -- permanently delete a memory.
- `brain_list` -- list memories with filtering (no vector search).

#### IDE subsystem (`pkg/mcp/ide/`)

Bridges the desktop IDE to a Laravel `core-agentic` backend over WebSocket.
Registers tools in three groups:

- **Chat**: `ide_chat_send`, `ide_chat_history`, `ide_session_list`,
  `ide_session_create`, `ide_plan_status`
- **Build**: `ide_build_status`, `ide_build_list`, `ide_build_logs`
- **Dashboard**: `ide_dashboard_overview`, `ide_dashboard_activity`,
  `ide_dashboard_metrics`

The IDE bridge (`Bridge`) maintains a persistent WebSocket connection to the
Laravel backend with exponential-backoff reconnection. Messages are forwarded
from Laravel to a local `ws.Hub` for real-time streaming to the IDE frontend.

### Transports

The Go server supports three transports, all using line-delimited JSON-RPC:

| Transport | Activation | Default address |
|-----------|-----------|-----------------|
| **Stdio** | No `MCP_ADDR` env var | stdin/stdout |
| **TCP** | `MCP_ADDR=host:port` | `127.0.0.1:9100` |
| **Unix** | `ServeUnix(ctx, path)` | caller-specified socket path |

TCP binds to `127.0.0.1` by default when the host component is empty. Binding
to `0.0.0.0` emits a security warning. Each accepted connection becomes a
session on the shared `mcp.Server`.

### REST bridge

`BridgeToAPI` populates a `go-api.ToolBridge` from the recorded tool
metadata. Each tool becomes a `POST` endpoint that:

1. Reads and size-limits the JSON body (10 MB max).
2. Calls the tool's `RESTHandler` (which deserialises to the correct input
   type).
3. Wraps the result in a standard `api.Response` envelope.

JSON parse errors return 400; all other errors return 500. This allows any
MCP tool to be called over plain HTTP without additional code.

### Data flow (Go)

```
AI Client (Claude Code, IDE)
  |
  | JSON-RPC over stdio / TCP / Unix
  v
mcp.Server  (go-sdk)
  |
  | typed handler dispatch
  v
Service.readFile / Service.writeFile / ...
  |
  | sandboxed I/O
  v
io.Medium  (go-io)

--- or via REST ---

HTTP Client
  |
  | POST /api/tools/{name}
  v
gin.Router -> BridgeToAPI -> RESTHandler -> typed handler
```

---

## PHP package (`src/php/`)

### Namespace structure

The PHP side is split into three namespace roots, each serving a different
stage of the Laravel request lifecycle:

| Namespace | Path | Purpose |
|-----------|------|---------|
| `Core\Front\Mcp` | `src/Front/Mcp/` | **Frontage** -- defines the `mcp` middleware group, fires `McpRoutesRegistering` and `McpToolsRegistering` lifecycle events |
| `Core\Mcp` | `src/Mcp/` | **Module** -- service provider, models, services, middleware, tools, admin panel, migrations |
| `Core\Website\Mcp` | `src/Website/Mcp/` | **Website** -- public-facing Livewire pages (playground, API explorer, metrics dashboard) |

### Boot sequence

1. **`Core\Front\Mcp\Boot`** (auto-discovered via `composer.json` extra) --
   registers the `mcp` middleware group with throttling and route-model
   binding, then fires `McpRoutesRegistering` and `McpToolsRegistering`
   lifecycle events.

2. **`Core\Mcp\Boot`** listens to those events via the `$listens` array:
   - `McpRoutesRegistering` -- registers MCP API routes under the configured
     domain with `mcp.auth` middleware.
   - `McpToolsRegistering` -- hook for other modules to register tool
     handlers.
   - `AdminPanelBooting` -- loads admin views and Livewire components.
   - `ConsoleBooting` -- registers artisan commands.

3. Services are bound as singletons: `ToolRegistry`, `ToolAnalyticsService`,
   `McpQuotaService`, `ToolDependencyService`, `AuditLogService`,
   `ToolVersionService`, `QueryAuditService`, `QueryExecutionService`.

### HTTP API

The `McpApiController` exposes five endpoints behind `mcp.auth` middleware:

| Method | Path | Handler |
|--------|------|---------|
| `GET` | `/servers.json` | List all MCP servers from YAML registry |
| `GET` | `/servers/{id}.json` | Server details with tool definitions |
| `GET` | `/servers/{id}/tools` | List tools for a server |
| `POST` | `/tools/call` | Execute a tool |
| `GET` | `/resources/{uri}` | Read a resource |

`POST /tools/call` accepts:

```json
{
    "server":    "hosthub-agent",
    "tool":      "brain_remember",
    "arguments": { "content": "...", "type": "observation" }
}
```

The controller validates arguments against the tool's JSON Schema, executes
via the `AgentToolRegistry`, logs the call, records quota usage, and
dispatches webhooks.

### Authentication

`McpApiKeyAuth` middleware extracts an API key from either:

- `Authorization: Bearer hk_xxx_yyy`
- `X-API-Key: hk_xxx_yyy`

It checks expiry, per-server access scopes, and records usage. The resolved
`ApiKey` model is attached to the request for downstream use.

### McpToolHandler contract

Tool handlers implement `Core\Front\Mcp\Contracts\McpToolHandler`:

```php
interface McpToolHandler
{
    public static function schema(): array;
    public function handle(array $args, McpContext $context): array;
}
```

`McpContext` abstracts the transport layer (stdio vs HTTP), providing:
session tracking, plan context, notification sending, and session logging.

### Tool registry

`Core\Mcp\Services\ToolRegistry` loads server and tool definitions from
YAML files in `resources/mcp/`. It provides:

- Server discovery (`getServers()`)
- Tool listing with optional version info (`getToolsForServer()`)
- Category grouping and search (`getToolsByCategory()`, `searchTools()`)
- Example input generation from JSON Schema
- 5-minute cache with manual invalidation

### SQL security

`QueryDatabase` is the most security-hardened tool. It implements seven
layers of defence:

1. **Keyword blocking** -- `INSERT`, `UPDATE`, `DELETE`, `DROP`, `ALTER`,
   `GRANT`, `SET`, and 20+ other dangerous keywords are rejected outright.
2. **Dangerous pattern detection** -- stacked queries, UNION injection,
   hex encoding, `SLEEP()`, `BENCHMARK()`, comment obfuscation, and
   `INFORMATION_SCHEMA` access are blocked before comment stripping.
3. **Whitelist matching** -- only queries matching predefined regex patterns
   (simple SELECT, COUNT, explicit column lists) are allowed.
4. **Blocked table list** -- configurable list of tables that cannot appear
   in FROM or JOIN clauses.
5. **Tier-based row limits** -- results are truncated with a warning when
   they exceed the configured limit.
6. **Query timeout** -- per-query time limit prevents runaway queries.
7. **Audit logging** -- every query attempt (allowed, blocked, or errored)
   is recorded with workspace, user, IP, and session context.

### Circuit breaker

`CircuitBreaker` provides fault tolerance for external service dependencies.
It implements the standard three-state pattern:

- **Closed** -- requests pass through normally; failures are counted.
- **Open** -- requests fail fast; a configurable timeout triggers transition
  to half-open.
- **Half-Open** -- a single trial request is allowed (with a lock to prevent
  concurrent trials); success closes the circuit, failure re-opens it.

Configuration is per-service via `config('mcp.circuit_breaker.{service}.*')`.

### Metrics and analytics

`McpMetricsService` provides dashboard data from `McpToolCallStat` aggregate
records:

- Overview stats with period-over-period trend comparison
- Daily call trends for charting
- Top tools by call count
- Per-tool performance percentiles (p50, p95, p99)
- Hourly distribution heatmap
- Error breakdown by tool and error code
- Plan activity tracking

`ToolAnalyticsService` and `ToolVersionService` handle deeper per-tool
analytics and schema versioning respectively.

### Quota management

`McpQuotaService` enforces per-workspace usage limits tracked in the
`mcp_usage_quotas` table. The `CheckMcpQuota` middleware blocks requests
when the quota is exhausted.

### Audit logging

`AuditLogService` records tamper-evident audit logs in the `mcp_audit_logs`
table. The `VerifyAuditLogCommand` artisan command checks log integrity.

### Admin panel

The module registers nine Livewire components for the admin panel:

- **ApiKeyManager** -- create, revoke, and scope API keys
- **Playground / McpPlayground** -- interactive tool testing
- **RequestLog** -- full request/response replay
- **ToolAnalyticsDashboard / ToolAnalyticsDetail** -- visual metrics
- **QuotaUsage** -- per-workspace quota status
- **AuditLogViewer** -- searchable audit trail
- **ToolVersionManager** -- schema versioning and deprecation

### Data flow (PHP)

```
AI Agent / External Client
  |
  | POST /tools/call  (Bearer hk_xxx_yyy)
  v
McpApiKeyAuth middleware
  |
  | auth + scope check
  v
CheckMcpQuota middleware
  |
  | quota enforcement
  v
McpApiController::callTool()
  |
  | schema validation
  v
AgentToolRegistry::execute()
  |
  | permission + dependency check
  v
Tool handler (e.g. QueryDatabase, brain_remember)
  |
  | result
  v
Log (McpToolCall, McpApiRequest, AuditLog, Webhook)
```

---

## Brain-seed utility (`cmd/brain-seed/`)

A standalone Go program that bulk-imports knowledge into OpenBrain via the
PHP MCP HTTP API. It discovers three sources:

1. **MEMORY.md** files from `~/.claude/projects/*/memory/`
2. **Plan documents** from `~/Code/*/docs/plans/`
3. **CLAUDE.md** files from `~/Code/` (up to 4 levels deep)

Each markdown file is split by headings into sections. Each section becomes a
`brain_remember` API call with inferred type (architecture, convention,
decision, bug, plan, research, observation), project tag, and confidence
level. Content is truncated to 3,800 characters to fit within embedding model
limits.

```bash
# Dry run (preview without storing)
go run ./cmd/brain-seed -dry-run

# Import memories
go run ./cmd/brain-seed -api-key YOUR_KEY

# Also import plans and CLAUDE.md files
go run ./cmd/brain-seed -api-key YOUR_KEY -plans -claude-md
```
