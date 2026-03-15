# Plan CRUD MCP Tools Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add MCP tools for plan lifecycle management — create plans from templates, track phase progress, record checkpoints. Agents can structure their work into plans with phases and tasks, enabling stop/ask/resume workflows.

**Architecture:** Thin HTTP wrappers calling the existing Laravel API at `api.lthn.sh/v1/plans/*`. Same pattern as `brain.NewDirect()` — reads `CORE_BRAIN_KEY` for auth, calls REST endpoints, returns structured results. No local storage needed (the API handles persistence).

**Tech Stack:** Go, MCP SDK, HTTP client, JSON

---

## Existing API Endpoints (Laravel — already built)

```
GET     /v1/plans                                     # List plans
GET     /v1/plans/{slug}                              # Get plan detail
POST    /v1/plans                                     # Create plan
PATCH   /v1/plans/{slug}                              # Update plan
PATCH   /v1/plans/{slug}/phases/{phase}               # Update phase
POST    /v1/plans/{slug}/phases/{phase}/checkpoint    # Add checkpoint
POST    /v1/plans/{slug}/phases/{phase}/tasks/{idx}/toggle  # Toggle task
```

## File Structure

| File | Action | Purpose |
|------|--------|---------|
| `pkg/mcp/agentic/plans.go` | Create | Plan CRUD MCP tools — create, status, list, checkpoint, update |
| `pkg/mcp/agentic/plans_test.go` | Create | Tests with mock HTTP server |

---

## Task 1: Plan MCP Tools

**Files:**
- Create: `pkg/mcp/agentic/plans.go`
- Modify: `pkg/mcp/agentic/prep.go` (register new tools)

- [ ] **Step 1: Write the plan tools**

Create `plans.go` with 4 MCP tools that wrap the Laravel API:

```go
// plans.go
package agentic

// Tool: agentic_plan_create
// Creates a plan from a template slug with variable substitution.
// POST /v1/plans with { template: slug, variables: {}, title: "", activate: bool }

// Tool: agentic_plan_status
// Gets plan detail including phases, tasks, and progress.
// GET /v1/plans/{slug}

// Tool: agentic_plan_list
// Lists all plans, optionally filtered by status.
// GET /v1/plans?status={status}

// Tool: agentic_plan_checkpoint
// Records a checkpoint on a phase (notes, completion status).
// POST /v1/plans/{slug}/phases/{phase}/checkpoint
```

Input/output types:

```go
// PlanCreateInput for agentic_plan_create
type PlanCreateInput struct {
    Template  string            `json:"template"`            // Template slug: bug-fix, code-review, feature, refactor
    Variables map[string]string `json:"variables,omitempty"` // Template variables
    Title     string            `json:"title,omitempty"`     // Override plan title
    Activate  bool              `json:"activate,omitempty"`  // Activate immediately (default: draft)
}

// PlanStatusInput for agentic_plan_status
type PlanStatusInput struct {
    Slug string `json:"slug"` // Plan slug
}

// PlanListInput for agentic_plan_list
type PlanListInput struct {
    Status string `json:"status,omitempty"` // Filter: draft, active, completed, archived
}

// PlanCheckpointInput for agentic_plan_checkpoint
type PlanCheckpointInput struct {
    Slug  string `json:"slug"`            // Plan slug
    Phase int    `json:"phase"`           // Phase order number
    Notes string `json:"notes,omitempty"` // Checkpoint notes
    Done  bool   `json:"done,omitempty"`  // Mark phase as completed
}
```

Each tool calls `s.apiCall()` (reuse the pattern from PrepSubsystem which already has `brainURL`, `brainKey`, and `client`).

- [ ] **Step 2: Implement agentic_plan_create**

```go
func (s *PrepSubsystem) planCreate(ctx context.Context, _ *mcp.CallToolRequest, input PlanCreateInput) (*mcp.CallToolResult, map[string]any, error) {
    if input.Template == "" {
        return nil, nil, fmt.Errorf("template is required")
    }

    body := map[string]any{
        "template":  input.Template,
        "variables": input.Variables,
        "activate":  input.Activate,
    }
    if input.Title != "" {
        body["title"] = input.Title
    }

    result, err := s.planAPICall(ctx, "POST", "/v1/plans", body)
    if err != nil {
        return nil, nil, fmt.Errorf("agentic_plan_create: %w", err)
    }

    return nil, result, nil
}
```

- [ ] **Step 3: Implement agentic_plan_status**

```go
func (s *PrepSubsystem) planStatus(ctx context.Context, _ *mcp.CallToolRequest, input PlanStatusInput) (*mcp.CallToolResult, map[string]any, error) {
    if input.Slug == "" {
        return nil, nil, fmt.Errorf("slug is required")
    }

    result, err := s.planAPICall(ctx, "GET", "/v1/plans/"+input.Slug, nil)
    if err != nil {
        return nil, nil, fmt.Errorf("agentic_plan_status: %w", err)
    }

    return nil, result, nil
}
```

- [ ] **Step 4: Implement agentic_plan_list**

```go
func (s *PrepSubsystem) planList(ctx context.Context, _ *mcp.CallToolRequest, input PlanListInput) (*mcp.CallToolResult, map[string]any, error) {
    path := "/v1/plans"
    if input.Status != "" {
        path += "?status=" + input.Status
    }

    result, err := s.planAPICall(ctx, "GET", path, nil)
    if err != nil {
        return nil, nil, fmt.Errorf("agentic_plan_list: %w", err)
    }

    return nil, result, nil
}
```

- [ ] **Step 5: Implement agentic_plan_checkpoint**

```go
func (s *PrepSubsystem) planCheckpoint(ctx context.Context, _ *mcp.CallToolRequest, input PlanCheckpointInput) (*mcp.CallToolResult, map[string]any, error) {
    if input.Slug == "" {
        return nil, nil, fmt.Errorf("slug is required")
    }

    path := fmt.Sprintf("/v1/plans/%s/phases/%d/checkpoint", input.Slug, input.Phase)
    body := map[string]any{
        "notes": input.Notes,
    }
    if input.Done {
        body["status"] = "completed"
    }

    result, err := s.planAPICall(ctx, "POST", path, body)
    if err != nil {
        return nil, nil, fmt.Errorf("agentic_plan_checkpoint: %w", err)
    }

    return nil, result, nil
}
```

- [ ] **Step 6: Add planAPICall helper**

Reuses `brainURL` and `brainKey` from PrepSubsystem (same API, same auth):

```go
func (s *PrepSubsystem) planAPICall(ctx context.Context, method, path string, body any) (map[string]any, error) {
    if s.brainKey == "" {
        return nil, fmt.Errorf("no API key (set CORE_BRAIN_KEY)")
    }

    var reqBody goio.Reader
    if body != nil {
        data, _ := json.Marshal(body)
        reqBody = bytes.NewReader(data)
    }

    req, _ := http.NewRequestWithContext(ctx, method, s.brainURL+path, reqBody)
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")
    req.Header.Set("Authorization", "Bearer "+s.brainKey)

    resp, err := s.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    respData, _ := goio.ReadAll(resp.Body)
    if resp.StatusCode >= 400 {
        return nil, fmt.Errorf("API %d: %s", resp.StatusCode, string(respData))
    }

    var result map[string]any
    json.Unmarshal(respData, &result)
    return result, nil
}
```

- [ ] **Step 7: Register tools in prep.go**

Add to `RegisterTools()`:

```go
s.registerPlanTools(server)
```

And the registration function:

```go
func (s *PrepSubsystem) registerPlanTools(server *mcp.Server) {
    mcp.AddTool(server, &mcp.Tool{
        Name:        "agentic_plan_create",
        Description: "Create an agent work plan from a template (bug-fix, code-review, feature, refactor). Plans have phases with tasks and checkpoints.",
    }, s.planCreate)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "agentic_plan_status",
        Description: "Get plan detail including phases, tasks, progress, and checkpoints.",
    }, s.planStatus)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "agentic_plan_list",
        Description: "List all plans, optionally filtered by status (draft, active, completed, archived).",
    }, s.planList)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "agentic_plan_checkpoint",
        Description: "Record a checkpoint on a plan phase. Include notes about progress, decisions, or blockers.",
    }, s.planCheckpoint)
}
```

- [ ] **Step 8: Verify compilation**

Run: `go vet ./pkg/mcp/agentic/`
Expected: clean

- [ ] **Step 9: Commit**

```bash
git add pkg/mcp/agentic/plans.go pkg/mcp/agentic/prep.go
git commit -m "feat(agentic): plan CRUD MCP tools — create, status, list, checkpoint

Thin HTTP wrappers calling api.lthn.sh/v1/plans/* REST endpoints.
Same auth pattern as brain tools (CORE_BRAIN_KEY).
Templates: bug-fix, code-review, feature, refactor.

Co-Authored-By: Virgil <virgil@lethean.io>"
```

---

## Summary

**Total: 1 task, 9 steps**

After completion, agents have 4 new MCP tools:
- `agentic_plan_create` — start structured work from a template
- `agentic_plan_status` — check progress
- `agentic_plan_list` — see all plans
- `agentic_plan_checkpoint` — record milestones

Combined with `agentic_status` + `agentic_resume`, this gives the full lifecycle:
1. Create plan → 2. Dispatch agent → 3. Agent works through phases → 4. Agent hits blocker → writes BLOCKED.md → 5. Reviewer answers → 6. Resume → 7. Agent checkpoints completion

Available templates (from PHP system):
- `bug-fix` — diagnose, fix, test, verify
- `code-review` — audit, report, fix, re-check
- `feature` — design, implement, test, document
- `refactor` — analyse, restructure, test, verify
