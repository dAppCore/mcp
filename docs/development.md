---
title: Development
description: How to build, test, and contribute to the core/mcp repository.
---

# Development

## Prerequisites

- **Go 1.26+** -- the module uses Go 1.26 features (range-over-func
  iterators, `reflect.Type.Fields()`)
- **PHP 8.2+** -- required by the Laravel package
- **Composer** -- for PHP dependency management
- **Core CLI** -- `core build`, `core go test`, etc. (built from
  `forge.lthn.ai/core/cli`)
- **Go workspace** -- this module is part of the workspace at `~/Code/go.work`

## Building

### Go binary

```bash
# From the repo root
core build              # produces ./core-mcp (arm64 by default)

# Or with Go directly
go build -o core-mcp ./cmd/core-mcp/
```

Build configuration lives in `.core/build.yaml`:

```yaml
project:
  name: core-mcp
  binary: core-mcp
```

### PHP package

The PHP code is consumed as a Composer package. There is no standalone build
step. To develop locally, symlink or use a Composer path repository in your
Laravel application:

```json
{
    "repositories": [
        {
            "type": "path",
            "url": "../core/mcp"
        }
    ]
}
```

Then run `composer require lthn/mcp:@dev`.

## Testing

### Go tests

```bash
# Run all tests
core go test

# Run a single test
core go test --run TestBridgeToAPI

# With coverage
core go cov
core go cov --open      # opens HTML report in browser

# Full QA (format + vet + lint + test)
core go qa
core go qa full         # also runs race detector, vuln scan, security audit
```

Test files follow the `_Good`, `_Bad`, `_Ugly` suffix convention:

| Suffix | Meaning |
|--------|---------|
| `_Good` | Happy path -- expected behaviour with valid inputs |
| `_Bad` | Error paths -- expected failures with invalid inputs |
| `_Ugly` | Edge cases -- panics, nil pointers, concurrent access |

Key test files:

| File | What it covers |
|------|----------------|
| `mcp_test.go` | Service creation, workspace sandboxing, file operations |
| `registry_test.go` | Tool recording, schema extraction, REST handler creation |
| `bridge_test.go` | `BridgeToAPI`, JSON error classification, 10 MB body limit |
| `subsystem_test.go` | Subsystem registration and shutdown |
| `transport_tcp_test.go` | TCP transport, loopback default, `0.0.0.0` warning |
| `transport_e2e_test.go` | End-to-end TCP client/server round-trip |
| `tools_metrics_test.go` | Duration parsing, metrics record/query |
| `brain/brain_test.go` | Brain subsystem registration and bridge-nil handling |
| `tools_process_test.go` | Process start/stop/kill/list/output/input |
| `tools_process_ci_test.go` | CI-safe process tests (no external binaries) |
| `tools_rag_test.go` | RAG query/ingest/collections |
| `tools_rag_ci_test.go` | CI-safe RAG tests (no Qdrant required) |
| `tools_webview_test.go` | Webview tool registration and error handling |
| `tools_ws_test.go` | WebSocket start/info tools |
| `iter_test.go` | Iterator helpers (`SubsystemsSeq`, `ToolsSeq`) |
| `integration_test.go` | Cross-subsystem integration |
| `ide/bridge_test.go` | IDE bridge connection, message dispatch |
| `ide/tools_test.go` | IDE tool registration |
| `brain/brain_test.go` | Brain subsystem registration and bridge-nil handling |

### PHP tests

```bash
# From the repo root (or src/php/)
composer test

# Single test
composer test -- --filter=SqlQueryValidatorTest
```

PHP tests use Pest syntax. Key test files:

| File | What it covers |
|------|----------------|
| `SqlQueryValidatorTest.php` | Blocked keywords, injection patterns, whitelist |
| `McpQuotaServiceTest.php` | Quota recording and enforcement |
| `QueryAuditServiceTest.php` | Audit log recording |
| `QueryExecutionServiceTest.php` | Query execution with limits and timeouts |
| `ToolAnalyticsServiceTest.php` | Analytics aggregation |
| `ToolDependencyServiceTest.php` | Dependency validation |
| `ToolVersionServiceTest.php` | Version management |
| `ValidateWorkspaceContextMiddlewareTest.php` | Workspace context validation |
| `WorkspaceContextSecurityTest.php` | Multi-tenant isolation |

## Code style

### Go

- Format with `core go fmt` (uses `gofmt`)
- Lint with `core go lint` (uses `golangci-lint`)
- Vet with `core go vet`
- All three run automatically via `core go qa`

### PHP

- Format with `composer lint` (uses Laravel Pint, PSR-12)
- Format only changed files: `./vendor/bin/pint --dirty`

### General conventions

- **UK English** in all user-facing strings and documentation (colour,
  organisation, centre, normalise, serialise).
- **Strict types** in every PHP file: `declare(strict_types=1);`
- **SPDX headers** in Go files: `// SPDX-License-Identifier: EUPL-1.2`
- **Type hints** on all PHP parameters and return types.
- Conventional commits: `type(scope): description`

## Project structure

```
core/mcp/
+-- .core/
|   +-- build.yaml          # Build configuration
+-- cmd/
|   +-- core-mcp/
|   |   +-- main.go          # Binary entry point
|   +-- mcpcmd/
|   |   +-- cmd_mcp.go       # CLI command registration
|   +-- brain-seed/
|       +-- main.go          # OpenBrain import utility
+-- pkg/
|   +-- mcp/
|       +-- mcp.go           # Service, file tools, Run()
|       +-- registry.go      # ToolRecord, addToolRecorded, schema extraction
|       +-- subsystem.go     # Subsystem interface, Options-based registration
|       +-- bridge.go        # BridgeToAPI (MCP-to-REST adapter)
|       +-- transport_stdio.go
|       +-- transport_tcp.go
|       +-- transport_unix.go
|       +-- tools_metrics.go # Metrics record/query
|       +-- tools_process.go # Process management tools
|       +-- tools_rag.go     # RAG query/ingest/collections
|       +-- tools_webview.go # Chrome DevTools automation
|       +-- tools_ws.go      # WebSocket server tools
|       +-- agentic/
|       +-- brain/
|       |   +-- brain.go     # Brain subsystem
|       |   +-- tools.go     # remember/recall/forget/list tools
|       +-- ide/
|           +-- ide.go       # IDE subsystem
|           +-- config.go    # Config, options, defaults
|           +-- bridge.go    # Laravel WebSocket bridge
|           +-- tools_chat.go
|           +-- tools_build.go
|           +-- tools_dashboard.go
+-- src/
|   +-- php/
|       +-- src/
|       |   +-- Front/Mcp/          # Frontage (middleware group, contracts)
|       |   +-- Mcp/                # Module (services, models, tools, admin)
|       |   +-- Website/Mcp/        # Public pages (playground, explorer)
|       +-- tests/
|       +-- config/
|       +-- routes/
+-- composer.json
+-- go.mod
+-- go.sum
```

## Running locally

### MCP server (stdio, for Claude Code)

Add to your Claude Code MCP configuration:

```json
{
    "mcpServers": {
        "core": {
            "command": "/path/to/core-mcp",
            "args": ["mcp", "serve", "--workspace", "/path/to/project"]
        }
    }
}
```

### MCP server (TCP, for multi-client)

```bash
MCP_ADDR=127.0.0.1:9100 ./core-mcp mcp serve
```

Connect with any JSON-RPC client over TCP. Each line is a complete JSON-RPC
message. Maximum message size is 10 MB.

### PHP development server

Use Laravel Valet or the built-in server:

```bash
cd /path/to/laravel-app
php artisan serve
```

The MCP API is available at the configured domain under the routes registered
by `Core\Mcp\Boot::onMcpRoutes`.

### Brain-seed

```bash
# Preview what would be imported
go run ./cmd/brain-seed -dry-run

# Import with API key
go run ./cmd/brain-seed \
    -api-key YOUR_KEY \
    -api https://lthn.sh/api/v1/mcp \
    -plans \
    -claude-md
```

## Adding a new Go tool

1. Define input and output structs with `json` tags:

```go
type MyToolInput struct {
    Query string `json:"query"`
    Limit int    `json:"limit,omitempty"`
}

type MyToolOutput struct {
    Results []string `json:"results"`
    Total   int      `json:"total"`
}
```

2. Write the handler function:

```go
func (s *Service) myTool(
    ctx context.Context,
    req *mcp.CallToolRequest,
    input MyToolInput,
) (*mcp.CallToolResult, MyToolOutput, error) {
    // Implementation here
    return nil, MyToolOutput{Results: results, Total: len(results)}, nil
}
```

3. Register in `registerTools()`:

```go
addToolRecorded(s, server, "mygroup", &mcp.Tool{
    Name:        "my_tool",
    Description: "Does something useful",
}, s.myTool)
```

The `addToolRecorded` generic function automatically generates JSON Schemas
from the struct tags and creates a REST-compatible handler. No additional
wiring is needed.

## Adding a new Go subsystem

1. Create a new package under `pkg/mcp/`:

```go
package mysubsystem

type Subsystem struct{}

func (s *Subsystem) Name() string { return "mysubsystem" }

func (s *Subsystem) RegisterTools(server *mcp.Server) {
    mcp.AddTool(server, &mcp.Tool{
        Name:        "my_subsystem_tool",
        Description: "...",
    }, s.handler)
}
```

2. Register when creating the service:

```go
mcp.New(mcp.Options{
    Subsystems: []mcp.Subsystem{
        &mysubsystem.Subsystem{},
    },
})
```

## Adding a new PHP tool

1. Create a tool class implementing `McpToolHandler`:

```php
namespace Core\Mcp\Tools;

use Core\Front\Mcp\Contracts\McpToolHandler;
use Core\Front\Mcp\McpContext;

class MyTool implements McpToolHandler
{
    public static function schema(): array
    {
        return [
            'name' => 'my_tool',
            'description' => 'Does something useful',
            'inputSchema' => [
                'type' => 'object',
                'properties' => [
                    'query' => ['type' => 'string'],
                ],
                'required' => ['query'],
            ],
        ];
    }

    public function handle(array $args, McpContext $context): array
    {
        return ['result' => 'done'];
    }
}
```

2. Register via the `McpToolsRegistering` lifecycle event in your module's
   Boot class.

## Contributing

- All changes must pass `core go qa` (Go) and `composer test` (PHP) before
  committing.
- Use conventional commits: `feat(mcp): add new tool`, `fix(mcp): handle nil
  input`, `docs(mcp): update architecture`.
- Include `Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>` when
  pair-programming with Claude.
- Licence: EUPL-1.2. All new files must include the appropriate SPDX header.
