---
title: Core MCP
description: Model Context Protocol server and tooling for AI agents -- Go binary + Laravel PHP package.
---

# Core MCP

`forge.lthn.ai/core/mcp` provides a complete Model Context Protocol (MCP)
implementation spanning two languages:

- **Go** -- a standalone MCP server binary (`core-mcp`) with file operations,
  ML inference, RAG, process management, webview automation, and WebSocket
  streaming.
- **PHP** -- a Laravel package (`lthn/mcp`) that adds an HTTP MCP API, tool
  registry, SQL query tools, quota enforcement, circuit breakers, audit
  logging, and an admin panel to any Host UK application.

Both halves speak the same protocol and can bridge to one another via REST or
WebSocket.

## Quick start

### Go binary

```bash
# Build
core build            # produces ./core-mcp

# Start on stdio (for Claude Code / IDE integration)
./core-mcp mcp serve

# Start on TCP
MCP_ADDR=127.0.0.1:9100 ./core-mcp mcp serve

# Restrict file operations to a directory
./core-mcp mcp serve --workspace /path/to/project
```

### PHP package

Add the Composer dependency to a Laravel application:

```bash
composer require lthn/mcp
```

The package auto-registers via `Core\Front\Mcp\Boot`. No manual provider
registration is needed. Run migrations, then visit the admin panel at
`/admin/mcp`.

## Package layout

| Path | Language | Purpose |
|------|----------|---------|
| `pkg/mcp/` | Go | Core MCP server: `Service`, transports, tool registry, REST bridge |
| `pkg/mcp/brain/` | Go | OpenBrain knowledge-store subsystem (remember/recall/forget/list) |
| `pkg/mcp/ide/` | Go | IDE subsystem: Laravel WebSocket bridge, chat, builds, dashboard |
| `cmd/core-mcp/` | Go | Binary entry point (`core-mcp`) |
| `cmd/mcpcmd/` | Go | CLI command registration (`mcp serve`) |
| `cmd/brain-seed/` | Go | Utility to import CLAUDE.md / MEMORY.md files into OpenBrain |
| `src/php/src/Mcp/` | PHP | Laravel service provider, models, services, middleware, tools |
| `src/php/src/Front/Mcp/` | PHP | MCP frontage: middleware group, `McpToolHandler` contract, `McpContext` |
| `src/php/src/Website/Mcp/` | PHP | Public-facing MCP pages: playground, API explorer, metrics |

## Dependencies

### Go

| Module | Role |
|--------|------|
| `forge.lthn.ai/core/go` | DI container and service lifecycle |
| `forge.lthn.ai/core/cli` | CLI framework (bubbletea TUI) |
| `forge.lthn.ai/core/go-io` | Filesystem abstraction (`Medium`, sandboxing) |
| `forge.lthn.ai/core/go-log` | Structured logger |
| `forge.lthn.ai/core/go-ai` | AI event metrics |
| `forge.lthn.ai/core/go-ml` | ML inference, scoring, capability probes |
| `forge.lthn.ai/core/go-inference` | Backend registry (Ollama, MLX, ROCm) |
| `forge.lthn.ai/core/go-rag` | Qdrant vector search and document ingestion |
| `forge.lthn.ai/core/go-process` | External process lifecycle management |
| `forge.lthn.ai/core/go-webview` | Chrome DevTools Protocol automation |
| `forge.lthn.ai/core/go-ws` | WebSocket hub and channel messaging |
| `forge.lthn.ai/core/go-api` | REST API framework and `ToolBridge` |
| `github.com/modelcontextprotocol/go-sdk` | Official MCP Go SDK |
| `github.com/gin-gonic/gin` | HTTP router (REST bridge) |
| `github.com/gorilla/websocket` | WebSocket client for IDE bridge |

### PHP

| Package | Role |
|---------|------|
| `lthn/php` (core/php) | Foundation framework: lifecycle events, modules, actions |
| Laravel 12 | Application framework |
| Livewire / Flux Pro | Admin panel components |

## Licence

EUPL-1.2. See the `SPDX-License-Identifier` headers in each source file.
