# Exceptions

Items from the Codex review that cannot be fixed, with reasons.

## 6. Compile-time interface assertions in subsystem packages

**Files:** `brain/brain.go`, `brain/direct.go`, `agentic/prep.go`, `ide/ide.go`

**Finding:** Add `var _ Subsystem = (*T)(nil)` compile-time assertions.

**Reason:** The `Subsystem` interface is defined in the parent `mcp` package. Subsystem packages (`brain`, `agentic`, `ide`) cannot import `mcp` because `mcp` already imports them via `Options.Subsystems` — this would create a circular import. The interface conformance is enforced at runtime when `RegisterTools` is called during `mcp.New()`.

## 7. Compile-time Notifier assertion on Service

**Finding:** Add `var _ Notifier = (*Service)(nil)`.

**Resolution:** Fixed — assertion added to `pkg/mcp/subsystem.go` (where the `Notifier` interface is defined). The TODO originally claimed this was already done in commit `907d62a`, but it was not present in the codebase.
