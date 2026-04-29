# Agent Notes

This repository builds the MCP surface for the core Go stack. The module
contains command entrypoints under `cmd/`, transport and tool registration in
`pkg/mcp`, optional subsystems under `pkg/mcp/{agentic,brain,ide}`, and local
shim packages under `internal/shims` that adapt core services such as process,
websocket, webview, API, RAG, and metrics.

Work should follow the `dappco.re/go` core wrappers that the rest of this
module uses. File operations, path handling, JSON, buffers, formatted output,
environment access, and assertions should come from core where a wrapper
exists. Tests live beside the source file they exercise, and public behaviour is
documented with sibling `*_example_test.go` files.

The compliance audit is the source of truth for repository shape. Before
handoff, run `GOWORK=off go mod tidy`, `GOWORK=off go vet ./...`,
`GOWORK=off go test -count=1 ./...`, `gofmt -l .`, and the v0.9.0 audit script
from the core Go test suite. A clean result means the module is consistent with
the current core/go consumer contract.
