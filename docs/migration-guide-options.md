# Migrating to `Options{}` for MCP Service Construction

## Before (functional options)

```go
svc, err := mcp.New(
    mcp.WithWorkspaceRoot("/path/to/project"),
    mcp.WithProcessService(processSvc),
    mcp.WithWSHub(hub),
    mcp.WithSubsystem(brainSub),
    mcp.WithSubsystem(ideSub),
)
```

## After (`Options{}` DTO)

```go
svc, err := mcp.New(mcp.Options{
    WorkspaceRoot:  "/path/to/project",
    ProcessService: processSvc,
    WSHub:         hub,
    Subsystems: []mcp.Subsystem{
        brainSub,
        ideSub,
    },
})
```

## Notification helpers

```go
// Broadcast to all MCP sessions.
svc.SendNotificationToAllClients(ctx, "info", "build", map[string]any{
    "event": "build.complete",
    "repo":  "go-io",
})

// Broadcast a named channel event.
svc.ChannelSend(ctx, "build.complete", map[string]any{
    "repo": "go-io",
    "status": "passed",
})

// Send to one session.
for session := range svc.Sessions() {
    svc.ChannelSendToSession(ctx, session, "agent.status", map[string]any{
        "state": "running",
    })
}
```
