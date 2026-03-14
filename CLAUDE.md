# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

Core MCP is a Model Context Protocol implementation in two halves: a **Go binary** (`core-mcp`) that speaks native MCP over stdio/TCP/Unix, and a **PHP Laravel package** (`lthn/mcp`) that adds an HTTP MCP API with auth, quotas, and analytics. Both halves bridge to each other via REST or WebSocket.

Module: `forge.lthn.ai/core/mcp` | Licence: EUPL-1.2

## Build and test commands

### Go

```bash
core build                          # Build binary (./core-mcp)
go build -o core-mcp ./cmd/core-mcp/  # Alternative without core CLI

core go test                        # Run all Go tests
core go test --run TestBridgeToAPI  # Run a single test
core go cov                         # Coverage report
core go cov --open                  # Open HTML coverage in browser
core go qa                          # Format + vet + lint + test
core go qa full                     # Also race detector, vuln scan, security audit
core go fmt                         # gofmt
core go lint                        # golangci-lint
core go vet                         # go vet
```

### PHP (from repo root or `src/php/`)

```bash
composer test                                       # Run all PHP tests (Pest)
composer test -- --filter=SqlQueryValidatorTest      # Single test
composer lint                                       # Laravel Pint (PSR-12)
./vendor/bin/pint --dirty                           # Format only changed files
```

### Running locally

```bash
./core-mcp mcp serve                              # Stdio transport (Claude Code / IDE)
./core-mcp mcp serve --workspace /path/to/project  # Sandbox file ops to directory
MCP_ADDR=127.0.0.1:9100 ./core-mcp mcp serve      # TCP transport
```

## Architecture

### Go server (`pkg/mcp/`)

`mcp.Service` is the central type, configured via functional options (`mcp.With*`). It owns the MCP server, a sandboxed filesystem `Medium`, optional subsystems, and an ordered `[]ToolRecord` that powers the REST bridge.

**Tool registration**: All tools use the generic `addToolRecorded[In, Out]()` function which simultaneously registers the MCP handler, reflects input/output structs into JSON Schemas, and creates a REST handler closure. No per-tool glue code needed.

**Tool groups** (registered in `registerTools()`):
- `files`, `language` — `mcp.go`
- `metrics` — `tools_metrics.go`
- `rag` — `tools_rag.go`
- `process` — `tools_process.go` (requires `WithProcessService`)
- `webview` — `tools_webview.go`
- `ws` — `tools_ws.go` (requires `WithWSHub`)

**Subsystem interface** (`Subsystem` / `SubsystemWithShutdown`): Pluggable tool groups registered via `WithSubsystem`. Three ship with the repo:
- `tools_ml.go` — ML inference subsystem (generate, score, probe, status, backends)
- `pkg/mcp/ide/` — IDE bridge to Laravel backend over WebSocket (chat, build, dashboard tools)
- `pkg/mcp/brain/` — OpenBrain knowledge store proxy (remember, recall, forget, list)

**Transports**: stdio (default), TCP (`MCP_ADDR` env var), Unix socket (`ServeUnix`). TCP binds `127.0.0.1` by default; `0.0.0.0` emits a security warning.

**REST bridge**: `BridgeToAPI` maps each `ToolRecord` to a `POST` endpoint via `api.ToolBridge`. 10 MB body limit.

### PHP package (`src/php/`)

Three namespace roots mapping to the Laravel request lifecycle:

| Namespace | Path | Role |
|-----------|------|------|
| `Core\Front\Mcp` | `src/Front/Mcp/` | Frontage — middleware group, `McpToolHandler` contract, lifecycle events |
| `Core\Mcp` | `src/Mcp/` | Module — service provider, models, services, tools, admin panel |
| `Core\Website\Mcp` | `src/Website/Mcp/` | Website — playground, API explorer, metrics dashboard |

Boot chain: `Core\Front\Mcp\Boot` (auto-discovered) fires `McpRoutesRegistering` / `McpToolsRegistering` → `Core\Mcp\Boot` listens and registers routes, tools, admin views, artisan commands.

Key services (bound as singletons): `ToolRegistry`, `ToolAnalyticsService`, `McpQuotaService`, `CircuitBreaker`, `AuditLogService`, `QueryExecutionService`.

`QueryDatabase` tool has 7-layer SQL security (keyword blocking, pattern detection, whitelist, table blocklist, row limits, timeouts, audit logging).

### Brain-seed utility (`cmd/brain-seed/`)

Bulk-imports MEMORY.md, plan docs, and CLAUDE.md files into OpenBrain via the PHP MCP API. Splits by headings, infers memory type, truncates to 3800 chars.

## Conventions

- **UK English** in all user-facing strings and docs (colour, organisation, centre, normalise)
- **SPDX headers** in Go files: `// SPDX-License-Identifier: EUPL-1.2`
- **`declare(strict_types=1);`** in every PHP file
- **Full type hints** on all PHP parameters and return types
- **Pest syntax** for PHP tests (not PHPUnit)
- **Flux Pro** components in Livewire views (not vanilla Alpine); **Font Awesome** icons (not Heroicons)
- **Conventional commits**: `type(scope): description` — e.g. `feat(mcp): add new tool`
- Go test names use `_Good` / `_Bad` / `_Ugly` suffixes (happy path / error path / edge cases)

## Adding a new Go tool

1. Define `Input` and `Output` structs with `json` tags
2. Write handler: `func (s *Service) myTool(ctx, *mcp.CallToolRequest, Input) (*mcp.CallToolResult, Output, error)`
3. Register in `registerTools()`: `addToolRecorded(s, server, "group", &mcp.Tool{...}, s.myTool)`

## Adding a new Go subsystem

1. Create package under `pkg/mcp/`, implement `Subsystem` (and optionally `SubsystemWithShutdown`)
2. Register: `mcp.New(mcp.WithSubsystem(&mysubsystem.Subsystem{}))`

## Adding a new PHP tool

1. Implement `Core\Front\Mcp\Contracts\McpToolHandler` (`schema()` + `handle()`)
2. Register via the `McpToolsRegistering` lifecycle event

## Key dependencies

| Go module | Role |
|-----------|------|
| `github.com/modelcontextprotocol/go-sdk` | Official MCP Go SDK |
| `forge.lthn.ai/core/go-io` | Filesystem abstraction + sandboxing |
| `forge.lthn.ai/core/go-ml` | ML inference, scoring, probes |
| `forge.lthn.ai/core/go-rag` | Qdrant vector search |
| `forge.lthn.ai/core/go-process` | Process lifecycle management |
| `forge.lthn.ai/core/api` | REST framework + `ToolBridge` |
| `forge.lthn.ai/core/go-ws` | WebSocket hub |

PHP: `lthn/php` (Core framework), Laravel 12, Livewire 3, Flux Pro.

Go workspace: this module is part of `~/Code/go.work`. Requires Go 1.26+, PHP 8.2+.
